package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/spf13/viper"
)

var log *slog.Logger

func read_config() {
	viper.SetDefault("spotiFy_client_id", "")
	viper.SetDefault("spotify_client_secret", "")
	viper.SetDefault("db_path", "ffs.db")
	viper.SetDefault("port", "8080")
	viper.SetDefault("log_level", "INFO")

	viper.AddConfigPath(".")
	viper.SetConfigName("ffs_config")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("fff")
	viper.AutomaticEnv()
}

func configure_logger() {
	var level slog.Level
	if err := level.UnmarshalText(
		[]byte(viper.GetString("log_level")),
	); err != nil {
		panic(err)
	}

	addSource := false
	if level <= slog.LevelDebug {
		addSource = true
	}

	levelVar := &slog.LevelVar{}
	levelVar.Set(level)
	log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: addSource,
		Level:     levelVar,
	}))

	slog.SetDefault(log)
}

func main() {
	read_config()
	configure_logger()

	ctx := context.Background()
	db, err := create_db(ctx, "ffs.db")
	if err != nil {
		log.Error("Error opening DB connection: %v", err)
		os.Exit(1)
	}
	defer db.Close()
	log.Info("Connected to database")
}
