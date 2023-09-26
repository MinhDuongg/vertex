package router

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vertex-center/vertex/pkg/ginutils"
)

type KernelRouter struct {
	server *http.Server
	engine *gin.Engine
}

func NewKernelRouter() KernelRouter {
	gin.SetMode(gin.ReleaseMode)

	r := KernelRouter{
		engine: gin.New(),
	}

	r.engine.Use(ginutils.ErrorHandler())
	r.engine.Use(ginutils.Logger("KERNEL"))
	r.engine.Use(gin.Recovery())
	r.engine.GET("/ping", handlePing)

	r.initAdapters()
	r.initServices()
	r.initAPIRoutes()

	return r
}

func (r *KernelRouter) Start() error {
	r.server = &http.Server{
		Addr:    ":6131",
		Handler: r.engine,
	}

	return r.server.ListenAndServe()
}

func (r *KernelRouter) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := r.server.Shutdown(ctx)
	if err != nil {
		return err
	}

	r.server = nil
	return nil
}

func (r *KernelRouter) initAdapters() {
	// TODO: Implement
}

func (r *KernelRouter) initServices() {
	// TODO: Implement
}

func (r *KernelRouter) initAPIRoutes() {
	// TODO: Implement
}
