package main

import (
	"container/heap"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
)

const maxMergeFanIn = 64

type runRecord struct {
	query string
	count int64
	pos   int64
}

type runReader struct {
	path   string
	file   *os.File
	reader *csv.Reader
}

func openRunReader(path string) (*runReader, error) {
	file, err := os.Open(path) // #nosec G304 -- run paths are created by this process in a private temporary directory.
	if err != nil {
		return nil, fmt.Errorf("open run file %q: %w", path, err)
	}

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	reader.FieldsPerRecord = 3

	return &runReader{path: path, file: file, reader: reader}, nil
}

func (reader *runReader) next() (runRecord, bool, error) {
	fields, err := reader.reader.Read()
	if errors.Is(err, io.EOF) {
		return runRecord{}, false, nil
	}
	if err != nil {
		return runRecord{}, false, fmt.Errorf("read run file %q: %w", reader.path, err)
	}

	count, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return runRecord{}, false, fmt.Errorf("parse count in run file %q: %w", reader.path, err)
	}
	pos, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return runRecord{}, false, fmt.Errorf("parse position in run file %q: %w", reader.path, err)
	}

	return runRecord{query: fields[0], count: count, pos: pos}, true, nil
}

func (reader *runReader) close() error {
	if reader.file == nil {
		return nil
	}

	err := reader.file.Close()
	reader.file = nil

	return err
}

type runWriter struct {
	path   string
	file   *os.File
	writer *csv.Writer
}

func newRunWriter(tempDir, prefix string) (*runWriter, error) {
	file, err := os.CreateTemp(tempDir, prefix+"*.tsv")
	if err != nil {
		return nil, fmt.Errorf("create temporary run file: %w", err)
	}

	writer := csv.NewWriter(file)
	writer.Comma = '\t'

	return &runWriter{path: file.Name(), file: file, writer: writer}, nil
}

func (writer *runWriter) write(record runRecord) error {
	return writer.writer.Write([]string{
		record.query,
		strconv.FormatInt(record.count, 10),
		strconv.FormatInt(record.pos, 10),
	})
}

func (writer *runWriter) close() error {
	if writer.file == nil {
		return nil
	}

	writer.writer.Flush()
	writeErr := writer.writer.Error()
	closeErr := writer.file.Close()
	writer.file = nil
	if writeErr != nil {
		return fmt.Errorf("flush temporary run file: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close temporary run file: %w", closeErr)
	}

	return nil
}

func (writer *runWriter) abort() {
	if writer.file == nil {
		return
	}

	_ = writer.file.Close()
	writer.file = nil
	_ = os.Remove(writer.path)
}

type runItem struct {
	record runRecord
	source int
	index  int
}

type runHeap struct {
	items []*runItem
	less  func(runRecord, runRecord) bool
}

var _ heap.Interface = (*runHeap)(nil)

func (heap runHeap) Len() int {
	return len(heap.items)
}

func (heap runHeap) Less(i, j int) bool {
	left := heap.items[i]
	right := heap.items[j]
	if heap.less(left.record, right.record) {
		return true
	}
	if heap.less(right.record, left.record) {
		return false
	}

	return left.source < right.source
}

func (heap runHeap) Swap(i, j int) {
	heap.items[i], heap.items[j] = heap.items[j], heap.items[i]
	heap.items[i].index = i
	heap.items[j].index = j
}

func (heap *runHeap) Push(value any) {
	item := value.(*runItem)
	item.index = len(heap.items)
	heap.items = append(heap.items, item)
}

func (heap *runHeap) Pop() any {
	items := heap.items
	last := len(items) - 1
	item := items[last]
	items[last] = nil
	item.index = -1
	heap.items = items[:last]

	return item
}

func externalSort(inputFile, outputFile string, memLimit int) error {
	if memLimit <= 0 {
		return fmt.Errorf("memory limit must be positive")
	}

	tempDir, err := os.MkdirTemp("", "uniq-search-frequency-")
	if err != nil {
		return fmt.Errorf("create temporary directory: %w", err)
	}
	defer func() {
		logUnhandledErr(os.RemoveAll(tempDir))
	}()

	countRuns, rows, err := createCountRuns(inputFile, tempDir, memLimit)
	if err != nil {
		return err
	}
	if len(countRuns) == 0 {
		return writeOutput(outputFile, nil)
	}

	fanIn := mergeFanIn(memLimit)
	aggregatedRun, err := mergeCountRuns(tempDir, countRuns, fanIn)
	if err != nil {
		return err
	}

	outputRuns, err := createOutputRuns(aggregatedRun, tempDir, memLimit)
	if err != nil {
		return err
	}
	if err := os.Remove(aggregatedRun); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove aggregated run: %w", err)
	}

	finalRun, err := mergeOutputRuns(tempDir, outputRuns, fanIn)
	if err != nil {
		return err
	}
	unique, err := writeOutputFromRun(outputFile, finalRun)
	if err != nil {
		return err
	}

	logProcessed(rows, unique)

	return nil
}

func createCountRuns(inputFile, tempDir string, memLimit int) ([]string, int64, error) {
	file, err := os.Open(inputFile) // #nosec G304 -- the CLI intentionally accepts an input path from the user.
	if err != nil {
		return nil, 0, fmt.Errorf("open input file: %w", err)
	}

	frequency := make(map[string]freq, initialCapacity(memLimit))
	runs := make([]string, 0)
	flush := func() error {
		if len(frequency) == 0 {
			return nil
		}

		path, err := writeRun(tempDir, "count-", sortSearchesByQuery(frequency))
		if err != nil {
			return err
		}
		runs = append(runs, path)
		frequency = make(map[string]freq, initialCapacity(memLimit))

		return nil
	}

	rows, readErr := forEachSearchQuery(file, func(query string, pos int64) error {
		value, found := frequency[query]
		if !found && len(frequency) == memLimit {
			if err := flush(); err != nil {
				return err
			}
			value, found = frequency[query]
		}

		if found {
			value.count++
			frequency[query] = value
			return nil
		}

		frequency[query] = freq{count: 1, pos: pos}

		return nil
	})
	closeErr := file.Close()
	if readErr != nil {
		return nil, rows, fmt.Errorf("read input file: %w", readErr)
	}
	if closeErr != nil {
		return nil, rows, fmt.Errorf("close input file: %w", closeErr)
	}
	if err := flush(); err != nil {
		return nil, rows, err
	}

	return runs, rows, nil
}

func createOutputRuns(inputRun, tempDir string, memLimit int) ([]string, error) {
	reader, err := openRunReader(inputRun)
	if err != nil {
		return nil, err
	}

	searches := make([]search, 0, initialCapacity(memLimit))
	runs := make([]string, 0)
	flush := func() error {
		if len(searches) == 0 {
			return nil
		}

		path, err := writeRun(tempDir, "output-", searches)
		if err != nil {
			return err
		}
		runs = append(runs, path)
		searches = make([]search, 0, initialCapacity(memLimit))

		return nil
	}

	for {
		record, found, err := reader.next()
		if err != nil {
			_ = reader.close()
			return nil, err
		}
		if !found {
			break
		}

		searches = append(searches, search(record))
		if len(searches) == memLimit {
			searches = sortOutputSearches(searches)
			if err := flush(); err != nil {
				_ = reader.close()
				return nil, err
			}
		}
	}
	if err := reader.close(); err != nil {
		return nil, fmt.Errorf("close aggregated run: %w", err)
	}
	if len(searches) > 0 {
		searches = sortOutputSearches(searches)
		if err := flush(); err != nil {
			return nil, err
		}
	}

	return runs, nil
}

func writeRun(tempDir, prefix string, searches []search) (string, error) {
	writer, err := newRunWriter(tempDir, prefix)
	if err != nil {
		return "", err
	}
	defer writer.abort()

	for _, value := range searches {
		if err := writer.write(runRecord(value)); err != nil {
			return "", fmt.Errorf("write temporary run record: %w", err)
		}
	}
	if err := writer.close(); err != nil {
		return "", err
	}

	return writer.path, nil
}

func mergeCountRuns(tempDir string, runs []string, fanIn int) (string, error) {
	return mergeRuns(tempDir, runs, fanIn, mergeCountRunGroup)
}

func mergeCountRunGroup(tempDir string, paths []string) (string, error) {
	readers, queue, err := openRunReaders(paths, lessRunByQuery)
	if err != nil {
		return "", err
	}
	defer closeRunReaders(readers)

	writer, err := newRunWriter(tempDir, "aggregate-")
	if err != nil {
		return "", err
	}
	defer writer.abort()

	var aggregate *runRecord
	for queue.Len() > 0 {
		item := heap.Pop(queue).(*runItem)
		if err := pushNextRecord(queue, readers, item.source); err != nil {
			return "", err
		}

		if aggregate == nil {
			record := item.record
			aggregate = &record
			continue
		}
		if aggregate.query == item.record.query {
			aggregate.count += item.record.count
			if item.record.pos < aggregate.pos {
				aggregate.pos = item.record.pos
			}
			continue
		}

		if err := writer.write(*aggregate); err != nil {
			return "", fmt.Errorf("write aggregated record: %w", err)
		}
		record := item.record
		aggregate = &record
	}
	if aggregate != nil {
		if err := writer.write(*aggregate); err != nil {
			return "", fmt.Errorf("write aggregated record: %w", err)
		}
	}
	if err := writer.close(); err != nil {
		return "", err
	}

	return writer.path, nil
}

func mergeOutputRuns(tempDir string, runs []string, fanIn int) (string, error) {
	return mergeRuns(tempDir, runs, fanIn, mergeOutputRunGroup)
}

func mergeRuns(tempDir string, runs []string, fanIn int, mergeGroup func(string, []string) (string, error)) (string, error) {
	if len(runs) == 0 {
		return "", fmt.Errorf("merge requires at least one run")
	}

	for len(runs) > 1 {
		nextRuns := make([]string, 0, (len(runs)+fanIn-1)/fanIn)
		for start := 0; start < len(runs); start += fanIn {
			end := minInt(start+fanIn, len(runs))
			if end-start == 1 {
				nextRuns = append(nextRuns, runs[start])
				continue
			}

			mergedRun, err := mergeGroup(tempDir, runs[start:end])
			if err != nil {
				return "", err
			}
			if err := removeRuns(runs[start:end]); err != nil {
				return "", err
			}
			nextRuns = append(nextRuns, mergedRun)
		}
		runs = nextRuns
	}

	return runs[0], nil
}

func mergeOutputRunGroup(tempDir string, paths []string) (string, error) {
	readers, queue, err := openRunReaders(paths, lessRunByOutput)
	if err != nil {
		return "", err
	}
	defer closeRunReaders(readers)

	writer, err := newRunWriter(tempDir, "sorted-")
	if err != nil {
		return "", err
	}
	defer writer.abort()

	for queue.Len() > 0 {
		item := heap.Pop(queue).(*runItem)
		if err := writer.write(item.record); err != nil {
			return "", fmt.Errorf("write sorted record: %w", err)
		}
		if err := pushNextRecord(queue, readers, item.source); err != nil {
			return "", err
		}
	}
	if err := writer.close(); err != nil {
		return "", err
	}

	return writer.path, nil
}

func openRunReaders(paths []string, less func(runRecord, runRecord) bool) ([]*runReader, *runHeap, error) {
	readers := make([]*runReader, 0, len(paths))
	queue := &runHeap{items: make([]*runItem, 0, len(paths)), less: less}
	heap.Init(queue)

	for index, path := range paths {
		reader, err := openRunReader(path)
		if err != nil {
			closeRunReaders(readers)
			return nil, nil, err
		}
		readers = append(readers, reader)
		if err := pushNextRecord(queue, readers, index); err != nil {
			closeRunReaders(readers)
			return nil, nil, err
		}
	}

	return readers, queue, nil
}

func pushNextRecord(queue *runHeap, readers []*runReader, source int) error {
	record, found, err := readers[source].next()
	if err != nil {
		return err
	}
	if found {
		heap.Push(queue, &runItem{record: record, source: source})
	}

	return nil
}

func closeRunReaders(readers []*runReader) {
	for _, reader := range readers {
		logUnhandledErr(reader.close())
	}
}

func writeOutputFromRun(outputFile, path string) (int, error) {
	reader, err := openRunReader(path)
	if err != nil {
		return 0, err
	}
	defer func() {
		logUnhandledErr(reader.close())
	}()

	unique := 0
	err = writeOutputRecords(outputFile, func(writer *csv.Writer) error {
		for {
			record, found, err := reader.next()
			if err != nil {
				return err
			}
			if !found {
				return nil
			}
			if err := writer.Write([]string{record.query, strconv.FormatInt(record.count, 10)}); err != nil {
				return fmt.Errorf("write output record: %w", err)
			}
			unique++
		}
	})

	return unique, err
}

func mergeFanIn(memLimit int) int {
	if memLimit < 2 {
		return 2
	}
	if memLimit < maxMergeFanIn {
		return memLimit
	}

	return maxMergeFanIn
}

func initialCapacity(memLimit int) int {
	if memLimit < maxMergeFanIn {
		return memLimit
	}

	return maxMergeFanIn
}

func minInt(left, right int) int {
	if left < right {
		return left
	}

	return right
}

func removeRuns(paths []string) error {
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove temporary run %q: %w", path, err)
		}
	}

	return nil
}

func lessRunByQuery(left, right runRecord) bool {
	if left.query != right.query {
		return left.query < right.query
	}

	return left.pos < right.pos
}

func lessRunByOutput(left, right runRecord) bool {
	return lessByOutput(search(left), search(right))
}
