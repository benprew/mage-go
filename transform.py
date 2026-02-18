#!/usr/bin/env python3
"""Transform mage-go codebase to idiomatic Go naming conventions."""

import re
import os

WORKTREE = "/Users/christian.carter/mage-go-cleanup"

# ============================================================
# Field renames for root package files
# ============================================================

FIELD_RENAMES = [
    # BaseCard fields - longest first to avoid partial matches
    ('GraveyardReturnMinCreatures_', 'graveyardReturnMinCreatures'),
    ('IntrinsicDoesNotUntap_', 'intrinsicDoesNotUntap'),
    ('CantBeBlockedByWalls_', 'cantBeBlockedByWalls'),
    ('CanBlockAdditional_', 'canBlockAdditional'),
    ('SacrificeUnlessLand_', 'sacrificeUnlessLand'),
    ('EntersWithXCountersSet', 'entersWithXCountersSet'),
    ('EntersWithXCounters_', 'entersWithXCounters'),
    ('DestroyAtEndOfTurn_', 'destroyAtEndOfTurn'),
    ('EntersTapped_', 'entersTapped'),
    ('Toughness_', 'toughness'),
    ('ManaCost_', 'manaCost'),
    ('Abilities_', 'abilities'),
    ('SubTypes_', 'subTypes'),
    ('Power_', 'power'),
    ('Types_', 'types'),
    ('Owner_', 'owner'),
    ('Name_', 'name'),
    ('ID_', 'id'),
    # BasePlayer fields
    ('ManaPool_', 'manaPool'),
    ('Graveyard_', 'graveyard'),
    ('Library_', 'library'),
    ('Hand_', 'hand'),
    ('Life_', 'life'),
    # BaseAbility fields
    ('Controller_', 'controller'),
    ('Source_', 'source'),
    ('Type_', 'abilityType'),
]

CONTINUOUS_RENAMES = [
    ('sourceID_', 'sourceID'),
]

ABILITY_FIELD_RENAMES = [
    ('Effs', 'effects'),
    ('Csts', 'costs'),
    ('Tgts', 'targets'),
]


def apply_renames(filepath, renames):
    """Apply word-boundary regex renames to a file."""
    with open(filepath, 'r') as f:
        content = f.read()
    for old, new in renames:
        content = re.sub(r'\b' + re.escape(old) + r'\b', new, content)
    with open(filepath, 'w') as f:
        f.write(content)


def split_args(s):
    """Split comma-separated Go arguments respecting string literals."""
    parts = []
    depth = 0
    current = []
    in_str = False

    for ch in s:
        if ch == '"':
            in_str = not in_str
            current.append(ch)
        elif ch == '(' and not in_str:
            depth += 1
            current.append(ch)
        elif ch == ')' and not in_str:
            depth -= 1
            current.append(ch)
        elif ch == ',' and depth == 0 and not in_str:
            parts.append(''.join(current).strip())
            current = []
        else:
            current.append(ch)

    if current:
        parts.append(''.join(current).strip())

    return parts


def transform_creature_definitions(filepath):
    """In cards/ files: update NewCreature calls, replace field assignments with setters."""
    with open(filepath, 'r') as f:
        lines = f.readlines()

    result = []
    i = 0
    while i < len(lines):
        line = lines[i]

        # Check for NewCreature call
        creature_match = re.search(r'(mage\.NewCreature\()(.+?)(\))', line)
        if creature_match:
            # Look ahead for Power_ and Toughness_ assignments
            power_val = None
            toughness_val = None
            power_line = None
            toughness_line = None

            for j in range(i + 1, min(i + 10, len(lines))):
                pm = re.match(r'\s*c\.Power_\s*=\s*(\d+)\s*$', lines[j])
                if pm:
                    power_val = pm.group(1)
                    power_line = j
                tm = re.match(r'\s*c\.Toughness_\s*=\s*(\d+)\s*$', lines[j])
                if tm:
                    toughness_val = tm.group(1)
                    toughness_line = j
                # Stop looking after non-assignment lines
                if (lines[j].strip()
                    and not re.match(r'\s*c\.\w+', lines[j])
                    and not re.match(r'\s*$', lines[j])
                    and not re.match(r'\s*//', lines[j])):
                    break

            if power_val is not None and toughness_val is not None:
                # Rewrite NewCreature call with power/toughness inserted
                args_str = creature_match.group(2)
                args = split_args(args_str)
                if len(args) >= 2:
                    new_args = args[:2] + [power_val, toughness_val] + args[2:]
                    new_args_str = ', '.join(new_args)
                    prefix = line[:creature_match.start(1)]
                    suffix = line[creature_match.end(3):]
                    result.append(f'{prefix}mage.NewCreature({new_args_str}){suffix}')
                else:
                    result.append(line)

                # Skip the Power_/Toughness_ lines
                skip = {power_line, toughness_line}
                i += 1
                while i < len(lines):
                    if i in skip:
                        i += 1
                        continue
                    break
                continue
            else:
                result.append(line)
                i += 1
                continue

        # Transform special field assignments to setter calls
        transformed = False

        # c.EntersWithXCountersSet = true → skip (handled by SetEntersWithXCounters)
        if re.match(r'\s*c\.EntersWithXCountersSet\s*=\s*true\s*$', line):
            i += 1
            continue

        # c.EntersWithXCounters_ = mage.P1P1 → c.SetEntersWithXCounters(mage.P1P1)
        m = re.match(r'(\s*)c\.EntersWithXCounters_\s*=\s*(.+?)\s*$', line)
        if m:
            result.append(f'{m.group(1)}c.SetEntersWithXCounters({m.group(2)})\n')
            transformed = True

        if not transformed:
            m = re.match(r'(\s*)c\.CantBeBlockedByWalls_\s*=\s*(.+?)\s*$', line)
            if m:
                result.append(f'{m.group(1)}c.SetCantBeBlockedByWalls({m.group(2)})\n')
                transformed = True

        if not transformed:
            m = re.match(r'(\s*)c\.EntersTapped_\s*=\s*(.+?)\s*$', line)
            if m:
                result.append(f'{m.group(1)}c.SetEntersTapped({m.group(2)})\n')
                transformed = True

        if not transformed:
            m = re.match(r'(\s*)c\.IntrinsicDoesNotUntap_\s*=\s*(.+?)\s*$', line)
            if m:
                result.append(f'{m.group(1)}c.SetIntrinsicDoesNotUntap({m.group(2)})\n')
                transformed = True

        if not transformed:
            m = re.match(r'(\s*)c\.SacrificeUnlessLand_\s*=\s*(.+?)\s*$', line)
            if m:
                result.append(f'{m.group(1)}c.SetSacrificeUnlessLand({m.group(2)})\n')
                transformed = True

        if not transformed:
            m = re.match(r'(\s*)c\.GraveyardReturnMinCreatures_\s*=\s*(.+?)\s*$', line)
            if m:
                result.append(f'{m.group(1)}c.SetGraveyardReturnMinCreatures({m.group(2)})\n')
                transformed = True

        if not transformed:
            result.append(line)

        i += 1

    with open(filepath, 'w') as f:
        f.writelines(result)


def main():
    # Phase 1: Transform cards/ directory FIRST (before making fields unexported)
    cards_dir = os.path.join(WORKTREE, 'cards')
    for f in sorted(os.listdir(cards_dir)):
        if f.endswith('.go') and not f.endswith('_test.go'):
            path = os.path.join(cards_dir, f)
            print(f"  cards/{f}: transforming creature definitions...")
            transform_creature_definitions(path)

    # Phase 2: Root package field renames
    for f in sorted(os.listdir(WORKTREE)):
        if f.endswith('.go'):
            path = os.path.join(WORKTREE, f)
            print(f"  {f}: renaming fields...")
            apply_renames(path, FIELD_RENAMES)

    # Phase 3: continuous.go sourceID_ renames
    continuous_path = os.path.join(WORKTREE, 'continuous.go')
    print(f"  continuous.go: renaming sourceID_ fields...")
    apply_renames(continuous_path, CONTINUOUS_RENAMES)

    # Phase 4: Ability field renames (Effs→effects, Csts→costs, Tgts→targets)
    ability_files = [
        'activated.go', 'triggered.go', 'spell.go',
        'game.go', 'interactive.go', 'aiplayer.go',
        'effect.go', 'continuous.go',
    ]
    for f in ability_files:
        path = os.path.join(WORKTREE, f)
        if os.path.exists(path):
            print(f"  {f}: renaming Effs/Csts/Tgts...")
            apply_renames(path, ABILITY_FIELD_RENAMES)

    print("\nTransformation complete!")


if __name__ == '__main__':
    main()
