package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// AIService is the single entry point for every LLM call on the platform.
//
// Two providers are supported, both speaking the OpenAI chat-completions
// protocol. The service tries the preferred provider first, and falls back
// to the other on a 5xx, timeout, or network error — never on a 4xx, which
// would mean the request itself is wrong and repeating it will not help.
type AIService struct {
	primary   providerConfig
	secondary *providerConfig
	client    *http.Client
}

type providerConfig struct {
	name    string
	apiKey  string
	baseURL string
	model   string
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	Temperature float64       `json:"temperature,omitempty"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// LastStatus records the outcome of the most recent Chat() call, for the
// /ai/status endpoint.
type LastStatus struct {
	Provider  string
	Model     string
	LatencyMs int64
	At        time.Time
	Err       string
}

var lastStatus LastStatus

// NewAIService resolves both provider configurations from the environment
// and returns a client with a preferred provider and an optional fallback.
func NewAIService() *AIService {
	groq := providerConfig{
		name:    "groq",
		apiKey:  strings.TrimSpace(os.Getenv("GROQ_API_KEY")),
		baseURL: "https://api.groq.com/openai/v1",
		model:   defaultModel("groq", strings.TrimSpace(os.Getenv("LLM_MODEL"))),
	}
	openrouter := providerConfig{
		name:    "openrouter",
		apiKey:  strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")),
		baseURL: "https://openrouter.ai/api/v1",
		model:   defaultModel("openrouter", strings.TrimSpace(os.Getenv("LLM_MODEL_OPENROUTER"))),
	}

	preferred := strings.ToLower(strings.TrimSpace(os.Getenv("LLM_PROVIDER")))

	var primary providerConfig
	var secondary *providerConfig

	hasKey := func(p providerConfig) bool { return p.apiKey != "" }

	switch {
	case preferred == "openrouter" && hasKey(openrouter):
		primary = openrouter
		if hasKey(groq) {
			secondary = &groq
		}
	case preferred == "groq" && hasKey(groq):
		primary = groq
		if hasKey(openrouter) {
			secondary = &openrouter
		}
	case hasKey(groq):
		primary = groq
		if hasKey(openrouter) {
			secondary = &openrouter
		}
	case hasKey(openrouter):
		primary = openrouter
	default:
		primary = providerConfig{name: "unconfigured"}
	}

	return &AIService{
		primary:   primary,
		secondary: secondary,
		client: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

func defaultModel(provider, override string) string {
	if override != "" {
		return override
	}
	switch provider {
	case "groq":
		return "openai/gpt-oss-120b"
	case "openrouter":
		return "openrouter/auto"
	}
	return ""
}

// ProviderName returns the currently preferred provider, for /ai/status.
func (s *AIService) ProviderName() string {
	if s.primary.name == "" {
		return "unconfigured"
	}
	return s.primary.name
}

// Model returns the model the preferred provider is configured with.
func (s *AIService) Model() string {
	return s.primary.model
}

// HasFallback reports whether a secondary provider is configured.
func (s *AIService) HasFallback() bool {
	return s.secondary != nil && s.secondary.apiKey != ""
}

// Status returns the last recorded call status, for /ai/status.
func Status() LastStatus { return lastStatus }

// Chat sends a chat completion request. Tries the primary provider, then
// the fallback on transient failures. Never retries on 4xx — a bad request
// will stay bad.
func (s *AIService) Chat(messages []ChatMessage) (string, error) {
	if s.primary.apiKey == "" {
		return "", fmt.Errorf("no LLM provider configured: set GROQ_API_KEY or OPENROUTER_API_KEY")
	}

	out, err := s.call(s.primary, messages)
	if err == nil {
		return out, nil
	}

	// 4xx errors mean the request is wrong; retrying with the same payload
	// against a different provider will not fix it.
	if isClientError(err) {
		return "", err
	}

	if s.secondary == nil || s.secondary.apiKey == "" {
		return "", err
	}

	// Try the fallback. If the fallback also fails, return the fallback's
	// error — that reflects the current live failure more accurately.
	return s.call(*s.secondary, messages)
}

type httpStatusError struct {
	status int
	body   string
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("llm %d: %s", e.status, e.body)
}

func isClientError(err error) bool {
	hse, ok := err.(*httpStatusError)
	if !ok {
		return false
	}
	return hse.status >= 400 && hse.status < 500
}

func (s *AIService) call(p providerConfig, messages []ChatMessage) (string, error) {
	startedAt := time.Now()

	body, _, err := s.doRequest(p, messages)
	elapsed := time.Since(startedAt).Milliseconds()

	if err != nil {
		lastStatus = LastStatus{
			Provider:  p.name,
			Model:     p.model,
			LatencyMs: elapsed,
			At:        time.Now(),
			Err:       err.Error(),
		}
		return "", err
	}

	lastStatus = LastStatus{
		Provider:  p.name,
		Model:     p.model,
		LatencyMs: elapsed,
		At:        time.Now(),
	}
	return body, nil
}

func (s *AIService) doRequest(p providerConfig, messages []ChatMessage) (string, int, error) {
	request := ChatRequest{
		Model:       p.model,
		Messages:    messages,
		Stream:      false,
		Temperature: 0.4,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", 0, err
	}

	req, err := http.NewRequest("POST", p.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	if p.name == "openrouter" {
		req.Header.Set("HTTP-Referer", "https://nativityguard.app")
		req.Header.Set("X-Title", "Nativity Guard")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet := strings.TrimSpace(string(raw))
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		var probe ChatResponse
		if json.Unmarshal(raw, &probe) == nil && probe.Error != nil {
			snippet = probe.Error.Message
		}
		return "", resp.StatusCode, &httpStatusError{status: resp.StatusCode, body: snippet}
	}

	var parsed ChatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", resp.StatusCode, err
	}
	if parsed.Error != nil {
		return "", resp.StatusCode, &httpStatusError{status: resp.StatusCode, body: parsed.Error.Message}
	}
	if len(parsed.Choices) == 0 {
		return "", resp.StatusCode, fmt.Errorf("empty response from provider")
	}
	return parsed.Choices[0].Message.Content, resp.StatusCode, nil
}

// Chatbot — general safety assistant. callerContext (optional) is prepended
// to the system prompt so replies can be specific to the caller's area.
func (s *AIService) Chatbot(question, userRole, callerContext string) (string, error) {
	system := `You are Nativity Guard AI, a helpful security assistant for Nigerian communities.
Your role is to:
1. Provide safety tips and security advice
2. Help users report incidents
3. Explain how the platform works
4. Give general security information
5. Be friendly and culturally aware

Answer in short, clear paragraphs. Never invent case numbers or statistics that are not in the context. If you don't know something, say so and suggest contacting their local security unit.`
	if callerContext != "" {
		system += "\n\nContext about the person asking:\n" + callerContext
	}

	messages := []ChatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: fmt.Sprintf("User role: %s\nQuestion: %s", userRole, question)},
	}
	return s.Chat(messages)
}

// AnalyzeImage — forensic image description review.
func (s *AIService) AnalyzeImage(imageDescription string) (string, error) {
	return s.Chat([]ChatMessage{
		{Role: "system", Content: `You are a forensic image analyst for Nativity Guard.
Analyze the image description and provide:
1. Key observations
2. Potential evidence identification
3. Safety implications
4. Recommended actions`},
		{Role: "user", Content: "Image description: " + imageDescription},
	})
}

// AnalyzeLocationRisk analyzes security risk for a specific location.
func (s *AIService) AnalyzeLocationRisk(latitude, longitude float64, locationName, recentIncidents string) (string, error) {
	return s.Chat([]ChatMessage{
		{Role: "system", Content: `You are a security intelligence analyst for Nativity Guard in Nigeria.
Analyze location security risks and provide actionable insights. Base the analysis only on the incidents provided; do not invent figures.`},
		{Role: "user", Content: fmt.Sprintf(`Analyze security risk for this location:
Location: %s (Lat: %f, Lng: %f)
Recent Incidents: %s

Provide:
1. Risk Level (Low/Medium/High/Extreme)
2. Key Risk Factors
3. Safety Recommendations
4. Emergency Contacts (if known)`, locationName, latitude, longitude, recentIncidents)},
	})
}

// AnalyzeNewsSentiment analyzes news articles for security sentiment.
func (s *AIService) AnalyzeNewsSentiment(newsContent string) (string, error) {
	return s.Chat([]ChatMessage{
		{Role: "system", Content: `You are a security news analyst for Nativity Guard.
Analyze news content for security implications. Focus on: threat level, affected areas, community impact. Be concise.`},
		{Role: "user", Content: "Analyze this news for security implications:\n" + newsContent + `

Provide:
1. Sentiment (Positive/Neutral/Negative)
2. Threat Level
3. Affected Locations
4. Key Risks Identified`},
	})
}

// GenerateSecurityWarning generates a security warning based on incidents.
func (s *AIService) GenerateSecurityWarning(incidentsData string) (string, error) {
	return s.Chat([]ChatMessage{
		{Role: "system", Content: `You are a security warning system for Nativity Guard.
Generate clear, actionable security warnings for communities. Be specific, practical, and culturally appropriate for Nigeria.`},
		{Role: "user", Content: "Based on these recent incidents, generate a security warning:\n" + incidentsData + `

Format:
[WARNING TYPE]
[Location/Area]
[Description]
[Recommended Actions]`},
	})
}

// AnalyzeIncidentPatterns analyzes incident patterns by location.
func (s *AIService) AnalyzeIncidentPatterns(location string, incidents []string) (string, error) {
	text := ""
	for i, inc := range incidents {
		text += fmt.Sprintf("%d. %s\n", i+1, inc)
	}
	return s.Chat([]ChatMessage{
		{Role: "system", Content: `You are a security pattern analyst for Nativity Guard.
Identify patterns, hotspots, and emerging threats. Base every claim on the incidents provided.`},
		{Role: "user", Content: fmt.Sprintf(`Analyze incident patterns in %s:
%s

Provide:
1. Pattern Summary
2. Hotspots Identified
3. Time Patterns
4. Recommendations`, location, text)},
	})
}

// GetSmartSafetyTips provides context-aware safety tips.
func (s *AIService) GetSmartSafetyTips(location, userRole, timeOfDay, recentThreats string) (string, error) {
	return s.Chat([]ChatMessage{
		{Role: "system", Content: `You are a safety advisor for Nativity Guard.
Provide personalized, context-aware safety tips. Be practical and specific.`},
		{Role: "user", Content: fmt.Sprintf(`Provide safety tips for:
Location: %s
Role: %s
Time: %s
Recent Threats: %s

Give 5 specific, actionable tips.`, location, userRole, timeOfDay, recentThreats)},
	})
}

// PredictRiskHotspots predicts potential risk hotspots.
func (s *AIService) PredictRiskHotspots(historicalData string) (string, error) {
	return s.Chat([]ChatMessage{
		{Role: "system", Content: `You are a predictive security analyst for Nativity Guard.
Analyze historical data to predict risk hotspots. Be explicit about uncertainty — do not overstate confidence from limited data.`},
		{Role: "user", Content: "Based on this historical incident data, predict risk hotspots:\n" + historicalData + `

Provide:
1. High-Risk Areas
2. Time Periods
3. Types of Incidents
4. Preventive Measures`},
	})
}

// SummarizeCase provides a concise case summary.
func (s *AIService) SummarizeCase(caseData string) (string, error) {
	return s.Chat([]ChatMessage{
		{Role: "system", Content: `You are a security case summarizer. Create concise, informative summaries.
Focus on key facts, status, and critical action items.`},
		{Role: "user", Content: "Summarize this case concisely:\n" + caseData},
	})
}