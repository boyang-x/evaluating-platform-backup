package handler

import (
	"errors"
	"strings"
	"testing"
)

func TestReadLimitedUploadRejectsTooLargeFile(t *testing.T) {
	_, err := readLimitedUpload(strings.NewReader("123456"), 5)
	if !errors.Is(err, errUploadTooLarge) {
		t.Fatalf("expected errUploadTooLarge, got %v", err)
	}
}

func TestReadLimitedUploadAllowsExactLimit(t *testing.T) {
	data, err := readLimitedUpload(strings.NewReader("12345"), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "12345" {
		t.Fatalf("unexpected data %q", string(data))
	}
}

func TestExpertDataUploadLimitMatchesFrontendProxyLimit(t *testing.T) {
	if maxExpertDataUploadBytes != 50<<20 {
		t.Fatalf("maxExpertDataUploadBytes = %d, want 50MiB to match frontend nginx client_max_body_size", maxExpertDataUploadBytes)
	}
}
