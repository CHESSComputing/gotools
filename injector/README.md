# FOXDEN Injector

**FOXDEN Injector** is a command-line tool to inject a series of JSON metadata files into the **FOXDEN meta-data service**.

> **Note:** To perform FOXDEN injection, you must first acquire a **FOXDEN token** with `write` scopes.

## Features

* Crawl directories and find JSON files by pattern (e.g., `meta*.json`)
* By default it runs in **dry-run** mode to preview files without injecting
  - if `-inject` flag is provided it will inject records to FOXDEN
* Concurrent injection using multiple workers
* Configurable HTTP timeouts
* Logs injection results to file or stdout
* Supports reading token from flag, environment variable, or token file

## Installation

Build the executable using:

```bash
make
```

This produces the `injector` binary.

## Usage

### Dry-run

By default, if the `-inject` flag is not specified, the tool runs in **dry-run mode**:

```bash
./injector -path /nfs -file 'meta*.json'
```

This will list all JSON files matching the pattern without sending them to the server.

### Inject with concurrency

```bash
./injector \
  -path /nfs \
  -file 'meta*.json' \
  -inject \
  -url http://localhost:8080/inject \
  -workers 8 \
  -timeout 5s \
  -output inject.log \
  -token /path/to/foxden.write.token
```

* `-path` : Root directory to search
* `-file` : JSON file pattern
* `-inject` : Enable injection (required for actual HTTP POST)
* `-url` : FOXDEN meta-data service endpoint
* `-workers` : Number of concurrent workers
* `-timeout` : HTTP request timeout
* `-output` : Optional log file for status
* `-token` : FOXDEN token or path to token file

## Example Output

After injection, the output may look like:

```
500     /Users/vk/Work/CHESS/FOXDEN/gotools/foxden/test/data/4b-meta.json       Record with did=/beamline=4b/btr=clancy-4592-a/cycle=2025-3/sample_name=tb2ti2o7_stuffed found in MetaData database [...]
200     /Users/vk/Work/CHESS/FOXDEN/gotools/foxden/test/data/4b-meta3.json
200     /Users/vk/Work/CHESS/FOXDEN/gotools/foxden/test/data/4b-meta2.json
200     /Users/vk/Work/CHESS/FOXDEN/gotools/foxden/test/data/4b-meta1.json
```

* First column: HTTP status code from the server
* Second column: JSON file path
* The rest is server error

## Token Handling

FOXDEN Injector resolves tokens as follows:

1. Use the `-token` flag if provided.
2. Fallback to environment variable `FOXDEN_WRITE_TOKEN`.
3. If the token points to a file, the token is read from the file.
4. Otherwise, the string is used as-is.

> Tokens must include the `write` scope to allow injection.
