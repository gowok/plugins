package policy

import (
	"database/sql"

	sqladapter "github.com/Blank-Xu/sql-adapter"
	redisadapter "github.com/casbin/redis-adapter/v3"
	"github.com/gowok/gowok"
	"github.com/gowok/gowok/must"
	"github.com/gowok/plugins/cache"
)

type Option func(*enforcer)

func withAdapter() Option {
	return func(p *enforcer) {
		var a any
		func() {
			conf, ok := gowok.Get().Config.SQLs["policy"]
			if !ok {
				return
			}
			gowok.Get().SQL("policy").IfPresent(func(db *sql.DB) {
				p.adapter = must.Must(sqladapter.NewAdapter(db, conf.Driver, "casbin_rule"))
			})
		}()
		if p.adapter != nil {
			return
		}

		func() {
			confs, err := cache.ConfigFromProject(gowok.Get())
			if err != nil {
				return
			}

			conf, ok := confs["policy"]
			if !ok {
				return
			}

			if !conf.Enabled {
				return
			}

			p.adapter = must.Must(redisadapter.NewAdapter("tcp", conf.DSN))
		}()
		if a != nil {
			return
		}

		// add more adapters here!

	}
}
