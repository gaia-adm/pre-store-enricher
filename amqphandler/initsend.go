package amqphandler

import (
	"github.com/gaia-adm/pre-store-enricher/log"
)

const (
	sendExchangeName           = "events-to-index"
)

var initSLogger = log.GetLogger("initsend")

func initForSend(readyToConsume chan<- InitResult, shutdownRequested chan struct{}) {
	go func() {
		return
	}()
}
