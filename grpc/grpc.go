package grpc

import (
	"log/slog"
	"net"

	"github.com/gowok/gowok"
	"github.com/gowok/gowok/singleton"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var _g = singleton.New(func() *grpc.Server {
	s := grpc.NewServer()
	reflection.Register(s)
	return s
})

func g() *grpc.Server {
	return *_g()
}

type Service interface {
	Description() *grpc.ServiceDesc
}

func Configure(services ...Service) func() {
	return func() {
		config, err := getConfig()
		if err != nil {
			slog.Warn("failed to start", "error", err)
			return
		}

		if !config.Enabled {
			return
		}

		slog.Info("starting GRPC", "host", config.Host)
		gowok.Config.Forever = true
		listen, err := net.Listen("tcp", config.Host)
		if err != nil {
			panic("grpc: failed to start: " + err.Error())
		}

		for _, s := range services {
			RegisterService(s.Description(), s)
		}

		go func() {
			err = g().Serve(listen)
			if err != nil {
				panic("grpc: failed to start: " + err.Error())
			}
		}()
	}
}

func RegisterService(sd *grpc.ServiceDesc, ss any) {
	g().RegisterService(sd, ss)
}

func GetServiceInfo() map[string]grpc.ServiceInfo {
	return g().GetServiceInfo()
}
