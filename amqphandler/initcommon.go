package amqphandler

import (
	"github.com/Sirupsen/logrus"
	"github.com/streadway/amqp"
)

type InitResult struct {
	Connection *amqp.Connection
	Err        error
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

func declareExchange(exchangeName string, channel *amqp.Channel, logger *logrus.Entry) (err error) {
	err = channel.ExchangeDeclare(
		exchangeName, // name of the exchange
		"topic",      // type
		true,         // durable
		false,        // delete when complete
		false,        // internal
		false,        // noWait
		nil,          // arguments
	)
	if err != nil {
		logger.Error(" error when declaring Exchange: ", exchangeName, ", error msg: ", err)
		return err
	} else {
		logger.Info("declared exchange: ", exchangeName)
		return nil
	}
}
