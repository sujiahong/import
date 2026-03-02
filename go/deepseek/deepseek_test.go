package deepseek

import (
	"os"
	"testing"
)

func TestGetAPIKey(t *testing.T) {
	// Test when environment variable is not set
	os.Unsetenv("DEEPSEEK_API_KEY")
	_, err := getAPIKey()
	if err == nil {
		t.Error("Expected error when DEEPSEEK_API_KEY is not set")
	}

	// Test when environment variable is set
	os.Setenv("DEEPSEEK_API_KEY", "test-api-key")
	apiKey, err := getAPIKey()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if apiKey != "test-api-key" {
		t.Errorf("Expected 'test-api-key', got '%s'", apiKey)
	}
}

func TestDSChat_EmptyQuestion(t *testing.T) {
	os.Setenv("DEEPSEEK_API_KEY", "test-api-key")
	_, err := DSChat("")
	if err == nil {
		t.Error("Expected error when question is empty")
	}
	if err.Error() != "question cannot be empty" {
		t.Errorf("Expected 'question cannot be empty', got '%s'", err.Error())
	}
}

func TestDSChat_NoAPIKey(t *testing.T) {
	os.Unsetenv("DEEPSEEK_API_KEY")
	_, err := DSChat("test question")
	if err == nil {
		t.Error("Expected error when DEEPSEEK_API_KEY is not set")
	}
}

func TestDSList_NoAPIKey(t *testing.T) {
	os.Unsetenv("DEEPSEEK_API_KEY")
	_, err := DSList()
	if err == nil {
		t.Error("Expected error when DEEPSEEK_API_KEY is not set")
	}
}
