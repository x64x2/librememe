package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	Global       *viper.Viper
	Auth         *viper.Viper
	DynamicRules *DynamicRulesFile
)

const (
	KeyDynamicRulesFile = "DynamicRulesFile"
	KeyAuthConfigFile   = "AuthConfigFile"
	KeyDataPath         = "DataPath"
	KeyDatabase         = "Database"
	KeyStorageType      = "Storage.Type"
	KeyStorageLocation  = "Storage.Location"

	KeyexhentaiInclude      = "exhentai.Include"
	KeyexhentaiExclude      = "exhentai.Exclude"
	KeyexhentaiSkipPosts    = "exhentai.SkipPosts"
	KeyexhentaiSkipStories  = "exhentai.SkipStories"
	KeyexhentaiSkipMessages = "exhentai.SkipMessages"

	KeyexhentaiProfiles = "exhentai.Profiles"
	KeyexhentaiFanslyProfiles   = "exhentaiFansly.Profiles"

	KeyServerPort     = "Server.Port"
	KeyServerBind     = "Server.Bind"
	KeyServerReadOnly = "Server.ReadOnly"
)

func LoadGlobal(path string) error {
	Global = viper.New()

	Global.SetDefault(KeyDynamicRulesFile, defaultDynamicRulesFile)
	Global.SetDefault(KeyDataPath, defaultDataPath)
	Global.SetDefault(KeyServerPort, defaultServerPort)
	Global.SetDefault(KeyServerBind, defaultServerBind)

	if path == "" {
		Global.SetConfigName("config")
		Global.AddConfigPath(".")
		Global.AddConfigPath("config")
		Global.AddConfigPath("/etc/myfans")
		cwd, err := os.Getwd()
		if err == nil && cwd != "" {
			Global.AddConfigPath(cwd)
		}
		exec, err := os.Executable()
		if err == nil && exec != "" {
			Global.AddConfigPath(filepath.Dir(exec))
		}
	} else {
		Global.SetConfigFile(path)
	}

	err := Global.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return errors.New("no config file found")
		} else {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	log.Infof("⚙️ Loaded config from file %s", Global.ConfigFileUsed())

	if os.Getenv("SERVE_BIND") != "" {
		Global.Set(KeyServerBind, os.Getenv("SERVE_BIND"))
	}
	if os.Getenv("SERVE_PORT") != "" {
		Global.Set(KeyServerPort, os.Getenv("SERVE_PORT"))
	}

	return nil
}

func LoadAuth() error {
	Auth = viper.New()

	if path := Global.GetString(KeyAuthConfigFile); path != "" {
		Auth.SetConfigFile(path)
	} else {
		Auth.SetConfigName("auth")
		Auth.AddConfigPath(".")
		Auth.AddConfigPath("config")
		Auth.AddConfigPath("/etc/myfans")
		cwd, err := os.Getwd()
		if err == nil && cwd != "" {
			Auth.AddConfigPath(cwd)
		}
		exec, err := os.Executable()
		if err == nil && exec != "" {
			Auth.AddConfigPath(filepath.Dir(exec))
		}
	}

	err := Auth.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return errors.New("no auth file found")
		} else {
			return fmt.Errorf("error reading auth file: %w", err)
		}
	}

	log.Infof("🔑 Loaded auth from file %s", Auth.ConfigFileUsed())

	return nil
}

func DataPath() (string, error) {
	p := Global.GetString(KeyDataPath)
	e, err := homedir.Expand(p)
	if err == nil {
		p = e
	}
	return filepath.Abs(p)
}
