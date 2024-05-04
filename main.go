package main

import (
	"cloudshell/internal/config"
	"cloudshell/internal/middleware"
	clog "cloudshell/pkg/log"
	"cloudshell/pkg/xtermjs"
	"cloudshell/ui"
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	LivenessPath  string = "/healthz"
	PprofPath     string = "/metric"
	ReadinessPath string = "/readyz"
	XtermPath     string = "/xterm.js"
)

func main() {
	// so I can access it everywhere within my computer
	_, filename, _, _ := runtime.Caller(0)
	projectDir := filepath.Dir(filename)

	wd, err := os.Getwd()
	if err != nil {
		message := fmt.Sprintf("failed to get working directory: %s", err)
		log.Fatal(message)
	}

	// initialise config
	configuration, err := config.Configuration(filepath.Join(projectDir, "./config.yaml"))
	if err != nil {
		message := fmt.Sprintf("failed to parse configuration: %s", err)
		log.Fatal(message)
	}

	// initialise the logger
	clog.Init(clog.Format(configuration.LogFormat), clog.Level(configuration.LogLevel))
	keepalivePingTimeout := time.Duration(configuration.KeepalivePingTimeout) * time.Second

	clog.Infof("working directory     : '%s'", wd)
	clog.Infof("command               : '%s'", configuration.Command)
	clog.Infof("arguments             : ['%s']", strings.Join(configuration.Arguments, "', '"))

	clog.Infof("allowed hosts         : ['%s']", strings.Join(configuration.AllowedHostnames, "', '"))
	clog.Infof("connection error limit: %v", configuration.ConnectionErrorLimit)
	clog.Infof("keepalive ping timeout: %v", keepalivePingTimeout)
	clog.Infof("max buffer size       : %v bytes", configuration.MaxBufferSizeBytes)
	clog.Infof("server address        : '%s' ", configuration.ServerAddress)
	clog.Infof("server port           : %v", configuration.Port)

	clog.Infof("liveness checks path  : '%s'", LivenessPath)
	clog.Infof("readiness checks path : '%s'", ReadinessPath)
	clog.Infof("metrics endpoint path : '%s'", PprofPath)
	clog.Infof("xtermjs endpoint path : '%s'", XtermPath)

	// configure routing
	router := mux.NewRouter()

	// this is the endpoint for xterm.js to connect to
	xtermjsHandlerOptions := xtermjs.HandlerOpts{
		AllowedHostnames:     configuration.AllowedHostnames,
		Arguments:            configuration.Arguments,
		Command:              configuration.Command,
		ConnectionErrorLimit: int(configuration.ConnectionErrorLimit),
		CreateLogger: func(connectionUUID string, r *http.Request) xtermjs.Logger {
			middleware.CreateRequestLog(r, map[string]interface{}{"connection_uuid": connectionUUID}).Infof("created logger for connection '%s'", connectionUUID)
			return middleware.CreateRequestLog(nil, map[string]interface{}{"connection_uuid": connectionUUID})
		},
		KeepalivePingTimeout: keepalivePingTimeout,
		MaxBufferSizeBytes:   configuration.MaxBufferSizeBytes,
	}
	router.HandleFunc(XtermPath, xtermjs.GetHandler(xtermjsHandlerOptions))

	// readiness probe endpoint
	router.HandleFunc(ReadinessPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// liveness probe endpoint
	router.HandleFunc(LivenessPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// metrics endpoint
	router.Handle(PprofPath, promhttp.Handler())

	// this is the endpoint for serving xterm.js assets
	router.PathPrefix("/assets").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ui.ServeAsset(w, r)
	})

	// this is the endpoint for the root path aka website
	public, err := fs.Sub(ui.StaticFS, "public")
	if err != nil {
		message := fmt.Sprintf("failed to get public embedded dir: %s", err)
		log.Fatal(message)
	}
	router.PathPrefix("/").Handler(http.FileServer(http.FS(public)))

	// start memory logging pulse
	logWithMemory := middleware.CreateMemoryLog()
	go func(tick *time.Ticker) {
		for {
			logWithMemory.Debug("tick")
			<-tick.C
		}
	}(time.NewTicker(time.Second * 60))

	// listen
	listenOnAddress := fmt.Sprintf("%s:%v", configuration.ServerAddress, configuration.Port)
	server := http.Server{
		Addr:    listenOnAddress,
		Handler: middleware.AddIncomingRequestLogging(router),
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	defer stop()

	clog.Infof("starting server on interface:port '%s'...", listenOnAddress)
	go func() {
		if err := server.ListenAndServe(); err != nil {
			clog.Infof("listen and serve returned err: %v\n", err)
		}
	}()

	<-ctx.Done()
	log.Println("got interruption signal")
	if err := server.Shutdown(context.TODO()); err != nil { // Use here context with a required timeout
		clog.Infof("server shutdown returned an err: %v\n", err)
	}

	clog.Info("server exit")
}
