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
	"strings"
)

const tempDir = "tmp"
const inputBatchPrefix = "input"
const outputBatchPrefix = "output"
const outputBatchFileTemplate = "%s-%d.tsv"
const inputBacthFileTemplate = "%s-%d.txt"

func externalSort(inputFile, outputFile string, memLimit int) error {
	dir, err := os.MkdirTemp(os.TempDir(), tempDir)
	if err != nil {
		return err
	}

	defer func() {
		err = os.RemoveAll(dir)
		logUnhandledErr(err)
	}()

	fmt.Println(dir)

	if err = countUniqueSearhes(dir, inputFile, 0, memLimit); err != nil {
		return err
	}

	return mergeFiles(dir, outputFile, memLimit)
}

func countUniqueSearhes(tmpDir string, inputFile string, batch, memLimit int) error {
	input, err := os.Open(inputFile)
	if err != nil {
		return err
	}

	defer func() {
		err = input.Close()
		logUnhandledErr(err)
	}()

	scanner := bufio.NewScanner(input)
	scanner.Split(bufio.ScanLines)

	var query string
	var rows int

	frequency := make(map[string]*freq, memLimit)
	var batchIn *os.File
	var batchInName string
	for scanner.Scan() {
		query = strings.TrimSpace(scanner.Text())

		rows++

		_, ok := frequency[query]
		if !ok && len(frequency) < memLimit {
			frequency[query] = &freq{1, rows}

			continue
		}

		if ok {
			frequency[query].count++

			continue
		}

		if batchIn == nil {
			batchInName = fmt.Sprintf(inputBacthFileTemplate, inputBatchPrefix, batch)
			batchIn, err = os.Create(filepath.Join(tmpDir, batchInName))
			if err != nil {
				return err
			}

			defer func() {
				err = batchIn.Close()
				logUnhandledErr(err)
			}()
		}

		if _, err = batchIn.Write(append([]byte(query), '\n')); err != nil {
			return err
		}
	}

	if scanner.Err() != nil {
		return err
	}

	batchOut, err := os.Create(filepath.Join(tmpDir, fmt.Sprintf(outputBatchFileTemplate, outputBatchPrefix, batch)))
	if err != nil {
		return err
	}

	defer func() {
		err = batchOut.Close()
		logUnhandledErr(err)
	}()

	searches := sortUniqSearches(frequency)
	for _, search := range searches {
		_, err = batchOut.WriteString(fmt.Sprintf("%s\t%d\n", search.query, search.freq.count))
		if err != nil {
			return err
		}
	}

	if batchIn == nil {
		return nil
	}

	return countUniqueSearhes(tmpDir, filepath.Join(tmpDir, batchInName), batch+1, memLimit)
}

func mergeFiles(tmpDir string, outputFile string, n int) error {
	chunkFiles, err := filepath.Glob(filepath.Join(tmpDir, fmt.Sprintf("%s-*.tsv", outputBatchPrefix)))
	if err != nil {
		return err
	}

	k := len(chunkFiles)

	log.Println(chunkFiles)

	batchFiles := make([]*os.File, k)
	for i := 0; i < k; i++ {
		batchFiles[i], err = os.Open(chunkFiles[i])
		if err != nil {
			return err
		}

		defer func(i int) {
			err = batchFiles[i].Close()
			logUnhandledErr(err)
		}(i)
	}

	out, err := os.Create(filepath.Clean(outputFile))
	if err != nil {
		return err
	}
	defer func() {
		err = out.Close()
		logUnhandledErr(err)
	}()

	pq := make(PriorityQueue, 0)
	heap.Init(&pq)

	var item *Item
	for i := 0; i < k; i++ {
		item = &Item{}

		_, err = fmt.Fscanf(batchFiles[i], outputTemplate, &item.value, &item.priority)
		if errors.Is(err, io.EOF) {
			continue
		}
		if err != nil {
			return err
		}

		item.batchIndex = i

		heap.Push(&pq, item)
	}

	for pq.Len() > 0 {
		query := heap.Pop(&pq)

		item = query.(*Item)

		fmt.Fprintf(out, "%s\t%d\n", item.value, item.priority)

		_, err = fmt.Fscanf(batchFiles[item.batchIndex], outputTemplate, &item.value, &item.priority)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return err
			}

			continue
		}

		heap.Push(&pq, item)
	}

	return nil
}
