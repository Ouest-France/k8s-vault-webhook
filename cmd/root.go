package cmd

import (
	"fmt"
	"os"

	"github.com/Ouest-France/k8s-vault-webhook/api"
	"github.com/Ouest-France/k8s-vault-webhook/vault"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:          "k8s-vault-webhook",
	Short:        "A kubernetes webhook to get secrets from Hashicorp Vault",
	SilenceUsage: true,
	PreRunE: func(cmd *cobra.Command, args []string) error {

		// Checks all required params are set
		required := []string{"cert", "key", "vault-addr", "vault-token", "vault-backend"}
		for _, param := range required {
			if viper.GetString(param) == "" {
				return fmt.Errorf("required parameter %q is not defined", param)
			}
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

		server := api.Server{
			Listen:       viper.GetString("listen"),
			Cert:         viper.GetString("cert"),
			Key:          viper.GetString("key"),
			Vault:        vc,
			VaultBackend: viper.GetString("vault-backend"),
			VaultPattern: viper.GetString("vault-pattern"),
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

	rootCmd.Flags().StringP("listen", "l", ":8443", "HTTPS Port to listen [$KVW_LISTEN]")
	rootCmd.Flags().StringP("cert", "c", "", "HTTPS certificate file (required) [$KVW_CERT]")
	rootCmd.Flags().StringP("key", "k", "", "HTTPS key file (required) [$KVW_KEY]")
	rootCmd.Flags().StringP("vault-addr", "v", "", "Vault address (required) [$KVW_VAULT-ADDR]")
	rootCmd.Flags().StringP("vault-token", "t", "", "Vault token path (required) [$KVW_VAULT-TOKEN]")
	rootCmd.Flags().StringP("vault-backend", "b", "", "Vault secret backend path (required) [$KVW_VAULT-BACKEND]")
	rootCmd.Flags().StringP("vault-pattern", "p", "{{namespace}}", "Vault search pattern [$KVW_VAULT-PATTERN]")

	flags := []string{"listen", "cert", "key", "vault-addr", "vault-token", "vault-backend", "vault-pattern"}
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
