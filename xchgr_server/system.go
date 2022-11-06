package xchgr_server

type System struct {
	port       int
	router     *Router
	httpServer *HttpServer
}

func NewSystem(port int) *System {
	var c System
	c.port = port
	c.router = NewRouter()
	c.httpServer = NewHttpServer()
	return &c
}

func (c *System) Start() {
	c.router.Start()
	c.httpServer.Start(c.router, c.port)
}

func (c *System) Stop() {
	c.httpServer.Stop()
	c.router.Stop()
}
