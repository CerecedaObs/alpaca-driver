package dome

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseResponse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Response
		expectError bool
	}{
		{
			name:  "Valid ACK without value",
			input: "_ACK_S;",
			expected: Response{
				Code:  cmdStatus,
				Error: false,
			},
			expectError: false,
		},
		{
			name:  "Valid ACK with value",
			input: "_ACK_V=(1.2.3);",
			expected: Response{
				Code:  cmdVersion,
				Value: "(1.2.3)",
				Error: false,
			},
			expectError: false,
		},
		{
			name:  "Valid NACK without value",
			input: "_NACK_V;",
			expected: Response{
				Code:  cmdVersion,
				Error: true,
			},
			expectError: false,
		},
		// {
		// 	name:        "Command with more than one character",
		// 	input:       "_ACK_CMD=123;",
		// 	expectError: true,
		// },
		// {
		// 	name:        "Nack of command with more than one character",
		// 	input:       "_NACK_CMD;",
		// 	expectError: true,
		// },
		{
			name:        "Too few underscores",
			input:       "ACK_C;",
			expectError: true,
		},
		{
			name:        "Invalid ack indicator",
			input:       "_NOTACK_V;",
			expectError: true,
		},
		{
			name:        "Invalid extra equals",
			input:       "_ACK_P=123=456;",
			expectError: true,
		},
		{
			name:        "No semicolon",
			input:       "_ACK_P=123",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := parseResponse(tc.input)
			if tc.expectError {
				assert.Error(t, err, "expected error for input: %s", tc.input)
			} else {
				assert.NoError(t, err, "unexpected error for input: %s", tc.input)
				assert.Equal(t, tc.expected.Code, resp.Code)
				assert.Equal(t, tc.expected.Value, resp.Value)
				assert.Equal(t, tc.expected.Error, resp.Error)
			}
		})
	}
}

func TestNormalizeAngle(t *testing.T) {
	assert.Equal(t, 0.0, normalizeAngle(0.0))
	assert.Equal(t, 45.0, normalizeAngle(45.0))
	assert.Equal(t, 0.0, normalizeAngle(360.0))
	assert.Equal(t, 0.0, normalizeAngle(-360.0))
	assert.Equal(t, 10.0, normalizeAngle(370.0))
	assert.Equal(t, 330.0, normalizeAngle(-30.0))
	assert.Equal(t, 320.0, normalizeAngle(-400.0))
	assert.Equal(t, 85.0, normalizeAngle(3685.0))
	assert.Equal(t, 30.0, normalizeAngle(-3570.0))
}
