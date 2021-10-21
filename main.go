package main

import (
	"os"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
)

func main() {
	app := gtk.NewApplication(
		"com.github.diamondburned.gappdash",
		gio.ApplicationIsService|gio.ApplicationIsLauncher)
	app.Connect("activate", activate)

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func activate() {

}
