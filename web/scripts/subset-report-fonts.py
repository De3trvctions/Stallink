#!/usr/bin/env python3
"""Build the sales-report PDF font subsets under web/public/fonts/report/.

Run this by hand when the character set or the upstream font changes; the web
build never runs it and never touches the network. It downloads pinned files
from the official Google Fonts repository, checks their SHA-256, pins the
variable font at Regular (wght 400), and keeps only:

  * the 8105 characters of 通用规范汉字表 (2013), read from the kTGH field of
    the Unicode Character Database Unihan 17.0.0 (Unicode License V3,
    https://www.unicode.org/license.txt). kTGH is the Unicode Consortium's
    mapping of every character in the 2013 table to its table index.
  * printable ASCII, Latin-1 and Latin Extended-A (Malay and most EU names)
  * general punctuation, CJK punctuation, full-width forms
  * currency symbols, and U+25A1 □ used to replace characters the subset lacks

Requires: Python 3.9+ and fontTools (pip install fonttools).
Usage:    python3 web/scripts/subset-report-fonts.py [--cache DIR]
"""

from __future__ import annotations

import argparse
import hashlib
import io
import json
import re
import sys
import urllib.request
import zipfile
from pathlib import Path

from fontTools import subset
from fontTools.ttLib import TTFont
from fontTools.varLib import instancer

GOOGLE_FONTS = "https://raw.githubusercontent.com/google/fonts"
UNIHAN_URL = "https://www.unicode.org/Public/17.0.0/ucd/Unihan.zip"
UNIHAN_SHA256 = "f7a48b2b545acfaa77b2d607ae28747404ce02baefee16396c5d2d7a8ef34b5e"
UNICODE_LICENSE_URL = "https://www.unicode.org/license.txt"

FONTS = {
    "sans": {
        "family": "Noto Sans SC",
        "url": f"{GOOGLE_FONTS}/a85815a42757630ce188fdad368c2dfc444d4773/ofl/notosanssc/NotoSansSC%5Bwght%5D.ttf",
        "license_url": f"{GOOGLE_FONTS}/a85815a42757630ce188fdad368c2dfc444d4773/ofl/notosanssc/OFL.txt",
        "sha256": "a3041811a78c361b1de50f953c805e0244951c21c5bd412f7232ef0d899af0da",
        "output": "noto-sans-sc-regular.subset.ttf",
    },
    "serif": {
        "family": "Noto Serif SC",
        "url": f"{GOOGLE_FONTS}/8b0a1d0f5983c89bc2b93f1b5fb55f9e252744b5/ofl/notoserifsc/NotoSerifSC%5Bwght%5D.ttf",
        "license_url": f"{GOOGLE_FONTS}/8b0a1d0f5983c89bc2b93f1b5fb55f9e252744b5/ofl/notoserifsc/OFL.txt",
        "sha256": "050080d9255a86808f2945bffac582b31ef32bc36411ce29563b4961670c66f9",
        "output": "noto-serif-sc-regular.subset.ttf",
    },
}

LEVEL_COUNTS = {1: 3500, 2: 3000, 3: 1605}

EXTRA_RANGES = [
    (0x0020, 0x007E),  # ASCII
    (0x00A0, 0x00FF),  # Latin-1 Supplement (× ÷ · ¥ £ etc.)
    (0x0100, 0x017F),  # Latin Extended-A
    (0x2010, 0x2027),  # dashes, quotes, ellipsis
    (0x2030, 0x203A),  # per mille, primes, angle quotes
    (0x20A0, 0x20C0),  # currency symbols
    (0x2116, 0x2116),  # №
    (0x2190, 0x2193),  # arrows
    (0x2212, 0x2212),  # minus sign
    (0x25A1, 0x25A1),  # □ missing-glyph replacement
    (0x3000, 0x303F),  # CJK symbols and punctuation
    (0xFF01, 0xFF5E),  # full-width ASCII forms
    (0xFFE0, 0xFFE6),  # full-width currency signs
]

# Kel accepted the 3.71 MB Noto Serif SC subset on 2026-11-26; this guard only
# catches an accidental jump such as a changed character set or hinting.
MAX_FIRST_CLICK_BYTES = 4_000_000


def fetch(url: str, cache: Path, expected: str | None) -> bytes:
    name = hashlib.sha256(url.encode()).hexdigest()[:16] + "-" + url.rsplit("/", 1)[-1].replace("%5B", "[").replace("%5D", "]")
    path = cache / name
    if not path.exists():
        print(f"download {url}", file=sys.stderr)
        with urllib.request.urlopen(url, timeout=120) as response:  # noqa: S310 - pinned https URL
            path.write_bytes(response.read())
    data = path.read_bytes()
    digest = hashlib.sha256(data).hexdigest()
    if expected and digest != expected:
        raise SystemExit(f"SHA-256 mismatch for {url}: got {digest}, want {expected}")
    print(f"sha256 {digest} {url}", file=sys.stderr)
    return data


def standard_characters(cache: Path) -> list[int]:
    archive = zipfile.ZipFile(io.BytesIO(fetch(UNIHAN_URL, cache, UNIHAN_SHA256)))
    indexes: dict[int, int] = {}
    for line in archive.read("Unihan_OtherMappings.txt").decode("utf-8").splitlines():
        if "\tkTGH\t" not in line:
            continue
        codepoint, _, value = line.split("\t")
        match = re.search(r"2013:(\d+)", value)
        if match:
            indexes[int(codepoint[2:], 16)] = int(match.group(1))
    levels = {1: 0, 2: 0, 3: 0}
    for index in indexes.values():
        levels[1 if index <= 3500 else 2 if index <= 6500 else 3] += 1
    if len(indexes) != 8105 or sorted(indexes.values()) != list(range(1, 8106)) or levels != LEVEL_COUNTS:
        raise SystemExit(f"Unihan kTGH should list 8105 characters (3500/3000/1605), got {len(indexes)} {levels}")
    return sorted(indexes, key=indexes.__getitem__)


def ranges(codepoints: list[int]) -> list[list[int]]:
    out: list[list[int]] = []
    for cp in sorted(set(codepoints)):
        if out and cp == out[-1][1] + 1:
            out[-1][1] = cp
        else:
            out.append([cp, cp])
    return out


def build(kind: str, spec: dict, wanted: list[int], cache: Path, out_dir: Path) -> tuple[set[int], dict]:
    source = cache / f"{kind}-source.ttf"
    source.write_bytes(fetch(spec["url"], cache, spec["sha256"]))
    font = TTFont(source)
    if "fvar" in font:
        font = instancer.instantiateVariableFont(font, {"wght": 400}, updateFontNames=True)
    options = subset.Options()
    options.layout_features = ["kern", "liga", "locl", "ccmp", "mark", "mkmk"]
    options.hinting = False
    options.name_IDs = ["*"]
    options.name_languages = ["*"]
    options.notdef_outline = True
    options.drop_tables += ["DSIG"]
    subsetter = subset.Subsetter(options)
    subsetter.populate(unicodes=wanted)
    subsetter.subset(font)
    target = out_dir / spec["output"]
    font.save(target)
    covered = set(TTFont(target).getBestCmap().keys())
    data = target.read_bytes()
    return covered, {
        "family": spec["family"],
        "file": spec["output"],
        "bytes": len(data),
        "sha256": hashlib.sha256(data).hexdigest(),
        "source": spec["url"],
    }


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--cache", type=Path, default=Path.home() / ".cache" / "stallink-report-fonts")
    parser.add_argument("--out", type=Path, default=Path(__file__).resolve().parents[1] / "public" / "fonts" / "report")
    args = parser.parse_args()
    args.cache.mkdir(parents=True, exist_ok=True)
    args.out.mkdir(parents=True, exist_ok=True)

    wanted = standard_characters(args.cache)
    for start, end in EXTRA_RANGES:
        wanted.extend(range(start, end + 1))

    fonts: dict[str, dict] = {}
    coverage: set[int] | None = None
    for kind, spec in FONTS.items():
        covered, info = build(kind, spec, wanted, args.cache, args.out)
        fonts[kind] = info
        coverage = covered if coverage is None else coverage & covered
        print(f"{info['file']}: {info['bytes']:,} bytes", file=sys.stderr)
    assert coverage is not None
    coverage.discard(0)

    licenses = {kind: fetch(spec["license_url"], args.cache, None).decode("utf-8") for kind, spec in FONTS.items()}
    marker = "SIL OPEN FONT LICENSE Version 1.1"
    bodies = {kind: " ".join(text.split(marker, 1)[1].split()) for kind, text in licenses.items()}
    if bodies["sans"] != bodies["serif"]:
        raise SystemExit("Noto Sans SC and Noto Serif SC OFL bodies differ; ship both files")
    notices = [f"{FONTS[kind]['family']}: {text.splitlines()[0].strip()}" for kind, text in licenses.items()]
    body = licenses["sans"][licenses["sans"].index("This Font Software is licensed"):]
    (args.out / "OFL.txt").write_text("\n".join(notices) + "\n\n" + body, encoding="utf-8")
    (args.out / "UNICODE-LICENSE.txt").write_text(fetch(UNICODE_LICENSE_URL, args.cache, None).decode("utf-8"), encoding="utf-8")

    manifest = {
        "note": "Generated by web/scripts/subset-report-fonts.py; do not edit by hand.",
        "characterTable": {"source": UNIHAN_URL, "sha256": UNIHAN_SHA256, "field": "kTGH", "count": 8105},
        "fonts": fonts,
        "codepointCount": len(coverage),
        "ranges": ranges(list(coverage)),
    }
    (args.out / "coverage.json").write_text(json.dumps(manifest, ensure_ascii=False, separators=(",", ":")) + "\n", encoding="utf-8")

    largest = max(info["bytes"] for info in fonts.values())
    print(f"codepoints covered by both fonts: {len(coverage):,}", file=sys.stderr)
    if largest > MAX_FIRST_CLICK_BYTES:
        raise SystemExit(f"largest subset is {largest:,} bytes, above the {MAX_FIRST_CLICK_BYTES:,}-byte byte guard")


if __name__ == "__main__":
    main()
