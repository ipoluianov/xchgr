package xchgr_server

type System struct {
	router     *Router
	httpServer *HttpServer
}

func NewSystem() *System {
	var c System
	c.router = NewRouter()
	c.httpServer = NewHttpServer()
	return &c
}

func (c *System) Start() {
	c.router.Start()
	c.httpServer.Start(c.router)
}

func (c *System) Stop() {
	c.httpServer.Stop()
	c.router.Stop()
}
