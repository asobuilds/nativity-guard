# AI frontend research and backend handoff

Status: frontend preparation only. AI requests are disabled unless `VITE_ENABLE_AI_ASSISTANT=true` is explicitly set at build time. This switch is a release control, **not** an authorization boundary.

## Basis for the design

- `AGENT.md` says unit access is not case access; facts need source, confidence, type and timestamp, and AI inference must not be represented as verified fact.
- NIST's [Generative AI Profile](https://nvlpubs.nist.gov/nistpubs/ai/NIST.AI.600-1.pdf) and [AI RMF](https://nvlpubs.nist.gov/nistpubs/ai/nist.ai.100-1.pdf) call for context-specific evaluation, measurement and human oversight. We apply that by keeping generated advice advisory and requiring review before a public warning.
- OWASP's [2025 LLM risks](https://genai.owasp.org/llm-top-10/) include [prompt injection](https://genai.owasp.org/llmrisk/llm01-prompt-injection/) and sensitive-information disclosure. Case text and user prompts are untrusted inputs; a friendly UI warning cannot replace server enforcement.

## Frontend prepared on this branch

- Authenticated `/assistant` screen in role navigation, with general question and safety-tip forms, accessible error states, and an Emergency SOS link.
- `POST /ai/chatbot` sends `{ question }` and reads `{ response }`; `POST /ai/smart-tips` sends `{ location }` and reads `{ tips }`. These match current Go handlers. Input is length-limited in the browser, but server limits remain necessary.
- Output is plain text labelled **AI suggestion**; there is no automatic action, claim of verified facts, stored chat history, inferred risk score, or predictive map overlay. The browser keeps exchanges only in page memory.
- Disabled by default even when the page is reachable. There is no fake mock answer; mock mode alone does not enable it.

## Backend work before enabling the switch

1. **Blocker: enforce case authorization.** Review `/ai/analyze-location`, `/ai/map-insights`, `/ai/security-warning`, `/ai/predict-hotspots` and `/ai/smart-tips` for case access. Current location/map queries use unit filters only for selected roles; a missing unit ID can remove that filter. Explicit warning `incidentIds` bypass unit filtering. Apply the canonical per-case policy from `AGENT.md` before constructing prompts; never send unauthorized case fields to OpenRouter. Test citizens, assigned/support/unassigned officers, unit admins and cross-unit IDs.
2. **Blocker: define data sharing and privacy.** Decide which user questions and case facts may be sent to OpenRouter, retention terms, permitted fields, redaction, logging, and user-facing disclosure. The frontend cannot enforce these rules. Avoid sending exact location or sensitive evidence by default.
3. **Blocker: harden AI requests.** Limit text and result sizes on the server; rate limit each AI route per user; cap provider cost and duration; choose a controlled model/version, and return stable generic errors without provider details. Check missing `OPENROUTER_API_KEY` in the release environment without printing its value.
4. **Contract: mark generated material.** For advisory text, return structured metadata such as `generatedAt`, `model`, `sourceType`, `sourceIds`, and `reviewStatus` where applicable. The precise fields must be agreed with the backend teammate before frontend consumption. Never mark a model's unsupported risk level as measured evidence.
5. **Human publication gate.** Security warnings and case summaries require explicit authorized human review, an audit trail and a separate publish action. Keep `security-warning`, `predict-hotspots`, `map-insights`, `analyze-location`, and `analyze-news` out of this frontend release.
6. **Evidence quality.** Location analysis must select geographically and temporally relevant, permitted cases; map insight must actually use the requested map bounds. Evaluate usefulness, false alarms and failure cases on representative Nigerian scenarios before calling outputs predictions.
7. **Endpoint clarity.** `/ai/analyze-image` currently accepts only a text description. Either rename the user-facing capability or implement real image input with its own privacy and permission review. `GenerateLocationRisk` is a deterministic count rule with a simulated delay, not a trained AI model. `AIAnalysis` is migrated but not populated by the AI handlers.

## Joint integration check

With a test account and a real backend, verify authorized chat and tips responses, 400 on empty/oversized input, 401 on an expired session, 429 under rate limit, 503 or safe error on provider failure, and that no prompt, private case detail or provider error leaks to other users or logs. Check keyboard/screen-reader use and mobile layout. Only then opt in to `VITE_ENABLE_AI_ASSISTANT=true` for a controlled deployment. No production AI behavior has been verified by this document.
