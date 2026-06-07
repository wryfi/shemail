package config

import (
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/wryfi/shemail/logging"
	"path/filepath"
	"strings"
)

// CfgFile is an optional explicit path to a configuration file; when set (e.g.
// via the --config flag) it overrides the default search paths.
var CfgFile string

// GetHome uses the homedir library to get the user's HOME directory in a
// cross-platform way.
func GetHome() string {
	home, err := homedir.Dir()
	if err != nil {
		// The terminating .Msg() is required: without it the event is never
		// dispatched, so Fatal neither logs nor exits and we would fall
		// through returning an empty home directory.
		log.Fatal().Err(err).Msg("failed to determine home directory")
	}
	return home
}

// setDefaults sets default values for configuration keys. All configuration
// values for the application should be defined here.
func setDefaults() {
	viper.SetDefault("log.level", "warn")
	viper.SetDefault("log.pretty", false)
	viper.SetDefault("timezone", "America/Los_Angeles")
}

// InitConfig initializes the viper configuration by reading the defaults set
// above and combining them with configuration file (if any) and environment
// variable values. CLI flags can be bound to viper where they are defined.
// The code below looks for configuration files in ~/.local/etc/shemail.(json|yaml)
// and /etc/shemail.(json|yaml) and reads environment variables prefixed with SHEMAIL__
// e.g. SHEMAIL__LOG__LEVEL="debug".
func InitConfig() {
	setDefaults()
	if CfgFile != "" {
		viper.SetConfigFile(CfgFile)
	} else {
		viper.SetConfigName("shemail")
		localconf := filepath.Join(GetHome(), ".local", "etc")
		viper.AddConfigPath(localconf)
		viper.AddConfigPath("/etc")
	}
	err := viper.ReadInConfig()
	if err != nil {
		log.Warn().Msgf("no configuration file will be used: %s", err)
	}
	viper.SetEnvPrefix("SHEMAIL_")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	viper.AutomaticEnv()
	logging.ConfigureLogger()
}
