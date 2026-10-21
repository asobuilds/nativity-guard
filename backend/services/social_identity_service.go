package services

import (
 "context"
 "crypto/rsa"
 "encoding/base64"
 "encoding/json"
 "errors"
 "io"
 "math/big"
 "net/http"
 "os"
 "strings"
 "sync"
 "time"

 "github.com/golang-jwt/jwt/v5"
)

type VerifiedSocialIdentity struct {
 Provider, Subject, Email, GivenName, FamilyName string
}

type identityClaims struct {
 Email string `json:"email"`
 EmailVerified json.RawMessage `json:"email_verified"`
 GivenName string `json:"given_name"`
 FamilyName string `json:"family_name"`
 Nonce string `json:"nonce"`
 jwt.RegisteredClaims
}

func verifiedEmail(raw json.RawMessage) bool {
 return string(raw) == "true" || string(raw) == `"true"`
}

type jwkList struct {
 Keys []struct { Kid string `json:"kid"`; Kty string `json:"kty"`; Alg string `json:"alg"`; Use string `json:"use"`; N string `json:"n"`; E string `json:"e"` } `json:"keys"`
}

var socialKeyCache = struct {
 sync.Mutex
 entries map[string]struct{ keys map[string]*rsa.PublicKey; expires time.Time }
}{entries: map[string]struct{ keys map[string]*rsa.PublicKey; expires time.Time }{}}

func identityPublicKey(ctx context.Context, provider, kid string) (*rsa.PublicKey, error) {
 source := map[string]string{
  "google": "https://www.googleapis.com/oauth2/v3/certs",
  "apple": "https://appleid.apple.com/auth/keys",
 }[provider]
 if source == "" || kid == "" { return nil, errors.New("invalid provider key") }
 socialKeyCache.Lock()
 cached, exists := socialKeyCache.entries[provider]
 if exists && time.Now().Before(cached.expires) && cached.keys[kid] != nil {
  key := cached.keys[kid]; socialKeyCache.Unlock(); return key,nil
 }
 socialKeyCache.Unlock()
 ctx, cancel := context.WithTimeout(ctx, 6*time.Second); defer cancel()
 req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
 if err != nil { return nil,err }
 res, err := http.DefaultClient.Do(req)
 if err != nil { return nil,errors.New("identity provider unavailable") }
 defer res.Body.Close()
 if res.StatusCode != http.StatusOK { return nil,errors.New("identity provider unavailable") }
 var document jwkList
 if err := json.NewDecoder(io.LimitReader(res.Body, 1024*1024)).Decode(&document); err != nil {
  return nil,errors.New("invalid provider signing keys")
 }
 keys := make(map[string]*rsa.PublicKey)
 for _, item := range document.Keys {
  if item.Kty != "RSA" || (item.Alg != "" && item.Alg != "RS256") ||
    (item.Use != "" && item.Use != "sig") { continue }
  nb, nerr := base64.RawURLEncoding.DecodeString(item.N)
  eb, eerr := base64.RawURLEncoding.DecodeString(item.E)
  if nerr != nil || eerr != nil || len(nb) < 256 || len(eb) == 0 || len(eb) > 4 { continue }
  exponent := big.NewInt(0).SetBytes(eb).Int64()
  if exponent < 3 || exponent > 65537 { continue }
  keys[item.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: int(exponent)}
 }
 socialKeyCache.Lock()
 socialKeyCache.entries[provider] = struct{ keys map[string]*rsa.PublicKey; expires time.Time }{
  keys: keys, expires: time.Now().Add(10*time.Minute),
 }
 key := keys[kid]
 socialKeyCache.Unlock()
 if key == nil { return nil,errors.New("unknown provider signing key") }
 return key,nil
}

func VerifySocialIDToken(ctx context.Context, provider, rawToken, expectedNonce string) (*VerifiedSocialIdentity,error) {
 if len(rawToken) > 16000 || len(rawToken) < 100 || expectedNonce == "" {
  return nil,errors.New("invalid identity proof")
 }
 var client, issuer string
 switch provider {
 case "google": client = os.Getenv("GOOGLE_CLIENT_ID"); issuer = "https://accounts.google.com"
 case "apple": client = os.Getenv("APPLE_SERVICE_ID"); issuer = "https://appleid.apple.com"
 default: return nil,errors.New("unsupported identity provider")
 }
 if client == "" { return nil,errors.New("identity provider not configured") }
 claims := &identityClaims{}
 token,err := jwt.ParseWithClaims(rawToken,claims,func(token *jwt.Token)(interface{},error){
  kid,ok := token.Header["kid"].(string); if !ok { return nil,errors.New("missing signing key") }
  return identityPublicKey(ctx,provider,kid)
 },jwt.WithValidMethods([]string{"RS256"}),jwt.WithAudience(client),jwt.WithExpirationRequired(),jwt.WithLeeway(30*time.Second))
 if err != nil || token == nil || !token.Valid { return nil,errors.New("identity token verification failed") }
 if provider == "google" {
  if claims.Issuer != issuer && claims.Issuer != "accounts.google.com" {
   return nil,errors.New("invalid identity issuer")
  }
 } else if claims.Issuer != issuer { return nil,errors.New("invalid identity issuer") }
 if claims.Nonce != expectedNonce { return nil,errors.New("sign-in challenge does not match") }
 if !verifiedEmail(claims.EmailVerified) || strings.TrimSpace(claims.Email) == "" || claims.Subject == "" {
  return nil,errors.New("provider did not verify an email address")
 }
 if !strings.Contains(claims.Email, "@") || strings.ContainsAny(claims.Email,"\r\n") {
  return nil, errors.New("invalid verified email")
 }
 return &VerifiedSocialIdentity{
  Provider:provider,Subject:claims.Subject,Email:strings.ToLower(strings.TrimSpace(claims.Email)),
  GivenName:strings.TrimSpace(claims.GivenName),FamilyName:strings.TrimSpace(claims.FamilyName),
 },nil
}

