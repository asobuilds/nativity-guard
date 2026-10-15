package services

import "security-solution/models"

// GenerateJWTWithJTI exposes the existing JWT generator for trusted
// authentication flows, including one-time sign-in links.
func (s *AuthService) GenerateJWTWithJTI(user *models.User) (string, string, error) {
	return s.generateJWTWithJTI(user)
}
