package cmd

import (
	app "github.com/qjoly/argocd-plugin-helm-envsubst/internal"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(generateCmd)
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "take a template, substitute env vars and output the result",
	Run: func(cmd *cobra.Command, args []string) {
		app.NewGenerator().Generate()
	},
}
