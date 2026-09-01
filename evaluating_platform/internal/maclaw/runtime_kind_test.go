package maclaw

import "testing"

func TestNormalizeRuntimeKindDefaultsToMaclawSrv(t *testing.T) {
	if got := NormalizeRuntimeKind(""); got != RuntimeKindMaclawSrv {
		t.Fatalf("NormalizeRuntimeKind empty = %q", got)
	}
}

func TestNewRuntimeClientRejectsRetiredLegacyRuntimeKind(t *testing.T) {
	if _, err := NewRuntimeClient("legacy-evaluation", Config{BaseURL: "http://maclaw.local", APIToken: "token", TimeoutSeconds: 1}); err == nil {
		t.Fatalf("expected legacy runtime kind to be rejected")
	}
}
