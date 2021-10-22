package appindex

import (
	"strings"

	"github.com/sahilm/fuzzy"
)

// Searcher describes a search service.
type Searcher interface {
	// Index indexes the searcher with the given strings for indexing.
	Index(entries []string)
	// Search searches up the given query and returns the list of indices
	// relative to the last entries given to Index.
	Search(query string) []int
}

type fuzzySearcher struct {
	data []string
	ints []int
}

// NewFuzzySearcher creates a new fuzzy searcher. Fuzzy searching is always
// case-insensitive.
func NewFuzzySearcher() Searcher {
	return &fuzzySearcher{}
}

func (f *fuzzySearcher) Index(entries []string) { f.data = entries }

func (f *fuzzySearcher) Search(query string) []int {
	f.ints = f.ints[:0]
	for _, match := range fuzzy.Find(query, f.data) {
		f.ints = append(f.ints, match.Index)
	}

	return f.ints
}

// TODO: https://pkg.go.dev/golang.org/x/text/search

type substringSearcher struct {
	data []string
	ints []int
	fold bool
}

// NewSubstringSearcher creates a new substring searcher. If caseSensitive is
// false, then matches are done regardless of the casing.
func NewSubstringSearcher(caseSensitive bool) Searcher {
	return &substringSearcher{
		fold: !caseSensitive,
	}
}

func (s *substringSearcher) Index(data []string) {
	if !s.fold {
		s.data = data
		return
	}

	if cap(s.data) < len(data) {
		s.data = make([]string, 0, len(data))
	} else {
		s.data = s.data[:0]
	}

	for _, str := range data {
		s.data = append(s.data, strings.ToLower(str))
	}
}

func (s *substringSearcher) Search(query string) []int {
	if s.fold {
		query = strings.ToLower(query)
	}

	s.ints = s.ints[:0]
	for i, str := range s.data {
		if strings.Contains(str, query) {
			s.ints = append(s.ints, i)
		}
	}

	return s.ints
}
