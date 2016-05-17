package amqphandler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/gaia-adm/pre-store-enricher/log"
	"github.com/jmoiron/jsonq"
	"github.com/streadway/amqp"
	"strings"
	"time"
)

const (
	consumerTag = "enricher-consumer"
	//use for getting the field location in the event that represent the event time
	eventTimeJsonpathHeaderName = "tsField"
)

type Processor struct {
	consumeChannel *amqp.Channel
	sendChannel    *amqp.Channel
	consumedQueue  string
	sentToExchange string
	processLogger  *logrus.Entry
	processorId    int
}

func NewProcessor(processorId int, consumeQueue string, sendExchange string) *Processor {
	p := Processor{consumedQueue: consumeQueue, sentToExchange: sendExchange, processorId: processorId}
	p.processLogger = log.GetLogger(fmt.Sprintf("%s%d", "processor", processorId))
	return &p
}

func (p *Processor) startConsume(consumeConn *amqp.Connection, sendConn *amqp.Connection) (err error) {

	p.processLogger.Info("getting amqpChannel from consume connection")
	p.consumeChannel, err = consumeConn.Channel()
	if err != nil {
		p.processLogger.Error("error when creating a consume amqpChannel: ", err)
		return err
	}

	go func() {
		p.processLogger.Infof("consume amqpChannel closed: %s", <-p.consumeChannel.NotifyClose(make(chan *amqp.Error)))
	}()

	p.sendChannel, err = sendConn.Channel()
	if err != nil {
		p.processLogger.Error("error when creating a send amqpChannel: ", err)
		return err
	}

	go func() {
		p.processLogger.Infof("send amqpChannel closed: %s", <-p.sendChannel.NotifyClose(make(chan *amqp.Error)))
	}()

	p.processLogger.Info("starting Consume messages from amqp using tag: ", consumerTag)
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
		p.processLogger.Error("error when trying to conume message from queue: ", p.consumedQueue)
		return err
	}

	p.processDeliveries(p.sendChannel, deliveries)

	return nil
}

func (p *Processor) shutdown() (err error) {

	if err := p.consumeChannel.Cancel(consumerTag, false); err != nil {
		p.processLogger.Errorf("Consumer channel cancel failed: %s", err)
		return err
	} else {
		p.processLogger.Infof("Sent Cancel to consume amqpChannel to drain the deliveries")
		return nil
	}
	//No need to Cancel the send channel. Cancel is used only to drain the deliveries chan
}

func (p *Processor) processDeliveries(sendChannel *amqp.Channel, deliveries <-chan amqp.Delivery) {

	for d := range deliveries {

		p.processLogger.Debugf(
			"got msg with length %dB delivery: [%v] %q",
			len(d.Body),
			d.DeliveryTag,
			d.Body,
		)

		eventTimeFieldLocation, _ := d.Headers[eventTimeJsonpathHeaderName].(string)
		jsonToSend, err := p.enrichMessage(&d.Body, eventTimeFieldLocation)

		if err != nil {
			p.processLogger.Error("failed to enrich msg, sending Nack to amqp and continue to next msg")
			d.Nack(false, false) //we use dead letter, no need to requeue
			continue
		}

		p.processLogger.Debug("marshalled the msg successfully after enriching it")

		p.processLogger.Debugf(
			"after enriching the msg, length is %dB, body is: %q",
			len(*jsonToSend),
			*jsonToSend)

		err = sendChannel.Publish(
			p.sentToExchange, // publish to an exchange
			d.RoutingKey,     // routing to 0 or more queues
			false,            // mandatory
			false,            // immediate
			amqp.Publishing{
				Headers:         amqp.Table{},
				ContentType:     "application/json",
				ContentEncoding: "",
				Body:            *jsonToSend,
				DeliveryMode:    amqp.Persistent, // 2=persistent
				Priority:        0,               // 0-9
			})

		if err != nil {
			p.processLogger.Error("failed to send to exchange: ", p.sentToExchange, " Nacking the msg, error is: ", err)
			d.Nack(false, false) //we use dead letter, no need to requeue
			continue
		}

		p.processLogger.Debug("published the msg successfully to exchange: ", p.sentToExchange)

		d.Ack(false) // Ack, msg sent successfully
	}

	p.processLogger.Info("deliveries channel closed")
}

func (p *Processor) enrichMessage(in *[]byte, eventTimeFieldLocation string) (out *[]byte, err error) {

	jsonDecoder := json.NewDecoder(bytes.NewReader(*in))
	jsonDecoder.UseNumber() //To avoid conversion number to float to avoid data conversion when marshaling back the json

	var f interface{}
	err = jsonDecoder.Decode(&f)

	if err != nil {
		p.processLogger.Error("failed to unmarshal msg, error is:", err)
		return nil, err
	}

	p.processLogger.Debug("unmarshalled the msg successfully")

	//Adding fields
	eventMap := f.(map[string]interface{})
	gaiaMap := make(map[string]interface{})
	timeNow := time.Now().Format(time.RFC3339)
	gaiaMap["gaia_time"] = timeNow
	gaiaMap["event_time"] = extractEventTime(eventMap, eventTimeFieldLocation, timeNow)
	eventMap["gaia"] = &gaiaMap

	jsonToSend, err := json.Marshal(eventMap)
	if err != nil {
		p.processLogger.Error("failed to marshal msg, error is:", err)
		return nil, err
	}

	return &jsonToSend, nil
}

//let's try to extract the event time, if we fail to do so, event time will be set to now()
func extractEventTime(eventMap map[string]interface{}, eventTimeFieldLocation string, timeNow string) (extractedEventTime string) {
	extractedEventTime = timeNow

	if eventTimeFieldLocation != "" {
		jq := jsonq.NewQuery(eventMap)
		path := convertFieldLocationToSlice(eventTimeFieldLocation)
		strVal, err := jq.String(path...)
		if err == nil {
			//the value is string format.
			//we assume it's a date in a string format so we can pass it as is
			extractedEventTime = strVal
		} else {
			_, err := jq.Int(path...)
			if err == nil {
				//If it's a number we assume it's milli seconds or seconds from 1970.
				//We will try to convert it to a string formated date
				//TODO: write a code to convert the number to string
				extractedEventTime = "needToWriteACodeToConvert"
			}
			//It's not a string nor a number - hence we will use the default now()
		}
	}

	return extractedEventTime
}

func convertFieldLocationToSlice(fieldLocation string) (fieldLocationSlice []string) {

	f := func(c rune) bool {
		return c == '.' || c == '[' || c == ']'
	}
	return strings.FieldsFunc(fieldLocation, f)
}
