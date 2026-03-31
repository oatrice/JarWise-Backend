package service

import (
	"testing"
	"time"
)

func TestParseMMTransactionDate_SupportsUnixMilliseconds(t *testing.T) {
	parsed, err := parseMMTransactionDate("1506749778498")
	if err != nil {
		t.Fatalf("expected unix millisecond timestamp to parse, got %v", err)
	}

	expected := time.UnixMilli(1506749778498).UTC()
	if !parsed.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected, parsed)
	}
}

func TestParseMMTransactionDate_SupportsLegacyDateLayouts(t *testing.T) {
	parsed, err := parseMMTransactionDate("2025-01-15 08:15:00")
	if err != nil {
		t.Fatalf("expected formatted timestamp to parse, got %v", err)
	}

	expected := time.Date(2025, time.January, 15, 8, 15, 0, 0, time.UTC)
	if !parsed.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected, parsed)
	}
}
