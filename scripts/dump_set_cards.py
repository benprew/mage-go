#!/usr/bin/env python3
"""Dump cards from a set JSON file in a format suitable for analysis.
Usage: python3 scripts/dump_set_cards.py <set_code> [set_code2 ...]

Outputs a condensed summary of each card: name, type, cost, keywords, oracle text.
"""

import json
import sys
import os

DATA_DIR = os.path.join(os.path.dirname(__file__), '..', 'data')

SET_NAMES = {
    'LEG': 'Legends', 'DRK': 'The Dark', 'FEM': 'Fallen Empires',
    'ICE': 'Ice Age', 'HML': 'Homelands', 'ALL': 'Alliances',
    'MIR': 'Mirage', 'VIS': 'Visions', 'WTH': 'Weatherlight',
    'TMP': 'Tempest', 'STH': 'Stronghold', 'EXO': 'Exodus',
    'USG': "Urza's Saga", 'ULG': "Urza's Legacy", 'UDS': "Urza's Destiny",
    'MMQ': 'Mercadian Masques', 'NEM': 'Nemesis', 'PCY': 'Prophecy',
    'INV': 'Invasion', 'PLS': 'Planeshift', 'APC': 'Apocalypse',
    'ODY': 'Odyssey', 'TOR': 'Torment', 'JUD': 'Judgment',
    'ONS': 'Onslaught', 'LGN': 'Legions', 'SCG': 'Scourge',
}


def dump_set(code):
    filepath = os.path.join(DATA_DIR, f'{code}.json')
    if not os.path.exists(filepath):
        print(f"File not found: {filepath}", file=sys.stderr)
        return

    with open(filepath) as f:
        cards = json.load(f)

    name = SET_NAMES.get(code, code)
    print(f"=== {code}: {name} ({len(cards)} cards) ===\n")

    # Deduplicate by name (multiple printings of same card)
    seen = set()
    unique_cards = []
    for c in cards:
        n = c.get('name', '')
        if n not in seen:
            seen.add(n)
            unique_cards.append(c)

    # Group by primary type
    groups = {}
    for c in unique_cards:
        tl = c.get('type_line', '')
        if 'Creature' in tl:
            g = 'Creatures'
        elif 'Instant' in tl:
            g = 'Instants'
        elif 'Sorcery' in tl:
            g = 'Sorceries'
        elif 'Enchantment' in tl:
            g = 'Enchantments'
        elif 'Artifact' in tl:
            g = 'Artifacts'
        elif 'Land' in tl:
            g = 'Lands'
        else:
            g = 'Other'
        groups.setdefault(g, []).append(c)

    for gname in ['Creatures', 'Instants', 'Sorceries', 'Enchantments', 'Artifacts', 'Lands', 'Other']:
        if gname not in groups:
            continue
        cards_in_group = groups[gname]
        print(f"--- {gname} ({len(cards_in_group)}) ---")
        for c in sorted(cards_in_group, key=lambda x: x.get('name', '')):
            name = c.get('name', '?')
            cost = c.get('mana_cost', '')
            tl = c.get('type_line', '')
            ot = c.get('oracle_text', '').replace('\n', ' // ')
            kws = c.get('keywords', [])
            p = c.get('power', '')
            t = c.get('toughness', '')

            parts = [f"[{name}]"]
            if cost:
                parts.append(cost)
            parts.append(tl)
            if p and t:
                parts.append(f"{p}/{t}")
            if kws:
                parts.append(f"KW:{','.join(kws)}")
            if ot:
                parts.append(f"| {ot}")

            print('  '.join(parts))
        print()


if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Usage: python3 scripts/dump_set_cards.py <SET_CODE> [SET_CODE2 ...]")
        sys.exit(1)

    for code in sys.argv[1:]:
        dump_set(code.upper())
