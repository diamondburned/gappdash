package desktopentry

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/diamondburned/gappdash/internal/sortutil"
	"github.com/rkoesters/xdg/desktop"
	"github.com/rkoesters/xdg/keyfile"
)

func appDirs() []string {
	// https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
	// https://github.com/nwg-piotr/nwg-drawer/blob/main/tools.go#L222
	var paths pathValidator

	homeDir, _ := os.UserHomeDir()

	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		paths.addPath(filepath.Join(dataHome, "applications"))
	} else if homeDir != "" {
		paths.addPath(filepath.Join(homeDir, ".local", "share", "applications"))
	}

	dataDirs := os.Getenv("XDG_DATA_DIRS")
	if dataDirs == "" {
		dataDirs = "/usr/local/share/:/usr/share/"
	}

	for _, dataDir := range strings.Split(dataDirs, ":") {
		paths.addPath(filepath.Join(dataDir, "applications"))
	}

	return paths
}

// AppDirs is a list of possible application directories in the application.
// Only valid directories will be there.
var AppDirs = appDirs()

type pathValidator []string

func (p *pathValidator) addPath(path string) {
	s, err := os.Stat(path)
	if err == nil && s.IsDir() {
		*p = append(*p, path)
	}
}

// ListErrors can contain multiple IO errors that are put in by
// ListDesktopEntries.
type ListErrors []error

// Error returns the first error and a count for all.
func (errs ListErrors) Error() string {
	if len(errs) == 0 {
		return ""
	}
	return fmt.Sprintf("%s (%d errors total)", errs[0], len(errs))
}

// EntrySortType describes possible types to sort entries by.
type EntrySortType uint8

const (
	// EntryUnsorted leaves the list of entries unsorted. The order of these
	// entries is undefined.
	EntryUnsorted EntrySortType = iota
	EntrySortedAlphabetically
	EntrySortedAlphabeticallyReverse
	EntrySortedModTime
	EntrySortedModTimeReverse
	entrySortedMax
)

type entrySorter struct {
	entries  []desktop.Entry
	stats    []fs.FileInfo
	sortType EntrySortType
}

var _ sort.Interface = (*entrySorter)(nil)

func (s *entrySorter) Len() int { return len(s.stats) }

func (s *entrySorter) Swap(i, j int) {
	s.stats[i], s.stats[j] = s.stats[j], s.stats[i]
	s.entries[i], s.entries[j] = s.entries[j], s.entries[i]
}

func (s *entrySorter) Less(i, j int) bool {
	switch s.sortType {
	case EntrySortedAlphabetically:
		return sortutil.LessFold(s.entries[i].Name, s.entries[j].Name)
	case EntrySortedAlphabeticallyReverse:
		return sortutil.LessFold(s.entries[j].Name, s.entries[i].Name)
	case EntrySortedModTime:
		return s.modTime(i).Before(s.modTime(j))
	case EntrySortedModTimeReverse:
		return s.modTime(i).After(s.modTime(j))
	default:
		return false
	}
}

func (s *entrySorter) modTime(i int) time.Time {
	if s.stats == nil || s.stats[i] == nil {
		return time.Time{}
	}
	return s.stats[i].ModTime()
}

// List searches the given paths for all valid desktop files. Invalid desktop
// files are ignored. Files that cannot be opened will be ignored as well, but
// the returned errors will be of type ListErrors.
func List(sortBy EntrySortType) ([]desktop.Entry, error) {
	type input struct {
		entry fs.DirEntry
		abs   string
	}

	type output struct {
		entry *desktop.Entry
		stat  fs.FileInfo
		err   error
	}

	var needStat bool

	switch sortBy {
	case EntrySortedModTime, EntrySortedModTimeReverse:
		needStat = true
	}

	var errs ListErrors
	var stats []fs.FileInfo
	var entries []desktop.Entry

	// Overengineering, go! Have maximum (nproc * 2) workers.
	parallel := runtime.GOMAXPROCS(-1) * 2
	outputCh := make(chan output)
	inputCh := make(chan input)
	doneSig := make(chan struct{})

	var outputWg sync.WaitGroup
	outputWg.Add(1)
	go func() {
		defer outputWg.Done()

		execs := make(map[string]struct{})

		for {
			select {
			case <-doneSig:
				return
			case output := <-outputCh:
				if output.err != nil {
					errs = append(errs, output.err)
					break
				}
				// Ignore duplicates.
				if _, ok := execs[output.entry.Exec]; ok {
					break
				}
				execs[output.entry.Exec] = struct{}{}
				stats = append(stats, output.stat)
				entries = append(entries, *output.entry)
			}
		}
	}()

	var inputWg sync.WaitGroup
	for i := 0; i < parallel; i++ {
		inputWg.Add(1)
		go func() {
			defer inputWg.Done()

			for input := range inputCh {
				e, err := parseDesktopEntryFile(input.abs)
				if err != nil {
					outputCh <- output{err: err}
					continue
				}

				var stat fs.FileInfo
				if needStat {
					stat, _ = input.entry.Info()
				}

				outputCh <- output{
					entry: e,
					stat:  stat,
				}
			}
		}()
	}

	for _, path := range AppDirs {
		d, err := os.ReadDir(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		for _, entry := range d {
			inputCh <- input{
				entry: entry,
				abs:   filepath.Join(path, entry.Name()),
			}
		}
	}

	// Close the path input channel, which would cause the workers to exit soon
	// after.
	close(inputCh)
	// Wait for the input goroutines to finish sending jobs.
	inputWg.Wait()
	// Signal the collector goroutine to stop.
	close(doneSig)
	// Wait for the collector goroutine to finish.
	outputWg.Wait()

	// Sort all entries if sortBy asks to.
	if sortBy > EntryUnsorted && sortBy < entrySortedMax {
		sort.Sort(&entrySorter{
			entries:  entries,
			stats:    stats,
			sortType: sortBy,
		})
	}

	return entries, errs
}

var locale = keyfile.DefaultLocale()

func parseDesktopEntryFile(path string) (*desktop.Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return desktop.NewWithLocale(f, locale)
}
