package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestCountSearchQueriesFreq(t *testing.T) {
	count, frequency, err := countSearchQueriesFreq(strings.NewReader("new\ntest\ntest\n"))
	if err != nil {
		t.Fatalf("count queries: %v", err)
	}

	expected := map[string]freq{
		"new":  {count: 1, pos: 1},
		"test": {count: 2, pos: 2},
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
	if !reflect.DeepEqual(frequency, expected) {
		t.Errorf("frequency = %#v, want %#v", frequency, expected)
	}
}
