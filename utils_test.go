package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_sortUniqSearches(t *testing.T) {
	tt := []struct {
		Name      string
		Freq      map[string]*freq
		ExpSearch []search
	}{
		{
			"empty freq",
			nil,
			nil,
		},
		{
			"with single search",
			map[string]*freq{
				"new": {1, 1},
			},
			[]search{
				{"new", &freq{1, 1}},
			},
		},
		{
			"search with sorting",
			map[string]*freq{
				"new":  {1, 1},
				"asd":  {1, 3},
				"test": {2, 2},
			},
			[]search{
				{"test", &freq{2, 2}},
				{"new", &freq{1, 1}},
				{"asd", &freq{1, 3}},
			},
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.Name, func(t *testing.T) {
			search := sortUniqSearches(tc.Freq)

			assert.Equal(t, tc.ExpSearch, search)
		})
	}
}
