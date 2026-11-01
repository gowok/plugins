package policy

import (
	"strings"
	"testing"
)

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
