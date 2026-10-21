package services

import (
 "context"
 "encoding/json"
 "testing"
)

func TestSocialVerifiedEmailClaim(t *testing.T) {
 if !verifiedEmail(json.RawMessage("true")) || !verifiedEmail(json.RawMessage(`"true"`)) {
  t.Fatal("verified provider emails should pass")
 }
 for _, raw := range []string{"false",`"false"`,"null",""} {
  if verifiedEmail(json.RawMessage(raw)) { t.Fatalf("unverified email accepted: %s",raw) }
 }
}
func TestRejectUnconfiguredAndUnknownSocialIdentity(t *testing.T) {
 if _,err:=VerifySocialIDToken(context.Background(),"untrusted","fake","nonce");err==nil {
  t.Fatal("unrecognized provider must be rejected")
 }
}
