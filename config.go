package main

import "time"

// Config is the main configuration struct type.
type Config struct {
	App        AppConfig        `toml:"gappdash"`
	LayerShell LayerShellConfig `toml:"layer-shell"`
	Window     WindowConfig     `toml:"window"`
}

// AppConfig is the GAppDash's configuration.
type AppConfig struct {
	Mode          string
	Daemonize     bool
	IndexAge      time.Duration `toml:"index-age"`
	Fuzzy         bool
	CaseSensitive bool `toml:"case-sensitive"`
}

// LayerShellConfig is the Layer Shell's configuration.
type LayerShellConfig struct {
	Enable  bool
	Layer   string
	Anchors []string
	Margins struct {
		Top    int
		Bottom int
		Left   int
		Right  int
	}
}

// WindowConfig is the main window's configuration.
type WindowConfig struct {
	Width  int
	Height int
}
