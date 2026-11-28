# Experiment protocol

Run focused tests, then measure the working tree incumbent from the mage-go root.

```bash
SIM_GAMES=1000 SIM_SEED=42 go run ./cmd/aibench
```

Set `SIM_GAMES` to a positive even number. The runner executes a HEAD-versus-HEAD control and a working-tree-versus-HEAD comparison. Each pair reuses the same opening deck orders and swaps the strategies between seats. Investigate a large or repeatable control bias before using the results.

Record the printed HEAD revision, seed, deck configuration, turn limit, wins, losses, actual draws, and turn-limit games. Keep the revision and match settings fixed for the incumbent and candidate runs. Use matching seeds for the two versions and different seeds across repeated trials.

## Acceptance criteria

Use the win rate among decided games as the main metric. Track actual draws and turn-limit games alongside it.

- Treat a change of about two percentage points or less in one 1,000-game run as inconclusive.
- Confirm a promising result with at least three runs of each version in the same environment. Pool the decided-game counts and check the result across runs and decks.
- Use the runner's 95% paired confidence interval for a comparison against `HEAD`. Assess gains over a modified incumbent with the matched before and after runs. Gather more games or leave a close result inconclusive when the interval includes no improvement.
- Reject a regression, illegal action, panic, or substantial unexplained increase in draws or turn-limit games. Require a deterministic behavior test for each accepted change.

Check the AI and evaluation packages after an accepted change.

```bash
go test ./pkg/mage/interactive/ai/... ./pkg/mage/interactive/eval/... -short -count=1
```

Run broader checks when required by the change. Inspect the diff if repository lint applies fixes.
