package datastore

import "go.uber.org/zap"

type Config struct {
	Logger         *zap.Logger
	DataSourceName string
}

var config Config

func Configure(c Config) {
	config = c
}
