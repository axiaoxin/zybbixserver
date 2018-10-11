package lib

import (
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func InitLog() {
	log.SetOutput(os.Stdout)

	logLevel, err := log.ParseLevel(strings.ToLower(viper.GetString("log_level")))
	if err != nil {
		log.Error("Log level parse failed:", err)
	}
	log.SetLevel(logLevel)

	if strings.ToLower(viper.GetString("log_formatter")) == "json" {
		log.SetFormatter(&log.JSONFormatter{})
		log.Debugf("log %s with json formatter", logLevel)
	} else {
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
		log.Debugf("log %s with text formatter", logLevel)
	}
}
