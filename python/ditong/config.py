"""Configuration loader for ditong.

Loads defaults from config.json at project root, with hardcoded fallbacks.
"""

import json
from pathlib import Path
from typing import Any

# Hardcoded fallback defaults
FALLBACK_DEFAULTS = {
    "languages": "en,tr",
    "min_length": 3,
    "max_length": 5,
    "output_dir": "output/dicts",
    "cache_dir": "sources",
    "parallel": True,
    "workers": 0,
    "ipa": False,
    "cursewords": False,
    "consolidate": False,
    "force": False,
    "quiet": False,
    "verbose": False,
    "metrics": True,
}

FALLBACK_LANGUAGES = ["en", "tr", "de", "fr", "es", "it", "pt", "nl", "pl", "ru"]

_config: dict[str, Any] | None = None


def _find_config() -> Path | None:
    """Find config.json by walking up from current file."""
    paths = [
        Path(__file__).parent.parent.parent / "config.json",  # python/ditong -> root
        Path(__file__).parent.parent.parent.parent / "config.json",  # extra level
        Path.cwd() / "config.json",
        Path.cwd().parent / "config.json",
    ]
    for path in paths:
        if path.exists():
            return path
    return None


def load() -> dict[str, Any]:
    """Load configuration from config.json or use fallbacks."""
    global _config
    if _config is not None:
        return _config

    config_path = _find_config()
    if config_path:
        try:
            with open(config_path) as f:
                _config = json.load(f)
                return _config
        except (json.JSONDecodeError, OSError):
            pass

    # Fallback
    _config = {
        "defaults": FALLBACK_DEFAULTS,
        "available_languages": FALLBACK_LANGUAGES,
    }
    return _config


def get_default(key: str, fallback: Any = None) -> Any:
    """Get a default value from config."""
    cfg = load()
    return cfg.get("defaults", {}).get(key, fallback)


def get_available_languages() -> list[str]:
    """Get list of available languages."""
    cfg = load()
    return cfg.get("available_languages", FALLBACK_LANGUAGES)


# Convenience accessors
def default_languages() -> str:
    return get_default("languages", FALLBACK_DEFAULTS["languages"])


def default_min_length() -> int:
    return get_default("min_length", FALLBACK_DEFAULTS["min_length"])


def default_max_length() -> int:
    return get_default("max_length", FALLBACK_DEFAULTS["max_length"])


def default_output_dir() -> str:
    return get_default("output_dir", FALLBACK_DEFAULTS["output_dir"])


def default_cache_dir() -> str:
    return get_default("cache_dir", FALLBACK_DEFAULTS["cache_dir"])
