/*
	TODO:
	- send / receive
	- Json processing
	- http server
	- tests
*/
package main

import (
	"github.com/Sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

var log = logrus.New()

func init() {

	//set log level according to env var
	logLevel := os.Getenv("PSE_LOG_LEVEL")
	if logLevel != "" {
		level, err := logrus.ParseLevel(logLevel)
		if err != nil {
			log.Error("Failed to set logger level: ", err)
		} else {
			log.Level = level
		}
	} else {
		log.Level = logrus.DebugLevel
	}
}

func main() {

	sigs := make(chan os.Signal)
	done := make(chan bool)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs

		log.Warn("Signal received: ", sig)
		done <- true
	}()

	log.Info("awaiting signal")
	<-done
	log.Warn("exiting")
}
