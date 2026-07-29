package ai

import (
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
)

// This file gives AIPlayer value-aware implementations of the resolution-time
// and cost-time choice hooks. Without these overrides the engine falls back to
// BasePlayer's naive defaults (pick the first option / the maximum / White),
// which discards bombs, sacrifices the best creature, and tutors arbitrary
// cards. Each override routes through the shared eval scoring so the same
// valuation that drives spell/target selection from hand also drives these
// decisions.

// gameOf recovers the concrete *mage.Game from a GameReader. The engine always
// passes the live game, but the interface uses GameReader; returning nil lets
// callers fall back to game-agnostic heuristics in the rare case it is absent.
func gameOf(g mage.GameReader) *mage.Game {
	if gg, ok := g.(*mage.Game); ok {
		return gg
	}
	return nil
}

// permChoiceValue scores a permanent for sacrifice/destroy/keep choices, using
// the full eval valuation when a game is available and a P/T- or CMC-based
// fallback otherwise.
func (ai *AIPlayer) permChoiceValue(perm *mage.Permanent, g mage.GameReader) int {
	if gg := gameOf(g); gg != nil {
		return eval.PermanentValueForTargeting(gg, perm, eval.TargetGeneric)
	}
	if perm.HasType(core.TypeCreature) {
		return perm.CurrentPower(g) + perm.CurrentToughness(g)
	}
	return perm.Card.ManaCost().CMC()
}

// ChoosePermanent picks a permanent based on why the choice is being made:
//   - sacrifice / return-to-hand costs: keep the best, give up the weakest.
//   - destroy / exile / "keep one" (legend rule): pick the most valuable.
//   - anything else: first candidate.
func (ai *AIPlayer) ChoosePermanent(candidates []*mage.Permanent, reason string, g mage.GameReader) *mage.Permanent {
	if len(candidates) == 0 {
		return nil
	}
	lower := strings.ToLower(reason)
	wantLowest := strings.Contains(lower, "sacrifice") || strings.Contains(lower, "return")
	wantHighest := strings.Contains(lower, "destroy") || strings.Contains(lower, "exile") || strings.Contains(lower, "keep")
	if !wantLowest && !wantHighest {
		return candidates[0]
	}

	best := candidates[0]
	bestVal := ai.permChoiceValue(best, g)
	for _, c := range candidates[1:] {
		val := ai.permChoiceValue(c, g)
		if (wantLowest && val < bestVal) || (wantHighest && val > bestVal) {
			best = c
			bestVal = val
		}
	}
	return best
}

// discardKeepValue scores how much the AI wants to keep a card when forced to
// discard. Lower values are discarded first. Surplus lands are cheap to pitch
// when the board is flooded but precious when mana-light; spells are valued by
// their context-sensitive SpellValue (so bombs survive over filler).
func (ai *AIPlayer) discardKeepValue(card mage.Card, g *mage.Game) int {
	if card.HasType(core.TypeLand) {
		if g == nil {
			return 8
		}
		landsInPlay := g.CountBattlefield(mage.And(mage.IsLand, mage.ControlledBy(ai.PlayerID())))
		switch {
		case landsInPlay <= 2:
			return 60
		case landsInPlay >= 5:
			return 2
		default:
			return 12
		}
	}
	cmc := card.ManaCost().CMC()
	if g == nil {
		return cmc + 3
	}
	v := max(eval.SpellValue(card, ai, g), cmc)
	return v + 3
}

// ChooseCardsFromHand returns the `amount` lowest keep-value cards to discard,
// keeping the AI's most valuable cards.
func (ai *AIPlayer) ChooseCardsFromHand(amount int, reason string, g mage.GameReader) []mage.Card {
	hand := ai.Hand()
	if amount >= len(hand) {
		return hand
	}
	if amount <= 0 {
		return nil
	}
	gg := gameOf(g)
	sorted := make([]mage.Card, len(hand))
	copy(sorted, hand)
	sort.SliceStable(sorted, func(i, j int) bool {
		return ai.discardKeepValue(sorted[i], gg) < ai.discardKeepValue(sorted[j], gg)
	})
	return sorted[:amount]
}

// ChooseCardFromHand picks one card from a pre-filtered candidate list. Discard
// and sacrifice prompts give up the lowest keep-value card; other (beneficial)
// selections take the highest-value card.
func (ai *AIPlayer) ChooseCardFromHand(candidates []mage.Card, reason string, g mage.GameReader) mage.Card {
	if len(candidates) == 0 {
		return nil
	}
	gg := gameOf(g)
	lower := strings.ToLower(reason)
	beneficial := strings.Contains(lower, "reveal") || strings.Contains(lower, "play") ||
		strings.Contains(lower, "cast") || strings.Contains(lower, "put")

	best := candidates[0]
	bestVal := ai.discardKeepValue(best, gg)
	for _, c := range candidates[1:] {
		val := ai.discardKeepValue(c, gg)
		if (beneficial && val > bestVal) || (!beneficial && val < bestVal) {
			best = c
			bestVal = val
		}
	}
	return best
}

// ChooseCardFromLibrary tutors for the highest-value card. Non-land spells are
// scored by SpellValue (so a bomb beats filler); lands score low and are only
// fetched when nothing better is available.
func (ai *AIPlayer) ChooseCardFromLibrary(candidates []mage.Card, reason string, g mage.GameReader) mage.Card {
	if len(candidates) == 0 {
		return nil
	}
	gg := gameOf(g)
	best := candidates[0]
	bestVal := ai.libraryFetchValue(best, gg)
	for _, c := range candidates[1:] {
		val := ai.libraryFetchValue(c, gg)
		if val > bestVal {
			best = c
			bestVal = val
		}
	}
	return best
}

func (ai *AIPlayer) libraryFetchValue(card mage.Card, g *mage.Game) int {
	if card.HasType(core.TypeLand) {
		return 0
	}
	if g == nil {
		return card.ManaCost().CMC() + 1
	}
	return eval.SpellValue(card, ai, g) + 1
}

// ChooseManaColor returns the color the AI most needs for the spells in hand,
// defaulting to White when the hand has no colored requirements.
func (ai *AIPlayer) ChooseManaColor(reason string) core.Color {
	colors := []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green}
	counts := make(map[core.Color]int, len(colors))
	for _, card := range ai.Hand() {
		mc := card.ManaCost()
		counts[core.White] += mc.White
		counts[core.Blue] += mc.Blue
		counts[core.Black] += mc.Black
		counts[core.Red] += mc.Red
		counts[core.Green] += mc.Green
	}
	best := core.White
	bestCount := -1
	for _, c := range colors {
		if counts[c] > bestCount {
			bestCount = counts[c]
			best = c
		}
	}
	return best
}

// ChooseNumber picks the minimum for costs the AI wants to minimize
// (discarding, sacrificing, paying life) and the maximum otherwise (e.g. how
// much damage to deal, how many cards to draw).
func (ai *AIPlayer) ChooseNumber(minimum, maximum int, reason string) int {
	lower := strings.ToLower(reason)
	if strings.Contains(lower, "discard") || strings.Contains(lower, "sacrifice") ||
		strings.Contains(lower, "pay") || strings.Contains(lower, "lose") {
		return minimum
	}
	return maximum
}

// ChooseDamageDistribution spreads divided damage to kill as many of the
// opponent's creatures as possible (assigning each exactly lethal, biggest
// first), then dumps any remainder on the opponent's face if it is a legal
// target, otherwise piling it onto an already-targeted creature.
func (ai *AIPlayer) ChooseDamageDistribution(possible []uuid.UUID, total int, reason string, g *mage.Game) map[uuid.UUID]int {
	if total <= 0 || len(possible) == 0 {
		return nil
	}
	if g == nil {
		return map[uuid.UUID]int{possible[0]: total}
	}
	myID := ai.PlayerID()

	type creatureTarget struct {
		id     uuid.UUID
		lethal int
		value  int
	}
	var creatures []creatureTarget
	oppPlayer := uuid.Nil
	for _, id := range possible {
		if perm := g.FindPermanent(id); perm != nil {
			if perm.ControllerID() == myID {
				continue
			}
			lethal := max(perm.CurrentToughness(g)-perm.Damage, 1)
			creatures = append(creatures, creatureTarget{
				id:     id,
				lethal: lethal,
				value:  eval.PermanentValueForTargeting(g, perm, eval.TargetBurn),
			})
		} else if pl := g.GetPlayer(id); pl != nil && pl.PlayerID() != myID {
			oppPlayer = id
		}
	}

	sort.SliceStable(creatures, func(i, j int) bool {
		return creatures[i].value > creatures[j].value
	})

	out := make(map[uuid.UUID]int)
	remaining := total
	for _, c := range creatures {
		if remaining <= 0 {
			break
		}
		if c.lethal <= remaining {
			out[c.id] = c.lethal
			remaining -= c.lethal
		}
	}

	if remaining > 0 {
		switch {
		case oppPlayer != uuid.Nil:
			out[oppPlayer] += remaining
		case len(out) > 0:
			for _, id := range possible {
				if _, ok := out[id]; ok {
					out[id] += remaining
					break
				}
			}
		default:
			out[possible[0]] = remaining
		}
	}

	return out
}

// ChooseTargets selects resolution-time targets (triggered abilities, copied
// spells) the AI could not pre-select at cast time. Lacking the effect's
// purpose at this point, it applies the common-case removal heuristic: prefer
// the opponent (face, then their most valuable permanents) over our own.
func (ai *AIPlayer) ChooseTargets(possible []uuid.UUID, minimum, maximum int, g *mage.Game) []uuid.UUID {
	if len(possible) < minimum {
		return nil
	}
	if g == nil || minimum <= 0 {
		count := min(minimum, len(possible))
		return possible[:count]
	}

	myID := ai.PlayerID()
	type scored struct {
		id    uuid.UUID
		score int
	}
	ranked := make([]scored, 0, len(possible))
	for _, id := range possible {
		switch {
		case g.FindPermanent(id) != nil:
			perm := g.FindPermanent(id)
			val := ai.permChoiceValue(perm, g)
			if perm.ControllerID() != myID {
				ranked = append(ranked, scored{id, 1000 + val})
			} else {
				ranked = append(ranked, scored{id, val})
			}
		case g.GetPlayer(id) != nil:
			if g.GetPlayer(id).PlayerID() != myID {
				ranked = append(ranked, scored{id, 2000})
			} else {
				ranked = append(ranked, scored{id, 0})
			}
		default:
			ranked = append(ranked, scored{id, 500})
		}
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})

	count := min(minimum, len(ranked))
	result := make([]uuid.UUID, count)
	for i := range count {
		result[i] = ranked[i].id
	}
	return result
}

// scryPlan partitions the revealed top cards into ones to keep on top (best
// draws first) and ones to send away (flooded surplus lands). The same logic
// drives scry (away = bottom of library) and surveil (away = graveyard).
func (ai *AIPlayer) scryPlan(top []mage.Card, g mage.GameReader) (keepTop, sendAway []uuid.UUID) {
	gg := gameOf(g)
	if gg == nil {
		ids := make([]uuid.UUID, len(top))
		for i, c := range top {
			ids[i] = c.ID()
		}
		return ids, nil
	}

	myID := ai.PlayerID()
	landsInPlay := gg.CountBattlefield(mage.And(mage.IsLand, mage.ControlledBy(myID)))
	availMana := eval.CountAvailableMana(gg, myID)

	type kept struct {
		id       uuid.UUID
		priority int
	}
	var keep []kept
	for _, c := range top {
		if c.HasType(core.TypeLand) {
			// Bottom surplus lands once we are no longer mana-light.
			if landsInPlay >= 5 {
				sendAway = append(sendAway, c.ID())
				continue
			}
			priority := 10
			if landsInPlay < 3 {
				priority = 50
			}
			keep = append(keep, kept{c.ID(), priority})
			continue
		}
		priority := eval.SpellValue(c, ai, gg)
		if c.ManaCost().CMC() <= availMana {
			priority += 20
		}
		keep = append(keep, kept{c.ID(), priority})
	}

	sort.SliceStable(keep, func(i, j int) bool {
		return keep[i].priority > keep[j].priority
	})
	keepTop = make([]uuid.UUID, len(keep))
	for i, k := range keep {
		keepTop[i] = k.id
	}
	return keepTop, sendAway
}

// ChooseScryPlacement keeps the best draws on top and bottoms flooded surplus
// lands.
func (ai *AIPlayer) ChooseScryPlacement(top []mage.Card, reason string, g mage.GameReader) (bottom, topOrder []uuid.UUID) {
	topOrder, bottom = ai.scryPlan(top, g)
	return bottom, topOrder
}

// ChooseSurveilPlacement keeps the best draws on top and mills flooded surplus
// lands to the graveyard.
func (ai *AIPlayer) ChooseSurveilPlacement(top []mage.Card, reason string, g mage.GameReader) (graveyard, topOrder []uuid.UUID) {
	topOrder, graveyard = ai.scryPlan(top, g)
	return graveyard, topOrder
}
