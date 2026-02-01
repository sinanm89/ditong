"""ditong - Multi-language lexicon toolkit.

A toolkit for building and analyzing cross-language word dictionaries.
Supports normalization, metadata tracking, and phonetic similarity.

Core concepts:
    - Words from multiple sources normalize to the same form
    - Each word tracks its origin sources and metadata
    - Synthesis configs filter words by source tags

Example:
    "care" (EN) + "çare" (TR) → normalized "care"
    Tagged with sources: ["hunspell_en_us", "hunspell_tr"]

Usage:
    from ditong.ingest import hunspell
    from ditong.builder import DictionaryBuilder, SynthesisBuilder, SynthesisConfig

    # Ingest dictionaries
    en_result = hunspell.download_and_ingest("en", cache_dir="./cache")
    tr_result = hunspell.download_and_ingest("tr", cache_dir="./cache")

    # Build per-language dictionaries
    builder = DictionaryBuilder(output_dir="./dicts")
    builder.add_words(en_result)
    builder.add_words(tr_result)
    stats = builder.build()

    # Build synthesis (merged unique words)
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
"""

__version__ = "0.1.0"
