package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/godbus/dbus/v5"

	"github.com/singularityos-lab/sinty-input/internal/bijoy"
	"github.com/singularityos-lab/sinty-input/internal/engine"
	"github.com/singularityos-lab/sinty-input/internal/hangul"
	"github.com/singularityos-lab/sinty-input/internal/kana"
	"github.com/singularityos-lab/sinty-input/internal/pinyin"
	"github.com/singularityos-lab/sinty-input/internal/service"
	"github.com/singularityos-lab/sinty-input/internal/simlish"
	"github.com/singularityos-lab/sinty-input/internal/telex"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("sinty-input: ")
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Fatal(err)
	}
	engines := []engine.Engine{pinyin.New(), kana.New(), hangul.New(), telex.New(), bijoy.New(), simlish.New()}
	if _, err := service.Export(conn, engines); err != nil {
		log.Fatal(err)
	}
	reply, err := conn.RequestName(service.Name, dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		log.Fatalf("%s is taken", service.Name)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	log.Printf("serving %s with %d engines", service.Name, len(engines))
	<-ctx.Done()
}
