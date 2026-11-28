---
name: autoresearch-heuristic-ai
description: Tune the mage-go heuristic AI through isolated, measured experiments against committed HEAD. Use for autonomous gameplay experiments.
---

# Improve the heuristic AI

Keep a change when a focused test proves the intended behavior and repeated comparisons provide evidence of better play.

## Boundaries

Work in the heuristic AI, combat solver, evaluation package, or shared decision code when the behavior belongs there. Hold the benchmark decks, turn limit, rules engine, and card definitions fixed. The runner compares the working tree's decision code with committed `HEAD`. Both players use the rules engine, card pool, and mtgish compiler from `HEAD`.

## Run the research loop

Read [the experiment protocol](references/experiment-protocol.md), `cmd/aibench/main.go`, `pkg/mage/interactive/ai/heuristic/sim_test.go`, and the affected decision code and tests.

1. State one gameplay weakness and a testable hypothesis. Measure the working tree incumbent before editing.
2. Add a deterministic behavior test and confirm it fails for the expected reason.
3. Make one coherent change and pass the focused tests before simulation.
4. Run the candidate comparison and record each result. Apply the acceptance criteria in the protocol.
5. Keep an accepted change as the next incumbent. Revert only the experiment if it fails the criteria.

Prefer general gameplay rules and existing metadata to card-name checks.

## Records and stopping

For a multi-experiment run, record results with the [experiment schema](references/experiment-record.schema.json). Keep the artifact in task notes or a temporary file. Validate a saved artifact with [the artifact validator](../scripts/validate_artifact.py).

Use the user's budget when given. Otherwise, stop after three consecutive well-formed hypotheses fail the acceptance criteria or no independent hypothesis remains. Report the outcome of each hypothesis, results, tests, changed files, and limits of the comparison.
