package main

import (
	"log/slog"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Spotify_client_id     string            `mapstructure:"spotify_client_id"`
	Spotify_client_secret string            `mapstructure:"spotify_client_secret"`
	Datasource            string            `mapstructure:"data_source"`
	BackupPath            string            `mapstructure:"backup_path"`
	Port                  string            `mapstructure:"port"`
	Log_level             string            `mapstructure:"log_level"`
	Redirect_url          string            `mapstructure:"redirect_url"`
	Shutdown_timeout      time.Duration     `mapstructure:"shutdown_timeout"`
	Users                 map[string]string `mapstructure:"users"`
	CheckpointInterval    time.Duration     `mapstructure:"checkpoint_interval"`
	CheckpointTimeout     time.Duration     `mapstructure:"checkpoint_timeout"`
}

func read_config() (Config, error) {
	viper.SetDefault("spotify_client_id", "")
	viper.SetDefault("spotify_client_secret", "")
	viper.SetDefault("data_source", "file:ffs.db?_journal_mode=WAL")
	viper.SetDefault("backup_path", "ffs.backup.db")
	viper.SetDefault("port", "8080")
	viper.SetDefault("log_level", "INFO")
	viper.SetDefault("redirect_url", "http://localhost:8080/spotifyauthentication")
	viper.SetDefault("shutdown_timeout", time.Second*10)
	viper.SetDefault("users", map[string]string{})
	viper.SetDefault("checkpoint_interval", 2*time.Hour)
	viper.SetDefault("checkpoint_timeout", 1*time.Minute)

	viper.SetEnvPrefix("FFS")
	viper.AutomaticEnv()

	viper.AddConfigPath(".")
	viper.SetConfigName("ffs_config")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	// users are read from environment if present
	viper.BindEnv("users")

	// check if users is set via env
	usersMapEnv := viper.GetString("users")
	if usersMapEnv != "" {
		// If environment variable is set, handle parsing
		slog.Info("Using environment variable for users")
		users := parseCommaSeparatedMap(usersMapEnv)
		viper.Set("users", users)
	}

	var config Config
	err := viper.Unmarshal(&config)
	return config, err
}

func parseCommaSeparatedMap(input string) map[string]string {
	result := make(map[string]string)
	pairs := strings.Split(input, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		}
	}
	return result
}
