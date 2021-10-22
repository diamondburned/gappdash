package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "embed"

	"github.com/diamondburned/gotk4-layer-shell/pkg/gtklayershell"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

// Config is the main configuration struct type.
type Config struct {
	App        AppConfig        `toml:"gappdash"`
	LayerShell LayerShellConfig `toml:"layer-shell"`
	Window     WindowConfig     `toml:"window"`
}

// AppMode is a string enum type.
type AppMode string

const (
	GridMode AppMode = "grid"
	ListMode AppMode = "list"
)

// AppConfig is the GAppDash's configuration.
type AppConfig struct {
	Mode          AppMode
	Daemonize     bool
	IndexAge      time.Duration `toml:"index-age"`
	Fuzzy         bool
	CaseSensitive bool `toml:"case-sensitive"`
	IconSize      int  `toml:"icon-size"`

	Grid GridConfig
}

// StockIconSize rounds the config's icon size.
func (a *AppConfig) StockIconSize() gtk.IconSize {
	switch {
	case a.IconSize > 48:
		return gtk.IconSizeDialog
	case a.IconSize > 32:
		return gtk.IconSizeDND
	case a.IconSize > 24:
		return gtk.IconSizeLargeToolbar
	default:
		return gtk.IconSizeSmallToolbar
	}
}

func (a *AppConfig) Validate() error {
	if a.Mode != GridMode && a.Mode != ListMode {
		return fmt.Errorf("unknown mode %q", a.Mode)
	}
	return nil
}

// GridConfig is the config for grid mode.
type GridConfig struct {
	MinChildrenPerLine uint `toml:"min-children-per-line"`
	MaxChildrenPerLine uint `toml:"max-children-per-line"`
}

// LayerShellAnchor is a string enum type.
type LayerShellAnchor string

const (
	LayerShellAnchorTop    LayerShellAnchor = "top"
	LayerShellAnchorBottom LayerShellAnchor = "bottom"
	LayerShellAnchorLeft   LayerShellAnchor = "left"
	LayerShellAnchorRight  LayerShellAnchor = "right"
)

func (a LayerShellAnchor) Edge() gtklayershell.Edge {
	switch a {
	case LayerShellAnchorTop:
		return gtklayershell.LayerShellEdgeTop
	case LayerShellAnchorBottom:
		return gtklayershell.LayerShellEdgeBottom
	case LayerShellAnchorLeft:
		return gtklayershell.LayerShellEdgeLeft
	case LayerShellAnchorRight:
		return gtklayershell.LayerShellEdgeRight
	default:
		return -1
	}
}

// LayerShellLayer is a string enum type.
type LayerShellLayer string

const (
	LayerShellLayerTop        LayerShellLayer = "top"
	LayerShellLayerBottom     LayerShellLayer = "bottom"
	LayerShellLayerBackground LayerShellLayer = "background"
	LayerShellLayerOverlay    LayerShellLayer = "overlay"
)

func (l LayerShellLayer) Layer() gtklayershell.Layer {
	switch l {
	case LayerShellLayerTop:
		return gtklayershell.LayerShellLayerTop
	case LayerShellLayerBottom:
		return gtklayershell.LayerShellLayerBottom
	case LayerShellLayerBackground:
		return gtklayershell.LayerShellLayerBackground
	case LayerShellLayerOverlay:
		return gtklayershell.LayerShellLayerOverlay
	default:
		return -1
	}
}

// LayerShellConfig is the Layer Shell's configuration.
type LayerShellConfig struct {
	Enable  bool
	Layer   LayerShellLayer
	Anchors []LayerShellAnchor // TODO: change to allow stretching
	Margins struct {
		Top    int
		Bottom int
		Left   int
		Right  int
	}
}

// TransformMargins transforms the Margins values into a map of the appropriate
// Layer Shell edges.
func (c *LayerShellConfig) TransformMargins() map[gtklayershell.Edge]int {
	return map[gtklayershell.Edge]int{
		gtklayershell.LayerShellEdgeTop:    c.Margins.Top,
		gtklayershell.LayerShellEdgeBottom: c.Margins.Bottom,
		gtklayershell.LayerShellEdgeLeft:   c.Margins.Left,
		gtklayershell.LayerShellEdgeRight:  c.Margins.Right,
	}
}

// Validate validates the Layer Shell config.
func (c *LayerShellConfig) Validate() error {
	if c.Layer.Layer() == -1 {
		return fmt.Errorf("invalid layer %q", c.Layer)
	}

	for _, anchor := range c.Anchors {
		if anchor.Edge() == -1 {
			return fmt.Errorf("invalid anchor %q", anchor)
		}
	}

	return checkPositiveInts(map[string]int{
		"margins.top":    c.Margins.Top,
		"margins.bottom": c.Margins.Bottom,
		"margins.left":   c.Margins.Left,
		"margins.right":  c.Margins.Right,
	})
}

func checkPositiveInts(ints map[string]int) error {
	for what, i := range ints {
		if i < 0 {
			return fmt.Errorf("%s: negative int %d not allowed", what, i)
		}
	}

	return nil
}

// WindowConfig is the main window's configuration.
type WindowConfig struct {
	Width  int
	Height int
}

func validate(validators ...interface{ Validate() error }) error {
	for _, validator := range validators {
		if err := validator.Validate(); err != nil {
			return err
		}
	}
	return nil
}

//go:embed config.example.toml
var defaultConfigTOML []byte

// ParseConfig parses the config from the given path to file.
func ParseConfig(path string) (*Config, error) {
	var cfg Config

	if err := toml.Unmarshal(defaultConfigTOML, &cfg); err != nil {
		log.Panicln("BUG: error parsing default config:", err)
	}

	if err := validate(&cfg.LayerShell, &cfg.App); err != nil {
		log.Panicln("BUG: error validating default config:", err)
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Ignore not-exist errors.
			return &cfg, nil
		}
		return nil, errors.Wrap(err, "failed to open file")
	}
	defer f.Close()

	// Parse the user config OVER the default config.
	if err := toml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}

	if err := validate(&cfg.LayerShell, &cfg.App); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func userConfigFile(filename string) (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", errors.Wrap(err, "failed to get config directory")
	}

	return filepath.Join(cfg, "gappdash", filename), nil
}

// ParseUserConfig parses the configuration file at the default location. If the
// file does not exist, then a new file with the defaults is created.
func ParseUserConfig() (*Config, error) {
	cfg, err := userConfigFile("config.toml")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get config path")
	}

	return ParseConfig(cfg)
}
