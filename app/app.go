package app

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ipoluianov/gomisc/logger"
	"github.com/ipoluianov/xchgr/xchgr_server"
	"github.com/kardianos/osext"
	"github.com/kardianos/service"
)

var ServiceName string
var ServiceDisplayName string
var ServiceDescription string
var ServiceRunFunc func() error
var ServiceStopFunc func()

func SetAppPath() {
	exePath, _ := osext.ExecutableFolder()
	err := os.Chdir(exePath)
	if err != nil {
		return
	}
}

func init() {
	SetAppPath()
}

func TryService() bool {
	serviceFlagPtr := flag.Bool("service", false, "Run as service")
	installFlagPtr := flag.Bool("install", false, "Install service")
	uninstallFlagPtr := flag.Bool("uninstall", false, "Uninstall service")
	startFlagPtr := flag.Bool("start", false, "Start service")
	stopFlagPtr := flag.Bool("stop", false, "Stop service")

	flag.Parse()

	if *serviceFlagPtr {
		runService()
		return true
	}

	if *installFlagPtr {
		InstallService()
		return true
	}

	if *uninstallFlagPtr {
		UninstallService()
		return true
	}

	if *startFlagPtr {
		StartService()
		return true
	}

	if *stopFlagPtr {
		StopService()
		return true
	}

	return false
}

func NewSvcConfig() *service.Config {
	var SvcConfig = &service.Config{
		Name:        ServiceName,
		DisplayName: ServiceDisplayName,
		Description: ServiceDescription,
	}
	SvcConfig.Arguments = append(SvcConfig.Arguments, "-service")
	return SvcConfig
}

func InstallService() {
	fmt.Println("Service installing")
	prg := &program{}
	s, err := service.New(prg, NewSvcConfig())
	if err != nil {
		log.Fatal(err)
	}
	err = s.Install()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Service installed")
}

func UninstallService() {
	fmt.Println("Service uninstalling")
	prg := &program{}
	s, err := service.New(prg, NewSvcConfig())
	if err != nil {
		log.Fatal(err)
	}
	err = s.Uninstall()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Service uninstalled")
}

func StartService() {
	fmt.Println("Service starting")
	prg := &program{}
	s, err := service.New(prg, NewSvcConfig())
	if err != nil {
		log.Fatal(err)
	}
	err = s.Start()
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("Service started")
	}
}

func StopService() {
	fmt.Println("Service stopping")
	prg := &program{}
	s, err := service.New(prg, NewSvcConfig())
	if err != nil {
		log.Fatal(err)
	}
	err = s.Stop()
	if err != nil {
		log.Fatal(err)
		return
	} else {
		fmt.Println("Service stopped")
	}
}

func runService() {
	prg := &program{}
	s, err := service.New(prg, NewSvcConfig())
	if err != nil {
		log.Fatal(err)
	}
	err = s.Run()
	if err != nil {
		logger.Error(err)
	}
}

type program struct{}

func (p *program) Start(_ service.Service) error {
	return ServiceRunFunc()
}

func (p *program) Stop(_ service.Service) error {
	ServiceStopFunc()
	return nil
}

// ///////////////////////////
var system *xchgr_server.System

var systems []*xchgr_server.System

func Start() error {
	logger.Println("[i]", "App::Start", "begin")
	TuneFDs()

	real := true
	if real {
		system = xchgr_server.NewSystem(8084)
		system.Start()
	} else {
		systems = make([]*xchgr_server.System, 0)
		for i := 8084; i < 8087; i++ {
			system = xchgr_server.NewSystem(i)
			system.Start()
			systems = append(systems, system)
		}
	}

	logger.Println("[i]", "App::Start", "end")

	return nil
}

func Stop() {
	system.Stop()
}

func RunConsole() {
	logger.Println("[i]", "App::RunConsole", "begin")
	err := Start()
	if err != nil {
		logger.Println("[ERROR]", "App::RunConsole", "Start error:", err)
		return
	}
	_, _ = fmt.Scanln()
	logger.Println("[i]", "App::RunConsole", "end")
}

func RunAsServiceF() error {
	return Start()
}

func StopServiceF() {
	Stop()
}
