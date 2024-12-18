package cmd

import (
	app "github.com/qjoly/argocd-plugin-helm-envsubst/internal"
	"github.com/spf13/cobra"
)

var (
	buildPath                    string
	repositoryConfigPath         string
	helmRegistrySecretConfigPath string
)

func init() {
	buildCmd.PersistentFlags().StringVar(&buildPath, "path", "", "Path to the application")
	buildCmd.PersistentFlags().StringVar(&repositoryConfigPath, "repository-path", "", "Repository config, default to /helm-working-dir/")
	buildCmd.PersistentFlags().StringVar(&helmRegistrySecretConfigPath, "helm-registry-secret-config-path", "", "Repository config, default to /helm-working-dir/plugin-repositories/repositories.yaml")
	rootCmd.AddCommand(buildCmd)
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Similar to helm dependency build",
	Run: func(cmd *cobra.Command, args []string) {
		app.NewBuilder().Build(buildPath, repositoryConfigPath, helmRegistrySecretConfigPath)
	},
}
