/*
	TODO:
	- SIGTERM
	- circleCI
	- send / receive
	- Json processing
	- log
	- http server
	- tests
 */
package main

import (
	"os/signal"
	"os"
	"syscall"
	"fmt"
)

func main() {

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println("Signal received:", sig)
		done <- true
	}()

	fmt.Println("awaiting signal")
	<-done
	fmt.Println("exiting")
}
