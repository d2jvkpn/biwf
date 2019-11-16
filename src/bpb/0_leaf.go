package bpb

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	. "x/src/bpa"
)

func Leaf(args *Args) {
	var (
		err     error
		lfs     *LFS
		timeout <-chan time.Time
	)

	// initialize
	lfs, err = CreateLeaf(args)
	ErrExit(err)

	log.Println(lfs.Name, "start")

	shutdown := make(chan struct{})
	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	if lfs.Timeout > 0 {
		timeout = time.After(lfs.Timeout)
	}

	go func() {
		select {
		case <-signalChannel:
			by, reply := "local", ""
			lfs.LFX.Leave(&by, &reply)
			lfs.Ch <- [2]string{by, "stop"}
		case <-timeout:
			by, reply := "timeout", ""
			lfs.LFX.Leave(&by, &reply)
			lfs.Ch <- [2]string{by, "stop"}
		case <-shutdown:
			return
		}
	}()

	for {
		msg := <-lfs.Ch
		switch msg[1] {
		case "start", "recover":
			go lfs.Run(msg)

		case "stop":
			lfs.Stop(msg[0])
			log.Printf("%s stopped by %s", lfs.Name, msg[0])

		default:
			tmp := "unexpected message recevied: %s, %s"
			lfs.Record("!!!", fmt.Sprintf(tmp, msg[0], msg[1]))
		}

		if msg[1] == "stop" {
			break
		}
	}

	close(shutdown)
}
