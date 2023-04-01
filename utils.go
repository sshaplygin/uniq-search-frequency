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

func sortUniqSearches(frequency map[string]*freq) []search {
	if len(frequency) == 0 {
		return nil
	}

	searches := make([]search, 0, len(frequency))
	for query, count := range frequency {
		searches = append(searches, search{query, count})
	}

	sort.Slice(searches, func(i, j int) bool {
		if searches[i].freq.count == searches[j].freq.count {
			return searches[i].freq.pos < searches[j].freq.pos
		}

		return searches[i].freq.count > searches[j].freq.count
	})

	return searches
}
