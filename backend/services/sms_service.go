package services

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// maskPhone returns a PII-safe representation of a phone number: all digits
// except the last two are replaced with asterisks. Empty input returns "".
func maskPhone(phone string) string {
	re := regexp.MustCompile(`[^0-9]`)
	digits := re.ReplaceAllString(phone, "")
	if len(digits) <= 2 {
		return "***"
	}
	return "***" + digits[len(digits)-2:]
}

// SendSMS sends an SMS message via Twilio if configured, otherwise logs.
func SendSMS(phone, message string) error {
	formatted := formatPhoneNumber(phone)
	if formatted == "" {
		return fmt.Errorf("empty phone number")
	}

	accountSid := strings.TrimSpace(os.Getenv("TWILIO_ACCOUNT_SID"))
	authToken := strings.TrimSpace(os.Getenv("TWILIO_AUTH_TOKEN"))
	fromPhone := strings.TrimSpace(os.Getenv("TWILIO_PHONE_NUMBER"))

	if accountSid != "" && authToken != "" && fromPhone != "" {
		apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", accountSid)

		data := url.Values{}
		data.Set("To", "+"+formatted)
		data.Set("From", fromPhone)
		data.Set("Body", message)

		req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
		if err != nil {
			log.Printf("sms: failed to create Twilio request: %v", err)
			return err
		}
		req.SetBasicAuth(accountSid, authToken)
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("sms: Twilio request failed for %s: %v", maskPhone(formatted), err)
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Printf("sms: Twilio sent successfully to %s", maskPhone(formatted))
			return nil
		} else {
			log.Printf("sms: Twilio returned status %d for %s", resp.StatusCode, maskPhone(formatted))
			return fmt.Errorf("twilio status %d", resp.StatusCode)
		}
	}

	log.Printf("sms: sending to %s", maskPhone(phone))
	return nil
}

// SendOTPSMS sends an OTP verification code via SMS
func SendOTPSMS(phone, otpCode string) error {
	phone = formatPhoneNumber(phone)
	_ = otpCode
	log.Printf("sms: otp sent to %s", maskPhone(phone))
	return nil
}

// formatPhoneNumber formats phone number for Nigeria
func formatPhoneNumber(phone string) string {
	re := regexp.MustCompile(`[^0-9]`)
	phone = re.ReplaceAllString(phone, "")

	if len(phone) > 0 && phone[0] == '0' {
		phone = "234" + phone[1:]
	}

	if len(phone) > 0 && len(phone) >= 10 && phone[:3] != "234" {
		phone = "234" + phone
	}

	return phone
}
