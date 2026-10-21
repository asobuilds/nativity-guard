package handlers

import (
 "bytes"
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "testing"

 "github.com/gin-gonic/gin"
)
func TestSocialPhoneValidation(t *testing.T) {
 for _,s:=range []string{"+2348012345678","08012345678"} {
  if !socialPhonePattern.MatchString(s) {t.Fatalf("expected valid number: %s",s)}
 }
 for _,s:=range []string{"oauth-pending","abc","+1","","<script>"} {
  if socialPhonePattern.MatchString(s){t.Fatalf("unsafe phone accepted: %s",s)}
 }
}
func TestUnconfiguredSocialProviderIsClosed(t *testing.T) {
 t.Setenv("GOOGLE_CLIENT_ID","")
 t.Setenv("APPLE_SERVICE_ID","")
 t.Setenv("APPLE_REDIRECT_URI","")
 gin.SetMode(gin.TestMode)
 r:=gin.New()
 r.POST("/start",BeginSocialLogin)
 body,_:=json.Marshal(map[string]string{"provider":"google"})
 req:=httptest.NewRequest(http.MethodPost,"/start",bytes.NewReader(body))
 req.Header.Set("Content-Type","application/json")
 w:=httptest.NewRecorder()
 r.ServeHTTP(w,req)
 if w.Code!=http.StatusServiceUnavailable {t.Fatalf("wanted 503, got %d",w.Code)}
}
func TestCompleteSocialProfileRequiresAuth(t *testing.T) {
 gin.SetMode(gin.TestMode)
 r:=gin.New();r.POST("/complete",CompleteSocialProfile)
 req:=httptest.NewRequest(http.MethodPost,"/complete",bytes.NewBufferString("{}"))
 req.Header.Set("Content-Type","application/json")
 w:=httptest.NewRecorder();r.ServeHTTP(w,req)
 if w.Code!=http.StatusUnauthorized{t.Fatalf("expected 401, got %d",w.Code)}
}
