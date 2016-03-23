package amqpconnector

import (
	"os"
	"time"
	"github.com/gaia-adm/pre-store-enricher/log"
	"github.com/streadway/amqp"
)

var pseAmqpUrl = "amqp://guest:guest@localhost:5672"

func init() {
	pseAmqpUrlEnvVar := os.Getenv("PSE_AMQP_URL")
	if (pseAmqpUrlEnvVar != "") {
		pseAmqpUrl = pseAmqpUrlEnvVar
	}
}

func InitRabbitConn(connected chan<- *amqp.Connection) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for ; ; <-ticker.C {
		conn, err := amqp.Dial(pseAmqpUrl)
		if err != nil {
			log.Log.Warn("failed to connect to rabbit (", pseAmqpUrl, "): ", err)
			log.Log.Warn("trying to reconnect to rabbit in 5 seconds...")
			continue
		} else {
			log.Log.Info("successfully connected to rabbit (", pseAmqpUrl, ")")
		}
		connected <- conn
		return
	}
}