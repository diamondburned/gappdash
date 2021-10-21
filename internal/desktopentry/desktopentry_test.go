package desktopentry

import (
	"testing"
)

func TestAppDirs(t *testing.T) {
	dirs := AppDirs()
	if len(dirs) == 0 {
		t.Fatal("missing app dirs")
	}

	for _, dir := range dirs {
		t.Log("got app dir", dir)
	}
}

func TestListDesktopEntries(t *testing.T) {
	entries, err := ListDesktopEntries(EntrySortedAlphabetically, AppDirs())
	if err != nil {
		for _, err := range err.(ListErrors) {
			t.Log("error reading entry:", err)
		}
	}

	if len(entries) == 0 {
		t.Fatal("no entries found")
	}

	for _, entry := range entries {
		t.Logf("got entry %s (%s)", entry.Name, entry.Exec)
	}
}

func BenchmarkListDesktopEntries(b *testing.B) {
	dirs := AppDirs()

	b.Run("sort-alphabetically", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ListDesktopEntries(EntrySortedAlphabetically, dirs)
		}
	})

	b.Run("sort-modtime", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ListDesktopEntries(EntrySortedModTime, dirs)
		}
	})
}
