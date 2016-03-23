/*
	TODO:
	- send / receive
	- Json processing
	- http server
	- tests
*/
package main

import (
	"os"
	"os/signal"
	"syscall"
	"github.com/streadway/amqp"
	"github.com/gaia-adm/pre-store-enricher/log"
	"github.com/gaia-adm/pre-store-enricher/amqpconnector"
)


func main() {

	sigs := make(chan os.Signal)
	done := make(chan bool)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs

		log.Log.Warn("Signal received: ", sig)
		done <- true
	}()

	connectedToRabbit := make(chan *amqp.Connection)
	go amqpconnector.InitRabbitConn(connectedToRabbit)

	log.Log.Info("awaiting signal")
	<-done
	log.Log.Warn("exiting")
}