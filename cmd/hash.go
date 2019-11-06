package cmd

import (
	"fmt"

	"github.com/howeyc/gopass"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
)

// hashCmd represents the hash command
var hashCmd = &cobra.Command{
	Use:   "hash",
	Short: "Hash a password for basic auth",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Password to hash (will be hidden): ")
		password, err := gopass.GetPasswd()
		if err != nil {
			return err
		}

		hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		fmt.Printf("Hashed password: %s\n", string(hash))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(hashCmd)
}
