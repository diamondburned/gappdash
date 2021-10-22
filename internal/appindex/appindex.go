// Package appindex provides an application file lister, indexer and searcher.
package appindex

import (
	"strings"
	"sync"
	"time"

	"github.com/diamondburned/gappdash/internal/desktopentry"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
)

// Index is the application indexer. All its methods are thread-safe.
type Index struct {
	Searcher Searcher
	MaxAge   time.Duration
	SortType desktopentry.EntrySortType

	searchResults []gio.AppInfor

	mutex   sync.Mutex
	entries entryIndex

	reindexing bool
}

type entryIndex struct {
	entries       []gio.AppInfor
	searchEntries []string
	lastIndexed   time.Time
}

// NewIndex creates a new indexer.
func NewIndex(searcher Searcher) *Index {
	return &Index{
		Searcher:      searcher,
		MaxAge:        30 * time.Minute,
		SortType:      desktopentry.EntrySortedModTimeReverse,
		searchResults: make([]gio.AppInfor, 0, 50),
	}
}

// AllEntries returns all entries.
func (i *Index) AllEntries() []gio.AppInfor {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	return i.entries.entries
}

// Search searches the index for the given query.
func (i *Index) Search(query string) []gio.AppInfor {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	if i.entries.lastIndexed.Add(i.MaxAge).Before(time.Now()) && !i.reindexing {
		// Expired. Queue a reindexing.
		i.reindexing = true
		go i.asyncReindex(func() { i.reindexing = false })
	}

	i.searchResults = i.searchResults[:0]
	for _, idx := range i.Searcher.Search(query) {
		i.searchResults = append(i.searchResults, i.entries.entries[idx])
	}

	return i.searchResults
}

// Reindex forces the index to be reindexed synchronously.
func (i *Index) Reindex() {
	i.asyncReindex(nil)
}

func (i *Index) asyncReindex(then func()) {
	// TODO: reuse a buffer.
	idx := entryIndex{}
	forceReindex(i.SortType, &idx)

	i.mutex.Lock()

	i.entries = idx
	// TODO: decouple indexing from the searcher.
	i.Searcher.Index(i.entries.searchEntries)

	if then != nil {
		then()
	}

	i.mutex.Unlock()
}

func forceReindex(sortType desktopentry.EntrySortType, idx *entryIndex) {
	idx.entries = desktopentry.List(sortType)

	if cap(idx.searchEntries) >= len(idx.entries) {
		idx.searchEntries = idx.searchEntries[:0]
	} else {
		idx.searchEntries = make([]string, 0, len(idx.entries))
	}

	for _, entry := range idx.entries {
		idx.searchEntries = append(idx.searchEntries, buildEntryQuery(entry))
	}

	idx.lastIndexed = time.Now()
}

func buildEntryQuery(entry gio.AppInfor) string {
	list := []string{
		entry.DisplayName(),
		entry.Description(),
		entry.Executable(),
	}

	return strings.Join(list, " ")
}
