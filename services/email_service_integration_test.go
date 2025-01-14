package services

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

type MailHogMessage struct {
	Content struct {
		Headers map[string][]string `json:"headers"`
		Body    string              `json:"body"`
	} `json:"Content"`
	Raw struct {
		From string   `json:"From"`
		To   []string `json:"To"`
	} `json:"Raw"`
}

type MailHogResponse struct {
	Items []MailHogMessage `json:"items"`
}

func TestEmailService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Configure email service to use MailHog
	service := NewEmailService(SMTPConfig{
		Host:     "localhost",
		Port:     1025, // MailHog SMTP port
		Username: "",   // No auth needed for MailHog
		Password: "",
		From:     "test@example.com",
	})

	tests := []struct {
		name    string
		to      string
		isReset bool // true for password reset, false for verification
		wantErr bool
	}{
		{
			name:    "Send password reset email",
			to:      "user@example.com",
			isReset: true,
			wantErr: false,
		},
		{
			name:    "Send verification email",
			to:      "user@example.com",
			isReset: false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear MailHog messages before test
			clearMailHogMessages(t)

			// Send email
			var err error
			if tt.isReset {
				err = service.SendPasswordResetEmail(tt.to, "test-token")
			} else {
				err = service.SendVerificationEmail(tt.to, "test-token")
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("SendEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Wait a bit for MailHog to process the email
				time.Sleep(100 * time.Millisecond)

				// Verify email was received by MailHog
				messages := getMailHogMessages(t)
				if len(messages.Items) != 1 {
					t.Errorf("Expected 1 email, got %d", len(messages.Items))
					return
				}

				msg := messages.Items[0]
				if len(msg.Raw.To) != 1 || msg.Raw.To[0] != tt.to {
					t.Errorf("Expected recipient %s, got %v", tt.to, msg.Raw.To)
				}

				expectedSubject := "Password Reset Request"
				if !tt.isReset {
					expectedSubject = "Verify Your Email"
				}

				if subjects, ok := msg.Content.Headers["Subject"]; !ok || len(subjects) != 1 || subjects[0] != expectedSubject {
					t.Errorf("Expected subject %s, got %v", expectedSubject, subjects)
				}

				if !tt.isReset && !contains(msg.Content.Body, "test-token") {
					t.Error("Verification email should contain the token")
				}
				if tt.isReset && !contains(msg.Content.Body, "test-token") {
					t.Error("Password reset email should contain the token")
				}
			}
		})
	}
}

func clearMailHogMessages(t *testing.T) {
	req, err := http.NewRequest(http.MethodDelete, "http://localhost:8025/api/v1/messages", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to clear messages: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to clear messages, status: %d", resp.StatusCode)
	}
}

func getMailHogMessages(t *testing.T) *MailHogResponse {
	resp, err := http.Get("http://localhost:8025/api/v2/messages")
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to get messages, status: %d", resp.StatusCode)
	}

	var messages MailHogResponse
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		t.Fatalf("Failed to decode messages: %v", err)
	}

	return &messages
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
