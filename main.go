package main

import (
	"flag"
	"log"
	"runtime/debug"
	"time"

	"github.com/AlekSi/pointer"
)

const withoutMemoryLimit = -1

var (
	inputFlag  = flag.String("input", "input.txt", "Set input filename. Defaul value: input.txt Example: --input=inuput.txt")
	outputFlag = flag.String("output", "output.tsv", "Set output filename. Default value: output.tsv Example: --output=output.tsv")
	nFlag      = flag.Int("n", withoutMemoryLimit, "Set memory limit for first uniques search queries. Defaul value: -1 - without limit. Example: --n=3")
)

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
	if n == 0 || n < withoutMemoryLimit {
		log.Println("input unique search limit is zero")

		return
	}

	inputFile := pointer.GetString(inputFlag)
	outputFile := pointer.GetString(outputFlag)

	sortFunc := externalSort
	if n == withoutMemoryLimit {
		sortFunc = inMemorySort
	}

	err := sortFunc(inputFile, outputFile, n)
	if err != nil {
		log.Println("sort result:", err)

		return
	}

	log.Println("data was written to output file", outputFile)
}
