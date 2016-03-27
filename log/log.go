package log

import (
	"github.com/Sirupsen/logrus"
	"os"
)

var log = logrus.New()
var logger = GetLogger("log")

func init() {

	//set log level according to env var
	logLevel := os.Getenv("PSE_LOG_LEVEL")
	if logLevel != "" {
		level, err := logrus.ParseLevel(logLevel)
		if err != nil {
			logger.Error("Failed to set logger level: ", err)
		} else {
			log.Level = level
		}
	} else {
		log.Level = logrus.DebugLevel
	}

}

func GetLogger(context string) *logrus.Entry {
	return log.WithFields(logrus.Fields{"context": context})
}
