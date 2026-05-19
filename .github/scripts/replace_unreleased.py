#!/usr/bin/env python3
"""Replace the `## [Unreleased]` block in a Keep-a-Changelog file."""

from __future__ import annotations

import re
import sys
from pathlib import Path

UNRELEASED_HEADER = "## [Unreleased]"
VERSION_HEADER = re.compile(r"^## \[(?:\d|Unreleased)")


def split_sections(text: str) -> tuple[str, str, str]:
    lines = text.splitlines(keepends=True)
    start = next(
        (i for i, line in enumerate(lines) if line.startswith(UNRELEASED_HEADER)),
        None,
    )
    if start is None:
        body_index = next(
            (i for i, line in enumerate(lines) if VERSION_HEADER.match(line)),
            len(lines),
        )
        header = "".join(lines[:body_index])
        tail = "".join(lines[body_index:])
        return header, "", tail

    header = "".join(lines[:start])
    end = next(
        (
            i
            for i in range(start + 1, len(lines))
            if VERSION_HEADER.match(lines[i]) and not lines[i].startswith(UNRELEASED_HEADER)
        ),
        len(lines),
    )
    unreleased = "".join(lines[start:end])
    tail = "".join(lines[end:])
    return header, unreleased, tail


def main() -> int:
    if len(sys.argv) != 3:
        print("usage: replace_unreleased.py CHANGELOG.md NEW_UNRELEASED.md", file=sys.stderr)
        return 2

    changelog = Path(sys.argv[1])
    new_unreleased = Path(sys.argv[2]).read_text(encoding="utf-8").strip()
    if not new_unreleased:
        return 0

    if not new_unreleased.endswith("\n"):
        new_unreleased += "\n"
    new_unreleased += "\n"

    original = changelog.read_text(encoding="utf-8")
    header, _, tail = split_sections(original)
    changelog.write_text(header + new_unreleased + tail, encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
