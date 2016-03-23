package log

import (
	"github.com/Sirupsen/logrus"
	"os"
)

var Log = logrus.New()

func init() {

	//set log level according to env var
	logLevel := os.Getenv("PSE_LOG_LEVEL")
	if logLevel != "" {
		level, err := logrus.ParseLevel(logLevel)
		if err != nil {
			Log.Error("Failed to set logger level: ", err)
		} else {
			Log.Level = level
		}
	} else {
		Log.Level = logrus.DebugLevel
	}
}

