package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
)

const withoutMemoryLimit = -1

var (
	outputFlag = flag.String("output", "output.tsv", "Set output filename. Default value: output.tsv Example: --output=output.tsv")
	inputFlag  = flag.String("input", "input.txt", "Set input filename. Defaul value is empty string. Example: --input=inuput.txt")
	nFlag      = flag.Int("n", withoutMemoryLimit, "Set memory limit for first uniques search queries. Defaul value: -1 - without limit. Example: --n=3")
)

type TextScanner interface {
	Text() string
	Scan() bool
}

type search struct {
	query string
	freq  *freq
}

type freq struct {
	count int
	pos   int
}

func main() {
	now := time.Now()
	defer log.Println("execution time:", time.Since(now))

	defer func() {
		if panicErr := recover(); panicErr != nil {
			log.Printf("recover panic: %+v\n%s\n", panicErr, debug.Stack())
		}
	}()

	flag.Parse()

	n := pointer.GetInt(nFlag)
	if n == 0 {
		log.Println("input unique search limit is zero")

		return
	}

	input := pointer.GetString(inputFlag)

	file, err := os.Open(filepath.Clean(input))
	if err != nil {
		log.Println("open input file", err)

		return
	}

	defer func() {
		if err = file.Close(); err != nil {
			log.Println("close input file", err)
		}
	}()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	countRows, searchFreq := countSearchQueriesFreq(scanner, n)

	log.Println("data was read from input file", input)

	defer log.Println("processed queries:", countRows)
	defer log.Println("unique queries:", len(searchFreq))

	uniqSearches := sortUniqSearches(searchFreq)

	output := pointer.GetString(outputFlag)

	f, err := os.Create(filepath.Clean(output))
	if err != nil {
		log.Println("create output file", err)

		return
	}

	defer func() {
		if err = f.Close(); err != nil {
			log.Println("close file", err)
		}
	}()

	for i := 0; i < len(uniqSearches); i++ {
		_, err = f.WriteString(fmt.Sprintf("%s\t%d\n", uniqSearches[i].query, uniqSearches[i].freq.count))
		if err != nil {
			log.Println("write data to output", err)

			return
		}
	}

	log.Println("data was written to output file", output)
}

func countSearchQueriesFreq(scanner TextScanner, memoryLimit int) (int, map[string]*freq) {
	if scanner == nil {
		return 0, nil
	}

	var frequency map[string]*freq
	if memoryLimit > 0 {
		frequency = make(map[string]*freq, memoryLimit)
	} else {
		frequency = make(map[string]*freq)
	}

	var query string
	var rows int
	for scanner.Scan() && (len(frequency) < memoryLimit || memoryLimit == withoutMemoryLimit) {
		query = strings.TrimSpace(scanner.Text())

		rows++

		if _, ok := frequency[query]; !ok {
			frequency[query] = &freq{0, rows}
		}

		frequency[query].count++
	}

	return rows, frequency
}

func sortUniqSearches(frequency map[string]*freq) []search {
	if len(frequency) == 0 {
		return nil
	}

	searches := make([]search, 0, len(frequency))
	for query, count := range frequency {
		count := count

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
