package amqphandler

import (
	"github.com/gaia-adm/pre-store-enricher/log"
	"github.com/streadway/amqp"
)

const (
	consumeExchangeName           = "events-to-enrich"
	consumeDeadLetterExchangeName = "events-to-enrich.deadletter"
	consumeQueueName              = "events-to-enrich-q"
	consumeDeadLetterQueueName    = "events-to-enrich-q.deadletter"
	consumerRoutingKey            = "event.#"
)

var initCLogger = log.GetLogger("initconsume")

//initForConsume connects to amqp in a separate goroutine defines exchange and queue to get
//the messages from (also dead letter exchange and queue) and send back the connection on the
//returned channel
func initForConsume(shutdownRequested chan struct{}) (readyToConsume chan InitResult)  {
	readyToConsume = make(chan InitResult)
	go func() {
		conn, channel, err := connAndChannel(shutdownRequested, initCLogger)

		//Something went wrong, probably shutdown requested
		if err != nil {
			readyToConsume <- InitResult{conn, err}
			return
		}

		err = declareExchange(consumeExchangeName, channel, initCLogger)
		if err != nil {
			readyToConsume <- InitResult{conn, err}
			return
		}

		err = declareExchange(consumeDeadLetterExchangeName, channel, initCLogger)
		if err != nil {
			readyToConsume <- InitResult{conn, err}
			return
		}

		if declareAndBindQueue(consumeQueueName, consumeExchangeName, conn, channel, amqp.Table{"x-dead-letter-exchange": consumeDeadLetterExchangeName}, readyToConsume) != nil {
			return
		}
		if declareAndBindQueue(consumeDeadLetterQueueName, consumeDeadLetterExchangeName, conn, channel, nil, readyToConsume) != nil {
			return
		}

		readyToConsume <- InitResult{conn, nil}
	}()

	return readyToConsume
}

func declareAndBindQueue(queueName string, exchangeName string, conn *amqp.Connection, channel *amqp.Channel, extraArgs amqp.Table, readyToConsume chan<- InitResult) (err error) {
	_, err = channel.QueueDeclare(
		queueName, // name of the queue
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // noWait
		extraArgs, // arguments
	)
	if err != nil {
		initCLogger.Error(" error when declaring Queue: ", queueName, ", error msg: ", err)
		readyToConsume <- InitResult{conn, err}
		return err
	} else {
		initCLogger.Info("declared queue: ", queueName)
	}

	err = channel.QueueBind(
		queueName,          // name of the queue
		consumerRoutingKey, // bindingKey
		exchangeName,       // sourceExchange
		false,              // noWait
		nil,                // arguments
	)
	if err != nil {
		initCLogger.Error(" error when binding Queue: ", queueName, ", to exchange: ", exchangeName, ", using routing key: ", consumerRoutingKey, ", error msg: ", err)
		readyToConsume <- InitResult{conn, err}
		return err
	} else {
		initCLogger.Info("binded queue: ", queueName, ", to exchange: ", exchangeName, ", using routing key: ", consumerRoutingKey)
		return nil
	}

}
