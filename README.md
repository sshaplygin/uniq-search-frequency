# uniq-search-frequency

CLI implementation of the task described in `task.txt`.

The output is ordered by descending frequency. Queries with the same frequency retain the order of their first appearance in the input.

## Usage

Run without building a binary:

```bash
make run
```

or

Verify that the project builds:

```bash
make build
```

## Flags

CLI support next flags:

- `n` — maximum number of unique queries aggregated in memory. The default `-1` processes the whole input in memory.
- `input` — input file path. Default: `input.txt`.
- `output` — output file path. Default: `output.tsv`.
- `h` — print flag help.

Example:

```bash
make run ARGS="--n=3 --input=test.txt --output=test1.tsv"
```

With a positive `--n`, the utility creates sorted temporary runs, merges and aggregates them with a bounded number of open files, then performs a second external sort by frequency. This lets it process more unique queries than the memory limit without recursive reprocessing of the entire input.

Output is tab-separated. Queries containing tabs or quotes are encoded using CSV-compatible quoting with a tab delimiter, so they remain round-trippable.

The command returns a non-zero exit code on invalid flags or I/O failures. The destination file is written atomically: a failed run leaves an existing output file unchanged.

## Development

The Makefile is the entry point for all development commands. Run the complete validation suite:

```bash
make check
```

Run individual tasks when needed:

```bash
make test
make coverage
make lint
make vet
make staticcheck
make format
make generate
```

## Links

Read more:

- External sort [link](https://www.geeksforgeeks.org/external-sorting/)
- Merge sort [link](https://www.geeksforgeeks.org/merge-k-sorted-arrays/)
- Impelemntation extenal sort in Go [link](https://rosettacode.org/wiki/External_sort#Go)
