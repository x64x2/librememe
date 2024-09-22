package main

import (
	"context"
	"errors"
	"os"
	"slices"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

)

var scrapeCmd *cobra.Command

func init() {
	var scrapeFlags struct {
		Config                 string
		Verbose                bool
		Quiet                  bool
		Profile                []string
		ForceRescan            bool
		SkipPastPurchasesCheck bool
		SkipUpdateProfiles     bool
	}

	scrapeCmd = &cobra.Command{
		Use:   "scrape",
		Short: "Run the scraper",
		Run: func(cmd *cobra.Command, args []string) {
			log.SetOutput(os.Stdout)
			switch {
			case scrapeFlags.Verbose:
				log.SetLevel(log.DebugLevel)
			case scrapeFlags.Quiet:
				log.SetLevel(log.WarnLevel)
			default:
				log.SetLevel(log.InfoLevel)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			termCh := make(chan os.Signal, 1)
			signal.Notify(termCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-termCh
				cancel()
				<-termCh
				log.Fatal("Force termination")
			}()*/

			err := config.LoadGlobal(scrapeFlags.Config)
			if err != nil {
				fatalErr(err, "Failed to load global config")
			}

			err = config.LoadAuth()
			if err != nil {
				fatalErr(err, "Failed to load auth info")
			}

			err = config.LoadDynamicRules(ctx)
			if err != nil {
				fatalErr(err, "Failed to load dynamic rules")
			}

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
			err = downloader.SharedWorker.Start(3, 12)
			if err != nil {
				fatalErr(err, "Failed to start download worker")
			}

			scraper := exhentai.NewScraper()
			scraper.ForceRescan = scrapeFlags.ForceRescan

			ownProfile, err := scraper.GetProfile(ctx)
			if err != nil || ownProfile == nil {
				fatalErr(err, "Failed to get own profile")
			}

			var allProfiles map[string]exhentai.ScrapeProfile
			if scrapeFlags.SkipUpdateProfiles {
				list, err := exhentai.LoadAllProfiles(ctx)
				if err != nil {
					fatalErr(err, "Failed to load profile list")
				}
				allProfiles = list.ToMapIndexedById()
				scraper.ProfilesBySourceId = list.ToMapIndexedBySourceId()
			} else {
				allProfiles, err = scraper.GetSubscriptions(ctx)
				if err != nil {
					fatalErr(err, "Failed to get subscriptions")
				}
				allProfiles[ownProfile.ID] = exhentai.ScrapeProfile{
					ID:       ownProfile.ID,
					SourceID: ownProfile.SourceID,
					Username: ownProfile.Name,
				}
			}

			if len(allProfiles) == 0 {
				log.Error("You are not subscribed to any user")
				return
			}

			if len(scrapeFlags.Profile) > 0 {
				config.Global.Set(config.KeyexhentaiInclude, scrapeFlags.Profile)
			}

			var scrapeProfiles []string
			include := config.Global.GetStringSlice(config.KeyexhentaiInclude)
			if len(include) > 0 {
				includeIDs, err := models.LoadProfileIDs(ctx, include)
				if err != nil {
					fatalErr(err, "Failed to load profile IDs")
				}

				scrapeProfiles = make([]string, len(includeIDs))
				added := map[string]bool{}
				n := 0
				for _, id := range includeIDs {
					p, ok := allProfiles[id]
					if ok && !added[id] {
						log.Infof("👩 Selected %s (%v)", p.Username, id)
						scrapeProfiles[n] = id
						n++
						added[id] = true
					}
				}
				scrapeProfiles = scrapeProfiles[:n]
			} else {
				exclude := config.Global.GetStringSlice(config.KeyexhentaiExclude)
				excludeIDs := make([]string, 0)
				if len(exclude) > 0 {
					excludeIDs, err = models.LoadProfileIDs(ctx, exclude)
					if err != nil {
						fatalErr(err, "Failed to load profile IDs")
					}
				}

				scrapeProfiles = make([]string, len(allProfiles))
				n := 0
				for id, p := range allProfiles {
					if !slices.Contains(excludeIDs, id) {
						log.Infof("👩 Selected %s (%v)", p.Username, id)
						scrapeProfiles[n] = id
						n++
					}
				}
				scrapeProfiles = scrapeProfiles[:n]
			}

			for _, id := range scrapeProfiles {
				if ctx.Err() != nil {
					break
				}

				log.Infof("🕵️ Scraping %s (%v)", allProfiles[id].Username, id)

				if !config.Global.GetBool(config.KeyexhentaiSkipStories) {
					err = scraper.ScrapeStories(ctx, allProfiles[id])
					if err != nil {
						if errors.Is(err, context.Canceled) {
							break
						}
						log.Errorf("Failed to scrape stories for profile %v: %v", id, err)
					}
				}

				if !config.Global.GetBool(config.KeyexhentaiSkipPosts) {
					err = scraper.ScrapePosts(ctx, allProfiles[id])
					if err != nil {
						if errors.Is(err, context.Canceled) {
							break
						}
						log.Errorf("Failed to scrape posts for profile %v: %v", id, err)
					}
				}

				if !config.Global.GetBool(config.KeyexhentaiSkipMessages) {
					err = scraper.ScrapeMessages(ctx, allProfiles[id])
					if err != nil {
						if errors.Is(err, context.Canceled) {
							break
						}
						log.Errorf("Failed to scrape messages for profile %v: %v", id, err)
					}
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
		},
	}
	scrapeCmd.Flags().StringVarP(&scrapeFlags.Config, "config", "c", "", "config file to load (optional)")
	scrapeCmd.Flags().BoolVarP(&scrapeFlags.Verbose, "verbose", "v", false, "verbose mode - logs more info")
	scrapeCmd.Flags().BoolVarP(&scrapeFlags.Quiet, "quiet", "q", false, "quiet mode - suppress most logs")
	scrapeCmd.Flags().StringSliceVarP(&scrapeFlags.Profile, "profile", "p", nil, "scrape only this profile (ID or username) - can be repeated")
	scrapeCmd.Flags().BoolVar(&scrapeFlags.ForceRescan, "force-rescan", false, "force complete rescan of all posts")
	scrapeCmd.Flags().BoolVar(&scrapeFlags.SkipUpdateProfiles, "skip-update-profiles", false, "skip updating profiles")

	scrapeCmd.Flags().MarkHidden("skip-update-profiles")
}

func fatalErr(err error, msg string) {
	if errors.Is(err, context.Canceled) {
		log.Errorf("Process stopped")
		os.Exit(1)
	}

	log.Fatal(msg + ": " + err.Error())
}
