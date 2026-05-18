# Search Ordering Follow-Up

## Goal

Improve alpha-beta pruning in the search AI by presenting promising legal candidates earlier without changing legal move generation, search depth, cloning behavior, or the resulting candidate set.

## Decision

Do not call the existing heuristic AI from inside search. Profiling realistic `cmd/gametest -rogue-decks` samples showed that exact per-node heuristic hints are slower than the default generated order, mainly because combat hints invoke the combat solver from within search combat branching.

Search should continue using generated candidate order until a cheaper ordering strategy is added.

## Future Ordering Requirements

- Ordering must not add or remove moves, attacker subsets, or blocker subsets.
- Sorting must be stable so equal-ranked candidates retain generation order.
- Scoring must avoid cloning and must not call another AI strategy.
- Combat ordering should use local board facts only, not the combat solver.
- Any new ordering should be guarded by paired profiles against the no-ordering baseline.

## Profiling Baseline

Use 20-turn rogue-deck samples for comparison because shorter games underrepresent complex board states:

- `./cmd/gametest -mode-a search -mode-b search -rogue-decks -quiet -games 1 -seed 3 -turns 20 -timeout 90s -loop-timing`
- `./cmd/gametest -mode-a search -mode-b search -rogue-decks -quiet -games 1 -seed 4 -turns 20 -timeout 90s -loop-timing`
