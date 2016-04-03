package amqphandler

import (
	"errors"
	"github.com/gaia-adm/pre-store-enricher/amqpinit"
	"github.com/gaia-adm/pre-store-enricher/log"
	"github.com/streadway/amqp"
	"runtime"
	"sync"
)

type dispatcher struct {
	closedOnShutdown chan struct{}
	consumeConn      *amqp.Connection
	sendConn         *amqp.Connection
	processors       []*Processor
}

var Dispatcher dispatcher
var dispatcherLogger = log.GetLogger("dispacher")

func init() {
	Dispatcher.closedOnShutdown = make(chan struct{})
}

func (d *dispatcher) RunAmqp() (err error) {

	//Connect to rabbit with two separate connections (one for consumers and one for senders
	readyToConsume := amqpinit.InitForConsume(d.closedOnShutdown)
	readyToSend := amqpinit.InitForSend(d.closedOnShutdown)

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

	//We span processors as the number of go max procs
	//GOMAXPROCS can be set using env var from outside
	//runtime.GOMAXPROCS(0) queries the value and does not change it
	var wg sync.WaitGroup
	numOfProcessors := runtime.GOMAXPROCS(0)
	d.processors = make([]*Processor, numOfProcessors)
	for i := 0; i < numOfProcessors; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			p := NewProcessor(index, amqpinit.ConsumeQueueName, amqpinit.SendExchangeName)
			d.processors[index] = p
			p.startConsume(d.consumeConn, d.sendConn)
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

			return &amqpinit.ShutDownError{}
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
