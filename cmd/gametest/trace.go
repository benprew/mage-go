package main

import (
	"fmt"
	"io"
	"math"
	"strings"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai/heuristic"
)

type positionScorer interface {
	PositionValue(*mage.Game, uuid.UUID) int
}

func playerStrategy(p mage.Player) ai.AIStrategy {
	if player, ok := p.(*ai.AIPlayer); ok {
		strategy := player.Strategy
		if traced, ok := strategy.(*decisionTraceStrategy); ok {
			strategy = traced.AIStrategy
		}
		return strategy
	}
	return nil
}

func predictedScore(g *mage.Game, p mage.Player) string {
	scorer, ok := playerStrategy(p).(positionScorer)
	if !ok {
		return "n/a"
	}
	switch score := scorer.PositionValue(g, p.PlayerID()); score {
	case math.MaxInt:
		return "WIN"
	case math.MinInt:
		return "LOSS"
	default:
		return fmt.Sprintf("%+d", score)
	}
}

func printPredictedOptions(out io.Writer, g *mage.Game, p mage.Player, options []interactive.ActionOption) {
	predictor, ok := playerStrategy(p).(*heuristic.Strategy)
	if !ok {
		predictor = heuristic.New(ai.MidrangeWeighted)
	}
	for i, option := range options {
		value := "prediction unavailable"
		if branch, ok := predictor.PredictOption(g, p.PlayerID(), option); ok {
			value = predictedScore(branch, p) + "/" + predictedScore(branch, g.GetOpponent(p.PlayerID()))
		}
		fmt.Fprintf(out, "    %d. %s (%s)\n", i+1, option.Label, value)
	}
}

func printPositionScores(out io.Writer, g *mage.Game) {
	for _, p := range g.AllPlayers() {
		value := "unavailable for this strategy"
		if player, ok := p.(*ai.AIPlayer); ok {
			strategy := player.Strategy
			if traced, ok := strategy.(*decisionTraceStrategy); ok {
				strategy = traced.AIStrategy
			}
			if scorer, ok := strategy.(interface {
				PositionValue(*mage.Game, uuid.UUID) int
			}); ok {
				score := scorer.PositionValue(g, p.PlayerID())
				switch score {
				case math.MaxInt:
					value = "WIN (terminal)"
				case math.MinInt:
					value = "LOSS (terminal)"
				default:
					value = fmt.Sprintf("%+d", score)
				}
			}
		}
		fmt.Fprintf(out, "    %s eval: %s\n", p.Name(), value)
	}
}

func printBoardState(out io.Writer, g *mage.Game) {
	for _, p := range g.AllPlayers() {
		hand := handStr(p.Hand())
		if hand == "" {
			hand = "(empty)"
		}
		fmt.Fprintf(out, "    %s: %d life, %d in library, hand: [%s]\n", p.Name(), p.Life(), len(p.Library()), hand)
		permanents := battlefieldFor(g, p.PlayerID())
		parts := make([]string, 0, len(permanents))
		for _, perm := range permanents {
			text := permStr(g, []*mage.Permanent{perm})
			var attrs []string
			for attr := core.Flying; attr <= core.Ward; attr++ {
				if perm.HasKeyword(attr) {
					attrs = append(attrs, attr.String())
				}
			}
			if perm.HasType(core.TypeCreature) && perm.HasAttr(core.AttrSummonSick) {
				attrs = append(attrs, "summoning sick")
			}
			if perm.Damage > 0 {
				attrs = append(attrs, fmt.Sprintf("%d damage", perm.Damage))
			}
			if perm.AttachedTo != uuid.Nil() {
				if target := g.FindPermanent(perm.AttachedTo); target != nil {
					attrs = append(attrs, "attached to "+target.Name())
				}
			}
			if len(attrs) > 0 {
				text += " [" + strings.Join(attrs, ", ") + "]"
			}
			parts = append(parts, text)
		}
		board := strings.Join(parts, ", ")
		if board == "" {
			board = "(empty)"
		}
		fmt.Fprintf(out, "    %s's battlefield: %s\n", p.Name(), board)
	}
}

type decisionTraceStrategy struct {
	ai.AIStrategy
	out io.Writer
}

func (s *decisionTraceStrategy) Attackers(p mage.Player, g *mage.Game) []uuid.UUID {
	attackers := s.AIStrategy.Attackers(p, g)
	if len(attackers) == 0 {
		return attackers
	}
	var names []string
	for _, id := range attackers {
		if perm := g.FindPermanent(id); perm != nil {
			names = append(names, permStr(g, []*mage.Permanent{perm}))
		}
	}
	if len(names) == 0 {
		names = append(names, "(none)")
	}
	fmt.Fprintf(s.out, "  [%s] %s declares attackers: %s\n", g.GetStep(), p.Name(), strings.Join(names, ", "))
	printPositionScores(s.out, g)
	return attackers
}

func (s *decisionTraceStrategy) Blockers(p mage.Player, g *mage.Game) []mage.BlockAssignment {
	blocks := s.AIStrategy.Blockers(p, g)
	if len(g.CombatGroups()) == 0 {
		return blocks
	}
	fmt.Fprintf(s.out, "  [%s] %s declares blocks:\n", g.GetStep(), p.Name())
	if len(blocks) == 0 {
		fmt.Fprintln(s.out, "    (none)")
	}
	for _, block := range blocks {
		blocker := g.FindPermanent(block.BlockerID)
		attacker := g.FindPermanent(block.AttackerID)
		if blocker != nil && attacker != nil {
			fmt.Fprintf(s.out, "    %s blocks %s\n", permStr(g, []*mage.Permanent{blocker}), permStr(g, []*mage.Permanent{attacker}))
		}
	}
	printPositionScores(s.out, g)
	return blocks
}
