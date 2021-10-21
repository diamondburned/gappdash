// Package appindex provides an application file lister, indexer and searcher.
package appindex

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/diamondburned/gappdash/internal/desktopentry"
	"github.com/rkoesters/xdg/desktop"
)

// Index is the application indexer.
type Index struct {
	Searcher Searcher
	MaxAge   time.Duration
	SortType desktopentry.EntrySortType

	searchResults []desktop.Entry

	mutex   sync.Mutex
	entries entryIndex

	reindexing bool
}

type entryIndex struct {
	entries       []desktop.Entry
	searchEntries []string
	lastIndexed   time.Time
}

// NewIndex creates a new indexer.
func NewIndex(searcher Searcher) *Index {
	return &Index{
		Searcher:      searcher,
		MaxAge:        10 * time.Second,
		searchResults: make([]desktop.Entry, 0, 50),
	}
}

// Search searches the index for the given query.
func (i *Index) Search(query string) []desktop.Entry {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	if i.entries.lastIndexed.Add(i.MaxAge).Before(time.Now()) && !i.reindexing {
		// Expired. Queue a reindexing.
		go i.asyncReindex(func() { i.reindexing = false })
	}

	i.searchResults = i.searchResults[:0]
	for _, idx := range i.Searcher.Search(query) {
		i.searchResults = append(i.searchResults, i.entries.entries[idx])
	}

	return i.searchResults
}

// ForceReindex forces the index to be reindexed synchronously. This method is
// safe to be called concurrently.
func (i *Index) ForceReindex() {
	i.asyncReindex(nil)
}

// func (i *Index) reindexLocked() {
// 	i.mutex.Lock()
// 	i.forceReindex(&i.entries)
// 	i.Searcher.Index(i.entries.searchEntries)
// 	i.mutex.Unlock()
// }

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
	idx.entries, _ = desktopentry.List(sortType)

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

func buildEntryQuery(entry desktop.Entry) string {
	list := []string{
		entry.Name,
		entry.GenericName,
		entry.Comment,
		strings.Join(entry.Keywords, " "),
		strings.Join(entry.Categories, " "),
		execBase(entry.Exec),
	}

	return strings.Join(list, " ")
}

func execBase(exec string) string {
	if exec == "" {
		return exec
	}
	return filepath.Base(strings.Split(exec, " ")[0])
}
