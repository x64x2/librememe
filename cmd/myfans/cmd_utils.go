package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var utilsCmd *cobra.Command

func init() {
	var utilsFlags struct {
		Config  string
		Verbose bool
		Quiet   bool
	}

	utilsCmd = &cobra.Command{
		Use:   "utils",
		Short: "Utilities",
	}

	utilsCmd.PersistentFlags().StringVarP(&utilsFlags.Config, "config", "c", "", "config file to load (optional)")
	utilsCmd.PersistentFlags().BoolVarP(&utilsFlags.Verbose, "verbose", "v", false, "verbose mode - logs more info")
	utilsCmd.PersistentFlags().BoolVarP(&utilsFlags.Quiet, "quiet", "q", false, "quiet mode - suppress most logs")

	utilsHashMissingCmd := &cobra.Command{
		Use:   "hash-missing",
		Short: "Computes the hash of downloaded files for which the hash is missing",
		Run: func(cmd *cobra.Command, args []string) {
			log.SetOutput(os.Stdout)
			switch {
			case utilsFlags.Verbose:
				log.SetLevel(log.DebugLevel)
			case utilsFlags.Quiet:
				log.SetLevel(log.WarnLevel)
			default:
				log.SetLevel(log.InfoLevel)
			}

			err := config.LoadGlobal(utilsFlags.Config)
			if err != nil {
				log.Fatalf("Failed to load global config: %v", err)
			}

			err = fs.InitShared(context.Background())
			if err != nil {
				log.Fatalf("Failed to create storage: %v", err)
			}

			err = db.Connect(false)
			if err != nil {
				log.Fatalf("Failed to connect to database: %v", err)
			}

			err = db.Migrate(context.Background())
			if err != nil {
				log.Fatalf("Failed to perform DB migrations: %v", err)
			}

			err = hashMissing(context.Background())
			if err != nil {
				log.Fatalf("Failed to compute missing hashes: %v", err)
			}
		},
	}

	utilsCmd.AddCommand(utilsHashMissingCmd)

	var utilsImportMediaFlags struct {
		Path string
	}

	utilsImportMediaCmd := &cobra.Command{
		Use:   "import-media",
		Short: "Imports media from a local directory",
		Long: `Copies media (photos, videos, audios) from a local directory into myfans' media folder, and adds a reference to the database. This command does not create posts for the imported files, nor adds them to a profile.

This command is useful when you have a folder of media already downloaded, and you want to run the scraper or importer after, without having to re-download a bunch of files.
`,
		Run: func(cmd *cobra.Command, args []string) {
			log.SetOutput(os.Stdout)
			switch {
			case utilsFlags.Verbose:
				log.SetLevel(log.DebugLevel)
			case utilsFlags.Quiet:
				log.SetLevel(log.WarnLevel)
			default:
				log.SetLevel(log.InfoLevel)
			}

			if utilsImportMediaFlags.Path == "" {
				log.Fatal("Specify the path to import with the '--path' flag")
			}

			path, err := filepath.Abs(utilsImportMediaFlags.Path)
			if err != nil {
				log.Fatalf("Invalid path: %v", err)
			}

			err = config.LoadGlobal(utilsFlags.Config)
			if err != nil {
				log.Fatalf("Failed to load global config: %v", err)
			}

			err = fs.InitShared(context.Background())
			if err != nil {
				log.Fatalf("Failed to create storage: %v", err)
			}

			err = db.Connect(false)
			if err != nil {
				log.Fatalf("Failed to connect to database: %v", err)
			}

			err = db.Migrate(context.Background())
			if err != nil {
				log.Fatalf("Failed to perform DB migrations: %v", err)
			}

			err = importPath(context.Background(), path)
			if err != nil {
				log.Fatalf("Failed to compute missing hashes: %v", err)
			}

			log.Info("Import done")
		},
	}

	utilsImportMediaCmd.Flags().StringVarP(&utilsImportMediaFlags.Path, "path", "p", "", "path to the folder containing the media to import")

	utilsCmd.AddCommand(utilsImportMediaCmd)
}

func hashMissing(ctx context.Context) error {
	log.Info("Loading rows from database")

	q := `
SELECT media_id, media_location, media_preview, media_hash, media_preview_hash
FROM ` + models.MediaBase.TableName() + `
WHERE
	(media_location <> '' AND media_hash IS NULL) OR 
	(media_preview <> '' AND media_preview_hash IS NULL)`
	rows, err := db.Get().QueryContext(ctx, q)
	if err != nil {
		return fmt.Errorf("failed to query database for rows: %w", err)
	}

	originals := make(map[int64]string)
	previews := make(map[int64]string)
	for rows.Next() {
		var (
			mediaID           int64
			location, preview string
			hash, previewHash []byte
		)
		err = rows.Scan(&mediaID, &location, &preview, &hash, &previewHash)
		if err != nil {
			return fmt.Errorf("error reading row: %w", err)
		}

		if location != "" && len(hash) == 0 {
			originals[mediaID] = location
		}
		if preview != "" && len(previewHash) == 0 {
			previews[mediaID] = preview
		}
	}
	rows.Close()

	hashFiles(ctx, originals, "original", "media_hash")
	hashFiles(ctx, previews, "preview", "media_preview_hash")

	return nil
}

func hashFiles(ctx context.Context, paths map[int64]string, typ string, col string) {
	conn := db.Get()

	var q string
	switch conn.DriverName() {
	case "sqlite":
		q = "UPDATE " + models.MediaBase.TableName() + " SET " + col + " = ? WHERE media_id = ?"

	case "pgx":
		q = "UPDATE " + models.MediaBase.TableName() + " SET " + col + " = $1 WHERE media_id = $2"
	}

	log.Infof("Need to compute hashes for %d %s", len(paths), typ+"s")
	i := 0
	for mediaID, path := range paths {
		i++
		log.Infof("(%d/%d) Computing hash of %s file '%s' for media %d", i, len(paths), typ, path, mediaID)
		hash, err := hashFile(ctx, path)
		if err != nil {
			log.Errorf("Failed to compute hash of file '%s' for media %v: %v", path, mediaID, err)
			continue
		}

		res, err := conn.ExecContext(ctx, q, hash, mediaID)
		if err != nil {
			log.Errorf("Failed to update hash of file '%s' for media %v: %v", path, mediaID, err)
			continue
		}

		affected, err := res.RowsAffected()
		if err != nil {
			log.Errorf("Failed to count affected rows after updating hash of file '%s' for media %v: %v", path, mediaID, err)
			continue
		}
		if affected == 0 {
			log.Errorf("No row was updated in database while updating hash of file '%s' for media %v", path, mediaID)
			continue
		}
	}
}

func hashFile(ctx context.Context, path string) ([]byte, error) {
	h := sha256.New()

	exists, err := fs.SharedFS.Get(ctx, path, 0, 0, h)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	} else if !exists {
		return nil, errors.New("file not found")
	}

	return h.Sum(nil), nil
}

func importPath(ctx context.Context, path string) error {
	log.Debugf("📂 Entering '%s'", path)

	rd, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to read directory '%s': %w", path, err)
	}

	dirs := make([]string, 0)

	for _, v := range rd {
		abs := filepath.Join(path, v.Name())
		if v.IsDir() {
			dirs = append(dirs, abs)
			continue
		}

		err = importFile(ctx, v.Name(), abs)
		if err != nil {
			return err
		}
	}

	for _, v := range dirs {
		err = importPath(ctx, v)
		if err != nil {
			return err
		}
	}

	return nil
}

func importFile(ctx context.Context, name string, abs string) (err error) {
	ft, err := util.GetFileType(name)
	if err != nil {
		log.Warnf("Skipping file '%s': failed to determine file type: %v", abs, err)
		return nil
	}

	log.Debugf("Importing '%s'", name)

	storeFilename, err := util.StoreFilename(abs)
	if err != nil {
		return fmt.Errorf("failed to generate stored file name: %w", err)
	}

	mm := models.NewMedia(db.SourceImported, uuid.Must(uuid.NewRandom()).String())
	mm.SetModified()
	mm.Type = ft
	mm.Visible = 1
	mm.OriginalLocation = abs
	mm.Location = path.Join("media", storeFilename)

	f, err := os.Open(abs)
	if err != nil {
		return fmt.Errorf("failed to open '%s' for reading: %w", abs, err)
	}
	defer f.Close()

	h := sha256.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return fmt.Errorf("failed to compute hash of file '%s': %w", abs, err)
	}

	mm.Hash = h.Sum(nil)

	dupID, err := findDuplicate(ctx, mm.Hash)
	if err != nil {
		return fmt.Errorf("failed to look for duplicate of file '%s': %w", abs, err)
	}
	if dupID != "" {
		log.Debugf("File already found with matching hash (media %v)", dupID)
		return nil
	}

	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek back to start of file '%s': %w", abs, err)
	}

	err = fs.SharedFS.Put(ctx, mm.Location, f)
	if err != nil {
		return fmt.Errorf("failed to store file '%s': %w", abs, err)
	}

	err = mm.Save(ctx, nil)
	if err != nil {
		rmErr := fs.SharedFS.Delete(ctx, mm.Location)
		if rmErr != nil {
			log.Warnf("Attempting to remove stored file '%s' after import error failed: %v", mm.Location, err)
		}

		return fmt.Errorf("failed to save media '%s': %w", abs, err)
	}

	return nil
}

func findDuplicate(ctx context.Context, hash []byte) (string, error) {
	if len(hash) == 0 {
		return "", nil
	}

	mid, loc, err := models.FindMediaByHash(ctx, hash)
	if err != nil {
		return "", fmt.Errorf("database error: %w", err)
	}

	if loc == "" {
		return "", nil
	}

	return mid, nil
}
