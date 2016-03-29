package amqphandler

import (
	"github.com/gaia-adm/pre-store-enricher/log"
	"github.com/streadway/amqp"
)

type dispatcher struct {
	shutdownRequested chan struct{}
	shutdownCompleted chan struct{}
	consumeConn       *amqp.Connection
	sendConn          *amqp.Connection
}

var Dispatcher dispatcher
var dispatcherLogger = log.GetLogger("dispacher")

func init() {
	Dispatcher.shutdownRequested = make(chan struct{})
	Dispatcher.shutdownCompleted = make(chan struct{})
}

func (d *dispatcher) RunAmqp(errorOccurred chan<- error) {

	go func() {
		//Connect to rabbit with two separate connections (one for consumers and one for senders
		readyToConsume := initForConsume(d.shutdownRequested)
		readyToSend := initForSend(d.shutdownRequested)

		//Wait for initForConsume to complete init
		select {
		case <-d.shutdownRequested:
			dispatcherLogger.Info("shutdown requested from RunAmqp (before validating consume), exiting")
			d.shutdownCompleted <- struct{}{}
			return
		case result := <-readyToConsume:
			if result.Err != nil {
				dispatcherLogger.Error("failed to init amqp connection for consume: ", result.Err)
				errorOccurred <- result.Err
			} else {
				dispatcherLogger.Info("ready for consume")
				d.consumeConn = result.Connection
			}
		}

		//Wait for initForSend to complete init
		select {
		case <-d.shutdownRequested:
			dispatcherLogger.Info("shutdown requested from RunAmqp (after validating init consume before validating send), exiting")
			d.shutdownCompleted <- struct{}{}
			return
		case result := <-readyToSend:
			if result.Err != nil {
				dispatcherLogger.Error("failed to init amqp connection for send: ", result.Err)
				errorOccurred <- result.Err
			} else {
				dispatcherLogger.Info("ready for send")
				d.sendConn = result.Connection
			}
		}

		p := Processor{consumedQueue: consumeQueueName, sentToExchange: sendExchangeName}
		p.startConsume(d.consumeConn, d.sendConn)

		<-d.shutdownRequested
		dispatcherLogger.Info("shutdown requested, going to cancel the consume channels and close the connections")
		p.shutdown()

		if err := d.consumeConn.Close(); err != nil {
			dispatcherLogger.Errorf("AMQP consume connection close error: %s. Still exiting...", err)
		} else {
			dispatcherLogger.Info("successfully closed the consume connection")
		}

		if err := d.sendConn.Close(); err != nil {
			dispatcherLogger.Errorf("AMQP send onnection close error: %s. Still exiting...", err)
		} else {
			dispatcherLogger.Info("successfully closed the send connection")
		}

		d.shutdownCompleted <- struct{}{}
	}()

}

func (d *dispatcher) ShutDown() {
	dispatcherLogger.Info("shutting down amqp (closing the shutdown channel)")
	//broadcast to all that listen to shutdownRequested
	close(d.shutdownRequested)
	//wait for RunAmqp to notify that shutdown was completed
	<-d.shutdownCompleted
}
