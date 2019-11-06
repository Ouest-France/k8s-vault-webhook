package cmd

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/Ouest-France/k8s-vault-webhook/api"
	"github.com/Ouest-France/k8s-vault-webhook/vault"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:          "k8s-vault-webhook",
	Short:        "A kubernetes webhook to get secrets from Hashicorp Vault",
	SilenceUsage: true,
	PreRunE: func(cmd *cobra.Command, args []string) error {

		// Checks all required params are set
		required := []string{"cert", "key", "vault-addr", "vault-token"}
		for _, param := range required {
			if viper.GetString(param) == "" {
				return fmt.Errorf("required parameter %q is not defined", param)
			}
		}

		// Check logformat
		validLogformat := func() bool {
			for _, validFormat := range []string{"text", "json"} {
				if viper.GetString("logformat") == validFormat {
					return true
				}
			}
			return false
		}()
		if !validLogformat {
			return fmt.Errorf("logformat is '%s', must be 'text' or 'json'", viper.GetString("logformat"))
		}

		// Check basicauth
		validBasicauth := func() bool {
			re := regexp.MustCompile(`^.+:.+$`)

			for _, userpass := range viper.GetStringSlice("basicauth") {
				if !re.MatchString(userpass) {
					return false
				}
			}
			return true
		}()
		if !validBasicauth {
			return errors.New("basicauth entries must match '^.+:.+$' regex")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		vc, err := vault.NewClient(
			viper.GetString("vault-addr"),
			viper.GetString("vault-token"),
		)
		if err != nil {
			return fmt.Errorf("failed to create new vault client: %s", err)
		}

		// Setup logger
		logger := logrus.New()
		level, err := logrus.ParseLevel(viper.GetString("loglevel"))
		if err != nil {
			return fmt.Errorf("failed to parse loglevel: %s", err)
		}
		logger.SetLevel(level)
		formatter := map[string]logrus.Formatter{
			"text": &logrus.TextFormatter{},
			"json": &logrus.JSONFormatter{FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyFunc:  "caller",
			}},
		}
		logger.SetFormatter(formatter[viper.GetString("logformat")])

		server := api.Server{
			Listen:       viper.GetString("address"),
			Cert:         viper.GetString("cert"),
			Key:          viper.GetString("key"),
			Vault:        vc,
			VaultPattern: viper.GetString("vault-pattern"),
			Logger:       logger,
			BasicAuth:    viper.GetStringSlice("basicauth"),
		}

		return server.Serve()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Flags().StringP("address", "a", ":8443", "HTTPS Port to listen [$KVW_LISTEN]")
	rootCmd.Flags().StringP("cert", "c", "", "HTTPS certificate file (required) [$KVW_CERT]")
	rootCmd.Flags().StringP("key", "k", "", "HTTPS key file (required) [$KVW_KEY]")
	rootCmd.Flags().StringP("vault-addr", "v", "", "Vault address (required) [$KVW_VAULT-ADDR]")
	rootCmd.Flags().StringP("vault-token", "t", "", "Vault token path (required) [$KVW_VAULT-TOKEN]")
	rootCmd.Flags().StringP("vault-pattern", "p", "{{namespace}}", "Vault search pattern [$KVW_VAULT-PATTERN]")
	rootCmd.Flags().StringP("loglevel", "l", "info", "Webhook loglevel [$KVW_LOGLEVEL]")
	rootCmd.Flags().StringP("logformat", "f", "text", "Webhook logformat (text or json) [$KVW_LOGFORMAT]")
	rootCmd.Flags().StringSliceP("basicauth", "b", []string{}, "Basic auth list of user:hashed_pass [$KVW_BASICAUTH]")

	flags := []string{"address", "cert", "key", "vault-addr", "vault-token", "vault-pattern", "loglevel", "logformat", "basicauth"}
	for _, flag := range flags {
		err := viper.BindPFlag(flag, rootCmd.Flags().Lookup(flag))
		if err != nil {
			fmt.Printf("failed to bind %q flag: %s\n", flag, err)
			os.Exit(1)
		}
	}
}

func initConfig() {
	viper.SetEnvPrefix("kvw")
	viper.AutomaticEnv()
}
