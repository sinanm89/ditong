"""Plain text word list ingestor.

Simple format: one word per line.
Supports comments with # and empty lines.

Use for:
- Custom word lists
- Curse word lists
- Banned word lists
- Any simple word-per-line format
"""

from pathlib import Path
from typing import Iterator, Optional

from .base import Ingestor


class PlainTextIngestor(Ingestor):
    """Ingestor for plain text word lists."""

    file_extensions = [".txt", ".list", ".words"]

    def __init__(
        self,
        language: str,
        category: str = "standard",
        min_length: int = 3,
        max_length: int = 10,
        comment_char: str = "#",
    ):
        super().__init__(language, category, min_length, max_length)
        self.comment_char = comment_char

    def parse(self, filepath: Path) -> Iterator[tuple[str, Optional[int]]]:
        """Parse plain text word list.

        Args:
            filepath: Path to text file.

        Yields:
            Tuples of (word, line_number).
        """
        with open(filepath, "r", encoding="utf-8", errors="ignore") as f:
            for line_num, line in enumerate(f, start=1):
                line = line.strip()

                # Skip empty lines and comments
                if not line or line.startswith(self.comment_char):
                    continue

                # Handle inline comments: "word # comment"
                if self.comment_char in line:
                    line = line.split(self.comment_char)[0].strip()

                if line:
                    yield line, line_num


def ingest(
    filepath: Path | str,
    language: str,
    category: str = "standard",
    min_length: int = 3,
    max_length: int = 10,
    comment_char: str = "#",
):
    """Convenience function to ingest a plain text word list.

    Args:
        filepath: Path to text file.
        language: Language code.
        category: Category tag.
        min_length: Minimum word length.
        max_length: Maximum word length.
        comment_char: Character that starts a comment.

    Returns:
        IngestResult with words.
    """
    ingestor = PlainTextIngestor(
        language=language,
        category=category,
        min_length=min_length,
        max_length=max_length,
        comment_char=comment_char,
    )
    return ingestor.ingest(filepath)
