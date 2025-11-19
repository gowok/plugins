package policy

import (
	sqladapter "github.com/Blank-Xu/sql-adapter"
	redisadapter "github.com/casbin/redis-adapter/v3"
	"github.com/gowok/gowok"
	"github.com/gowok/gowok/sql"
	"github.com/gowok/plugins/cache"
)

type Option func(*enforcer)

func withAdapter() Option {
	return func(p *enforcer) {
		if conf, ok := gowok.Get().Config.SQLs["policy"]; ok {
			if db, ok := sql.DBNoDefault("policy").Get(); ok {
				p.adapter = gowok.Must(sqladapter.NewAdapter(db, conf.Driver, "casbin_rule"))
				return
			}
		}

		if confs, err := cache.ConfigFromProject(gowok.Get()); err == nil {
			if conf, ok := confs["policy"]; ok {
				if !conf.Enabled {
					return
				}
				p.adapter = gowok.Must(redisadapter.NewAdapter("tcp", conf.DSN))
				return
			}
		}

		// add more adapters here!

	}
}
