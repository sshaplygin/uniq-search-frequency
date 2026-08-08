package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	withoutMemoryLimit = -1
	defaultInputFile   = "data/input.txt"
	defaultOutputFile  = "data/output.tsv"
)

var (
	inputFlag  = flag.String("input", defaultInputFile, "Set input filename. Default value: data/input.txt. Example: --input=custom-input.txt")
	outputFlag = flag.String("output", defaultOutputFile, "Set output filename. Default value: data/output.tsv. Example: --output=custom-output.tsv")
	nFlag      = flag.Int("n", withoutMemoryLimit, "Set memory limit for unique search queries. Default value: -1 (without limit). Example: --n=3")
)

func main() {
	startedAt := time.Now()
	flag.Parse()

	err := run(*inputFlag, *outputFlag, *nFlag)
	if err != nil {
		log.Println("sort result:", err)
		log.Println("execution time:", time.Since(startedAt))
		os.Exit(1)
	}

	log.Println("data was written to output file:", filepath.Clean(*outputFlag))
	log.Println("execution time:", time.Since(startedAt))
}

func run(inputFile, outputFile string, n int) error {
	if n == 0 || n < withoutMemoryLimit {
		return fmt.Errorf("input unique search limit is unsupported")
	}

	if inputFile == "" {
		return fmt.Errorf("input filename is required")
	}
	if outputFile == "" {
		return fmt.Errorf("output filename is required")
	}

	inputFile = filepath.Clean(inputFile)
	outputFile = filepath.Clean(outputFile)

	sortFunc := externalSort
	if n == withoutMemoryLimit {
		sortFunc = inMemorySort
	}

	return sortFunc(inputFile, outputFile, n)
}
