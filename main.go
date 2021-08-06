package main

import (
	_ "expvar"
	_ "net/http/pprof"

	"github.com/byxorna/jot/cmd"
)

var (
	pprofPort = 6060
)

// Uncomment this if you want to use pprof on :6060
//func init() {
//	fmt.Printf("Listening for pprof on :%d\n", pprofPort)
//	go http.ListenAndServe(fmt.Sprintf(":%d", pprofPort), nil)
//}

func main() {
	cmd.Execute()
}
