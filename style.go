package main

import (
	"log"
	"os"
	"strings"

	_ "embed"

	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
)

var (
	//go:embed style.css
	defaultCSS      string
	defaultProvider *gtk.CSSProvider

	userProvider *gtk.CSSProvider
)

func initStyles() {
	defaultProvider = gtk.NewCSSProvider()
	defaultProvider.Connect("parsing-error", cssErrorPrinter("built-in CSS", defaultCSS))
	defaultProvider.LoadFromData(defaultCSS)

	userProvider = gtk.NewCSSProvider()

	if userCSSPath, err := userConfigFile("style.css"); err == nil {
		if userCSS, err := os.ReadFile(userCSSPath); err == nil {
			userProvider.Connect("parsing-error", cssErrorPrinter("user CSS", string(userCSS)))
			userProvider.LoadFromData(string(userCSS))
		}
	}

	manager := gdk.DisplayManagerGet()
	manager.ConnectDisplayOpened(func(display gdk.Display) {
		styleNewDisplay(&display)
	})
	for _, display := range manager.ListDisplays() {
		styleNewDisplay(&display)
	}
}

func cssErrorPrinter(name, blob string) func(sect *gtk.CSSSection, err error) {
	return func(sect *gtk.CSSSection, err error) {
		lines := strings.Split(blob, "\n")
		line := lines[sect.StartLine()]
		log.Printf("%s: built-in CSS error (%s) at line:\n\t%s", name, err, line)
	}
}

func styleNewDisplay(display *gdk.Display) {
	gtk.StyleContextAddProviderForScreen(
		display.DefaultScreen(),
		defaultProvider,
		gtk.STYLE_PROVIDER_PRIORITY_APPLICATION,
	)

	gtk.StyleContextAddProviderForScreen(
		display.DefaultScreen(),
		userProvider,
		gtk.STYLE_PROVIDER_PRIORITY_USER,
	)

	display.Connect("closed", func() {
		gtk.StyleContextRemoveProviderForScreen(display.DefaultScreen(), defaultProvider)
		gtk.StyleContextRemoveProviderForScreen(display.DefaultScreen(), userProvider)
	})
}
