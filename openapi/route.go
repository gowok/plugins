package openapi

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-openapi/spec"
	"github.com/gowok/fp/slices"
	"github.com/gowok/gowok"
	"github.com/ngamux/ngamux"
)

type Router struct {
	group *ngamux.HttpServeMux
	opts  RouteOption
}

type Operation struct {
	*spec.Operation
	handler http.Handler
}

func (o *Operation) Handler(handler http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) {
	ngamuxMiddlewares := slices.Map(middlewares, func(m func(http.HandlerFunc) http.HandlerFunc) ngamux.MiddlewareFunc {
		return m
	})
	o.handler = ngamux.WithMiddlewares(ngamuxMiddlewares...)(handler)
}

type RouteOption func(*Operation)

type routeEntry struct {
	method, path string
}

func (r routeEntry) Method() string { return r.method }
func (r routeEntry) Path() string   { return r.path }

func (r *Router) Group(prefix string, opts ...RouteOption) *Router {
	return &Router{group: gowok.Web.Group(prefix), opts: func() RouteOption {
		if len(opts) <= 0 {
			return func(o *Operation) {}
		}

		return opts[0]
	}()}
}

func (r *Router) GroupFunc(prefix string, fn func(*Router), opts ...RouteOption) {
	fn(r.Group(prefix, opts...))
}

func joinPath(prefix, path string) string {
	if prefix == "" {
		return path
	}
	if path == "" {
		return prefix
	}
	return strings.TrimSuffix(prefix, "/") + "/" + strings.TrimPrefix(path, "/")
}

func (docs *Router) Route(method, path string, opt RouteOption) {
	pattern := fmt.Sprintf("%s %s", method, path)
	op := &Operation{}
	Docs().Add(pattern, func(o *spec.Operation) {
		op.Operation = o
		if opt != nil {
			opt(op)
		}
	})(routeEntry{method: method, path: path})

	if docs.group != nil {
		docs.group.HandleFunc(method, path, http.HandlerFunc(op.handler.ServeHTTP))
		return
	}

	gowok.Web.HandleFunc(method, path, http.HandlerFunc(op.handler.ServeHTTP))
}

func (docs *Router) Get(path string, opts RouteOption) {
	docs.Route(http.MethodGet, path, opts)
}

func (docs *Router) Post(path string, opts RouteOption) {
	docs.Route(http.MethodPost, path, opts)
}

func (docs *Router) Put(path string, opts RouteOption) {
	docs.Route(http.MethodPut, path, opts)
}

func (docs *Router) Patch(path string, opts RouteOption) {
	docs.Route(http.MethodPatch, path, opts)
}

func (docs *Router) Delete(path string, opts RouteOption) {
	docs.Route(http.MethodDelete, path, opts)
}
