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

func TestGrantAndRevoke(t *testing.T) {
	e, err := NewPolicyRBAC()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ok, err := e.Enforce("alice", "data1", ActionRead)
	if err != nil || ok {
		t.Fatalf("before grant: got (%v, %v), want (false, nil)", ok, err)
	}

	if added, err := e.Grant("alice", "data1", ActionRead); err != nil || !added {
		t.Fatalf("grant: got (%v, %v), want (true, nil)", added, err)
	}
	if ok, err := e.Enforce("alice", "data1", ActionRead); err != nil || !ok {
		t.Fatalf("after grant: got (%v, %v), want (true, nil)", ok, err)
	}

	if removed, err := e.Revoke("alice", "data1", ActionRead); err != nil || !removed {
		t.Fatalf("revoke: got (%v, %v), want (true, nil)", removed, err)
	}
	if ok, err := e.Enforce("alice", "data1", ActionRead); err != nil || ok {
		t.Fatalf("after revoke: got (%v, %v), want (false, nil)", ok, err)
	}

	if _, err := (*enforcer)(nil).Grant("alice", "data1", ActionRead); err != ErrNilEnforcer {
		t.Fatalf("grant on nil enforcer: got %v, want %v", err, ErrNilEnforcer)
	}
}

func TestGrantAndRevokeRole(t *testing.T) {
	e, err := NewPolicyRBAC()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := e.Grant("admin", "data1", ActionRead); err != nil {
		t.Fatalf("grant: unexpected error: %v", err)
	}

	if ok, err := e.Enforce("alice", "data1", ActionRead); err != nil || ok {
		t.Fatalf("before role grant: got (%v, %v), want (false, nil)", ok, err)
	}
	if ok, err := e.HasRole("alice", "admin"); err != nil || ok {
		t.Fatalf("has role before grant: got (%v, %v), want (false, nil)", ok, err)
	}

	if added, err := e.GrantRole("alice", "admin"); err != nil || !added {
		t.Fatalf("grant role: got (%v, %v), want (true, nil)", added, err)
	}
	if ok, err := e.HasRole("alice", "admin"); err != nil || !ok {
		t.Fatalf("has role after grant: got (%v, %v), want (true, nil)", ok, err)
	}
	if ok, err := e.Enforce("alice", "data1", ActionRead); err != nil || !ok {
		t.Fatalf("after role grant: got (%v, %v), want (true, nil)", ok, err)
	}

	if removed, err := e.RevokeRole("alice", "admin"); err != nil || !removed {
		t.Fatalf("revoke role: got (%v, %v), want (true, nil)", removed, err)
	}
	if ok, err := e.HasRole("alice", "admin"); err != nil || ok {
		t.Fatalf("has role after revoke: got (%v, %v), want (false, nil)", ok, err)
	}
	if ok, err := e.Enforce("alice", "data1", ActionRead); err != nil || ok {
		t.Fatalf("after role revoke: got (%v, %v), want (false, nil)", ok, err)
	}
}

func TestRoleHelpersOnNilEnforcer(t *testing.T) {
	if _, err := (*enforcer)(nil).GrantRole("alice", "admin"); err != ErrNilEnforcer {
		t.Fatalf("grant role on nil enforcer: got %v, want %v", err, ErrNilEnforcer)
	}
	if _, err := (*enforcer)(nil).RevokeRole("alice", "admin"); err != ErrNilEnforcer {
		t.Fatalf("revoke role on nil enforcer: got %v, want %v", err, ErrNilEnforcer)
	}
	if _, err := (*enforcer)(nil).HasRole("alice", "admin"); err != ErrNilEnforcer {
		t.Fatalf("has role on nil enforcer: got %v, want %v", err, ErrNilEnforcer)
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
