package amqphandler

import (
	"github.com/streadway/amqp"
	"github.com/gaia-adm/pre-store-enricher/log"
	"encoding/json"
	"time"
)

const (
	consumerTag = "enricher-consumer"
)
var processLogger = log.GetLogger("processor")

type Processor struct {
	consumeConn       *amqp.Connection
	sendConn          *amqp.Connection
	consumedQueue     string
	sentToExchange    string
}

func (p *Processor) startConsume() (err error){

	processLogger.Info("getting Channel from consume connection")
	channel, err := p.consumeConn.Channel()
	if err != nil {
		processLogger.Error("error when creating a consume channel: ", err)
		return err
	}

	sendChannel, err := p.sendConn.Channel()
	if err != nil {
		processLogger.Error("error when creating a send channel: ", err)
		return err
	}

	processLogger.Info("starting Consume using tag: ", consumerTag)
	deliveries, err := channel.Consume(
		p.consumedQueue, // name
		consumerTag,      // consumerTag,
		false,      //  noAutoAck
		false,      // exclusive
		false,      // noLocal
		false,      // noWait
		nil,        // arguments
	)
	if err != nil {
		processLogger.Error("error when trying to conume message from queue: ", p.consumedQueue)
		return err
	}

	for d := range deliveries {

		processLogger.Debugf(
			"got msg with length %dB delivery: [%v] %q",
			len(d.Body),
			d.DeliveryTag,
			d.Body,
		)

		var f interface{}
		err := json.Unmarshal(d.Body, &f)
		if err != nil {
			processLogger.Error("failed to unmarshal msg, sending Nack to amqp, error is:", err)
			d.Nack(false, false) //we use dead letter, no need to requeue
			continue
		}

		processLogger.Debug("unmarshalled the msg successfully")

		m := f.(map[string]interface{})
		gaiaMap := make(map[string]interface{})
		t := time.Now()
		gaiaMap["incoming_time"] = t.Format(time.RFC3339)
		m["gaia"] = &gaiaMap
		jsonToSend, err := json.Marshal(m)
		if err != nil {
			processLogger.Error("failed to marshal msg, sending Nack to amqp, error is:", err)
			d.Nack(false, false) //we use dead letter, no need to requeue
			continue
		}

		processLogger.Debug("marshalled the msg successfully after enriching it")

		err = sendChannel.Publish(
			p.sentToExchange,   // publish to an exchange
			d.RoutingKey, // routing to 0 or more queues
			false,      // mandatory
			false,      // immediate
			amqp.Publishing{
				Headers:         amqp.Table{},
				ContentType:     "application/json",
				ContentEncoding: "",
				Body:            jsonToSend,
				DeliveryMode:    amqp.Persistent, // 2=persistent
				Priority:        0,              // 0-9
			})

		if err != nil {
			processLogger.Error("failed to send to exchange: ", p.sentToExchange, " Nacking the msg, error is: ", err)
			d.Nack(false, false) //we use dead letter, no need to requeue
			continue
		}

		processLogger.Debug("published the msg successfully to exchange: ", p.sentToExchange)

		d.Ack(false) // Ack, msg sent successfully
	}
	processLogger.Warn("deliveries channel closed")

	return nil
}
