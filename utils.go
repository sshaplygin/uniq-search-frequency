package main

import (
	"log"
	"sort"
)

func logUnhandledErr(err error) {
	if err != nil {
		log.Println(err)
	}
}

func logProcessed(rows int64, unique int) {
	log.Println("processed queries:", rows)
	log.Println("unique queries:", unique)
}

func sortUniqSearches(frequency map[string]freq) []search {
	return sortSearches(frequency, lessByOutput)
}

func sortSearchesByQuery(frequency map[string]freq) []search {
	return sortSearches(frequency, lessByQuery)
}

func sortOutputSearches(searches []search) []search {
	sort.Slice(searches, func(i, j int) bool {
		return lessByOutput(searches[i], searches[j])
	})

	return searches
}

func sortSearches(frequency map[string]freq, less func(search, search) bool) []search {
	if len(frequency) == 0 {
		return nil
	}

	searches := make([]search, 0, len(frequency))
	for query, value := range frequency {
		searches = append(searches, search{query: query, count: value.count, pos: value.pos})
	}

	sort.Slice(searches, func(i, j int) bool {
		return less(searches[i], searches[j])
	})

	return searches
}

func lessByOutput(left, right search) bool {
	if left.count != right.count {
		return left.count > right.count
	}
	if left.pos != right.pos {
		return left.pos < right.pos
	}

	return left.query < right.query
}

func lessByQuery(left, right search) bool {
	if left.query != right.query {
		return left.query < right.query
	}

	return left.pos < right.pos
}
