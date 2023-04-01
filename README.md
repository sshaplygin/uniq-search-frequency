# uniq-search-frequency

This is the CLI for `task.txt`. The test task is written in the format .txt in Russian.

## Usage

Step 1. Without build binary file: `go run main.go`

or

Step 1. Build app to binary. `go build -o ./bin/cli` \
Step 2. Run cli `./bin/cli`

## Flags

CLI support next flags:

- n - memory limit for first uniques search queries. `default value = -1`
- input - input filepath `default = input.txt`
- output - output filepath `default = output.tsv`
- h - print helps about supported cli flags

Example:

```bash
    cli --n=3 --input=test.txt --output=test1.tsv
```

## Links

Read more:

- External sort [link](https://www.geeksforgeeks.org/external-sorting/)
- Merge sort [link](https://www.geeksforgeeks.org/merge-k-sorted-arrays/)
- Impelemntation extenal sort in Go [link](https://rosettacode.org/wiki/External_sort#Go)
