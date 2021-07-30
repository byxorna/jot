package main

import (
	_ "expvar"
	"fmt"
	_ "net/http/pprof"

	"net/http"

	"github.com/byxorna/jot/cmd"
)

var (
	pprofPort = 6060
)

func init() {
	fmt.Printf("Listening for pprof on :%d\n", pprofPort)
	go http.ListenAndServe(fmt.Sprintf(":%d", pprofPort), nil)
}

func main() {
	cmd.Execute()
}
