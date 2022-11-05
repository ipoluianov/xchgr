package main

import (
	"github.com/ipoluianov/gomisc/logger"
	"github.com/ipoluianov/xchgr/app"
)

func main() {
	logger.InitNearExe()

	app.ServiceName = "xchgr"
	app.ServiceDisplayName = "Xchg router"
	app.ServiceDescription = "Xchg router"
	app.ServiceRunFunc = app.RunAsServiceF
	app.ServiceStopFunc = app.StopServiceF

	if !app.TryService() {
		app.RunConsole()
	}
}
