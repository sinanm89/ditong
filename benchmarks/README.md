# Ditong Benchmark Suite

Benchmark configurations for testing parallel processing performance across different language combinations.

## Quick Start

```bash
# Linux/macOS
./run.sh

# Windows PowerShell
.\run.ps1

# Or directly with Go
go run runner.go
```

## Benchmark Groups

### Group 1: English + Turkish (`en_tr`)
Basic two-language setup testing parallel download and processing.

### Group 2: English + French + German (`en_fr_de`)
Three-language setup for testing moderate parallelism.

### Group 3: English + French + German + Turkish (`en_fr_de_tr`)
Four-language setup for maximum parallel benefit.

## Configurations (5 per group = 15 total)

| Config | Description |
|--------|-------------|
| Sequential | Baseline with no parallelism |
| Parallel Downloads | Parallel language downloads |
| 2 Workers | Parallel with 2 workers |
| 4 Workers | Parallel with 4 workers |
| 8 Workers | Parallel with 8 workers |

## Options

```bash
# Run specific group
go run runner.go --group en_tr

# Multiple iterations for averaging
go run runner.go --iterations 3

# Force re-download (no cache)
go run runner.go --force
```

## Output

Results are saved to `results/benchmark_YYYY-MM-DD_HH-MM-SS.json`:

```json
{
  "timestamp": "2026-02-01T15:30:00Z",
  "iterations": 1,
  "results": [
    {
      "config_id": "en_tr_sequential",
      "group": "en_tr",
      "languages": "en,tr",
      "duration_ms": 5000,
      "throughput": 10000.0,
      "words": 50000,
      "files": 16,
      "parallel": false,
      "workers": 1
    }
  ]
}
```

## Interpreting Results

The summary table shows speedup relative to the sequential baseline:

```
Config                          Duration      Words  Speedup
----------------------------------------------------------------------
[en_tr]
en_tr_sequential                   5000ms     50000        -
en_tr_parallel_4w                  1500ms     50000    3.33x
en_tr_parallel_8w                  1200ms     50000    4.17x
```

A speedup of `3.33x` means the parallel version completed in ~30% of the sequential time.
