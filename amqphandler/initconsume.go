package amqphandler

import (
	"github.com/Sirupsen/logrus"
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

func initForConsume(readyToConsume chan<- InitResult, shutdownRequested chan struct{}) {
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
}

func connAndChannel(shutdownRequested chan struct{}, logger *logrus.Entry) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := initRabbitConn(shutdownRequested, logger)

	//Something went wrong, probably shutdown requested
	if err != nil {
		return conn, nil, err
	}

	go func() {
		logger.Info("closing conn: %s", <-conn.NotifyClose(make(chan *amqp.Error)))
	}()

	logger.Info("got Connection, getting Channel")
	channel, err := conn.Channel()
	if err != nil {
		logger.Error("error when creating a channel: ", err)
		return conn, channel, err
	}

	return conn, channel, nil
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
