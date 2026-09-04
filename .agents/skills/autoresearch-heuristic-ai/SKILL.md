---
name: autoresearch-heuristic-ai
description: Iteratively improve the mage-go heuristic AI with focused tests and head-to-head simulation against the comparison baseline. Use for autonomous, hypothesis-driven heuristic AI tuning where only measured improvements should remain.
---

# Autoresearch the heuristic AI

Improve the heuristic AI through repeated, measured experiments. Keep a change only when it is correct and the comparison harness gives credible evidence that it improves play.

## Scope

Work primarily in:

- `pkg/mage/interactive/ai/heuristic/`
- `pkg/mage/interactive/ai/combatsolver/`
- `pkg/mage/interactive/eval/`
- shared AI code in `pkg/mage/interactive/ai/` when the behavior belongs there

Do not change cards or game rules to make the AI score better. Do not weaken tests, reduce the game set, lower the turn limit, or change the baseline during an experiment. Preserve unrelated work. Do not commit unless the user asks.

Commit `27f0af2` introduced the comparison harness. Read these files before the first experiment:

- `pkg/mage/interactive/ai/heuristic/sim_test.go`
- `pkg/mage/interactive/ai/heuristic/baseline_test.go`
- the candidate code that the proposed change will affect
- the closest focused tests

The baseline is not fully isolated. `baselineStrategy` calls some current heuristic helpers, the combat solver, and the evaluation package. Before an experiment, check whether the proposed code path is shared by the baseline. A shared change can affect both players and can hide or distort its measured effect. State this limitation in the experiment record. Do not edit the baseline to favor the candidate.

## Establish the incumbent

Inspect `git status --short` and identify pre-existing changes before editing. Never remove them when an experiment fails.

Run the fast relevant tests first. Then measure the unchanged incumbent against the baseline:

```bash
go test ./pkg/mage/interactive/ai/heuristic -run '^TestSimulation_CombatTricksAndTiming$' -count=1 -v
```

At the start of a research run, also run the control once unless a recent result from the same checkout and machine is available:

```bash
go test ./pkg/mage/interactive/ai/heuristic -run '^TestSimulation_BaselineVsBaseline$' -count=1 -v
```

The control should be close to 50% of decided games. Investigate a large or repeatable bias before trusting candidate results.

Record for every run:

- hypothesis and files changed;
- candidate wins, baseline wins, and draws;
- candidate win rate among decided games;
- test failures, panics, and unusual runtime;
- whether the experiment affects code shared with the baseline.

Keep this ledger in task notes or an untracked temporary file. Do not add benchmark logs to the repository unless the user asks.

## Experiment loop

Run one coherent experiment at a time.

1. Find a concrete weakness from code inspection, a focused test, or observed game behavior. Prefer a general game-state rule over card-name logic.
2. State why the proposed decision should improve match outcomes and which decks or board states it can affect.
3. Add a small deterministic test that expresses the desired decision. Run it before implementation and confirm that it fails for the expected reason.
4. Make the smallest complete AI change that satisfies the test. Keep Magic rules and legal-action checks exact.
5. Run the focused test and the relevant package tests with `-count=1`. Do not benchmark a candidate with failing tests.
6. Run one 5,000-game candidate-versus-baseline simulation as a screen.
7. If the screen is clearly worse, revert only this experiment and record the rejection. Use a patch or other targeted edit. Do not use a destructive Git command.
8. If the screen is promising or close enough to be noise, run more independent 5,000-game simulations. Compare their pooled result with repeated incumbent results from the same environment.
9. Keep the experiment only if the correctness tests pass and the evidence clears the acceptance gate. Otherwise, revert only the experiment.
10. After an accepted experiment, treat it as the new incumbent. Measure new experiments against its recorded results and continue with a different hypothesis.

Do not combine several unmeasured ideas in one patch. Do not tune repeatedly to one random run. A failed experiment is useful evidence; change the hypothesis instead of making arbitrary constants drift.

## Acceptance gate

The harness fixes deck matchup selection, alternates play order, and runs 5,000 games. Deck shuffles still use a concurrent global random source, so exact results vary between runs.

Use candidate win rate among decided games as the primary printed metric. Track draw rate separately. Do not accept a change that appears better only because it causes materially more draws or time-limit games.

Use these rules:

- Treat a change of about two percentage points or less in one 1,000-game run as inconclusive.
- Confirm a promising result with at least three total candidate runs and comparable incumbent data. Use more runs for small effects.
- Pool wins and losses across runs. Exclude draws from the printed win-rate calculation, but compare draw rates separately.
- For close results, compute a two-proportion 95% confidence interval for the difference between candidate and incumbent decided-game win rates. Do not retain the change on performance grounds while the lower bound is at or below zero; gather more games or reject it.
- Require a deterministic behavior test even when the aggregate result is strong.
- Reject any change that introduces a regression, illegal action, panic, or substantial unexplained draw-rate increase.

When an improvement changes code used by both candidate and baseline, the head-to-head result does not isolate that improvement. Add direct behavior tests and, when practical, a candidate-only comparison path. If isolation is not practical, report weaker confidence and do not describe the result as proven.

## Choosing hypotheses

Useful areas include combat decisions, threat and permanent valuation, target selection, spell timing, mana use, mulligans, activated abilities, and resource preservation. Search existing tests and metadata before adding new special cases. Prefer using `EffectProperties`, `AIHint`, legal targets, combat results, and current game state.

Avoid optimizations that exploit the benchmark decks without expressing a general Magic decision. Do not use card names unless no reusable engine metadata can represent the behavior and the user accepts that limitation.

## Verification and stopping

After each accepted change, run:

```bash
go test ./pkg/mage/interactive/ai/... ./pkg/mage/interactive/eval/... -short -count=1
```

At the end, run the broader tests and lint in proportion to the files changed. `make test` includes the long simulation tests, so account for that cost and do not substitute cached results. Run `make lint` last, and inspect any automatic edits before keeping them.

If the user gives a time, token, game, or iteration budget, use it as the stopping rule. Otherwise, continue after each accepted change and stop after three consecutive well-formed hypotheses fail the acceptance gate, or when no safe independent hypothesis remains. Do not stop after the first successful experiment.

Report:

- every accepted and rejected hypothesis;
- incumbent and final pooled results, including draws and run counts;
- focused and broader tests run;
- files that remain changed;
- noise, shared-baseline effects, or other limits on confidence;
- the next best hypotheses if the stopping rule ends the run.
