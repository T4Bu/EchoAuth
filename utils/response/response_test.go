package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSONResponse(t *testing.T) {
	tests := []struct {
		name           string
		data           interface{}
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success response with data",
			data: map[string]string{
				"message": "success",
				"token":   "test-token",
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"success","token":"test-token"}`,
		},
		{
			name:           "Empty response",
			data:           map[string]string{},
			expectedStatus: http.StatusOK,
			expectedBody:   `{}`,
		},
		{
			name: "Response with numbers",
			data: map[string]interface{}{
				"id":     1,
				"active": true,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"active":true,"id":1}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			JSONResponse(w, tt.data, tt.expectedStatus)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Header().Get("Content-Type") != "application/json" {
				t.Error("Content-Type header not set to application/json")
			}

			// Normalize the JSON response for comparison
			var got, want interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatalf("Failed to unmarshal response body: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.expectedBody), &want); err != nil {
				t.Fatalf("Failed to unmarshal expected body: %v", err)
			}

			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestJSONError(t *testing.T) {
	tests := []struct {
		name         string
		message      string
		status       int
		expectedBody string
	}{
		{
			name:         "Bad request error",
			message:      "Invalid input",
			status:       http.StatusBadRequest,
			expectedBody: `{"error":"Invalid input"}`,
		},
		{
			name:         "Unauthorized error",
			message:      "Invalid credentials",
			status:       http.StatusUnauthorized,
			expectedBody: `{"error":"Invalid credentials"}`,
		},
		{
			name:         "Internal server error",
			message:      "Internal server error",
			status:       http.StatusInternalServerError,
			expectedBody: `{"error":"Internal server error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			JSONError(w, tt.message, tt.status)

			if w.Code != tt.status {
				t.Errorf("Expected status code %d, got %d", tt.status, w.Code)
			}

			if w.Header().Get("Content-Type") != "application/json" {
				t.Error("Content-Type header not set to application/json")
			}

			// Normalize the JSON response for comparison
			var got, want interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatalf("Failed to unmarshal response body: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.expectedBody), &want); err != nil {
				t.Fatalf("Failed to unmarshal expected body: %v", err)
			}

			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, w.Body.String())
			}
		})
	}
}
