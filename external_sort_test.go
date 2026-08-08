package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type outputRecord struct {
	query string
	count string
}

func TestSortModesProduceStableTSV(t *testing.T) {
	queries := []string{"red apple", "blue\tberry", "solo"}
	for i := 0; i < 12; i++ {
		queries = append(queries, fmt.Sprintf("query %02d", i))
	}
	queries = append(queries, "red apple", "blue\tberry", "red apple")

	want := []outputRecord{
		{query: "red apple", count: "3"},
		{query: "blue\tberry", count: "2"},
		{query: "solo", count: "1"},
	}
	for i := 0; i < 12; i++ {
		want = append(want, outputRecord{query: fmt.Sprintf("query %02d", i), count: "1"})
	}

	inputFile := writeTestInput(t, strings.Join(queries, "\n")+"\n")
	tests := []struct {
		name string
		sort func(string, string, int) error
		n    int
	}{
		{name: "in memory", sort: inMemorySort, n: withoutMemoryLimit},
		{name: "external with one unique slot", sort: externalSort, n: 1},
		{name: "external with three unique slots", sort: externalSort, n: 3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outputFile := filepath.Join(t.TempDir(), "output.tsv")
			if err := test.sort(inputFile, outputFile, test.n); err != nil {
				t.Fatalf("sort: %v", err)
			}

			got := readOutput(t, outputFile)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("output = %#v, want %#v", got, want)
			}
		})
	}
}

func TestSortModesSupportLongQueries(t *testing.T) {
	longQuery := strings.Repeat("x", 128*1024)
	inputFile := writeTestInput(t, longQuery+"\nother search\n"+longQuery+"\n")
	want := []outputRecord{
		{query: longQuery, count: "2"},
		{query: "other search", count: "1"},
	}

	for _, test := range []struct {
		name string
		sort func(string, string, int) error
		n    int
	}{
		{name: "in memory", sort: inMemorySort, n: withoutMemoryLimit},
		{name: "external", sort: externalSort, n: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			outputFile := filepath.Join(t.TempDir(), "output.tsv")
			if err := test.sort(inputFile, outputFile, test.n); err != nil {
				t.Fatalf("sort: %v", err)
			}
			if got := readOutput(t, outputFile); !reflect.DeepEqual(got, want) {
				t.Errorf("output differs from expected result")
			}
		})
	}
}

func TestRunRejectsUnsupportedMemoryLimit(t *testing.T) {
	if err := run("input.txt", "output.tsv", 0); err == nil {
		t.Fatal("run returned nil error for zero memory limit")
	}
}

func TestWriteOutputKeepsExistingFileOnFailure(t *testing.T) {
	outputFile := filepath.Join(t.TempDir(), "output.tsv")
	const existingOutput = "existing\t1\n"
	if err := os.WriteFile(outputFile, []byte(existingOutput), 0o600); err != nil {
		t.Fatalf("write existing output: %v", err)
	}

	err := writeOutputRecords(outputFile, func(writer *csv.Writer) error {
		if err := writer.Write([]string{"new value", "2"}); err != nil {
			return err
		}

		return errors.New("simulated failure")
	})
	if err == nil {
		t.Fatal("writeOutputRecords returned nil error")
	}

	content, err := os.ReadFile(outputFile) // #nosec G304 -- the test reads a path it created in its own temporary directory.
	if err != nil {
		t.Fatalf("read output after failed write: %v", err)
	}
	if string(content) != existingOutput {
		t.Errorf("output = %q, want %q", content, existingOutput)
	}
}

func writeTestInput(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "input.txt")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	return path
}

func readOutput(t *testing.T, path string) []outputRecord {
	t.Helper()

	file, err := os.Open(path) // #nosec G304 -- the test opens a path it created in its own temporary directory.
	if err != nil {
		t.Fatalf("open output: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Errorf("close output: %v", err)
		}
	}()

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	reader.FieldsPerRecord = 2

	var records []outputRecord
	for {
		fields, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return records
			}
			t.Fatalf("read output: %v", err)
		}
		records = append(records, outputRecord{query: fields[0], count: fields[1]})
	}
}
