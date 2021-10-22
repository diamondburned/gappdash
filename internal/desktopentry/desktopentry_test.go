package desktopentry

import "testing"

func TestListDesktopEntries(t *testing.T) {
	entries := List(EntrySortedAlphabetically)
	if len(entries) == 0 {
		t.Fatal("no entries found")
	}

	for _, entry := range entries {
		t.Logf("got entry %s (%s)", entry.DisplayName(), entry.Executable())
	}
}

func BenchmarkListDesktopEntries(b *testing.B) {
	b.Run("sort-alphabetically", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			List(EntrySortedAlphabetically)
		}
	})

	b.Run("sort-modtime", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			List(EntrySortedModTime)
		}
	})
}
