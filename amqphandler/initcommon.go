package amqphandler

import (
	"github.com/Sirupsen/logrus"
	"github.com/streadway/amqp"
)

type InitResult struct {
	Connection *amqp.Connection
	Err        error
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
