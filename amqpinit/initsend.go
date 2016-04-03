package amqpinit

import (
	"github.com/gaia-adm/pre-store-enricher/log"
)

const (
	SendExchangeName = "events-to-index"
)

var initSLogger = log.GetLogger("initsend")

//initForSend connects to amqp in a separate goroutine define a exchange to send
//the messages to and send back the connection on the returned channel
func InitForSend(closedOnShutdown chan struct{}) (readyToSend chan InitResult) {
	readyToSend = make(chan InitResult)
	go func() {
		conn, channel, err := connAndChannel(closedOnShutdown, initSLogger)

		//Something went wrong, probably shutdown requested
		if err != nil {
			readyToSend <- InitResult{conn, err}
			return
		}

		err = declareExchange(SendExchangeName, channel, initSLogger)
		if err != nil {
			readyToSend <- InitResult{conn, err}
			return
		}

		readyToSend <- InitResult{conn, nil}
	}()

	return readyToSend
}
