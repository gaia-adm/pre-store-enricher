package amqphandler

import (
	"errors"
	"github.com/gaia-adm/pre-store-enricher/log"
	"github.com/streadway/amqp"
	"sync"
	"runtime"
)

type dispatcher struct {
	closedOnShutdown  chan struct{}
	consumeConn       *amqp.Connection
	sendConn          *amqp.Connection
	processors        []*Processor
}

type ShutDownError struct {
}

func (s *ShutDownError) Error() string {
	return "shutdown requested"
}

var Dispatcher dispatcher
var dispatcherLogger = log.GetLogger("dispacher")

func init() {
	Dispatcher.closedOnShutdown = make(chan struct{})
}

func (d *dispatcher) RunAmqp() (err error) {

	//Connect to rabbit with two separate connections (one for consumers and one for senders
	readyToConsume := initForConsume(d.closedOnShutdown)
	readyToSend := initForSend(d.closedOnShutdown)

	//Wait for initForConsume to complete init
	result := <-readyToConsume
	if result.Err != nil {
		return result.Err
	} else {
		dispatcherLogger.Info("amqp consume connection is ready")
		d.consumeConn = result.Connection
	}

	//Wait for initForSend to complete init
	result = <-readyToSend
	if result.Err != nil {
		return result.Err
	} else {
		dispatcherLogger.Info("amqp send connection is ready")
		d.sendConn = result.Connection
	}

	var wg sync.WaitGroup

	//We span processors as the number of go max procs
	//GOMAXPROCS can be set using env var from outside
	//runtime.GOMAXPROCS(0) queries the value and does not change it
	numOfProcessors := runtime.GOMAXPROCS(0)
	d.processors = make([]*Processor, numOfProcessors)
	for i := 0; i < numOfProcessors; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			p := Processor{consumedQueue: consumeQueueName, sentToExchange: sendExchangeName}
			d.processors[index] = &p
			p.startConsume(d.consumeConn, d.sendConn, index)
		}(i)
	}
	wg.Wait()

	dispatcherLogger.Info("all processors exit, going to check why...")

	//check if we exit due to shutdown
	select {
	case _, ok := <-d.closedOnShutdown:
		if ok == false {
			//we exit due to shutdown as the shutdownRequested channel was closed
			dispatcherLogger.Info("all processors shuted down, going to close connections")

			if err := d.consumeConn.Close(); err != nil {
				dispatcherLogger.Errorf("AMQP consume connection close error: %s. Still exiting...", err)
			} else {
				dispatcherLogger.Info("successfully closed the consume connection")
			}

			if err := d.sendConn.Close(); err != nil {
				dispatcherLogger.Errorf("AMQP send connection close error: %s. Still exiting...", err)
			} else {
				dispatcherLogger.Info("successfully closed the send connection")
			}

			return &ShutDownError{}
		}
	default:
		//Abnormal termination
	}

	dispatcherLogger.Error("all processors exit due to abnormal termination")
	return errors.New("all processors exit due to abnormal termination")
}

func (d *dispatcher) ShutDown() {
	dispatcherLogger.Info("shutting down amqp")
	//broadcast to all that listen to shutdownRequested
	close(d.closedOnShutdown)

	//shutting down the processors
	for _, p := range d.processors {
		if p != nil {
			go func(proc *Processor) {
				proc.shutdown()
			}(p)
		}
	}
}
