package appindex

// Searcher describes a search service.
type Searcher interface {
	// Index indexes the searcher with the given strings for indexing.
	Index(entries []string)
	// Search searches up the given query and returns the list of indices
	// relative to the last entries given to Index.
	Search(query string) []int
}

func NewSubstringSearcher(caseSensitive bool) Searcher {}

func NewFuzzySearcher() Searcher {

}
