#!/bin/bash

#running rabbit and pse
docker run -d -p 5672:5672 -p 15672:15672 -e RABBITMQ_USER="myuser" -e RABBITMQ_PASS="mypass" --name rabbitmq tutum/rabbitmq
docker run -d -e PSE_AMQP_URL="amqp://myuser:mypass@rabbitmq:5672" --link rabbitmq:rabbitmq --name pse gaiaadm/pre-store-enricher

#creating dummy queue (test-pse-q) and bind it to event-to-index exchange
curl --fail -S -i -u myuser:mypass -H "content-type:application/json" -XPUT -d'{"vhost":"/","name":"test-pse-q","durable":"true","auto_delete":"false","arguments":{}}' http://localhost:15672/api/queues/%2F/test-pse-q
curl --fail -S -i -u myuser:mypass -H "content-type:application/json" -XPOST -d'{"vhost":"/","source":"events-to-index","destination_type":"q","destination":"test-pse-q","routing_key":"event.#","arguments":{}}' http://localhost:15672/api/bindings/%2F/e/events-to-index/q/test-pse-q

#send message to events-to-enrich exchange
curl --fail -S -i -u myuser:mypass -H "content-type:application/json" -XPOST -d'{"vhost":"/","name":"events-to-enrich","properties":{"delivery_mode":2,"headers":{}},"routing_key":"event.testme","delivery_mode":"2","payload":"{\"key\": \"value\"}","headers":{},"props":{},"payload_encoding":"string"}' http://localhost:15672/api/exchanges/%2f/events-to-enrich/publish

#test that the message reached the dummy queue with the enrichment of gaia section
curl --fail -S -i -u myuser:mypass -H "content-type:application/json" -XPOST -d'{"vhost":"/","name":"test-pse-q","truncate":"50000","requeue":"true","encoding":"auto","count":"1"}' http://localhost:15672/api/queues/%2F/test-pse-q/get | grep gaia | grep incoming_time2
