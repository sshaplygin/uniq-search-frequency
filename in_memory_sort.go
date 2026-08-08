package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func inMemorySort(inputFile, outputFile string, _ int) error {
	file, err := os.Open(inputFile) // #nosec G304 -- the CLI intentionally accepts an input path from the user.
	if err != nil {
		return fmt.Errorf("open input file: %w", err)
	}

	rows, frequency, readErr := countSearchQueriesFreq(file)
	closeErr := file.Close()
	if readErr != nil {
		return fmt.Errorf("read input file: %w", readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close input file: %w", closeErr)
	}

	searches := sortUniqSearches(frequency)
	if err := writeOutput(outputFile, searches); err != nil {
		return err
	}

	logProcessed(rows, len(frequency))

	return nil
}

func countSearchQueriesFreq(reader io.Reader) (int64, map[string]freq, error) {
	frequency := make(map[string]freq)

	rows, err := forEachSearchQuery(reader, func(query string, pos int64) error {
		value, found := frequency[query]
		if !found {
			frequency[query] = freq{count: 1, pos: pos}
			return nil
		}

		value.count++
		frequency[query] = value

		return nil
	})
	if err != nil {
		return rows, nil, err
	}

	return rows, frequency, nil
}

func forEachSearchQuery(reader io.Reader, callback func(query string, pos int64) error) (int64, error) {
	bufferedReader := bufio.NewReader(reader)

	var rows int64
	for {
		line, err := bufferedReader.ReadString('\n')
		if len(line) > 0 {
			rows++
			if callbackErr := callback(strings.TrimSpace(line), rows); callbackErr != nil {
				return rows, callbackErr
			}
		}

		if err == io.EOF {
			return rows, nil
		}
		if err != nil {
			return rows, fmt.Errorf("read query at row %d: %w", rows+1, err)
		}
	}
}

func writeOutput(outputFile string, searches []search) (err error) {
	return writeOutputRecords(outputFile, func(writer *csv.Writer) error {
		for _, value := range searches {
			if err := writer.Write([]string{value.query, strconv.FormatInt(value.count, 10)}); err != nil {
				return fmt.Errorf("write output record: %w", err)
			}
		}

		return nil
	})
}

func writeOutputRecords(outputFile string, writeRecords func(*csv.Writer) error) (err error) {
	outputFile = filepath.Clean(outputFile)
	temporaryFile, err := os.CreateTemp(filepath.Dir(outputFile), "."+filepath.Base(outputFile)+"-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary output file: %w", err)
	}

	temporaryName := temporaryFile.Name()
	completed := false
	defer func() {
		if !completed {
			_ = temporaryFile.Close()
			_ = os.Remove(temporaryName)
		}
	}()

	writer := csv.NewWriter(temporaryFile)
	writer.Comma = '\t'
	if err := writeRecords(writer); err != nil {
		return err
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush output file: %w", err)
	}
	if err := temporaryFile.Close(); err != nil {
		return fmt.Errorf("close output file: %w", err)
	}
	if err := os.Rename(temporaryName, outputFile); err != nil {
		return fmt.Errorf("replace output file: %w", err)
	}

	completed = true

	return nil
}
