package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/godbus/dbus/v5"

	"github.com/singularityos-lab/sinty-input/internal/cli"
	"github.com/singularityos-lab/sinty-input/internal/ibus"
	"github.com/singularityos-lab/sinty-input/internal/registry"
	"github.com/singularityos-lab/sinty-input/internal/service"
	"github.com/singularityos-lab/sinty-input/internal/userdict"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "dict":
			os.Exit(cli.Dict(os.Args[2:], os.Stdout, os.Stderr))
		case "ibus-engines":
			ibus.EnginesXML(os.Stdout, registry.Engines())
			return
		case "ibus":
			serveIBus()
			return
		}
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

func serveIBus() {
	log.SetFlags(0)
	log.SetPrefix("sinty-input: ")
	addr, err := ibus.Address()
	if err != nil {
		log.Fatal(err)
	}
	conn, err := dbus.Connect(addr)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := ibus.Serve(conn, registry.New); err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	select {
	case <-ctx.Done():
	case <-closed(conn):
	}
	userdict.FlushAll()
}

func closed(conn *dbus.Conn) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		<-conn.Context().Done()
		close(ch)
	}()
	return ch
}
