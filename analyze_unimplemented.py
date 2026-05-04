#!/usr/bin/env python3
"""
Compare cards listed in scryfall_cards.json against cards implemented in mage-go.

A card is considered:
  - "implemented" if registered via Register("Name", ...) AND not preceded by a
    "// TODO: implement" comment within the preceding 8 lines.
  - "stub" if registered but the preceding block contains "// TODO: implement".
  - "unregistered" if not registered at all.

"unimplemented" = stub + unregistered.

Usage:
    python3 analyze_unimplemented.py [path-to-scryfall-cards.json[.zst]]
        [--cards-dir cards] [--format text|json]
"""

import argparse
import json
import os
import re
import subprocess
import sys
from pathlib import Path

REGISTER_LITERAL_RE = re.compile(r'Register\("([^"]+)"')
REGISTER_LOOP_RE = re.compile(r'\bRegister\(\s*[A-Za-z_][A-Za-z_0-9]*\s*,')
STRUCT_ENTRY_RE = re.compile(r'^\s*\{"([^"]+)"\s*,')
TODO_RE = re.compile(r'//\s*TODO:\s*implement', re.IGNORECASE)


def load_scryfall(path: Path):
    if path.suffix == ".zst":
        out = subprocess.run(
            ["zstd", "-dc", str(path)], check=True, capture_output=True
        ).stdout
        return json.loads(out)
    return json.loads(path.read_text())


def scan_cards_dir(cards_dir: Path):
    """Return (implemented, stubs) sets of card names.

    Detects two registration patterns:
      1. Literal:  Register("Card Name", func() Card { ... })
      2. Looped:   Register(name, func() Card { ... }) where `name` comes
                   from iterating a []struct{...} literal whose first field
                   is the card name. We pick up entries `{"Card Name", ...}`
                   in any file that contains a non-literal Register call.
    """
    implemented = set()
    stubs = set()
    for go_file in cards_dir.rglob("*.go"):
        if go_file.name.endswith("_test.go"):
            continue
        text = go_file.read_text()
        lines = text.splitlines()

        for i, line in enumerate(lines):
            m = REGISTER_LITERAL_RE.search(line)
            if not m:
                continue
            name = m.group(1)
            # Walk backward through the contiguous comment block immediately
            # preceding this Register (skipping blank lines, stopping at any
            # non-comment line). This avoids bleeding a previous card's TODO
            # into this one.
            j = i - 1
            while j >= 0 and lines[j].strip() == "":
                j -= 1
            comment_block = []
            while j >= 0 and lines[j].lstrip().startswith("//"):
                comment_block.append(lines[j])
                j -= 1
            if any(TODO_RE.search(c) for c in comment_block):
                stubs.add(name)
            else:
                implemented.add(name)

        if REGISTER_LOOP_RE.search(text):
            for line in lines:
                m = STRUCT_ENTRY_RE.match(line)
                if m:
                    implemented.add(m.group(1))

    stubs -= implemented
    return implemented, stubs


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "scryfall_path",
        nargs="?",
        default="../s30/assets/card_info/scryfall_cards.json",
        help="Path to scryfall cards JSON (or .zst). Auto-detects .zst if .json missing.",
    )
    parser.add_argument("--cards-dir", default="cards", help="Path to mage-go cards directory")
    parser.add_argument("--format", choices=["text", "json"], default="text")
    parser.add_argument(
        "--by-set",
        action="store_true",
        help="Break down unimplemented counts by Scryfall SetID",
    )
    args = parser.parse_args()

    scry_path = Path(args.scryfall_path)
    if not scry_path.exists():
        zst = scry_path.with_suffix(scry_path.suffix + ".zst")
        if zst.exists():
            scry_path = zst
        else:
            sys.exit(f"not found: {scry_path} (also tried {zst})")

    cards = load_scryfall(scry_path)
    # Card names may repeat across sets; dedupe by CardName.
    scry_names = {c["CardName"] for c in cards if c.get("CardName")}

    implemented, stubs = scan_cards_dir(Path(args.cards_dir))

    unregistered = sorted(scry_names - implemented - stubs)
    stub_in_scry = sorted(scry_names & stubs)
    impl_in_scry = sorted(scry_names & implemented)

    if args.format == "json":
        payload = {
            "totals": {
                "scryfall_unique_cards": len(scry_names),
                "implemented_in_mage_go_total": len(implemented),
                "stubs_in_mage_go_total": len(stubs),
                "scryfall_implemented": len(impl_in_scry),
                "scryfall_stub": len(stub_in_scry),
                "scryfall_unregistered": len(unregistered),
            },
            "stubs": stub_in_scry,
            "unregistered": unregistered,
        }
        if args.by_set:
            by_set = {}
            unimpl = set(stub_in_scry) | set(unregistered)
            for c in cards:
                if c["CardName"] in unimpl:
                    s = c.get("SetID", "?")
                    by_set.setdefault(s, []).append(c["CardName"])
            payload["by_set"] = {k: sorted(set(v)) for k, v in by_set.items()}
        print(json.dumps(payload, indent=2))
        return

    print(f"Scryfall unique cards:           {len(scry_names)}")
    print(f"  implemented in mage-go:        {len(impl_in_scry)}")
    print(f"  registered but stubbed:        {len(stub_in_scry)}")
    print(f"  not registered at all:         {len(unregistered)}")
    print(f"  unimplemented (stub+missing):  {len(stub_in_scry) + len(unregistered)}")
    print(f"\nmage-go totals: {len(implemented)} implemented + {len(stubs)} stubs")
    print(f"  (cards in mage-go but NOT in scryfall list: "
          f"{len(implemented - scry_names) + len(stubs - scry_names)})")

    if args.by_set:
        unimpl = set(stub_in_scry) | set(unregistered)
        by_set = {}
        for c in cards:
            if c["CardName"] in unimpl:
                s = c.get("SetID", "?")
                by_set.setdefault(s, set()).add(c["CardName"])
        print("\nUnimplemented by set:")
        for s in sorted(by_set, key=lambda k: -len(by_set[k])):
            print(f"  {s}: {len(by_set[s])}")

    print("\n--- Stubs (registered, TODO: implement) ---")
    for n in stub_in_scry:
        print(n)
    print("\n--- Unregistered ---")
    for n in unregistered:
        print(n)


if __name__ == "__main__":
    main()
