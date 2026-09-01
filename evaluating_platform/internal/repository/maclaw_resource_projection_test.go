package repository

import (
	"reflect"
	"testing"
)

func TestResourcePublicationSearchTokensSplitsChineseAndEnglishTerms(t *testing.T) {
	got := resourcePublicationSearchTokens("文言文 越狱 CCBOS/prompt_injection")
	want := []string{"文言文", "越狱", "ccbos", "prompt", "injection"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tokens = %#v, want %#v", got, want)
	}
}

func TestResourcePublicationSearchTokensDedupesAndDropsShortTerms(t *testing.T) {
	got := resourcePublicationSearchTokens("LLM llm a 文言文 文言文")
	want := []string{"llm", "文言文"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tokens = %#v, want %#v", got, want)
	}
}
