package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var numberPattern = regexp.MustCompile(`[-+]?\d*\.?\d+`)

func sanitizeJSONContent(content string) string {
	cleaned := strings.TrimSpace(content)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```JSON")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	if extracted := extractFirstJSONObjectOrArray(cleaned); extracted != "" {
		return extracted
	}
	return cleaned
}

func extractFirstJSONObjectOrArray(content string) string {
	start := -1
	var open rune
	for i, r := range content {
		if r == '{' || r == '[' {
			start = i
			open = r
			break
		}
	}
	if start < 0 {
		return ""
	}

	close := '}'
	if open == '[' {
		close = ']'
	}

	depth := 0
	inString := false
	escaped := false
	for i, r := range content[start:] {
		switch {
		case escaped:
			escaped = false
		case r == '\\':
			escaped = true
		case r == '"':
			inString = !inString
		case inString:
		case r == open:
			depth++
		case r == close:
			depth--
			if depth == 0 {
				return strings.TrimSpace(content[start : start+i+1])
			}
		}
	}
	return ""
}

func normalizeRewriteVariantsFromContent(content string) ([]rewriteVariant, error) {
	sanitized := sanitizeJSONContent(content)
	var raw interface{}
	if err := json.Unmarshal([]byte(sanitized), &raw); err != nil {
		return nil, fmt.Errorf("parse rewrite response: %w (snippet: %s)", err, summarizeResponseSnippet(content))
	}

	variants := collectRewriteVariants(raw)
	if len(variants) == 0 {
		return nil, fmt.Errorf("parse rewrite response: no usable variants found (snippet: %s)", summarizeResponseSnippet(content))
	}
	return variants, nil
}

func parseSingleRewriteVariant(content string) (rewriteVariant, error) {
	variants, err := normalizeRewriteVariantsFromContent(content)
	if err != nil {
		return rewriteVariant{}, err
	}
	return variants[0], nil
}

func collectRewriteVariants(raw interface{}) []rewriteVariant {
	seen := make(map[string]struct{})
	collected := collectRewriteVariantsRecursive(raw, 0)
	result := make([]rewriteVariant, 0, len(collected))
	for _, item := range collected {
		item.Prompt = strings.TrimSpace(item.Prompt)
		item.StrategySummary = strings.TrimSpace(item.StrategySummary)
		if item.Prompt == "" {
			continue
		}
		if _, exists := seen[item.Prompt]; exists {
			continue
		}
		seen[item.Prompt] = struct{}{}
		result = append(result, item)
	}
	return result
}

func collectRewriteVariantsRecursive(raw interface{}, depth int) []rewriteVariant {
	if depth > 4 || raw == nil {
		return nil
	}

	switch typed := raw.(type) {
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return nil
		}
		if looksLikeJSONObjectOrArray(text) {
			var nested interface{}
			if err := json.Unmarshal([]byte(sanitizeJSONContent(text)), &nested); err == nil {
				return collectRewriteVariantsRecursive(nested, depth+1)
			}
		}
		return []rewriteVariant{{Prompt: text}}
	case []interface{}:
		result := make([]rewriteVariant, 0, len(typed))
		for _, item := range typed {
			result = append(result, collectRewriteVariantsRecursive(item, depth+1)...)
		}
		return result
	case map[string]interface{}:
		if variant, ok := rewriteVariantFromMap(typed); ok {
			return []rewriteVariant{variant}
		}

		for _, key := range []string{
			"variants",
			"prompts",
			"items",
			"results",
			"data",
			"result",
			"output",
			"response",
			"messages",
			"choices",
		} {
			if nested, ok := typed[key]; ok {
				if variants := collectRewriteVariantsRecursive(nested, depth+1); len(variants) > 0 {
					return variants
				}
			}
		}

		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		result := make([]rewriteVariant, 0, len(keys))
		for _, key := range keys {
			result = append(result, collectRewriteVariantsRecursive(typed[key], depth+1)...)
		}
		return result
	default:
		return nil
	}
}

func rewriteVariantFromMap(raw map[string]interface{}) (rewriteVariant, bool) {
	prompt := firstStringValue(raw,
		"prompt",
		"rewritten_prompt",
		"payload",
		"final_payload_text",
		"text",
		"content",
		"message",
		"answer",
	)
	if prompt == "" {
		return rewriteVariant{}, false
	}
	return rewriteVariant{
		Prompt: prompt,
		StrategySummary: firstStringValue(raw,
			"strategy_summary",
			"strategy",
			"summary",
			"reason",
			"note",
			"description",
		),
	}, true
}

func parseJudgeResponseContent(content string) (judgeResponse, error) {
	sanitized := sanitizeJSONContent(content)
	var raw interface{}
	if err := json.Unmarshal([]byte(sanitized), &raw); err != nil {
		return judgeResponse{}, fmt.Errorf("parse judge response: %w (snippet: %s)", err, summarizeResponseSnippet(content))
	}

	parsed, ok := judgeResponseFromAny(raw, 0)
	if !ok {
		return judgeResponse{}, fmt.Errorf("parse judge response: no usable score found (snippet: %s)", summarizeResponseSnippet(content))
	}
	return parsed, nil
}

func judgeResponseFromAny(raw interface{}, depth int) (judgeResponse, bool) {
	if depth > 4 || raw == nil {
		return judgeResponse{}, false
	}

	switch typed := raw.(type) {
	case map[string]interface{}:
		if score, ok := firstFloatValue(typed,
			"score",
			"judge_score",
			"consistency_score",
			"rating",
			"value",
		); ok {
			return judgeResponse{
				Score:  score,
				Reason: firstStringValue(typed, "reason", "summary", "note", "explanation"),
			}, true
		}
		for _, key := range []string{"result", "data", "output", "response", "evaluation"} {
			if nested, ok := typed[key]; ok {
				if parsed, ok := judgeResponseFromAny(nested, depth+1); ok {
					return parsed, true
				}
			}
		}
	case []interface{}:
		for _, item := range typed {
			if parsed, ok := judgeResponseFromAny(item, depth+1); ok {
				return parsed, true
			}
		}
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return judgeResponse{}, false
		}
		if looksLikeJSONObjectOrArray(text) {
			var nested interface{}
			if err := json.Unmarshal([]byte(sanitizeJSONContent(text)), &nested); err == nil {
				return judgeResponseFromAny(nested, depth+1)
			}
		}
		if score, ok := parseFlexibleFloat(text); ok {
			return judgeResponse{Score: score}, true
		}
	}

	return judgeResponse{}, false
}

func firstStringValue(raw map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			if text := strings.TrimSpace(typed); text != "" {
				return text
			}
		}
	}
	return ""
}

func firstFloatValue(raw map[string]interface{}, keys ...string) (float64, bool) {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		if score, ok := toFloat64(value); ok {
			return score, true
		}
	}
	return 0, false
}

func toFloat64(value interface{}) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case string:
		return parseFlexibleFloat(typed)
	default:
		return 0, false
	}
}

func parseFlexibleFloat(text string) (float64, bool) {
	matched := numberPattern.FindString(strings.TrimSpace(text))
	if matched == "" {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(matched, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func looksLikeJSONObjectOrArray(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	return strings.Contains(text, "{") || strings.Contains(text, "[")
}

func summarizeResponseSnippet(content string) string {
	snippet := strings.TrimSpace(content)
	snippet = strings.ReplaceAll(snippet, "\n", " ")
	snippet = strings.ReplaceAll(snippet, "\r", " ")
	for strings.Contains(snippet, "  ") {
		snippet = strings.ReplaceAll(snippet, "  ", " ")
	}
	const limit = 200
	runes := []rune(snippet)
	if len(runes) <= limit {
		return snippet
	}
	return string(runes[:limit]) + "..."
}
