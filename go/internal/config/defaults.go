// Package config provides centralized configuration defaults for ditong.
package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// ConfigFile represents the structure of config.toml
type ConfigFile struct {
	Defaults           Defaults `toml:"defaults"`
	AvailableLanguages []string `toml:"available_languages"`
}

// Defaults holds all default values
type Defaults struct {
	Languages   string `toml:"languages"`
	MinLength   int    `toml:"min_length"`
	MaxLength   int    `toml:"max_length"`
	OutputDir   string `toml:"output_dir"`
	CacheDir    string `toml:"cache_dir"`
	Parallel    bool   `toml:"parallel"`
	Workers     int    `toml:"workers"`
	IPA         bool   `toml:"ipa"`
	Cursewords  bool   `toml:"cursewords"`
	Consolidate bool   `toml:"consolidate"`
	Force       bool   `toml:"force"`
	Quiet       bool   `toml:"quiet"`
	Verbose     bool   `toml:"verbose"`
	Metrics     bool   `toml:"metrics"`
}

// Hardcoded fallback defaults (used if config.toml not found)
var fallbackDefaults = Defaults{
	Languages:   "en,tr",
	MinLength:   3,
	MaxLength:   5,
	OutputDir:   "output/dicts",
	CacheDir:    "sources",
	Parallel:    true,
	Workers:     0,
	IPA:         false,
	Cursewords:  false,
	Consolidate: false,
	Force:       false,
	Quiet:       false,
	Verbose:     false,
	Metrics:     true,
}

var fallbackLanguages = []string{"en", "tr", "de", "fr", "es", "it", "pt", "nl", "pl", "ru"}

// loaded holds the parsed config (nil if not loaded yet)
var loaded *ConfigFile

// Load reads config.toml from the project root
func Load() *ConfigFile {
	if loaded != nil {
		return loaded
	}

	// Try to find config.toml by walking up from executable or cwd
	paths := []string{
		"config.toml",
		"../config.toml",
		"../../config.toml",
	}

	// Also try from executable location
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		paths = append(paths,
			filepath.Join(dir, "config.toml"),
			filepath.Join(dir, "..", "config.toml"),
			filepath.Join(dir, "..", "..", "config.toml"),
		)
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			var cfg ConfigFile
			if _, err := toml.DecodeFile(path, &cfg); err == nil {
				loaded = &cfg
				return loaded
			}
		}
	}

	// Return fallback if config.toml not found
	loaded = &ConfigFile{
		Defaults:           fallbackDefaults,
		AvailableLanguages: fallbackLanguages,
	}
	return loaded
}

// Convenience accessors that load config on first access
var (
	DefaultLanguages      = func() string { return Load().Defaults.Languages }
	DefaultMinLength      = func() int { return Load().Defaults.MinLength }
	DefaultMaxLength      = func() int { return Load().Defaults.MaxLength }
	DefaultOutputDir      = func() string { return Load().Defaults.OutputDir }
	DefaultCacheDir       = func() string { return Load().Defaults.CacheDir }
	DefaultParallel       = func() bool { return Load().Defaults.Parallel }
	DefaultWorkers        = func() int { return Load().Defaults.Workers }
	DefaultIPA            = func() bool { return Load().Defaults.IPA }
	DefaultCursewords     = func() bool { return Load().Defaults.Cursewords }
	DefaultConsolidate    = func() bool { return Load().Defaults.Consolidate }
	DefaultForce          = func() bool { return Load().Defaults.Force }
	DefaultQuiet          = func() bool { return Load().Defaults.Quiet }
	DefaultVerbose        = func() bool { return Load().Defaults.Verbose }
	DefaultMetrics        = func() bool { return Load().Defaults.Metrics }
	DefaultParallelIngest = func() bool { return Load().Defaults.Parallel }
	DefaultParallelBuild  = func() bool { return Load().Defaults.Parallel }
)

// MaxWorkers is the cap for parallel workers
const MaxWorkers = 8

// AvailableLanguagesStr returns available languages as comma-separated string.
func AvailableLanguagesStr() string {
	return strings.Join(Load().AvailableLanguages, ", ")
}
