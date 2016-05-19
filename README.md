# Preface
Enrich the event with data pre storing it into the DB.

The service consumes the events from amqp exchange **events-to-enrich** and publishs the enriched events into amqp exchange **events-to-index**

**Today we enrich the event with two fields (that are always exist):**

1. **gaia_time** - Contains the time when the event riched pre-store-enricher

2. **event_time** - In case that rabbitmq header **tdField** was presented and contained a vaild path, **event_time** will contain the field value of that path. If the field was a number, we assume it is Unix time and convert it into RFC3339 (without the millisec part if it was presented). If it was a string we assume it is already a date in a valid format as copy it as is. If we do not find the path or tsField header was not presented - **event_time** will contain the exact time as described in **gaia_time**


Here is an example of event post enrichment (the **"gaia"** json part has the enrichement info. **gaia_time** contains the arrival time to pre-store-enricher and **event_time** was extracted according to **tdField Rabbitmq header** that contained the path: **info.time**):
```javascript
{
  "gaia":
    {"gaia_time":"2016-05-19T12:42:40Z", 
     "event_time":"2016-04-17T13:29:38+03:00"},
  //event properties
  "key":"value" 
  "info": 
    { "time": 1460888978 }
}
```

# Building the service
The service is written in [golang](https://golang.org/)

If you want to manipulate the service but do not know golang it is mostly recommanded to follow at least the **Installing Go** and **Learning Go** sections in [go doc site](https://golang.org/doc/). Put extra attention to [How to write Go code](https://golang.org/doc/code.html) 

The service can be packed into **Docker image**, There is a [Dockerfile](https://github.com/gaia-adm/pre-store-enricher/blob/master/Dockerfile) that contains instructions for building the image. To build the image simply run 
```docker
docker build -t myimagetag .
```

# Running the service
There are few env vars you need to be aware of:
```
PSE_LOG_LEVEL (by deafault debug, mostly adviced to change to info on production env)
```

```
PSE_AMQP_URL (for instance: amqp://admin:admin@172.17.8.101:5672)
```

```
GOMAXPROCS (Go env var, contains by default the number of cpu cores in the machine, We span the number of processors according to this env var)
```

In [circle.yml](https://github.com/gaia-adm/pre-store-enricher/blob/master/circle.yml) and in the [fleet unit](https://github.com/gaia-adm/pre-store-enricher/blob/master/pre-store-enricher.service) you can find examples for how to run the docker image

# Implementation details
On startup the service creates two separate amqp connections on a two separate gorutines, one for consumers and one for producers as adviced in [amqp go client lib](https://godoc.org/github.com/streadway/amqp):
*"Note: RabbitMQ will rather use TCP pushback on the network connection instead of sending basic.flow. This means that if a single channel is producing too much on the same connection, all channels using that connection will suffer, including acknowledgments from deliveries. Use different Connections if you desire to interleave consumers and producers in the same process to avoid your basic.ack messages from getting rate limited with your basic.publish messages."*

Even if amqp is down, the service will keep trying to connect every 5 sec until the connection is established or someone shutdowned the service (pre-store-enricher service)

We define the amqp exchanges and queues (we also define dead letter exchange and queue for events-to-enrich so on Nacks the events won't disappear but will be kept in the dead letter queue - **event-to-enrich-q.deadletter**)

Then we span processors according the "GOMAXPROCS" env var, each processor runs two goroutines - one for consume (managed by the amqp lib) and one for produce (managed by pre-store-enricher).

The service sends Nacks for events that it failed to process for any reason, but does not support confirms on produce.
The events sent to the next exchange as "persist" events

The service supports "graceful shutdown" (listen to SIGTERM and SIGINT) and also knows to shutdown automaticlly in case that all of the processors die (The processor can die in case that amqp connection is broken for instance)

# Testing

We have unit test [processor_test.go](https://github.com/gaia-adm/pre-store-enricher/blob/master/amqphandler/processor_test.go) to check the enrichment logic

We also have system test [SystemTest.sh](https://github.com/gaia-adm/pre-store-enricher/blob/master/SystemTest.sh) to check the end-to-end flow (from queue to queue)

Both tests are executed during building the service in [CircleCI](https://circleci.com/gh/gaia-adm/pre-store-enricher)
