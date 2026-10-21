package handlers

import (
 "crypto/rand"
 "crypto/sha256"
 "encoding/base64"
 "encoding/hex"
 "errors"
 "net/http"
 "os"
 "regexp"
 "strings"
 "time"

 "github.com/gin-gonic/gin"
 "github.com/google/uuid"
 "golang.org/x/crypto/bcrypt"
 "gorm.io/gorm"

 "security-solution/config"
 "security-solution/content"
 "security-solution/models"
 "security-solution/services"
)

const socialChallengeDuration = 5 * time.Minute

var socialPhonePattern = regexp.MustCompile(`^\+?[0-9][0-9 -]{6,22}[0-9]$`)

func socialEnabled(provider string) bool {
 switch provider {
 case "google": return strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")) != ""
 case "apple": return strings.TrimSpace(os.Getenv("APPLE_SERVICE_ID")) != "" &&
    strings.TrimSpace(os.Getenv("APPLE_REDIRECT_URI")) != ""
 }
 return false
}

// SocialProviderConfig exposes public client identifiers only, never provider secrets.
func SocialProviderConfig(c *gin.Context) {
 google, apple := socialEnabled("google"), socialEnabled("apple")
 c.JSON(http.StatusOK, gin.H{
  "googleClientId": func() string { if google { return os.Getenv("GOOGLE_CLIENT_ID") }; return "" }(),
  "appleServiceId": func() string { if apple { return os.Getenv("APPLE_SERVICE_ID") }; return "" }(),
  "appleRedirectUri": func() string { if apple { return os.Getenv("APPLE_REDIRECT_URI") }; return "" }(),
 })
}

// BeginSocialLogin issues a random, one-use nonce before opening the provider
// popup. The provider's signed ID token must contain exactly this nonce.
func BeginSocialLogin(c *gin.Context) {
 var input struct{ Provider string `json:"provider" binding:"required"` }
 if c.ShouldBindJSON(&input) != nil || !socialEnabled(input.Provider) {
  c.JSON(http.StatusServiceUnavailable,gin.H{"error":"this sign-in method is not configured"})
  return
 }
 nonceBytes := make([]byte,32)
 if _,err:=rand.Read(nonceBytes); err!=nil {
  c.JSON(http.StatusServiceUnavailable,gin.H{"error":"sign-in temporarily unavailable"});return
 }
 nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)
 digest := sha256.Sum256([]byte(nonce))
 now := time.Now().UTC()
 record := models.LoginChallenge{
  Provider: input.Provider, NonceHash:hex.EncodeToString(digest[:]),
  ExpiresAt: now.Add(socialChallengeDuration),
 }
 if err:=config.DB.Create(&record).Error; err!=nil {
  c.JSON(http.StatusServiceUnavailable,gin.H{"error":"could not start sign-in"});return
 }
 c.Header("Cache-Control","no-store")
 c.JSON(http.StatusOK,gin.H{"nonce":nonce,"expiresInSeconds":300})
}

func consumeSocialChallenge(tx *gorm.DB, provider, nonce string) error {
 digest:=sha256.Sum256([]byte(nonce))
 now:=time.Now().UTC()
 result:=tx.Model(&models.LoginChallenge{}).
  Where("nonce_hash = ? AND provider = ? AND used_at IS NULL AND expires_at > ?",
   hex.EncodeToString(digest[:]),provider,now).
  Update("used_at",now)
 if result.Error!=nil{return result.Error}
 if result.RowsAffected!=1{return errors.New("sign-in attempt expired or already used")}
 return nil
}

func issueSocialSession(c *gin.Context,user *models.User) {
 authSvc:=services.NewAuthService()
 token,jti,err:=authSvc.GenerateJWTWithJTI(user)
 if err!=nil {c.JSON(500,gin.H{"error":"could not create session"});return}
 expiresAt:=time.Now().UTC().Add(24*time.Hour)
 session,err:=services.NewSessionService().Create(user.ID,jti,"",
  c.GetHeader("User-Agent"),c.ClientIP(),expiresAt)
 if err!=nil {c.JSON(500,gin.H{"error":"could not create session"});return}
 refresh,_,err:=services.NewRefreshTokenService().Issue(user.ID,&session.ID)
 if err!=nil {c.JSON(500,gin.H{"error":"could not create session"});return}
 c.Header("Cache-Control","no-store")
 c.JSON(http.StatusOK,gin.H{"token":token,"refreshToken":refresh,
  "user":gin.H{"id":user.ID,"email":user.Email,"firstName":user.FirstName,
   "lastName":user.LastName,"role":user.Role,"accountAccessLevel":user.AccountAccessLevel}})
}

// FinishSocialLogin verifies the provider signature, audience, issuer, expiry,
// email verification and nonce, then maps the immutable provider subject to an
// existing account. Identical emails do not silently link to password accounts.
func FinishSocialLogin(c *gin.Context) {
 var input struct{
  Provider string `json:"provider" binding:"required"`
  IDToken string `json:"idToken" binding:"required"`
  Nonce string `json:"nonce" binding:"required"`
  DateOfBirth string `json:"dateOfBirth"`
  TermsVersion string `json:"termsVersion"`
  ConsentTerms bool `json:"consentTerms"`
  ConsentDataProcessing bool `json:"consentDataProcessing"`
 }
 if c.ShouldBindJSON(&input)!=nil || !socialEnabled(input.Provider) {
  c.JSON(400,gin.H{"error":"invalid social sign-in request"});return
 }
 if len(input.Nonce)<30 || len(input.Nonce)>120 {
  c.JSON(400,gin.H{"error":"invalid sign-in challenge"});return
 }
 verified,err:=services.VerifySocialIDToken(c.Request.Context(),input.Provider,input.IDToken,input.Nonce)
 if err!=nil {c.JSON(http.StatusUnauthorized,gin.H{"error":"identity verification failed; restart sign-in"});return}
 // New accounts must satisfy the existing registration age policy BEFORE
 // any user row or session is created. Returning provider subjects do not
 // need to submit their date of birth again.
 var knownIdentity models.SocialIdentity
 lookup := config.DB.Where("provider = ? AND subject = ?", input.Provider, verified.Subject).First(&knownIdentity)
 if lookup.Error != nil && !errors.Is(lookup.Error, gorm.ErrRecordNotFound) {
  c.JSON(http.StatusServiceUnavailable,gin.H{"error":"sign-in temporarily unavailable"});return
 }
 var eligibleDOB time.Time
 var eligibleMinorStatus string
 var termsDocument, privacyDocument *models.TermsDocument
 if errors.Is(lookup.Error, gorm.ErrRecordNotFound) {
  if strings.TrimSpace(input.DateOfBirth)=="" {
   c.JSON(http.StatusPreconditionRequired,gin.H{"error":"date_of_birth_required"})
   return
  }
  eligibleDOB,eligibleMinorStatus,err = validateDOB(input.DateOfBirth)
  if err!=nil {c.JSON(http.StatusBadRequest,gin.H{"error":err.Error()});return}
  if !input.ConsentTerms || !input.ConsentDataProcessing {
   c.JSON(http.StatusBadRequest,gin.H{"error":"terms_consent_required"});return
  }
  if input.TermsVersion!=content.TermsVersion {
   c.JSON(http.StatusConflict,gin.H{"error":"terms_version_stale"});return
  }
  var docs []models.TermsDocument
  if err:=config.DB.Where("version = ? AND is_active = ?",content.TermsVersion,true).
   Where("(kind = ? AND role = ?) OR (kind = ? AND role = ?)",
    "terms","citizen","privacy","all").Find(&docs).Error;err!=nil {
   c.JSON(http.StatusServiceUnavailable,gin.H{"error":"terms_unavailable"});return
  }
  for i:=range docs {
   if docs[i].Kind=="terms" && docs[i].Role=="citizen" {termsDocument=&docs[i]}
   if docs[i].Kind=="privacy" && docs[i].Role=="all" {privacyDocument=&docs[i]}
  }
  if termsDocument==nil || privacyDocument==nil {
   c.JSON(http.StatusServiceUnavailable,gin.H{"error":"terms_unavailable"});return
  }
 }
 var user models.User
 err=config.DB.Transaction(func(tx *gorm.DB)error{
  if err:=consumeSocialChallenge(tx,input.Provider,input.Nonce);err!=nil{return err}
  var identity models.SocialIdentity
  match:=tx.Where("provider = ? AND subject = ?",input.Provider,verified.Subject).First(&identity)
  if match.Error==nil {return tx.First(&user,"id = ?",identity.UserID).Error}
  if !errors.Is(match.Error,gorm.ErrRecordNotFound){return match.Error}
  // A matching email may belong to an unrelated local account. Require a
  // separate authenticated linking flow instead of takeover by email alone.
  var existing int64
  if err:=tx.Model(&models.User{}).Where("LOWER(email) = ?",verified.Email).Count(&existing).Error;err!=nil{return err}
  if existing>0 {return errSocialEmailExists}
  id:=uuid.New()
  secret:=make([]byte,32)
  if _,err:=rand.Read(secret);err!=nil{return err}
  hashed,err:=bcrypt.GenerateFromPassword([]byte(base64.RawURLEncoding.EncodeToString(secret)),bcrypt.DefaultCost)
  if err!=nil{return err}
  first,last:=verified.GivenName,verified.FamilyName
  if first==""{first="Community"}
  if last==""{last="Member"}
  user=models.User{
   ID:id,Email:verified.Email,Phone:"pending-"+id.String(),
   FirstName:first,LastName:last,Password:string(hashed),
   Role:"citizen",Status:"pending",MinorStatus:eligibleMinorStatus,DateOfBirth:&eligibleDOB,DOBVerified:false,
   AccountAccessLevel:"limited",
  }
  if err:=tx.Create(&user).Error;err!=nil{return err}
  identity=models.SocialIdentity{UserID:id,Provider:input.Provider,Subject:verified.Subject}
  if err:=tx.Create(&identity).Error;err!=nil{return err}
  // All acceptance rows and the limited account are committed atomically.
  now:=time.Now().UTC()
  ip, agent := clampStr(c.ClientIP(),64),clampStr(c.Request.UserAgent(),400)
  for _,doc:=range []*models.TermsDocument{termsDocument,privacyDocument} {
   row:=models.TermsAcceptance{UserID:id,DocumentID:doc.ID,
    DocumentKind:doc.Kind,DocumentRole:doc.Role,DocumentVersion:doc.Version,
    AcceptedAt:now,IPAddress:ip,UserAgent:agent}
   if err:=tx.Create(&row).Error;err!=nil{return err}
  }
  return nil
 })
 if errors.Is(err,errSocialEmailExists) {
  c.JSON(http.StatusConflict,gin.H{"error":"An account already uses this email. Sign in with your existing method; accounts are not linked automatically."});return
 }
 if err!=nil {c.JSON(http.StatusUnauthorized,gin.H{"error":"sign-in could not be completed; please restart"});return}
 issueSocialSession(c,&user)
}

var errSocialEmailExists=errors.New("social email exists")

// CompleteSocialProfile upgrades only a signed-in limited citizen. This is the
// server-side gate: frontend hiding is never sufficient to authorize SOS or reports.
func CompleteSocialProfile(c *gin.Context) {
 raw,_:=c.Get("user")
 user,ok:=raw.(*models.User)
 if !ok {c.JSON(http.StatusUnauthorized,gin.H{"error":"authentication required"});return}
 if user.AccountAccessLevel!="limited" || user.Role!="citizen" {
  c.JSON(http.StatusConflict,gin.H{"error":"this account does not need social setup"});return
 }
 var input struct{
  Phone string `json:"phone" binding:"required"`
  FirstName string `json:"firstName" binding:"required"`
  LastName string `json:"lastName" binding:"required"`
  DateOfBirth string `json:"dateOfBirth"`
  TermsVersion string `json:"termsVersion" binding:"required"`
  ConsentTerms bool `json:"consentTerms"`
  ConsentDataProcessing bool `json:"consentDataProcessing"`
 }
 if c.ShouldBindJSON(&input)!=nil{c.JSON(400,gin.H{"error":"name, phone and current terms are required"});return}
 input.Phone=strings.TrimSpace(input.Phone)
 if !socialPhonePattern.MatchString(input.Phone) ||
   strings.TrimSpace(input.FirstName)=="" || strings.TrimSpace(input.LastName)=="" ||
   len(strings.TrimSpace(input.FirstName))>100 ||
   len(strings.TrimSpace(input.LastName))>100 {
  c.JSON(400,gin.H{"error":"enter a valid contact phone and name"});return
 }
 if !input.ConsentTerms || !input.ConsentDataProcessing {
  c.JSON(400,gin.H{"error":"terms_consent_required"});return
 }
 if input.TermsVersion!=content.TermsVersion {
  c.JSON(http.StatusConflict,gin.H{"error":"terms_version_stale"});return
 }
 // Age eligibility was checked at first social registration, but the
 // self-declared birth date is NOT independently verified. Never set
 // DOBVerified without the separately audited admin verification flow.
 if user.DateOfBirth==nil || (user.MinorStatus!="minor" && user.MinorStatus!="adult") {
  c.JSON(http.StatusForbidden,gin.H{"error":"age_verification_required"});return
 }
 var docs []models.TermsDocument
 if err:=config.DB.Where("version = ? AND is_active = ?",content.TermsVersion,true).
  Where("(kind = ? AND role = ?) OR (kind = ? AND role = ?)",
   "terms","citizen","privacy","all").Find(&docs).Error;err!=nil {
  c.JSON(503,gin.H{"error":"terms_unavailable"});return
 }
 var term,privacy *models.TermsDocument
 for i:=range docs {if docs[i].Kind=="terms" && docs[i].Role=="citizen"{term=&docs[i]}
  if docs[i].Kind=="privacy" && docs[i].Role=="all"{privacy=&docs[i]}}
 if term==nil||privacy==nil{c.JSON(503,gin.H{"error":"terms_unavailable"});return}
 err:=config.DB.Transaction(func(tx *gorm.DB)error{
  var other int64
  if err:=tx.Model(&models.User{}).Where("phone = ? AND id <> ?",input.Phone,user.ID).Count(&other).Error;err!=nil{return err}
  if other>0{return errSocialPhoneExists}
  result:=tx.Model(&models.User{}).
   Where("id = ? AND account_access_level = ?",user.ID,"limited").
   Updates(map[string]interface{}{
    "phone":input.Phone,"first_name":strings.TrimSpace(input.FirstName),
    "last_name":strings.TrimSpace(input.LastName),"account_access_level":"full",
   })
  if result.Error!=nil{return result.Error}
  if result.RowsAffected!=1{return errors.New("account setup already changed")}
  now:=time.Now().UTC();ip:=clampStr(c.ClientIP(),64);agent:=clampStr(c.Request.UserAgent(),400)
  for _,doc:=range []*models.TermsDocument{term,privacy} {
   acceptance:=models.TermsAcceptance{UserID:user.ID,DocumentID:doc.ID,
    DocumentKind:doc.Kind,DocumentRole:doc.Role,DocumentVersion:doc.Version,
    AcceptedAt:now,IPAddress:ip,UserAgent:agent}
   if err:=tx.Create(&acceptance).Error;err!=nil{return err}
  }
  return nil
 })
 if errors.Is(err,errSocialPhoneExists){c.JSON(409,gin.H{"error":"phone number is already registered"});return}
 if err!=nil {c.JSON(500,gin.H{"error":"could not complete your profile"});return}
 c.JSON(http.StatusOK,gin.H{"message":"profile completed","accountAccessLevel":"full"})
}
var errSocialPhoneExists=errors.New("phone exists")
