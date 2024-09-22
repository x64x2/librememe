package main

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var serveCmd *cobra.Command

func init() {
	var serveFlags struct {
		Config  string
		Verbose bool
		Quiet   bool
	}

	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start a server to browse content",
		Run: func(cmd *cobra.Command, args []string) {
			log.SetOutput(os.Stdout)
			switch {
			case serveFlags.Verbose:
				log.SetLevel(log.DebugLevel)
			case serveFlags.Quiet:
				log.SetLevel(log.WarnLevel)
			default:
				log.SetLevel(log.InfoLevel)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			err := config.LoadGlobal(serveFlags.Config)
			if err != nil {
				log.Fatalf("Failed to load global config: %v", err)
			}

			err = fs.InitShared(ctx)
			if err != nil {
				log.Fatalf("Failed to create storage: %v", err)
			}

			err = db.Connect(config.Global.GetBool(config.KeyServerReadOnly))
			if err != nil {
				log.Fatalf("Failed to connect to database: %v", err)
			}

			err = db.Migrate(ctx)
			if err != nil {
				log.Fatalf("Failed to perform DB migrations: %v", err)
			}

			server.StartServer()
		},
	}
	serveCmd.Flags().StringVarP(&serveFlags.Config, "config", "c", "", "config file to load (optional)")
	serveCmd.Flags().BoolVarP(&serveFlags.Verbose, "verbose", "v", false, "verbose mode - logs more info")
	serveCmd.Flags().BoolVarP(&serveFlags.Quiet, "quiet", "q", false, "quiet mode - suppress most logs")
}
