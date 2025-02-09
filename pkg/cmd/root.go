package cmd

import (
	"os"

	"github.com/heathcliff26/valkey-keepalived/pkg/config"
	failoverclient "github.com/heathcliff26/valkey-keepalived/pkg/failover-client"
	"github.com/heathcliff26/valkey-keepalived/pkg/version"
	"github.com/spf13/cobra"
)

const (
	flagNameConfig = "config"
	flagNameEnv    = "env"
)

func NewRootCommand() *cobra.Command {
	cobra.AddTemplateFunc(
		"ProgramName", func() string {
			return version.Name
		},
	)

	rootCmd := &cobra.Command{
		Use:   version.Name,
		Short: version.Name + " failover a group of valkey databases based on a virtual ip",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := cmd.Flags().GetString(flagNameConfig)
			if err != nil {
				return err
			}

			env, err := cmd.Flags().GetBool(flagNameEnv)
			if err != nil {
				return err
			}

			return run(cfg, env)
		},
	}

	rootCmd.Flags().StringP(flagNameConfig, "c", "", "Path to config file")
	err := rootCmd.MarkFlagFilename(flagNameConfig, "yaml", "yml")
	if err != nil {
		rootCmd.PrintErrln("Fatal: " + err.Error())
		os.Exit(1)
	}

	rootCmd.Flags().Bool(flagNameEnv, false, "Expand enviroment variables in config file")

	rootCmd.AddCommand(
		version.NewCommand(),
	)

	return rootCmd
}

func Execute() {
	cmd := NewRootCommand()
	err := cmd.Execute()
	if err != nil {
		cmd.PrintErrln("Fatal: " + err.Error())
		os.Exit(1)
	}
}

func run(configPath string, env bool) error {
	cfg, err := config.LoadConfig(configPath, env)
	if err != nil {
		return err
	}

	failoverclient.NewFailoverClient(cfg.Valkey).Run()
	return nil
}
