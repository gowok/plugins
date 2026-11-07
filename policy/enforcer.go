package policy

import (
	"errors"
	"slices"

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

// GrantRole assigns role to sub, so sub inherits every rule granted to role.
func (e *enforcer) GrantRole(sub, role string) (bool, error) {
	if e == nil || e.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return e.AddRoleForUser(sub, role)
}

// RevokeRole removes the role assignment from sub.
func (e *enforcer) RevokeRole(sub, role string) (bool, error) {
	if e == nil || e.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return e.DeleteRoleForUser(sub, role)
}

// HasRole reports whether sub is assigned to role.
func (e *enforcer) HasRole(sub, role string) (bool, error) {
	if e == nil || e.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	roles, err := e.GetRolesForUser(sub)
	if err != nil {
		return false, err
	}
	return slices.Contains(roles, role), nil
}
