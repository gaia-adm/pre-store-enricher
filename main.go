/*
	TODO:
	- multiple processors
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

	sigs := make(chan os.Signal)
	signalReceived := make(chan struct{})

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs

		logger.Warn("Signal received in main: ", sig)
		signalReceived <- struct{}{}
	}()

	amqpError := make(chan error)
	amqphandler.Dispatcher.RunAmqp(amqpError)
	logger.Info("awaiting signal in main")

	select {
	case <-signalReceived:
		amqphandler.Dispatcher.ShutDown()
		logger.Warn("exiting from main due to signal")
	case err := <-amqpError:
		logger.Error("exiting! amqp throwed an error: ", err)
	}

}
