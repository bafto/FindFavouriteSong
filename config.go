package main

import (
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
	viper.SetDefault("checkpoint_interval", 1*time.Minute)

	viper.SetEnvPrefix("ffs")
	viper.AutomaticEnv()

	viper.AddConfigPath(".")
	viper.SetConfigName("ffs_config")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	var config Config
	err := viper.Unmarshal(&config)
	return config, err
}
