package log

import (
	"encoding/json"
	"go.uber.org/zap"
)

var rootLogger *zap.Logger

func Init(level string) (*zap.Logger, error) {
	rawJSON := []byte(`{
		"level": "debug",
		"encoding": "json",
		"outputPaths": ["stdout"],
		"errorOutputPaths": ["stderr"],
		"encoderConfig": {
			"timeKey": "time",
			"timeEncoder": "iso8601",
	    	"messageKey": "message",
	    	"levelKey": "level",
			"levelEncoder": "lowercase",
			"callerKey": "caller",
			"callerEncoder": "short"
		}
	}`)

	var cfg zap.Config

	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		return rootLogger, err
	}
	if err := cfg.Level.UnmarshalText([]byte(level)); err != nil {
		return rootLogger, err
	}

	var err error
	rootLogger, err = cfg.Build()
	if err != nil {
		return rootLogger, err
	}

	rootLogger.Debug("Logging initialised")
	return rootLogger, nil
}

func Get() *zap.Logger {
	if rootLogger == nil {
		panic("Logging is not initialised (must call Init() before Get())")
	}
	return rootLogger
}
