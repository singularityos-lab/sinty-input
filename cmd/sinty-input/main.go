package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/godbus/dbus/v5"

	"github.com/singularityos-lab/sinty-input/internal/cli"
	"github.com/singularityos-lab/sinty-input/internal/registry"
	"github.com/singularityos-lab/sinty-input/internal/service"
	"github.com/singularityos-lab/sinty-input/internal/userdict"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "dict" {
		os.Exit(cli.Dict(os.Args[2:], os.Stdout, os.Stderr))
	}
	log.SetFlags(0)
	log.SetPrefix("sinty-input: ")
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Fatal(err)
	}
	engines := registry.Engines()
	svc, err := service.Export(conn, engines)
	if err != nil {
		log.Fatal(err)
	}
	svc.Rebuild = registry.Engines
	reply, err := conn.RequestName(service.Name, dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		log.Fatalf("%s is taken", service.Name)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	log.Printf("serving %s with %d engines", service.Name, len(engines))
	<-ctx.Done()
	userdict.FlushAll()
}
