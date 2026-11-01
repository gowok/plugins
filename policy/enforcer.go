package policy

import (
	"errors"

	"github.com/casbin/casbin/v2"
)

// ErrNilEnforcer is returned when the enforcer used by a rule helper is not initialized.
var ErrNilEnforcer = errors.New("policy: enforcer is not initialized")

type enforcer struct {
	*casbin.Enforcer
	adapter any
}

// Grant adds a policy rule allowing sub to act on obj.
func (e *enforcer) Grant(sub, obj, act string) (bool, error) {
	if e == nil || e.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return e.AddPolicy(sub, obj, act)
}

// Revoke removes the policy rule allowing sub to act on obj.
func (e *enforcer) Revoke(sub, obj, act string) (bool, error) {
	if e == nil || e.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return e.RemovePolicy(sub, obj, act)
}
