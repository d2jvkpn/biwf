package bpb

import (
	"github.com/gin-gonic/gin"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
	. "x/src/bpa"
)

func Node(args *Args) {
	var (
		err error
		nds *NDS
		ndx *NDX
	)

	nds, ndx = new(NDS), new(NDX)
	nds.NDX, ndx.NDS = ndx, nds
	nds.Node, nds.Port, nds.DataPath = args.Node, args.Port, args.DataPath
	nds.Created, nds.JobMap = time.Now(), new(sync.Map)
	nds.Active, nds.Engine = true, gin.Default()

	ErrExit(os.MkdirAll(nds.DataPath, 0755))

	nds.SetAPI()
	go nds.Engine.Run(":" + strconv.Itoa(nds.Port))

	rpc.Register(ndx)
	rpc.HandleHTTP()
	nds.Listener, err = net.Listen("tcp", ":"+strconv.Itoa(nds.Node))
	ErrExit(err)
	go http.Serve(nds.Listener, nil)

	log.Printf("Node RPC service start: %d, %s\n", nds.Node, nds.DataPath)

	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	<-signalChannel
	log.Println("Received os.Interrupt or syscall.SIGTERM")
	nds.Stop()
}
