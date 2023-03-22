# uniq-search-frequency

This is the CLI for `task.txt`. The test task is written in the format .txt in Russian.

## Usage

Step 1. Without build binary file: `go run main.go`

or

Step 1. Build app to binary. `go build -o ./bin/cli` \
Step 2. Run cli `./bin/cli`

## Flags

CLI support next flags:

- n - `default value = -1`
- input - `default = input.txt`
- output - `default = output.tsv`

Example:

```bash
    cli --n=3 --input=test.txt --output=test1.tsv
```
