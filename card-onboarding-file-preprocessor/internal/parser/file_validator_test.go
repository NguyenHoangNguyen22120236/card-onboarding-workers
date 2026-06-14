package parser

import (
	"strings"
	"testing"
)

func TestValidateFile_ValidCSVFile(t *testing.T) {
	err := ValidateFile("cards.csv", 128, 1024)
	if err != nil {
		t.Fatalf("ValidateFile returned error: %v", err)
	}
}

func TestValidateFile_InvalidExtension(t *testing.T) {
	err := ValidateFile("cards.txt", 128, 1024)
	if err == nil {
		t.Fatal("ValidateFile returned nil error")
	}

	if !strings.Contains(err.Error(), "only .csv files are supported") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "only .csv files are supported")
	}
}

func TestValidateFile_EmptyFile(t *testing.T) {
	err := ValidateFile("cards.csv", 0, 1024)
	if err == nil {
		t.Fatal("ValidateFile returned nil error")
	}

	if !strings.Contains(err.Error(), "greater than 0") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "greater than 0")
	}
}

func TestValidateFile_FileExceedsMaxSize(t *testing.T) {
	err := ValidateFile("cards.csv", 2048, 1024)
	if err == nil {
		t.Fatal("ValidateFile returned nil error")
	}

	if !strings.Contains(err.Error(), "exceeds max file size") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "exceeds max file size")
	}
}
