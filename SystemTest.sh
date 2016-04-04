#!/bin/bash

#We assume that rabbitmq and pse services are already running (check out circleci.yml for that)

#creating dummy queue (test-pse-q) and bind it to event-to-index exchange
curl --fail -S -i -u admin:admin -H "content-type:application/json" -XPUT -d'{"vhost":"/","name":"test-pse-q","durable":"true","auto_delete":"false","arguments":{}}' http://localhost:15672/api/queues/%2F/test-pse-q
curl --fail -S -i -u admin:admin -H "content-type:application/json" -XPOST -d'{"vhost":"/","source":"events-to-index","destination_type":"q","destination":"test-pse-q","routing_key":"event.#","arguments":{}}' http://localhost:15672/api/bindings/%2F/e/events-to-index/q/test-pse-q

#send message to events-to-enrich exchange
curl --fail -S -i -u admin:admin -H "content-type:application/json" -XPOST -d'{"vhost":"/","name":"events-to-enrich","properties":{"delivery_mode":2,"headers":{}},"routing_key":"event.testme","delivery_mode":"2","payload":"{\"key\": \"value\"}","headers":{},"props":{},"payload_encoding":"string"}' http://localhost:15672/api/exchanges/%2f/events-to-enrich/publish

#test that the message reached the dummy queue with the enrichment of gaia section
curl --fail -S -i -u admin:admin -H "content-type:application/json" -XPOST -d'{"vhost":"/","name":"test-pse-q","truncate":"50000","requeue":"true","encoding":"auto","count":"1"}' http://localhost:15672/api/queues/%2F/test-pse-q/get | grep gaia | grep incoming_time2
