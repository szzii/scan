package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config represents application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Scanner  ScannerConfig  `mapstructure:"scanner"`
	Storage  StorageConfig  `mapstructure:"storage"`
	AutoScan AutoScanConfig `mapstructure:"autoscan"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	ESCLEnabled  bool   `mapstructure:"escl_enabled"`
	ESCLPort     int    `mapstructure:"escl_port"`
}

// ScannerConfig represents scanner configuration
type ScannerConfig struct {
	DefaultResolution int    `mapstructure:"default_resolution"`
	DefaultColorMode  string `mapstructure:"default_color_mode"`
	DefaultFormat     string `mapstructure:"default_format"`
	ScanTimeout       int    `mapstructure:"scan_timeout"` // seconds
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	OutputDir      string `mapstructure:"output_dir"`
	MaxStorageSize int64  `mapstructure:"max_storage_size"` // bytes
	CleanupEnabled bool   `mapstructure:"cleanup_enabled"`
	RetentionDays  int    `mapstructure:"retention_days"`
}

// AutoScanConfig represents auto-scan configuration
type AutoScanConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	LidCloseDelay  int    `mapstructure:"lid_close_delay"`  // seconds to wait after lid close
	ScannerID      string `mapstructure:"scanner_id"`       // specific scanner to monitor
	DefaultParams  DefaultScanParams `mapstructure:"default_params"`
}

// DefaultScanParams represents default scan parameters for auto-scan
type DefaultScanParams struct {
	Resolution int    `mapstructure:"resolution"`
	ColorMode  string `mapstructure:"color_mode"`
	Format     string `mapstructure:"format"`
	UseDuplex  bool   `mapstructure:"use_duplex"`
	UseFeeder  bool   `mapstructure:"use_feeder"`
}

// Load loads configuration from file or environment variables
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Read from config file if provided
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Environment variables
	v.AutomaticEnv()
	v.SetEnvPrefix("SCANNER")

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.escl_enabled", true)
	v.SetDefault("server.escl_port", 8080)

	// Scanner defaults
	v.SetDefault("scanner.default_resolution", 300)
	v.SetDefault("scanner.default_color_mode", "Color")
	v.SetDefault("scanner.default_format", "PDF")
	v.SetDefault("scanner.scan_timeout", 300)

	// Storage defaults
	v.SetDefault("storage.output_dir", "./scans")
	v.SetDefault("storage.max_storage_size", int64(10*1024*1024*1024)) // 10GB
	v.SetDefault("storage.cleanup_enabled", true)
	v.SetDefault("storage.retention_days", 30)

	// Auto-scan defaults
	v.SetDefault("autoscan.enabled", false)
	v.SetDefault("autoscan.lid_close_delay", 2)
	v.SetDefault("autoscan.scanner_id", "")
	v.SetDefault("autoscan.default_params.resolution", 300)
	v.SetDefault("autoscan.default_params.color_mode", "Color")
	v.SetDefault("autoscan.default_params.format", "PDF")
	v.SetDefault("autoscan.default_params.use_duplex", false)
	v.SetDefault("autoscan.default_params.use_feeder", false)
}
