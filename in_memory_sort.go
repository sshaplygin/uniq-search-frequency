package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

func inMemorySort(inputFile, outputFile string, _ int) error {
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("open input file: %w", err)
	}

	defer func() {
		err = file.Close()
		logUnhandledErr(err)
	}()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	countRows, searchFreq := countSearchQueriesFreq(scanner)

	if scanner.Err() != nil {
		return fmt.Errorf("scanner after read from input file: %w", err)
	}

	log.Println("processed queries:", countRows)
	log.Println("unique queries:", len(searchFreq))

	uniqSearches := sortUniqSearches(searchFreq)

	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}

	defer func() {
		err = f.Close()
		logUnhandledErr(err)
	}()

	for i := 0; i < len(uniqSearches); i++ {
		_, err = f.WriteString(fmt.Sprintf("%s\t%d\n", uniqSearches[i].query, uniqSearches[i].freq.count))
		if err != nil {
			return fmt.Errorf("write data to output: %w", err)
		}
	}

	return nil
}

func countSearchQueriesFreq(scanner TextScanner) (int, map[string]*freq) {
	if scanner == nil {
		return 0, nil
	}

	frequency := make(map[string]*freq)

	var query string
	var rows int
	for scanner.Scan() {
		query = strings.TrimSpace(scanner.Text())

		rows++

		if _, ok := frequency[query]; !ok {
			frequency[query] = &freq{0, rows}
		}

		frequency[query].count++
	}

	return rows, frequency
}
