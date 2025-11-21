package policy

import (
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/gowok/gowok"
	"github.com/gowok/gowok/singleton"
)

const (
	ActionCreate = "CREATE"
	ActionRead   = "READ"
	ActionUpdate = "UPDATE"
	ActionDelete = "DELETE"
)

var Actions = []string{
	ActionCreate,
	ActionRead,
	ActionUpdate,
	ActionDelete,
}

type enforcer struct {
	*casbin.Enforcer
	adapter any
}

func NewPolicy(modelStr string, opts ...Option) (*enforcer, error) {
	m, err := model.NewModelFromString(modelStr)
	if err != nil {
		return nil, err
	}

	ee := &enforcer{}
	for _, opt := range opts {
		opt(ee)
	}

	params := make([]any, 0)
	params = append(params, m)
	if ee.adapter != nil {
		params = append(params, ee.adapter)
	}

	e, err := casbin.NewEnforcer(params...)
	if err != nil {
		return nil, err
	}
	ee.Enforcer = e

	return ee, nil
}

func NewPolicyRBAC(opts ...Option) (*enforcer, error) {
	model := `
	[request_definition]
	r = sub, obj, act

	[policy_definition]
	p = sub, obj, act

	[role_definition]
	g = _, _

	[policy_effect]
	e = some(where (p.eft == allow))

	[matchers]
  m = (g(r.sub, p.sub) || r.sub == p.sub) && r.obj == p.obj && r.act == p.act
	`

	return NewPolicy(model, opts...)
}

func NewPolicyABAC(opts ...Option) (*enforcer, error) {
	model := `
  [request_definition]
  r = sub, obj, act

  [policy_definition]
  p = sub, obj, act

  [role_definition]
  g = _, _

  [policy_effect]
  e = some(where (p.eft == allow))

  [matchers]
  m = r.sub == p.sub && r.obj == p.obj && r.act == p.act
	`

	return NewPolicy(model, opts...)
}

var policy = singleton.New(func() *enforcer {
	return &enforcer{}
})

func Configure(model string, opts ...Option) func() {
	opts = append(opts, withAdapter())
	return func() {
		p := &enforcer{}
		switch model {
		case "rbac", "RBAC":
			p = gowok.Must(NewPolicyRBAC(opts...))
		case "abac", "ABAC":
			p = gowok.Must(NewPolicyRBAC(opts...))
		}
		policy(p)
	}
}

func ConfigureRBAC(opts ...Option) func() {
	return Configure("rbac", opts...)
}

func ConfigureABAC(opts ...Option) func() {
	return Configure("abac", opts...)
}

func Enforcer() *enforcer {
	p := policy()
	return *p
}
