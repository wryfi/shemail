package logging

import (
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"os"
	"strings"
)

var Logger zerolog.Logger

// ConfigureLogger sets up an instance of zerolog that is configured with
// relevant settings (log level and output format) taken from our viper
// configuration registry.
func ConfigureLogger() {
	level := viper.GetString("log.level")
	pretty := viper.GetBool("log.pretty")
	if strings.ToLower(level) == "debug" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else if strings.ToLower(level) == "info" {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else if strings.ToLower(level) == "error" {
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}
	if pretty {
		output := zerolog.ConsoleWriter{Out: os.Stderr}
		Logger = zerolog.New(output).With().Timestamp().Logger()
	} else {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	}
}
