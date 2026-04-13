package chat

import "testing"

func TestPayloadLogsFromExecuteOutputParsesSnakeCaseResults(t *testing.T) {
	output := `{"results":[{"index":1,"original_content":"????","enhanced_content":"??+??","target_response":"????","attack_success":true,"attack_reason":"???????","error":""}]}`

	logs := payloadLogsFromExecuteOutput(output)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].ToolName != "payload_1" {
		t.Fatalf("unexpected tool name: %s", logs[0].ToolName)
	}
	if logs[0].Input == "" {
		t.Fatalf("expected non-empty input")
	}
	if logs[0].Output != "????" {
		t.Fatalf("unexpected output: %s", logs[0].Output)
	}
	if logs[0].Severity != "high" {
		t.Fatalf("expected high severity, got %s", logs[0].Severity)
	}
}

func TestPayloadLogsFromExecuteOutputParsesLegacyPascalCaseResults(t *testing.T) {
	output := `{"results":[{"Index":2,"OriginalContent":"???","EnhancedContent":"???+??","TargetResponse":"???","AttackSuccess":false,"AttackReason":"????","Error":""}]}`

	logs := payloadLogsFromExecuteOutput(output)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].ToolName != "payload_2" {
		t.Fatalf("unexpected tool name: %s", logs[0].ToolName)
	}
	if logs[0].Input == "" {
		t.Fatalf("expected non-empty input")
	}
	if logs[0].Output != "???" {
		t.Fatalf("unexpected output: %s", logs[0].Output)
	}
	if logs[0].Severity != "info" {
		t.Fatalf("expected info severity, got %s", logs[0].Severity)
	}
}
