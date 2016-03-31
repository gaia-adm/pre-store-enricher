package amqphandler

import (
	"github.com/Sirupsen/logrus"
	"github.com/streadway/amqp"
	"os"
	"time"
)

var pseAmqpUrl = "amqp://admin:admin@172.17.8.101:5672"

func init() {
	pseAmqpUrlEnvVar := os.Getenv("PSE_AMQP_URL")
	if pseAmqpUrlEnvVar != "" {
		pseAmqpUrl = pseAmqpUrlEnvVar
	}
}

func initRabbitConn(closedOnShutdown chan struct{}, logger *logrus.Entry) (conn *amqp.Connection, err error) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			conn, err := amqp.Dial(pseAmqpUrl)
			if err != nil {
				logger.Warn("failed to connect to rabbit (", pseAmqpUrl, "): ", err)
				logger.Warn("trying to reconnect to rabbit in 5 seconds...")
				continue
			} else {
				logger.Info("successfully connected to rabbit (", pseAmqpUrl, ")")
			}

			return conn, nil

		case <-closedOnShutdown:
			logger.Info("shutdown requested from initRabbitConn, exiting")
			return nil, &ShutDownError{}
		}
	}
}
