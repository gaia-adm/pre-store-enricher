package amqphandler

import (
	"encoding/json"
	"github.com/gaia-adm/pre-store-enricher/log"
	"github.com/streadway/amqp"
	"time"
)

const (
	consumerTag = "enricher-consumer"
)

var processLogger = log.GetLogger("processor")

type Processor struct {
	consumeChannel *amqp.Channel
	sendChannel    *amqp.Channel
	consumedQueue  string
	sentToExchange string
	done           chan struct{}
}

func (p *Processor) startConsume(consumeConn *amqp.Connection, sendConn *amqp.Connection) (err error) {

	processLogger.Info("getting Channel from consume connection")
	p.consumeChannel, err = consumeConn.Channel()
	if err != nil {
		processLogger.Error("error when creating a consume channel: ", err)
		return err
	}

	p.sendChannel, err = sendConn.Channel()
	if err != nil {
		processLogger.Error("error when creating a send channel: ", err)
		return err
	}

	processLogger.Info("starting Consume using tag: ", consumerTag)
	deliveries, err := p.consumeChannel.Consume(
		p.consumedQueue, // name
		consumerTag,     // consumerTag,
		false,           //  noAutoAck
		false,           // exclusive
		false,           // noLocal
		false,           // noWait
		nil,             // arguments
	)
	if err != nil {
		processLogger.Error("error when trying to conume message from queue: ", p.consumedQueue)
		return err
	}

	p.done = make(chan struct{})
	go p.processDeliveries(p.sendChannel, deliveries, p.done)

	return nil
}

func (p *Processor) shutdown() (err error) {

	if err := p.consumeChannel.Cancel(consumerTag, false); err != nil {
		processLogger.Errorf("Consumer channel cancel failed: %s", err)
		return err
	}

	//No need to Cancel the send channel. Cancel is used only to drain the deliveries chan
	//we wait to deliveries chan to drain before returning from the function
	<-p.done
	processLogger.Info("deliveries chan drained, exiting from shutdown function")
	return nil
}

func (p *Processor) processDeliveries(sendChannel *amqp.Channel, deliveries <-chan amqp.Delivery, done chan<- struct{}) {

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
			p.sentToExchange, // publish to an exchange
			d.RoutingKey,     // routing to 0 or more queues
			false,            // mandatory
			false,            // immediate
			amqp.Publishing{
				Headers:         amqp.Table{},
				ContentType:     "application/json",
				ContentEncoding: "",
				Body:            jsonToSend,
				DeliveryMode:    amqp.Persistent, // 2=persistent
				Priority:        0,               // 0-9
			})

		if err != nil {
			processLogger.Error("failed to send to exchange: ", p.sentToExchange, " Nacking the msg, error is: ", err)
			d.Nack(false, false) //we use dead letter, no need to requeue
			continue
		}

		processLogger.Debug("published the msg successfully to exchange: ", p.sentToExchange)

		d.Ack(false) // Ack, msg sent successfully
	}

	processLogger.Info("deliveries channel closed")
	done <- struct{}{} //inform that we exit
}
