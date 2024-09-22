package main

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

)

var importCmd *cobra.Command

func init() {
	var importFlags struct {
		Config      string
		ForceRescan bool
		Verbose     bool
		Quiet       bool
	}

	importCmd = &cobra.Command{
		Use:   "import",
		Short: "Import from various sources",
	}
	importCmd.PersistentFlags().StringVarP(&importFlags.Config, "config", "c", "", "config file to load (optional)")
	importCmd.PersistentFlags().BoolVar(&importFlags.ForceRescan, "force-rescan", false, "force complete rescan of all posts")
	importCmd.PersistentFlags().BoolVarP(&importFlags.Verbose, "verbose", "v", false, "verbose mode - logs more info")
	importCmd.PersistentFlags().BoolVarP(&importFlags.Quiet, "quiet", "q", false, "quiet mode - suppress most logs")

	var importexhentaiFlags struct {
		Profile []string
	}

	importexhentai := func(site string) func(cmd *cobra.Command, args []string) {
		return func(cmd *cobra.Command, args []string) {
			log.SetOutput(os.Stdout)
			switch {
			case importFlags.Verbose:
				log.SetLevel(log.DebugLevel)
			case importFlags.Quiet:
				log.SetLevel(log.WarnLevel)
			default:
				log.SetLevel(log.InfoLevel)
			}

			var profilesConfigKey string
			switch site {
			case "exhentai":
				profilesConfigKey = config.KeyexhentaiProfiles
			case "fansly":
				profilesConfigKey = config.KeyexhentaiFanslyProfiles
			default:
				panic("invalid service name: " + site)
			}

			err := config.LoadGlobal(importFlags.Config)
			if err != nil {
				fatalErr(err, "Failed to load global config")
			}

			if len(importexhentaiFlags.Profile) == 0 {
				importexhentaiFlags.Profile = config.Global.GetStringSlice(profilesConfigKey)
			}

			if len(importexhentaiFlags.Profile) == 0 {
				log.Fatalf("No profile to import: list them in the configuration '%s', or specify one or more profiles with the '--user' ('-u') flag", profilesConfigKey)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			/*termCh := make(chan os.Signal, 1)
			signal.Notify(termCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-termCh
				cancel()
			}()*/

			err = fs.InitShared(ctx)
			if err != nil {
				fatalErr(err, "Failed to create storage")
			}

			err = db.Connect(false)
			if err != nil {
				fatalErr(err, "Failed to connect to database")
			}

			err = db.Migrate(ctx)
			if err != nil {
				fatalErr(err, "Failed to perform DB migrations")
			}

			downloader.SharedWorker = downloader.NewDownloadWorker()
			err = downloader.SharedWorker.Start(2, 10)
			if err != nil {
				fatalErr(err, "Failed to start download worker")
			}

			for _, p := range importexhentaiFlags.Profile {
				log.Infof("🕵️ Scraping %s profile %s", site, p)
				ci := exhentai.NewexhentaiImporter(site, importFlags.ForceRescan)
				err = ci.ImportProfile(ctx, p)
				if err != nil {
					fatalErr(err, "Failed to scrape profile "+p)
				}
			}

			downloader.SharedWorker.Done()

			log.Debug("Waiting for all workers to return")
			downloader.SharedWorker.Wait()
			log.Debug("Workers done")

			err = downloader.SharedWorker.Stop()
			if err != nil {
				fatalErr(err, "Failed to stop workers")
			}
		}
	}

	importexhentai := &cobra.Command{
		Use:   "exhentai-exhentai",
		Short: "Import exhentai profiles from exhentai.org",
		Run:   importexhentai("exhentai"),
	}

	importexhentai.Flags().StringSliceVarP(&importexhentaiFlags.Profile, "profile", "p", []string{}, "exhentai profile(s) to import, as name(s)")

	importCmd.AddCommand(importexhentai)

	importexhentaiFansly := &cobra.Command{
		Use:   "exhentai-fansly",
		Short: "Import Fansly profiles from exhentai.org",
		Run:   importexhentai("fansly"),
	}

	importexhentaiFansly.Flags().StringSliceVarP(&importexhentaiFlags.Profile, "profile", "p", []string{}, "Fansly profile(s) to import, as ID(s)")

	importCmd.AddCommand(importexhentaiFansly)
}
