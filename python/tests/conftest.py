"""Pytest configuration and fixtures."""

import pytest


@pytest.fixture
def sample_hunspell_content():
    """Sample Hunspell dictionary content."""
    return """100
hello
world
testing/ABC
sample/XYZ
python
"""


@pytest.fixture
def sample_turkish_content():
    """Sample Turkish dictionary content."""
    return """50
merhaba
çare
şeker
görmek
ışık
"""


@pytest.fixture
def sample_wordlist_content():
    """Sample plain text word list."""
    return """# Word list
hello
world
test
sample
"""
