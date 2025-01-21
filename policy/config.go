package policy

import (
	sqladapter "github.com/Blank-Xu/sql-adapter"
	redisadapter "github.com/casbin/redis-adapter/v3"
	"github.com/gowok/gowok"
	"github.com/gowok/gowok/must"
	"github.com/gowok/gowok/sql"
	"github.com/gowok/plugins/cache"
)

type Option func(*enforcer)

func withAdapter() Option {
	return func(p *enforcer) {
		if conf, ok := gowok.Get().Config.SQLs["policy"]; ok {
			if db, ok := sql.GetNoDefault("policy").Get(); ok {
				p.adapter = must.Must(sqladapter.NewAdapter(db, conf.Driver, "casbin_rule"))
				return
			}
		}

		if confs, err := cache.ConfigFromProject(gowok.Get()); err == nil {
			if conf, ok := confs["policy"]; ok {
				if !conf.Enabled {
					return
				}
				p.adapter = must.Must(redisadapter.NewAdapter("tcp", conf.DSN))
				return
			}
		}

		// add more adapters here!

	}
}
