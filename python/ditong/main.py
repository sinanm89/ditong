"""ditong CLI - Multi-language lexicon toolkit.

Usage:
    python -m ditong.main --languages en,tr --min-length 5 --max-length 8
    python -m ditong.main --languages en,tr,de --synthesis en_tr_de_standard
"""

import argparse
import sys
from pathlib import Path

from ingest import hunspell
from builder.dictionary import DictionaryBuilder
from builder.synthesis import SynthesisBuilder, SynthesisConfig
from . import config as cfg


def main() -> int:
    """Main entry point."""
    # Load defaults from config.json
    defaults = cfg.load().get("defaults", cfg.FALLBACK_DEFAULTS)
    project_root = Path(__file__).parent.parent.parent

    parser = argparse.ArgumentParser(
        description="ditong - Multi-language lexicon toolkit"
    )
    parser.add_argument(
        "--languages",
        "-l",
        type=str,
        default=defaults.get("languages", "en,tr"),
        help=f"Comma-separated language codes (default: {defaults.get('languages', 'en,tr')})",
    )
    parser.add_argument(
        "--min-length",
        type=int,
        default=defaults.get("min_length", 3),
        help=f"Minimum word character length (default: {defaults.get('min_length', 3)})",
    )
    parser.add_argument(
        "--max-length",
        type=int,
        default=defaults.get("max_length", 5),
        help=f"Maximum word character length (default: {defaults.get('max_length', 5)})",
    )
    parser.add_argument(
        "--output-dir",
        "-o",
        type=Path,
        default=project_root / defaults.get("output_dir", "output/dicts"),
        help="Output directory for dictionaries",
    )
    parser.add_argument(
        "--cache-dir",
        "-c",
        type=Path,
        default=project_root / defaults.get("cache_dir", "sources"),
        help="Cache directory for downloaded sources",
    )
    parser.add_argument(
        "--synthesis",
        "-s",
        type=str,
        help="Name for synthesis dictionary (default: auto-generated)",
    )
    parser.add_argument(
        "--force",
        "-f",
        action="store_true",
        default=defaults.get("force", False),
        help="Force re-download of dictionaries",
    )
    parser.add_argument(
        "--no-split",
        action="store_true",
        help="Don't split synthesis by first letter",
    )

    args = parser.parse_args()

    languages = [l.strip() for l in args.languages.split(",")]

    print("=" * 60)
    print("ditong - Multi-language Lexicon Toolkit")
    print("=" * 60)
    print(f"Languages: {', '.join(languages)}")
    print(f"Length range: {args.min_length}-{args.max_length}")
    print(f"Output: {args.output_dir}")
    print()

    # Initialize builders
    dict_builder = DictionaryBuilder(
        output_dir=args.output_dir,
        min_length=args.min_length,
        max_length=args.max_length,
    )
    synth_builder = SynthesisBuilder(output_dir=args.output_dir)

    # Ingest each language
    print("[1/3] Downloading and ingesting dictionaries...")
    for lang in languages:
        print(f"\n  [{lang}] ", end="")
        try:
            result = hunspell.download_and_ingest(
                language=lang,
                cache_dir=args.cache_dir / lang,
                min_length=args.min_length,
                max_length=args.max_length,
                force=args.force,
            )
            print(f"OK - {result.total_valid:,} words")
            dict_builder.add_words(result)
            synth_builder.add_words(result.words)
        except ValueError as e:
            print(f"SKIP - {e}")
        except Exception as e:
            print(f"ERROR - {e}")
            return 1

    # Build per-language dictionaries
    print("\n[2/3] Building per-language dictionaries...")
    stats = dict_builder.build()
    print(f"  Total words: {stats.total_words:,}")
    print(f"  Files written: {len(stats.files_written)}")

    for lang, count in sorted(stats.by_language.items()):
        print(f"    {lang}: {count:,}")

    # Build synthesis dictionary
    print("\n[3/3] Building synthesis dictionary...")
    synth_name = args.synthesis or "_".join(sorted(languages)) + "_standard"

    config = SynthesisConfig(
        name=synth_name,
        include_languages=set(languages),
        include_categories={"standard"},
        min_length=args.min_length,
        max_length=args.max_length,
        split_by_letter=not args.no_split,
    )

    synth_stats = synth_builder.build(config)
    print(f"  Synthesis name: {synth_name}")
    print(f"  Unique words: {synth_stats.total_words:,}")
    print(f"  Files written: {len(synth_stats.files_written)}")

    print("\n  By length:")
    for length, count in sorted(synth_stats.by_length.items()):
        print(f"    {length}-c: {count:,}")

    print("\n" + "=" * 60)
    print("Done!")
    print("=" * 60)

    return 0


if __name__ == "__main__":
    sys.exit(main())
