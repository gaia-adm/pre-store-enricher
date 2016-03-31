/*
	TODO:
	- arrange code in a better way (better encapsulation)
	- tests
	- run go fmt, go vet, go doc as part of pipeline
*/

package main

import (
	"github.com/gaia-adm/pre-store-enricher/amqphandler"
	"github.com/gaia-adm/pre-store-enricher/log"
	"os"
	"os/signal"
	"syscall"
)

var logger = log.GetLogger("main")

func main() {

	//handle SIGINT or SIGTERM
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		logger.Info("Signal received in main: ", sig, ", going to shutdown dispacher")
		amqphandler.Dispatcher.ShutDown()
	}()

	err := amqphandler.Dispatcher.RunAmqp()

	if _, ok := err.(*amqphandler.ShutDownError); ok {
		logger.Info("exiting from main, due to shotdown: ", err)
	} else {
		logger.Error("exiting from main unexpectedly!: ", err)
	}
}
