#!/usr/bin/env python3
"""Analyze downloaded MTG set JSON files for mechanics and keyword breakdowns."""

import json
import re
import os
import sys

DATA_DIR = os.path.join(os.path.dirname(__file__), '..', 'data')

SETS_ORDER = [
    'LEG', 'DRK', 'FEM',        # Early expansions
    'ICE', 'HML', 'ALL',        # Ice Age block (+Homelands)
    'MIR', 'VIS', 'WTH',        # Mirage block
    'TMP', 'STH', 'EXO',        # Tempest block
    'USG', 'ULG', 'UDS',        # Urza's block
    'MMQ', 'NEM', 'PCY',        # Masques block
    'INV', 'PLS', 'APC',        # Invasion block
    'ODY', 'TOR', 'JUD',        # Odyssey block
    'ONS', 'LGN', 'SCG',        # Onslaught block
]

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

MECHANIC_PATTERNS = [
    ('tokens', r'[Cc]reate.+token'),
    ('sacrifice_cost', r'[Ss]acrifice .+:'),
    ('cantrips', r'[Dd]raw a card'),
    ('regenerate', r'[Rr]egenerate'),
    ('counterspell', r'[Cc]ounter target'),
    ('tutor', r'[Ss]earch your library'),
    ('discard', r'[Dd]iscard'),
    ('graveyard_recursion', r'from your graveyard'),
    ('bounce', r'[Rr]eturn .+ to .+ hand'),
    ('coin_flip', r'[Ff]lip a coin'),
    ('alt_cost', r'rather than pay|without paying'),
    ('X_spell', r'\{X\}'),
    ('enchant_world', r'World Enchantment'),
    ('phasing', r'[Pp]hasing|[Pp]hases? out'),
    ('flanking', r'[Ff]lanking'),
    ('shadow', r'[Ss]hadow'),
    ('buyback', r'[Bb]uyback'),
    ('cycling', r'[Cc]ycling'),
    ('echo', r'[Ee]cho'),
    ('fading', r'[Ff]ading'),
    ('kicker', r'[Kk]icker|[Kk]icked'),
    ('morph', r'[Mm]orph'),
    ('flashback', r'[Ff]lashback'),
    ('threshold', r'[Tt]hreshold'),
    ('madness', r'[Mm]adness'),
    ('storm', r'[Ss]torm'),
    ('amplify', r'[Aa]mplify'),
    ('provoke', r'[Pp]rovoke'),
    ('domain', r'[Dd]omain'),
    ('split_card', None),  # detected via card_faces
    ('protection', r'[Pp]rotection from'),
    ('bands_with', r'[Bb]ands with'),
    ('cumulative_upkeep', r'[Cc]umulative upkeep'),
    ('rampage', r'[Rr]ampage'),
]

def analyze_set(filepath):
    with open(filepath) as f:
        cards = json.load(f)

    counts = {}
    type_counts = {'creature': 0, 'instant': 0, 'sorcery': 0,
                   'enchantment': 0, 'artifact': 0, 'land': 0}
    multicolor = 0
    legendary = 0
    unique_keywords = set()
    unique_subtypes = set()

    for c in cards:
        ot = c.get('oracle_text', '')
        tl = c.get('type_line', '')
        colors = c.get('colors', [])
        keywords = c.get('keywords', [])

        unique_keywords.update(keywords)

        # Count types
        tl_lower = tl.lower()
        for t in type_counts:
            if t in tl_lower:
                type_counts[t] += 1

        # Count multicolor/legendary
        if len(colors) > 1:
            multicolor += 1
        if 'Legendary' in tl:
            legendary += 1

        # Extract creature subtypes
        if 'Creature' in tl and '—' in tl:
            subtypes = tl.split('—')[1].strip().split()
            unique_subtypes.update(subtypes)

        # Count mechanic patterns
        for label, pat in MECHANIC_PATTERNS:
            if label == 'split_card':
                if c.get('card_faces'):
                    counts[label] = counts.get(label, 0) + 1
            elif pat and re.search(pat, ot):
                counts[label] = counts.get(label, 0) + 1

    return {
        'total': len(cards),
        'types': type_counts,
        'multicolor': multicolor,
        'legendary': legendary,
        'keywords': sorted(unique_keywords),
        'subtypes': sorted(unique_subtypes),
        'mechanics': counts,
    }


def main():
    for code in SETS_ORDER:
        filepath = os.path.join(DATA_DIR, f'{code}.json')
        if not os.path.exists(filepath):
            print(f"--- {code} ({SET_NAMES.get(code, '?')}) --- NOT FOUND")
            continue

        info = analyze_set(filepath)
        name = SET_NAMES.get(code, '?')
        print(f"=== {code}: {name} ({info['total']} cards) ===")
        print(f"  Types: {', '.join(f'{k}:{v}' for k,v in info['types'].items() if v)}")
        print(f"  Multicolor: {info['multicolor']}, Legendary: {info['legendary']}")
        print(f"  Keywords: {', '.join(info['keywords'])}")

        # Only show non-zero mechanics
        mechs = {k: v for k, v in info['mechanics'].items() if v}
        if mechs:
            print(f"  Mechanics: {', '.join(f'{k}:{v}' for k,v in sorted(mechs.items()))}")

        # Show notable subtypes (creature types)
        if info['subtypes']:
            print(f"  Creature subtypes: {', '.join(sorted(info['subtypes']))}")
        print()


if __name__ == '__main__':
    main()
