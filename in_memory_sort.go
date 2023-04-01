package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func inMemorySort(inputFile, outputFile string, _ int) error {
	file, err := os.Open(filepath.Clean(inputFile))
	if err != nil {
		log.Println("open input file", err)

		return err
	}

	defer func() {
		if err = file.Close(); err != nil {
			log.Println("close input file", err)
		}
	}()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	countRows, searchFreq := countSearchQueriesFreq(scanner)

	if scanner.Err() != nil {
		log.Println("scanner after read", err)

		return err
	}

	log.Println("data was read from input file", inputFile)

	log.Println("processed queries:", countRows)
	log.Println("unique queries:", len(searchFreq))

	uniqSearches := sortUniqSearches(searchFreq)

	f, err := os.Create(filepath.Clean(outputFile))
	if err != nil {
		log.Println("create output file", err)

		return err
	}

	defer func() {
		err = f.Close()
		checkErr(err)
	}()

	for i := 0; i < len(uniqSearches); i++ {
		_, err = f.WriteString(fmt.Sprintf("%s\t%d\n", uniqSearches[i].query, uniqSearches[i].freq.count))
		if err != nil {
			log.Println("write data to output", err)

			return err
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
