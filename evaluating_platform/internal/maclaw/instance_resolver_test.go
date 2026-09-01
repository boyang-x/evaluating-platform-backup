package maclaw

import "testing"

func TestInstanceResolverPrefersExactUserMapping(t *testing.T) {
	resolver := NewInstanceResolver("inst_default", []InstanceMapping{
		{Role: "enterprise", InstanceID: "inst_role"},
		{UserID: "user_1", InstanceID: "inst_user"},
	})

	instanceID, ok := resolver.Resolve(RuntimeIdentity{UserID: "user_1", Role: "enterprise"})
	if !ok {
		t.Fatalf("expected mapping")
	}
	if instanceID != "inst_user" {
		t.Fatalf("instance id = %q, want user mapping", instanceID)
	}
}

func TestInstanceResolverFallsBackToRoleThenDefault(t *testing.T) {
	resolver := NewInstanceResolver("inst_default", []InstanceMapping{
		{Role: "enterprise", InstanceID: "inst_enterprise"},
	})

	instanceID, ok := resolver.Resolve(RuntimeIdentity{UserID: "user_2", Role: "enterprise"})
	if !ok || instanceID != "inst_enterprise" {
		t.Fatalf("role instance = %q ok=%v", instanceID, ok)
	}

	instanceID, ok = resolver.Resolve(RuntimeIdentity{UserID: "user_3", Role: "admin"})
	if !ok || instanceID != "inst_default" {
		t.Fatalf("default instance = %q ok=%v", instanceID, ok)
	}
}

func TestInstanceResolverRejectsEmptyResolution(t *testing.T) {
	resolver := NewInstanceResolver("", []InstanceMapping{{UserID: "user_1"}})
	if instanceID, ok := resolver.Resolve(RuntimeIdentity{UserID: "user_1"}); ok || instanceID != "" {
		t.Fatalf("instance id = %q ok=%v", instanceID, ok)
	}
}
