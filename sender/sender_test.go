package sender

import (
	"errors"
	"testing"
)

func TestExtractErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "nil error returns 200",
			err:      nil,
			expected: 200,
		},
		{
			name:     "no HTTP code in error message",
			err:      errors.New("Forbidden: bot was blocked by the user"),
			expected: 0,
		},
		{
			name:     "HTTP 400 Bad Request",
			err:      errors.New("Bad Request: 400 - invalid parameters"),
			expected: 400,
		},
		{
			name:     "HTTP 401 Unauthorized",
			err:      errors.New("Unauthorized: 401 access denied"),
			expected: 401,
		},
		{
			name:     "HTTP 403 Forbidden",
			err:      errors.New("Forbidden: 403 bot blocked"),
			expected: 403,
		},
		{
			name:     "HTTP 404 Not Found",
			err:      errors.New("Not Found: 404 chat not found"),
			expected: 404,
		},
		{
			name:     "HTTP 429 Rate Limited",
			err:      errors.New("Too Many Requests: 429 rate limit exceeded"),
			expected: 429,
		},
		{
			name:     "HTTP 500 Internal Server Error",
			err:      errors.New("Internal Server Error: 500"),
			expected: 500,
		},
		{
			name:     "HTTP 502 Bad Gateway",
			err:      errors.New("Bad Gateway: 502 upstream error"),
			expected: 502,
		},
		{
			name:     "HTTP 503 Service Unavailable",
			err:      errors.New("Service Unavailable: 503"),
			expected: 503,
		},
		{
			name:     "HTTP 504 Gateway Timeout",
			err:      errors.New("Gateway Timeout: 504"),
			expected: 504,
		},
		{
			name:     "non-HTTP number should be ignored",
			err:      errors.New("Some error with number 123 but not HTTP code"),
			expected: 0,
		},
		{
			name:     "number out of 4xx/5xx range should be ignored",
			err:      errors.New("Error 999 not in 4xx/5xx range"),
			expected: 0,
		},
		{
			name:     "multiple HTTP codes - should return first one",
			err:      errors.New("Multiple codes: 400 and 500"),
			expected: 400,
		},
		{
			name:     "HTTP code at the beginning",
			err:      errors.New("400: Bad Request"),
			expected: 400,
		},
		{
			name:     "HTTP code at the end",
			err:      errors.New("Request failed with code 403"),
			expected: 403,
		},
		{
			name:     "HTTP code in parentheses",
			err:      errors.New("Request failed (status: 404)"),
			expected: 404,
		},
		{
			name:     "phone number should not be confused with HTTP code",
			err:      errors.New("User phone: +1-429-555-0123 is invalid"),
			expected: 0,
		},
		{
			name:     "year should not be confused with HTTP code",
			err:      errors.New("Error occurred in year 2023, code 500"),
			expected: 500,
		},
		{
			name:     "partial HTTP code should be ignored",
			err:      errors.New("Error 40 or 50 occurred"),
			expected: 0,
		},
		{
			name:     "HTTP code with extra digits should be ignored",
			err:      errors.New("Error 4001 occurred"),
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractErrorCode(tt.err)
			if result != tt.expected {
				t.Errorf("extractErrorCode() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

// Benchmark test to ensure regex performance is acceptable
func BenchmarkExtractErrorCode(b *testing.B) {
	err := errors.New("Too Many Requests: 429 rate limit exceeded")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractErrorCode(err)
	}
}

// Test edge cases with malformed error messages
func TestExtractErrorCodeEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "empty error message",
			err:      errors.New(""),
			expected: 0,
		},
		{
			name:     "only numbers",
			err:      errors.New("429"),
			expected: 429,
		},
		{
			name:     "HTTP code with punctuation",
			err:      errors.New("Error: 403!"),
			expected: 403,
		},
		{
			name:     "HTTP code with newlines",
			err:      errors.New("Error\n500\noccurred"),
			expected: 500,
		},
		{
			name:     "multiple spaces around HTTP code",
			err:      errors.New("Error   404   not found"),
			expected: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractErrorCode(tt.err)
			if result != tt.expected {
				t.Errorf("extractErrorCode() = %d, expected %d", result, tt.expected)
			}
		})
	}
}
