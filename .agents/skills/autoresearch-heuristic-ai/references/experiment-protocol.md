# Experiment protocol

The comparison harness currently runs 1,000 games per invocation. Confirm `totalGames` in `pkg/mage/interactive/ai/heuristic/sim_test.go` before relying on this protocol.

## Establish the incumbent

Run focused tests first. Then measure the unchanged incumbent:

```bash
go test ./pkg/mage/interactive/ai/heuristic -run '^TestSimulation_CombatTricksAndTiming$' -count=1 -v
```

Run the control once at the start unless a recent result from the same checkout and machine exists:

```bash
go test ./pkg/mage/interactive/ai/heuristic -run '^TestSimulation_BaselineVsBaseline$' -count=1 -v
```

Investigate a large or repeatable control bias before trusting candidate results.

## Acceptance gate

Use candidate win rate among decided games as the main metric and track draws separately.

- Treat a change of about two percentage points or less in one 1,000-game run as inconclusive.
- Confirm a promising result with at least three candidate runs and comparable incumbent runs from the same environment.
- Pool wins and losses. Exclude draws from decided-game win rate, but compare draw rates.
- For a close result, compute a two-proportion 95% confidence interval for the candidate-minus-incumbent difference. Gather more games or reject the change when the lower bound is at or below zero.
- Reject a change with a regression, illegal action, panic, or substantial unexplained increase in draws or time-limit games.
- Require a deterministic behavior test even when aggregate results are strong.

A change to code shared by the candidate and baseline is not isolated. Add direct tests and a candidate-only comparison path when practical. Otherwise, report weaker confidence and do not describe the result as proven.

After an accepted change, run:

```bash
go test ./pkg/mage/interactive/ai/... ./pkg/mage/interactive/eval/... -short -count=1
```

At the end, run broader tests and lint for the changed packages. Inspect edits made by lint before keeping them.
