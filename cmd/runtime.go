package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"shadiff/internal/config"
)

type appRuntime struct {
	ConfigPath string
	Config     *config.AppConfig
	DataDir    string
	LogDir     string
}

var runtimeCtx *appRuntime

func initRuntime() error {
	store, err := config.NewStoreWithPath(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cfg := store.Get()
	if err := config.Validate(cfg); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	dataDir := store.DataDir()
	logDir := cfg.Log.LogDir
	if logDir == "" {
		logDir = filepath.Join(dataDir, "logs")
	}

	configPath := cfgFile
	if configPath == "" {
		configPath, err = config.DefaultPath()
		if err != nil {
			return err
		}
	}

	runtimeCtx = &appRuntime{
		ConfigPath: configPath,
		Config:     cfg,
		DataDir:    dataDir,
		LogDir:     logDir,
	}
	return nil
}

func mustRuntime() *appRuntime {
	if runtimeCtx == nil {
		panic("runtime not initialized")
	}
	return runtimeCtx
}

func currentConfig() *config.AppConfig {
	return mustRuntime().Config
}

func currentDataDir() string {
	return mustRuntime().DataDir
}

func currentLogDir() string {
	return mustRuntime().LogDir
}

func currentConfigPath() string {
	return mustRuntime().ConfigPath
}

func effectiveLogLevel() string {
	switch {
	case quiet:
		return "error"
	case verbose:
		return "debug"
	default:
		return strings.ToLower(currentConfig().Log.Level)
	}
}

func effectiveString(cmdFlagChanged bool, flagValue, cfgValue string) string {
	if cmdFlagChanged {
		return flagValue
	}
	if cfgValue != "" {
		return cfgValue
	}
	return flagValue
}

func effectiveStrings(cmdFlagChanged bool, flagValue, cfgValue []string) []string {
	if cmdFlagChanged {
		return flagValue
	}
	if len(cfgValue) > 0 {
		return cfgValue
	}
	return flagValue
}

func effectiveBool(cmdFlagChanged bool, flagValue, cfgValue bool) bool {
	if cmdFlagChanged {
		return flagValue
	}
	return cfgValue
}

func effectiveInt(cmdFlagChanged bool, flagValue, cfgValue int) int {
	if cmdFlagChanged {
		return flagValue
	}
	if cfgValue != 0 {
		return cfgValue
	}
	return flagValue
}
