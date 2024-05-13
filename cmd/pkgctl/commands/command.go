package commands

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/commands/pkgcmd"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/commands/repocmd"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/commands/secretcmd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
)

const (
	defaultConfigFileSubDir  = "pkgctl"
	defaultConfigFileName    = "pkgctl"
	defaultConfigFileNameExt = "yaml"
	defaultConfigEnvPrefix   = "PKGCTL"
	//defaultDBPath            = "package_db"
)

var (
	configFile string
)

func GetMain(ctx context.Context) *cobra.Command {
	//showVersion := false
	cmd := &cobra.Command{
		Use:          "pkgctl",
		Short:        "pkgctl is a cli tool for pkg management",
		Long:         "pkgctl is a cli tool for pkg management",
		SilenceUsage: true,
		// We handle all errors in main after return from cobra so we can
		// adjust the error message coming from libraries
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// initialize viper settings
			initConfig()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := cmd.Flags().GetBool("help")
			if err != nil {
				return err
			}
			if h {
				return cmd.Help()
			}
			return cmd.Usage()
		},
	}

	pf := cmd.PersistentFlags()

	kubeflags := genericclioptions.NewConfigFlags(true)
	kubeflags.AddFlags(pf)

	var k8s bool
	cmd.Flags().BoolVarP(&k8s, "k8s", "k", false, "when set to true execute the command on the k8s cluster, otherwise it is execute locally")

	kubeflags.WrapConfigFn = func(rc *rest.Config) *rest.Config {
		rc.UserAgent = fmt.Sprintf("pkg/%s", version)
		return rc
	}

	// ensure the viper config directory exists
	cobra.CheckErr(os.MkdirAll(path.Join(xdg.ConfigHome, defaultConfigFileSubDir), 0700))
	// initialize viper settings
	initConfig()

	cmd.AddCommand(pkgcmd.GetCommand(ctx, version, kubeflags, k8s))
	cmd.AddCommand(secretcmd.GetCommand(ctx, version, kubeflags, k8s))
	cmd.AddCommand(repocmd.GetCommand(ctx, version, kubeflags, k8s))
	//cmd.PersistentFlags().StringVar(&configFile, "config", "c", fmt.Sprintf("Default config file (%s/%s/%s.%s)", xdg.ConfigHome, defaultConfigFileSubDir, defaultConfigFileName, defaultConfigFileNameExt))

	return cmd
}

type Runner struct {
	Command *cobra.Command
	//Ctx     context.Context
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if configFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(configFile)
	} else {

		viper.AddConfigPath(filepath.Join(xdg.ConfigHome, defaultConfigFileSubDir))
		viper.SetConfigType(defaultConfigFileNameExt)
		viper.SetConfigName(defaultConfigFileName)

		_ = viper.SafeWriteConfig()
	}

	//viper.Set("kubecontext", kubecontext)
	//viper.Set("kubeconfig", kubeconfig)

	viper.SetEnvPrefix(defaultConfigEnvPrefix)
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		_ = 1
	}
}
