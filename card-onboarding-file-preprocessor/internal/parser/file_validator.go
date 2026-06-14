package parser

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

func ValidateFile(sourceFileName string, fileSizeBytes int64, maxFileSizeBytes int64) error {
	if !strings.EqualFold(filepath.Ext(sourceFileName), ".csv") {
		return fmt.Errorf("invalid file extension for %q: only .csv files are supported", sourceFileName)
	}

	if fileSizeBytes <= 0 {
		return errors.New("file size must be greater than 0 bytes")
	}

	if fileSizeBytes > maxFileSizeBytes {
		return fmt.Errorf("file size %d bytes exceeds max file size %d bytes", fileSizeBytes, maxFileSizeBytes)
	}

	return nil
}
