package desktopentry

import (
	"context"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"sort"
	"time"

	"github.com/diamondburned/gappdash/internal/sortutil"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
)

// EntrySortType describes possible types to sort entries by.
type EntrySortType uint8

const (
	// EntryUnsorted leaves the list of entries unsorted. The order of these
	// entries is undefined.
	EntryUnsorted EntrySortType = iota
	EntrySortedAlphabetically
	EntrySortedAlphabeticallyReverse
	EntrySortedModTime // exec
	EntrySortedModTimeReverse
	entrySortedMax
)

type entrySorter struct {
	entries  []gio.AppInfor
	names    []string
	stats    []fs.FileInfo
	sortType EntrySortType
}

var _ sort.Interface = (*entrySorter)(nil)

func (s *entrySorter) Len() int { return len(s.entries) }

func (s *entrySorter) Swap(i, j int) {
	s.entries[i], s.entries[j] = s.entries[j], s.entries[i]

	if s.names != nil {
		s.names[i], s.names[j] = s.names[j], s.names[i]
	}

	if s.stats != nil {
		s.stats[i], s.stats[j] = s.stats[j], s.stats[i]
	}
}

func (s *entrySorter) Less(i, j int) bool {
	switch s.sortType {
	case EntrySortedAlphabetically:
		return sortutil.LessFold(s.names[i], s.names[j])
	case EntrySortedAlphabeticallyReverse:
		return sortutil.LessFold(s.names[j], s.names[i])
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
func List(sortBy EntrySortType) []gio.AppInfor {
	apps := gio.AppInfoGetAll()

	filtered := apps[:0]

	for _, app := range apps {
		if app.ShouldShow() {
			filtered = append(filtered, app)
		}
	}

	apps = filtered

	if sortBy <= EntryUnsorted || sortBy >= entrySortedMax {
		return apps
	}

	var names []string
	var stats []fs.FileInfo

	switch sortBy {
	case EntrySortedAlphabetically, EntrySortedAlphabeticallyReverse:
		names = make([]string, len(apps))

		for i, app := range apps {
			names[i] = app.DisplayName()
		}

	case EntrySortedModTime, EntrySortedModTimeReverse:
		stats = make([]fs.FileInfo, len(apps))

		for i, app := range apps {
			s, err := os.Stat(app.Executable())
			if err == nil {
				stats[i] = s
			}
		}
	}

	sort.Sort(&entrySorter{
		entries:  apps,
		names:    names,
		stats:    stats,
		sortType: sortBy,
	})

	return apps
}

// Exec launches the given desktop entry.
func Exec(entry gio.AppInfor) {
	entry.LaunchURIsAsync(context.Background(), nil, nil, func(result gio.AsyncResulter) {
		if err := entry.LaunchURIsFinish(result); err != nil {
			log.Println("failed to launch normally:", err)

			exe := entry.Executable()
			log.Println("trying with os/exec for executable", exe)

			if err := exec.Command(exe).Start(); err != nil {
				log.Println("os/exec failed:", err)
			}
		}
	})
}
