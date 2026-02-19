# mage-go

A Go reimplementation of [XMage](https://github.com/magefree/mage) (Magic, Another Game Engine) — an open-source Java engine that enforces full MTG rules for 28,000+ cards.

## Overview

`mage-go` is a Go library (`github.com/mage/mage`) implementing the Magic: The Gathering game engine. It handles game state, cards, abilities, effects, combat, mana, targeting, the stack, and a test harness for scripted game scenarios.

## Package Layout

```
pkg/mage/        # game engine (package mage)
cards/limited/   # Alpha (Limited Edition) card set (package limited)
cards/arabian/   # Arabian Nights card set (package arabian)
cards/custom/    # custom/test cards (package custom)
cmd/tui/         # terminal UI (package main)
cmd/fetchset/    # set fetcher utility (package main)
```

## Build & Test

```bash
make test    # go test ./...
make build   # go build ./...
make vet     # go vet ./...
make lint    # golangci-lint + staticcheck
```

## Credits

This project is a Go port of XMage. All rules logic, card implementations, and test scenarios are derived from or inspired by [XMage](https://github.com/magefree/mage), which is itself licensed under GPL-2.0.

Magic: The Gathering is a trademark of Wizards of the Coast LLC.

## License

GPL-2.0. See [LICENSE](LICENSE) for full text.
