# ditong

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)

A multi-language lexicon toolkit for building cross-language word dictionaries with full metadata tracking.

## Quick Start

```bash
# Install
cd go && go build -o ditong ./cmd

# Run interactive mode
./ditong

# Or with flags
./ditong --languages en,tr --min-length 5 --max-length 8 --parallel
```

## Features

- **Multi-language normalization** — Turkish, German, French, Spanish → ASCII
- **Parallel processing** — Bounded worker pools with channel-based job distribution
- **Similarity search** — BK-tree for fuzzy matching (~3.7µs per query)
- **IPA transcription** — Rule-based phonetic transcription
- **Synthesis builder** — Cross-language word unions with filtering

## Performance

```
BenchmarkLevenshteinDistance    11M ops    107 ns/op    128 B/op
BenchmarkBKTreeSearch           350K ops   3.7 µs/op    4.1 KB/op
```

The Levenshtein implementation uses a two-row matrix (O(min(n,m)) space) rather than the full matrix. BK-tree queries exploit the triangle inequality for pruning, typically searching <10% of nodes.

## CLI Options

| Flag | Default | Description |
|------|---------|-------------|
| `--languages` | `en` | Comma-separated language codes |
| `--min-length` | `5` | Minimum word length |
| `--max-length` | `8` | Maximum word length |
| `--parallel` | `true` | Enable parallel processing |
| `--workers` | `0` (auto) | Number of parallel workers |
| `--ipa` | `false` | Generate IPA transcriptions |
| `--cursewords` | `false` | Include profanity dictionaries |
| `--quiet` | `false` | Suppress progress output |

## How It Works

**Normalization**: Characters like `ç`, `ş`, `ğ` (Turkish) or `ä`, `ö`, `ü` (German) map to ASCII equivalents. This means `care` (EN) and `çare` (TR) become the same identifier, with both sources tracked.

**Parallel Build**: Uses a bounded worker pool pattern—goroutines pull from a buffered channel rather than spawning unbounded. This keeps memory predictable under load.

**Similarity Search**: BK-trees partition words by edit distance. For a query, only branches where `|node_distance - query_distance| ≤ max_distance` need searching. This gives sublinear lookup for fuzzy matching.

## Output

```json
{
  "normalized": "care",
  "length": 4,
  "sources": [
    {"language": "en", "original_form": "care"},
    {"language": "tr", "original_form": "çare"}
  ],
  "languages": ["en", "tr"],
  "ipa": "/kɛər/"
}
```

## Supported Languages

en, tr, de, fr, es, it, pt, nl, pl, ru

## Development

```bash
cd go
go test -v ./...
go test -bench=. ./internal/similarity/
```

## License

GPL-3.0 — see [LICENSE](LICENSE). Commercial licensing: sales@rahatol.com
