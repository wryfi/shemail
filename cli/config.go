package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Account represents an email account configuration
type Account struct {
	Name     string      `yaml:"name"`
	User     string      `yaml:"user"`
	Password SecretValue `yaml:"password"`
	Server   string      `yaml:"server"`
	Port     int         `yaml:"port"`
	TLS      bool        `yaml:"tls"`
	Default  bool        `yaml:"default"`
	Purge    bool        `yaml:"purge"`
}

// Config represents the root configuration structure
type Config struct {
	Log struct {
		Level  string `yaml:"level"`
		Pretty bool   `yaml:"pretty"`
	} `yaml:"log"`
	Accounts []Account `yaml:"accounts"`
}

// SecretValue is a custom type that obfuscates its value when marshaled to YAML
type SecretValue string

// MarshalYAML implements the yaml.Marshaler interface
func (s SecretValue) MarshalYAML() (interface{}, error) {
	if s == "" {
		return nil, nil
	}
	return "********", nil
}

// ConfigurationCommand returns a cobra command for reading the application's
// configuration and writing to stdout for inspection in yaml format.
func ConfigurationCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "config",
		Short:   "Print shemail configuration",
		Aliases: []string{"cf", "cfg"},
		Long:    `Prints all of shemail's known configuration values`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile := viper.ConfigFileUsed()
			if configFile == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "no configuration file in use")
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "configuration file: %s\n", configFile)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "---")

			// Get all settings from viper
			settings := viper.AllSettings()

			// Create a Config struct
			var config Config

			// Convert map to yaml bytes
			settingsYaml, err := yaml.Marshal(settings)
			if err != nil {
				return err
			}

			// Unmarshal into our custom Config struct
			if err := yaml.Unmarshal(settingsYaml, &config); err != nil {
				return err
			}

			// Marshal the Config struct back to YAML
			out, err := yaml.Marshal(&config)
			if err != nil {
				return err
			}

			cmd.Printf("%s\n", out)
			return nil
		},
	}
}
