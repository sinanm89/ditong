"""Dictionary builder module.

Builds organized dictionary outputs:
- Per-language, per-length JSON files
- Synthesis dictionaries (multi-language unions)
- First-letter split files for large dictionaries
"""

from .dictionary import DictionaryBuilder
from .synthesis import SynthesisBuilder, SynthesisConfig

__all__ = [
    "DictionaryBuilder",
    "SynthesisBuilder",
    "SynthesisConfig",
]
