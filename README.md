# ditong

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![Python Version](https://img.shields.io/badge/Python-3.11+-3776AB?logo=python&logoColor=white)](https://python.org/)
[![CI](https://github.com/snn/ditong/actions/workflows/ci.yml/badge.svg)](https://github.com/snn/ditong/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/snn/ditong/branch/main/graph/badge.svg)](https://codecov.io/gh/snn/ditong)

A multi-language lexicon toolkit for building cross-language word dictionaries with full metadata tracking.

## Features

- **Multi-language normalization** — Turkish, German, French, Spanish, and more → ASCII
- **Pluggable ingestors** — Hunspell dictionaries, plain text, extensible for Urban Dictionary, Ekşi Sözlük, etc.
- **Per-word metadata** — Track sources, categories, languages, and custom tags
- **Synthesis builder** — Generate filtered cross-language word unions
- **Pluggable IPA transcription** — epitran, espeak-ng, phonemizer, g2p, gruut backends
- **Dual implementation** — Python and Go with identical output formats

## Use Cases

- Generate unique identifier spaces from multiple language dictionaries
- Find phonetically similar words across languages
- Build filtered word lists (exclude profanity, include only standard dictionary words, etc.)
- Create synthesis dictionaries for specific language combinations

## Installation

### Python

```bash
cd python
pip install -e .
```

### Go

```bash
cd go
go build ./cmd
```

## Quick Start

### Python

```python
from ditong.ingest import hunspell
from ditong.builder import DictionaryBuilder, SynthesisBuilder, SynthesisConfig

# Ingest dictionaries
en_result = hunspell.download_and_ingest("en", cache_dir="./sources/en")
tr_result = hunspell.download_and_ingest("tr", cache_dir="./sources/tr")

# Build per-language dictionaries
builder = DictionaryBuilder(output_dir="./dicts")
builder.add_words(en_result)
builder.add_words(tr_result)
stats = builder.build()

# Build synthesis (unique words across both languages)
synth = SynthesisBuilder(output_dir="./dicts")
synth.add_words(en_result.words)
synth.add_words(tr_result.words)

config = SynthesisConfig(
    name="en_tr_standard",
    include_languages={"en", "tr"},
    include_categories={"standard"},
    min_length=5,
    max_length=8,
)
synth.build(config)
```

### Go

```go
package main

import (
    "ditong/internal/ingest"
    "ditong/internal/builder"
)

func main() {
    // Ingest dictionaries
    enConfig := ingest.DefaultConfig("en")
    enResult, _ := ingest.DownloadAndIngest("en", "./sources/en", enConfig, false)

    trConfig := ingest.DefaultConfig("tr")
    trResult, _ := ingest.DownloadAndIngest("tr", "./sources/tr", trConfig, false)

    // Build dictionaries
    dictBuilder := builder.NewDictionaryBuilder("./dicts", 5, 8)
    dictBuilder.AddWords(enResult.Words, "en")
    dictBuilder.AddWords(trResult.Words, "tr")
    dictBuilder.Build()
}
```

### CLI

```bash
# Python
python -m ditong.main --languages en,tr --min-length 5 --max-length 8

# Go
go run ./cmd --languages en,tr --min-length 5 --max-length 8
```

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
      "dict_filepath": "/path/to/en.dic",
      "language": "en",
      "original_form": "care",
      "line_number": 12345,
      "category": "standard"
    },
    {
      "dict_name": "hunspell_tr",
      "dict_filepath": "/path/to/tr.dic",
      "language": "tr",
      "original_form": "çare",
      "line_number": 6789,
      "category": "standard"
    }
  ],
  "categories": ["standard"],
  "languages": ["en", "tr"],
  "tags": [],
  "ipa": null
}
```

### Directory Structure

```
dicts/
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

## Phonetics (Optional)

```python
from ditong.phonetics import transcribe, get_transcriber, list_transcribers

# Available backends: epitran, espeak, phonemizer, g2p_en, gruut
print(list_transcribers())

# Transcribe using default (epitran)
ipa = transcribe("hello", "en")

# Use specific backend
ipa = transcribe("hello", "en", backend="espeak")

# Batch transcribe
from ditong.phonetics import batch_transcribe
results = batch_transcribe(["hello", "world"], "en")
```

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
# Run tests
make test

# Run tests with coverage
make coverage

# Lint
make lint

# Format
make fmt
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the GPL-3.0 License - see the [LICENSE](LICENSE) file for details.

For commercial licensing, contact: sales@rahatol.com
