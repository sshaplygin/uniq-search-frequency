package main

import (
	"bufio"
	"container/heap"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const benchmarkUniqueQueries = 4096
const benchmarkMemoryLimit = 64

func BenchmarkExternalSort(b *testing.B) {
	inputFile := writeBenchmarkInput(b, benchmarkUniqueQueries)
	silenceBenchmarkLogs(b)

	benchmarks := []struct {
		name string
		sort func(string, string, int) error
	}{
		{name: "legacy_recursive_batches", sort: legacyExternalSort},
		{name: "new_bounded_merge", sort: externalSort},
	}

	for _, benchmark := range benchmarks {
		b.Run(benchmark.name, func(b *testing.B) {
			outputFile := filepath.Join(b.TempDir(), "output.tsv")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := benchmark.sort(inputFile, outputFile, benchmarkMemoryLimit); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestLegacyExternalSortBenchmarkBaseline(t *testing.T) {
	inputFile := writeBenchmarkInput(t, benchmarkMemoryLimit*2)
	outputFile := filepath.Join(t.TempDir(), "output.tsv")
	if err := legacyExternalSort(inputFile, outputFile, benchmarkMemoryLimit); err != nil {
		t.Fatalf("run legacy baseline: %v", err)
	}

	records := readOutput(t, outputFile)
	if len(records) != benchmarkMemoryLimit*2 {
		t.Fatalf("output records = %d, want %d", len(records), benchmarkMemoryLimit*2)
	}
	for _, record := range records {
		if record.count != "1" {
			t.Errorf("query %q has count %s, want 1", record.query, record.count)
		}
	}
}

func writeBenchmarkInput(testingObject testing.TB, uniqueQueries int) string {
	testingObject.Helper()

	path := filepath.Join(testingObject.TempDir(), "input.txt")
	file, err := os.Create(path) // #nosec G304 -- the benchmark creates the file in its own temporary directory.
	if err != nil {
		testingObject.Fatalf("create benchmark input: %v", err)
	}

	writer := bufio.NewWriter(file)
	for i := 0; i < uniqueQueries; i++ {
		if _, err := fmt.Fprintf(writer, "query-%06d\n", i); err != nil {
			testingObject.Fatalf("write benchmark input: %v", err)
		}
	}
	if err := writer.Flush(); err != nil {
		testingObject.Fatalf("flush benchmark input: %v", err)
	}
	if err := file.Close(); err != nil {
		testingObject.Fatalf("close benchmark input: %v", err)
	}

	return path
}

func silenceBenchmarkLogs(b *testing.B) {
	b.Helper()

	previousOutput := log.Writer()
	log.SetOutput(io.Discard)
	b.Cleanup(func() {
		log.SetOutput(previousOutput)
	})
}

// legacyExternalSort is a test-only copy of the recursive algorithm from main
// before commit 1fe76cd. It intentionally preserves its multi-pass spill model
// so the benchmark measures the cost removed by the production implementation.
func legacyExternalSort(inputFile, outputFile string, memLimit int) error {
	tempDir, err := os.MkdirTemp("", "uniq-search-frequency-legacy-")
	if err != nil {
		return fmt.Errorf("create temporary directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	if err := legacyCountUniqueSearches(tempDir, inputFile, 0, memLimit); err != nil {
		return err
	}

	return legacyMergeFiles(tempDir, outputFile)
}

func legacyCountUniqueSearches(tempDir, inputFile string, batch, memLimit int) error {
	input, err := os.Open(inputFile) // #nosec G304 -- the benchmark supplies an input it created itself.
	if err != nil {
		return fmt.Errorf("open input file: %w", err)
	}
	defer func() {
		_ = input.Close()
	}()

	scanner := bufio.NewScanner(input)
	frequency := make(map[string]legacyFrequency, memLimit)
	var spill *os.File
	var spillName string
	var rows int64
	for scanner.Scan() {
		query := strings.TrimSpace(scanner.Text())
		rows++

		value, found := frequency[query]
		if !found && len(frequency) < memLimit {
			frequency[query] = legacyFrequency{count: 1, pos: rows}
			continue
		}
		if found {
			value.count++
			frequency[query] = value
			continue
		}

		if spill == nil {
			spillName = filepath.Join(tempDir, fmt.Sprintf("spill-%d.txt", batch))
			spill, err = os.Create(spillName) // #nosec G304 -- the benchmark builds this path inside its temporary directory.
			if err != nil {
				return fmt.Errorf("create spill file: %w", err)
			}
			defer func() {
				_ = spill.Close()
			}()
		}
		if _, err := spill.WriteString(query + "\n"); err != nil {
			return fmt.Errorf("write spill file: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan input file: %w", err)
	}

	batchFile := filepath.Join(tempDir, fmt.Sprintf("batch-%d.tsv", batch))
	output, err := os.Create(batchFile) // #nosec G304 -- the benchmark builds this path inside its temporary directory.
	if err != nil {
		return fmt.Errorf("create batch file: %w", err)
	}
	defer func() {
		_ = output.Close()
	}()

	for _, value := range legacySortSearches(frequency) {
		if _, err := fmt.Fprintf(output, "%s\t%d\n", value.query, value.count); err != nil {
			return fmt.Errorf("write batch file: %w", err)
		}
	}

	if spill == nil {
		return nil
	}

	return legacyCountUniqueSearches(tempDir, spillName, batch+1, memLimit)
}

type legacyFrequency struct {
	count int64
	pos   int64
}

type legacySearch struct {
	query string
	count int64
	pos   int64
}

func legacySortSearches(frequency map[string]legacyFrequency) []legacySearch {
	searches := make([]legacySearch, 0, len(frequency))
	for query, value := range frequency {
		searches = append(searches, legacySearch{query: query, count: value.count, pos: value.pos})
	}

	sort.Slice(searches, func(i, j int) bool {
		if searches[i].count != searches[j].count {
			return searches[i].count > searches[j].count
		}

		return searches[i].pos < searches[j].pos
	})

	return searches
}

func legacyMergeFiles(tempDir, outputFile string) error {
	batchFiles, err := filepath.Glob(filepath.Join(tempDir, "batch-*.tsv"))
	if err != nil {
		return fmt.Errorf("find batch files: %w", err)
	}

	inputs := make([]*os.File, 0, len(batchFiles))
	for _, path := range batchFiles {
		input, err := os.Open(path) // #nosec G304 -- batch paths are generated in the benchmark's temporary directory.
		if err != nil {
			legacyCloseFiles(inputs)
			return fmt.Errorf("open batch file: %w", err)
		}
		inputs = append(inputs, input)
	}
	defer legacyCloseFiles(inputs)

	output, err := os.Create(outputFile) // #nosec G304 -- the benchmark creates its output in a temporary directory.
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() {
		_ = output.Close()
	}()

	queue := &legacyHeap{}
	heap.Init(queue)
	for source, input := range inputs {
		if err := legacyPushNext(queue, input, source); err != nil {
			return err
		}
	}

	for queue.Len() > 0 {
		item := heap.Pop(queue).(*legacyItem)
		if _, err := fmt.Fprintf(output, "%s\t%d\n", item.query, item.count); err != nil {
			return fmt.Errorf("write output file: %w", err)
		}
		if err := legacyPushNext(queue, inputs[item.source], item.source); err != nil {
			return err
		}
	}

	return nil
}

func legacyCloseFiles(files []*os.File) {
	for _, file := range files {
		_ = file.Close()
	}
}

type legacyItem struct {
	query  string
	count  int64
	source int
	index  int
}

type legacyHeap []*legacyItem

func (queue legacyHeap) Len() int {
	return len(queue)
}

func (queue legacyHeap) Less(i, j int) bool {
	if queue[i].count != queue[j].count {
		return queue[i].count > queue[j].count
	}

	return queue[i].source < queue[j].source
}

func (queue legacyHeap) Swap(i, j int) {
	queue[i], queue[j] = queue[j], queue[i]
	queue[i].index = i
	queue[j].index = j
}

func (queue *legacyHeap) Push(value any) {
	item := value.(*legacyItem)
	item.index = len(*queue)
	*queue = append(*queue, item)
}

func (queue *legacyHeap) Pop() any {
	items := *queue
	last := len(items) - 1
	item := items[last]
	items[last] = nil
	item.index = -1
	*queue = items[:last]

	return item
}

func legacyPushNext(queue *legacyHeap, input *os.File, source int) error {
	var item legacyItem
	_, err := fmt.Fscanf(input, "%s\t%d", &item.query, &item.count)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read batch file: %w", err)
	}

	item.source = source
	heap.Push(queue, &item)

	return nil
}
