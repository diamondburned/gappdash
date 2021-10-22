package main

import (
	"log"
	"os"

	"github.com/diamondburned/gappdash/internal/appindex"
	"github.com/diamondburned/gappdash/internal/desktopentry"
	"github.com/diamondburned/gotk4-layer-shell/pkg/gtklayershell"
	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gotk4/pkg/pango"
)

const appID = "com.github.diamondburned.gappdash"

func main() {
	glib.LogUseDefaultLogger()

	app := gtk.NewApplication(appID, 0)
	app.Connect("activate", activate)

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

var app struct {
	*gtk.Application
	cfg *Config
	idx *appindex.Index

	// previously opened window
	window *window

	reindexing bool
}

func activate(gapp *gtk.Application) {
	if app.Application == nil {
		app.Application = gapp

		initStyles()

		cfg, err := ParseUserConfig()
		if err != nil {
			log.Fatalln("config error:", err)
		}

		app.cfg = cfg

		if app.cfg.LayerShell.Enable && !gtklayershell.IsSupported() {
			log.Fatalln("layer-shell not supported; disable them in the config")
		}

		if cfg.App.Daemonize {
			app.Hold()
		}

		var searcher appindex.Searcher
		if cfg.App.Fuzzy {
			searcher = appindex.NewFuzzySearcher()
		} else {
			searcher = appindex.NewSubstringSearcher(cfg.App.CaseSensitive)
		}

		app.idx = appindex.NewIndex(searcher)
		app.idx.SortType = desktopentry.EntrySortedAlphabetically
		app.idx.MaxAge = cfg.App.IndexAge
		app.idx.Reindex()

		// app.pbc = pixbufcache.NewCache(app.cfg.App.IconSize)
	} else {
		// Asynchronously refresh the cache. This will be pretty much instant.
		if !app.reindexing {
			app.reindexing = true

			go func() {
				app.idx.Reindex()
				glib.IdleAdd(func() { app.reindexing = false })
			}()
		}
	}

	// See if we already have a window. Reuse that if possible.
	if app.window != nil {
		app.window.Show()
		app.window.entry.SetText("")
		app.window.entry.GrabFocus()
		return
	}

	app.window = openWindow()
	app.window.Connect("destroy", func() {
		// Invalidate the global window on destroying.
		app.window = nil
	})
}

type window struct {
	*gtk.ApplicationWindow
	entry *gtk.Entry
}

func openWindow() *window {
	w := gtk.NewApplicationWindow(app.Application)
	w.SetTitle("gappdash")
	w.SetSizeRequest(app.cfg.Window.Width, app.cfg.Window.Height)

	if lshell := app.cfg.LayerShell; lshell.Enable {
		gtklayershell.InitForWindow(&w.Window)
		gtklayershell.SetLayer(&w.Window, lshell.Layer.Layer())
		for _, anchor := range lshell.Anchors {
			gtklayershell.SetAnchor(&w.Window, anchor.Edge(), false)
		}
		for edge, margin := range lshell.TransformMargins() {
			gtklayershell.SetMargin(&w.Window, edge, margin)
		}
	} else {
		header := gtk.NewHeaderBar()
		header.SetTitle("gappdash")
		header.SetShowCloseButton(false)

		// Only show a hide button.
		hide := gtk.NewButtonFromIconName("window-close-symbolic", int(gtk.IconSizeButton))
		hide.ConnectClicked(shutWindow)

		header.PackEnd(hide)

		w.SetTitlebar(header)
	}

	w.Show()

	grid := gtk.NewFlowBox()
	grid.SetActivateOnSingleClick(true)
	grid.SetVAlign(gtk.AlignStart)
	grid.SetHAlign(gtk.AlignCenter)
	grid.SetHomogeneous(true)
	grid.SetMinChildrenPerLine(app.cfg.App.Grid.MinChildrenPerLine)
	grid.SetMaxChildrenPerLine(app.cfg.App.Grid.MaxChildrenPerLine)
	grid.SetSelectionMode(gtk.SelectionSingle)
	grid.Show()
	addCSSClass(grid, "app-grid")

	entries := app.idx.AllEntries()

	grid.Connect("child-activated", func(child *gtk.FlowBoxChild) {
		desktopentry.Exec(entries[child.Index()])
		shutWindow()
	})

	noResults := noResultsPage()

	stack := gtk.NewStack()
	stack.AddNamed(grid, "main")
	stack.AddNamed(noResults, "no-results")
	stack.SetTransitionDuration(100)
	stack.SetTransitionType(gtk.StackTransitionTypeCrossfade)
	stack.Show()

	update := func() {
		// Remove all children.
		for _, widget := range grid.Children() {
			widget.BaseWidget().Destroy()
		}

		if len(entries) == 0 {
			stack.SetVisibleChild(noResults)
			return
		}

		stack.SetVisibleChild(grid)

		for i, entry := range entries {
			icon := gtk.NewImageFromGIcon(entry.Icon(), int(app.cfg.App.StockIconSize()))
			name := entry.DisplayName()

			label := gtk.NewLabel(name)
			label.SetTooltipText(name)
			label.SetYAlign(1)
			singlelineLabel(label)

			overlay := gtk.NewOverlay()
			overlay.Add(icon)
			overlay.AddOverlay(label)

			evbox := gtk.NewEventBox()
			addCSSClass(evbox, "grid-item")
			evbox.AddEvents(int(gdk.EnterNotifyMask | gdk.LeaveNotifyMask))
			evbox.Connect("enter-notify-event", func() {
				multilineLabel(label)
				addCSSClass(evbox, "hover")
			})
			evbox.Connect("leave-notify-event", func() {
				singlelineLabel(label)
				removeCSSClass(evbox, "hover")
			})
			evbox.Add(overlay)

			child := gtk.NewFlowBoxChild()
			child.Add(evbox)

			if i == 0 {
				grid.SelectChild(child)
			}

			grid.Add(child)
		}

		grid.ShowAll()
	}

	update()

	scroll := gtk.NewScrolledWindow(nil, nil)
	scroll.Add(stack)
	scroll.Show()

	buffer := gtk.NewEntryBuffer("", -1)

	updateBuffer := func() {
		if text := buffer.Text(); text != "" {
			entries = app.idx.Search(text)
		} else {
			entries = app.idx.AllEntries()
		}
		update()
	}
	buffer.Connect("deleted-text", updateBuffer)
	buffer.Connect("inserted-text", updateBuffer)

	entry := gtk.NewEntryWithBuffer(buffer)
	entry.SetHAlign(gtk.AlignCenter)
	entry.SetVAlign(gtk.AlignCenter)
	entry.SetVExpand(true)
	entry.SetPlaceholderText("Search...")
	addCSSClass(entry, "search-entry")

	entry.Connect("activate", func() {
		// On Enter, activate entry if any.
		if selected := grid.SelectedChildren(); len(selected) > 0 {
			selected[0].Activate()
		}
	})

	// Focus on the input if the window is focused.
	w.Connect("notify::is-active", func() {
		if w.IsActive() {
			entry.GrabFocus()
		}
	})

	entryBox := gtk.NewBox(gtk.OrientationVertical, 0)
	entryBox.SetVAlign(gtk.AlignStart)
	entryBox.SetHExpand(true)
	entryBox.Add(entry)
	addCSSClass(entryBox, "search-entry-box")

	overlay := gtk.NewOverlay()
	overlay.Add(scroll)
	overlay.AddOverlay(entryBox)

	addCSSClass(w, "gappdash-window")
	w.Add(overlay)
	w.SetDeletable(false)
	w.ShowAll()

	// w.ConnectAfter("key-press-event", func(key *gdk.EventKey) bool {
	// 	log.Println("keyVal =", key.Keyval())
	// 	return false
	// 	// return true
	// })

	return &window{
		ApplicationWindow: w,
		entry:             entry,
	}
}

// shutWindow shuts the current window. It does nothing if the window isn't
// there.
func shutWindow() {
	if app.window == nil {
		return
	}

	app.window.Hide()
}

func multilineLabel(label *gtk.Label) {
	label.SetLineWrapMode(pango.WrapWordChar)
	label.SetLineWrap(true)
	label.SetLines(2)
	label.SetSingleLineMode(false)
	label.SetEllipsize(pango.EllipsizeEnd)
}

func singlelineLabel(label *gtk.Label) {
	label.SetLineWrap(false)
	label.SetLines(1)
	label.SetSingleLineMode(true)
	label.SetEllipsize(pango.EllipsizeEnd)
}

func noResultsPage() gtk.Widgetter {
	label := gtk.NewLabel("No results.")
	addCSSClass(label, "no-results")
	return label
}

func addCSSClass(w gtk.Widgetter, classes ...string) {
	ctx := w.BaseWidget().StyleContext()
	for _, class := range classes {
		ctx.AddClass(class)
	}
}

func removeCSSClass(w gtk.Widgetter, classes ...string) {
	ctx := w.BaseWidget().StyleContext()
	for _, class := range classes {
		ctx.RemoveClass(class)
	}
}
