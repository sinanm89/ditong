# ditong

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)

A multi-language lexicon toolkit for building cross-language word dictionaries with full metadata tracking.

## Quick Start

### Install

```bash
# From source
cd go && go build -o ditong ./cmd

# Or download binary from releases
```

### Run (Interactive)

```bash
./ditong
```

The interactive CLI guides you through language selection, word length filters, and output options.

### Run (CLI flags)

```bash
# Basic: English + Turkish, 5-8 character words
./ditong --languages en,tr --min-length 5 --max-length 8

# With IPA transcription and cursewords
./ditong --languages en,tr,de --ipa --cursewords

# Parallel processing (faster)
./ditong --languages en,tr --parallel --workers 8

# Quiet mode for scripts
./ditong --languages en --quiet --output-dir ./my-dicts
```

### As a Library

```go
package main

import (
    "ditong/internal/ingest"
    "ditong/internal/builder"
)

func main() {
    // Ingest dictionaries
    enResult, _ := ingest.DownloadAndIngest("en", "./sources/en", ingest.DefaultConfig("en"), false)
    trResult, _ := ingest.DownloadAndIngest("tr", "./sources/tr", ingest.DefaultConfig("tr"), false)

    // Build dictionaries
    dictBuilder := builder.NewDictionaryBuilder("./dicts", 5, 8)
    dictBuilder.AddWords(enResult.Words, "en")
    dictBuilder.AddWords(trResult.Words, "tr")
    dictBuilder.Build()
}
```

## Features

- **Multi-language normalization** — Turkish, German, French, Spanish, and more → ASCII
- **Pluggable ingestors** — Hunspell dictionaries, plain text, extensible
- **Per-word metadata** — Track sources, categories, languages, and custom tags
- **Synthesis builder** — Generate filtered cross-language word unions
- **IPA transcription** — Optional phonetic transcription support
- **Parallel processing** — Fast ingestion and build with configurable workers
- **Curseword dictionaries** — Optional inclusion of profanity dictionaries
- **Similarity search** — BK-tree based fuzzy matching
- **Benchmarking** — Built-in benchmark suite for performance testing

## CLI Options

| Flag | Default | Description |
|------|---------|-------------|
| `--languages` | `en` | Comma-separated language codes |
| `--min-length` | `5` | Minimum word length |
| `--max-length` | `8` | Maximum word length |
| `--output-dir` | `./output` | Output directory |
| `--ipa` | `false` | Generate IPA transcriptions |
| `--cursewords` | `false` | Include curseword dictionaries |
| `--parallel` | `true` | Enable parallel processing |
| `--workers` | `0` (auto) | Number of parallel workers |
| `--consolidate` | `false` | Generate consolidated output files |
| `--quiet` | `false` | Suppress progress output |
| `--force` | `false` | Re-download dictionaries |

## Rationale

- **Linguistic Pluralism** — Why limit your vernacular to a single language when multiple languages offer broader expression?
- **Cursewords** — Comprehensive profanity dictionaries that are hard to find elsewhere
- **Portability** — Build your dictionary once, use it anywhere
- **Efficient Sharing** — Generate unique identifier spaces from multiple language dictionaries

## Output Format

### Word Schema

```json
{
  "normalized": "care",
  "length": 4,
  "type": "4-c",
  "sources": [
    {
      "dict_name": "hunspell_en",
      "language": "en",
      "original_form": "care",
      "category": "standard"
    },
    {
      "dict_name": "hunspell_tr",
      "language": "tr",
      "original_form": "çare",
      "category": "standard"
    }
  ],
  "categories": ["standard"],
  "languages": ["en", "tr"],
  "ipa": "/kɛər/"
}
```

### Directory Structure

```
output/
├── en/
│   ├── 3-c.json
│   ├── 4-c.json
│   └── ...
├── tr/
│   └── ...
└── synthesis/
    └── en_tr_standard/
        ├── _config.json
        ├── 5-c/
        │   ├── a.json
        │   ├── b.json
        │   └── ...
        └── ...
```

## Normalization

Characters are normalized to ASCII equivalents:

| Language | Original | Normalized |
|----------|----------|------------|
| Turkish | ç, ş, ğ, ı, ö, ü | c, s, g, i, o, u |
| German | ä, ö, ü, ß | a, o, u, ss |
| French | é, è, ê, à, ç | e, e, e, a, c |
| Spanish | ñ, á, é, í, ó, ú | n, a, e, i, o, u |

This means `care` (EN) and `çare` (TR) normalize to the same identifier, tracked with both sources.

## Supported Languages

| Code | Language | Hunspell | IPA |
|------|----------|----------|-----|
| en | English | ✓ | ✓ |
| tr | Turkish | ✓ | ✓ |
| de | German | ✓ | ✓ |
| fr | French | ✓ | ✓ |
| es | Spanish | ✓ | ✓ |
| it | Italian | ✓ | ✓ |
| pt | Portuguese | ✓ | ✓ |
| nl | Dutch | ✓ | ✓ |
| pl | Polish | ✓ | ✓ |
| ru | Russian | ✓ | ✓ |

## Development

```bash
cd go

# Run tests
go test -v ./...

# Build
go build ./cmd

# Build fuzzy search tool
go build ./cmd/fuzzy

# Run benchmarks
cd ../benchmarks && go run runner.go
```

## License

GPL-3.0 License - see [LICENSE](LICENSE) for details.

Commercial licensing: sales@rahatol.com
