---
name: autoresearch-heuristic-ai
description: Improve the mage-go heuristic AI through isolated, measured experiments against the comparison baseline. Use for autonomous heuristic tuning, not ordinary AI bug fixes or unmeasured feature work.
---

# Improve the heuristic AI

Keep a change only when focused tests prove the intended behavior and repeated comparison runs provide credible evidence of better play.

## Boundaries

Work in the heuristic AI, combat solver, evaluation package, or shared AI code when the behavior belongs there. Do not change cards, game rules, benchmark decks, the turn limit, or the baseline to improve a candidate result.

The baseline calls some current heuristic helpers, combat solver code, and evaluation code. Identify shared paths before each experiment because a shared change weakens the comparison.

## Run the research loop

Before the first experiment, read:

- `pkg/mage/interactive/ai/heuristic/sim_test.go`;
- `pkg/mage/interactive/ai/heuristic/baseline_test.go`;
- the affected candidate code and closest tests;
- [references/experiment-protocol.md](references/experiment-protocol.md).

For each experiment:

1. State one general game-play weakness and a testable hypothesis.
2. Add a deterministic behavior test and confirm that it fails for the expected reason.
3. Make one coherent change and pass focused tests before simulation.
4. Run the candidate comparison and record every result.
5. Accept the change only when it passes the protocol gate. Otherwise, revert only that experiment with a targeted patch.
6. Treat an accepted change as the new incumbent before testing another hypothesis.

Do not combine unmeasured ideas or tune constants to one random result. Prefer game-state rules and existing metadata over card-name logic.

## Records and stopping

For a multi-experiment run, record results with [references/experiment-record.schema.json](references/experiment-record.schema.json). Keep the artifact in task notes or a temporary file, then validate it with:

```bash
python3 .agents/skills/scripts/validate_artifact.py .agents/skills/autoresearch-heuristic-ai/references/experiment-record.schema.json <artifact.json>
```

Use the user's budget when given. Otherwise, stop after three consecutive well-formed hypotheses fail the acceptance gate or no safe independent hypothesis remains. Report accepted and rejected hypotheses, pooled results, tests, changed files, confidence limits, and the next best hypothesis.
