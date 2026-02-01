"""Dictionary ingestion module.

Provides pluggable ingestors for various dictionary formats:
- Hunspell .dic files
- Urban Dictionary JSON
- Eksi Sozluk exports
- Plain text word lists
- Custom formats

Usage:
    from ditong.ingest import hunspell, urban_dictionary, plain_text

    words = hunspell.ingest("path/to/en_US.dic", language="en")
    words = urban_dictionary.ingest("path/to/urban.json", language="en")
    words = plain_text.ingest("path/to/words.txt", language="en")
"""

from .base import Ingestor, IngestResult
from . import hunspell
from . import plain_text

# Register available ingestors
INGESTORS: dict[str, type[Ingestor]] = {
    "hunspell": hunspell.HunspellIngestor,
    "plain_text": plain_text.PlainTextIngestor,
}


def get_ingestor(name: str) -> type[Ingestor]:
    """Get ingestor class by name."""
    if name not in INGESTORS:
        raise ValueError(f"Unknown ingestor: {name}. Available: {list(INGESTORS.keys())}")
    return INGESTORS[name]


def register_ingestor(name: str, ingestor_cls: type[Ingestor]) -> None:
    """Register a custom ingestor."""
    INGESTORS[name] = ingestor_cls


__all__ = [
    "Ingestor",
    "IngestResult",
    "hunspell",
    "plain_text",
    "get_ingestor",
    "register_ingestor",
    "INGESTORS",
]
