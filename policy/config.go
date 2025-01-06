package policy

import (
	sqladapter "github.com/Blank-Xu/sql-adapter"
	redisadapter "github.com/casbin/redis-adapter/v3"
	"github.com/gowok/gowok"
	"github.com/gowok/gowok/exception"
	"github.com/gowok/gowok/must"
	"github.com/gowok/plugins/cache"
)

type Option func(*enforcer)

func WithAdapter() Option {
	return func(p *enforcer) {
		var a any
		func() {
			conf, ok := gowok.Get().Config.SQLs["policy"]
			if !ok {
				return
			}
			db := gowok.Get().SQL("policy").OrPanic(exception.NoDatabaseFound)
			a = must.Must(sqladapter.NewAdapter(db, conf.Driver, "casbin_rule"))
		}()

		func() {
			confs, err := cache.ConfigFromProject(gowok.Get())
			if err != nil {
				return
			}

			conf, ok := confs["policy"]
			if !ok {
				return
			}

			a = must.Must(redisadapter.NewAdapter("tcp", conf.DSN))
		}()

		if a != nil {
			p.adapter = a
		}
	}
}
