package policy

import (
	"strings"
	"testing"
)

func TestEnforcerWithoutConfigure(t *testing.T) {
	e := Enforcer()
	if e.Enforcer == nil {
		t.Fatal("Enforcer() must fall back to a default model instead of a nil enforcer")
	}

	policies, err := e.GetPolicy()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(policies) != 0 {
		t.Fatalf("default enforcer must have no policy, got %v", policies)
	}

	ok, err := e.Enforce("alice", "data1", ActionRead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("enforce on an empty default policy must be false")
	}
}

func TestConfigureDoesNotWriteIntoCallerOptions(t *testing.T) {
	opts := make([]Option, 1, 2)
	opts[0] = func(*enforcer) {}

	Configure("rbac", opts...)

	if spare := opts[:cap(opts)]; spare[1] != nil {
		t.Fatalf("Configure wrote into the caller's slice backing array: %v", spare[1])
	}
}

func TestConfigureABACUsesABACModel(t *testing.T) {
	ConfigureABAC()()

	e := Enforcer()
	if e == nil || e.Enforcer == nil {
		t.Fatal("enforcer is not configured")
	}

	matcher := e.GetModel()["m"]["m"].Value
	if strings.Contains(matcher, "g(r_sub") {
		t.Fatalf("ABAC model must not use the RBAC matcher, got %q", matcher)
	}
}

func TestConfigureCustomModel(t *testing.T) {
	const customModel = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub == p.sub && r.obj == p.obj && r.act == p.act
`

	Configure(customModel)()

	e := Enforcer()
	if e == nil || e.Enforcer == nil {
		t.Fatal("enforcer is not configured")
	}

	matcher := e.GetModel()["m"]["m"].Value
	if matcher != "r_sub == p_sub && r_obj == p_obj && r_act == p_act" {
		t.Fatalf("custom model is not loaded, got matcher %q", matcher)
	}
}

func TestNewPolicySkipsNilOption(t *testing.T) {
	e, err := NewPolicyRBAC(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e == nil || e.Enforcer == nil {
		t.Fatal("enforcer is not initialized")
	}

	ok, err := e.Enforce("alice", "data1", "READ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("enforce without any policy must be false")
	}
}
