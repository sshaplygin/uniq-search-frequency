package main

import (
	"reflect"
	"testing"
)

func Test_sortUniqSearches(t *testing.T) {
	tt := []struct {
		Name      string
		Freq      map[string]freq
		ExpSearch []search
	}{
		{
			"empty freq",
			nil,
			nil,
		},
		{
			"with single search",
			map[string]freq{
				"new": {count: 1, pos: 1},
			},
			[]search{
				{query: "new", count: 1, pos: 1},
			},
		},
		{
			"search with sorting",
			map[string]freq{
				"new":  {count: 1, pos: 1},
				"asd":  {count: 1, pos: 3},
				"test": {count: 2, pos: 2},
			},
			[]search{
				{query: "test", count: 2, pos: 2},
				{query: "new", count: 1, pos: 1},
				{query: "asd", count: 1, pos: 3},
			},
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.Name, func(t *testing.T) {
			search := sortUniqSearches(tc.Freq)

			if !reflect.DeepEqual(tc.ExpSearch, search) {
				t.Errorf("search = %#v, want %#v", search, tc.ExpSearch)
			}
		})
	}
}
