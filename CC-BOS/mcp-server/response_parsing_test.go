package main

import "testing"

func TestSanitizeJSONContentExtractsFirstObject(t *testing.T) {
	raw := "1. ```json\n{\"variants\":[{\"prompt\":\"甲\",\"strategy_summary\":\"隐写\"}]}\n```"
	got := sanitizeJSONContent(raw)
	want := "{\"variants\":[{\"prompt\":\"甲\",\"strategy_summary\":\"隐写\"}]}"
	if got != want {
		t.Fatalf("sanitizeJSONContent() = %q, want %q", got, want)
	}
}

func TestNormalizeRewriteVariantsFromStandardShape(t *testing.T) {
	raw := "{\"variants\":[{\"prompt\":\"甲\",\"strategy_summary\":\"隐写\"},{\"prompt\":\"乙\",\"strategy_summary\":\"借喻\"}]}"
	variants, err := normalizeRewriteVariantsFromContent(raw)
	if err != nil {
		t.Fatalf("normalizeRewriteVariantsFromContent() error = %v", err)
	}
	if len(variants) != 2 {
		t.Fatalf("len(variants) = %d, want 2", len(variants))
	}
	if variants[0].Prompt != "甲" || variants[1].Prompt != "乙" {
		t.Fatalf("unexpected variants: %#v", variants)
	}
}

func TestNormalizeRewriteVariantsFromSinglePromptObject(t *testing.T) {
	raw := "{\"prompt\":\"文言改写结果\",\"strategy_summary\":\"借古喻今\"}"
	variants, err := normalizeRewriteVariantsFromContent(raw)
	if err != nil {
		t.Fatalf("normalizeRewriteVariantsFromContent() error = %v", err)
	}
	if len(variants) != 1 {
		t.Fatalf("len(variants) = %d, want 1", len(variants))
	}
	if variants[0].Prompt != "文言改写结果" {
		t.Fatalf("variant prompt = %q, want 文言改写结果", variants[0].Prompt)
	}
}

func TestNormalizeRewriteVariantsFromPromptArray(t *testing.T) {
	raw := "{\"prompts\":[\"文言一\",\"文言二\"]}"
	variants, err := normalizeRewriteVariantsFromContent(raw)
	if err != nil {
		t.Fatalf("normalizeRewriteVariantsFromContent() error = %v", err)
	}
	if len(variants) != 2 {
		t.Fatalf("len(variants) = %d, want 2", len(variants))
	}
	if variants[0].Prompt != "文言一" || variants[1].Prompt != "文言二" {
		t.Fatalf("unexpected prompt array parse: %#v", variants)
	}
}

func TestParseJudgeResponseContentSupportsNestedStringScore(t *testing.T) {
	raw := "{\"result\":{\"score\":\"4/5\",\"reason\":\"较强命中\"}}"
	parsed, err := parseJudgeResponseContent(raw)
	if err != nil {
		t.Fatalf("parseJudgeResponseContent() error = %v", err)
	}
	if parsed.Score != 4 {
		t.Fatalf("parsed.Score = %v, want 4", parsed.Score)
	}
	if parsed.Reason != "较强命中" {
		t.Fatalf("parsed.Reason = %q, want 较强命中", parsed.Reason)
	}
}
