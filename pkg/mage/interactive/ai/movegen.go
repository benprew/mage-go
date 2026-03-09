package ai

import (
	"sort"

	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/interactive"
	"github.com/mage/mage/pkg/mage/interactive/eval"
)

// Move represents a single action the AI can take during a priority window.
type Move struct {
	Type         interactive.ActionType
	CardID       uuid.UUID
	CardName     string
	Targets      []uuid.UUID
	PermanentID  uuid.UUID
	AbilityIndex int
	Attackers    []uuid.UUID

	IsCreature bool
	heuristic  int
}

// GeneratePriorityMoves enumerates legal moves for playerID at the current
// priority window, sorted by heuristic value (descending).
func GeneratePriorityMoves(g *mage.Game, p mage.Player, landsPlayed int, mainPhase bool) []Move {
	playerID := p.PlayerID()
	var moves []Move

	if mainPhase {
		if landsPlayed < 1 {
			for _, c := range p.Hand() {
				if c.HasType(core.TypeLand) {
					moves = append(moves, Move{
						Type:      interactive.ActionPlayLand,
						CardID:    c.ID(),
						CardName:  c.Name(),
						heuristic: 10,
					})
					break
				}
			}
		}

		for _, card := range g.GetCastableSpells(playerID) {
			if card.HasType(core.TypeInstant) {
				continue
			}
			if eval.SpellIsWorthless(card, p, g) {
				continue
			}
			moves = append(moves, expandSpellMoves(p, g, card)...)
		}
	}

	for _, card := range p.Hand() {
		if !card.HasType(core.TypeInstant) {
			continue
		}
		if !g.CanAfford(playerID, card.ManaCost()) {
			continue
		}
		hasUsableEffect := false
		for _, a := range card.Abilities() {
			if sa, ok := a.(*mage.SpellAbility); ok {
				if mage.SpellOutcome(sa.Effects()) != mage.OutcomeUnknown {
					hasUsableEffect = true
					break
				}
			}
		}
		if !hasUsableEffect {
			continue
		}
		if eval.SpellIsWorthless(card, p, g) {
			continue
		}
		moves = append(moves, expandSpellMoves(p, g, card)...)
	}

	for _, info := range g.GetActivatableAbilities(playerID) {
		q := abilityQualityFromInfo(g, info)
		if q < 3 {
			continue
		}
		moves = append(moves, Move{
			Type:         interactive.ActionActivateAbility,
			PermanentID:  info.PermanentID,
			CardName:     info.PermanentName,
			AbilityIndex: info.AbilityIndex,
			heuristic:    q,
		})
	}

	moves = append(moves, Move{
		Type:      interactive.ActionPass,
		heuristic: 0,
	})

	sort.Slice(moves, func(i, j int) bool {
		return moves[i].heuristic > moves[j].heuristic
	})

	return moves
}

// GenerateAttackerSets produces a pruned set of attacker combinations.
func GenerateAttackerSets(g *mage.Game, playerID uuid.UUID) [][]uuid.UUID {
	eligible := getEligibleAttackers(g, playerID)
	if len(eligible) == 0 {
		return [][]uuid.UUID{nil}
	}

	var sets [][]uuid.UUID

	all := make([]uuid.UUID, len(eligible))
	for i, p := range eligible {
		all[i] = p.ID()
	}
	sets = append(sets, all)

	sets = append(sets, nil)

	if len(eligible) > 1 {
		for _, p := range eligible {
			sets = append(sets, []uuid.UUID{p.ID()})
		}
	}

	var evasive []uuid.UUID
	for _, p := range eligible {
		if p.HasKeyword(core.Flying) ||
			p.HasKeyword(core.UnblockableKW) ||
			p.HasKeyword(core.Fear) ||
			p.HasKeyword(core.Islandwalk) ||
			p.HasKeyword(core.Swampwalk) ||
			p.HasKeyword(core.Forestwalk) ||
			p.HasKeyword(core.Mountainwalk) ||
			p.HasKeyword(core.Plainswalk) {
			evasive = append(evasive, p.ID())
		}
	}
	if len(evasive) > 0 && len(evasive) < len(eligible) {
		sets = append(sets, evasive)
	}

	return sets
}

func getEligibleAttackers(g *mage.Game, playerID uuid.UUID) []*mage.Permanent {
	var eligible []*mage.Permanent
	for _, perm := range g.Battlefield {
		if perm.Controller != playerID || !perm.CanDeclareAsAttacker(g) {
			continue
		}
		eligible = append(eligible, perm)
	}
	return eligible
}

func abilityQualityFromInfo(g *mage.Game, info mage.ActivatableInfo) int {
	perm := g.FindPermanent(info.PermanentID)
	if perm == nil {
		return 0
	}
	if info.AbilityIndex >= len(perm.RuntimeAbilities) {
		return 0
	}
	a := perm.RuntimeAbilities[info.AbilityIndex]
	aa, ok := a.(mage.ActivatedAbility)
	if !ok {
		return 0
	}
	return eval.AbilityQuality(aa)
}

func expandSpellMoves(p mage.Player, g *mage.Game, card mage.Card) []Move {
	playerID := p.PlayerID()
	sv := eval.SpellValue(card, p, g)

	var possibleTargets []uuid.UUID
	for _, a := range card.Abilities() {
		sa, ok := a.(*mage.SpellAbility)
		if !ok {
			continue
		}
		for _, t := range sa.Targets() {
			possibleTargets = t.Possible(playerID, card, g)
			break
		}
	}

	if len(possibleTargets) == 0 {
		return []Move{{
			Type:       interactive.ActionCastSpell,
			CardID:     card.ID(),
			CardName:   card.Name(),
			IsCreature: card.HasType(core.TypeCreature),
			heuristic:  sv,
		}}
	}

	var moves []Move
	for _, tid := range possibleTargets {
		h := sv
		if tp := g.GetPlayer(tid); tp != nil && tp.PlayerID() != playerID {
			for _, a := range card.Abilities() {
				sa, ok := a.(*mage.SpellAbility)
				if !ok {
					continue
				}
				for _, e := range sa.Effects() {
					if dv := e.Properties().DamageValue; dv != nil {
						dmg := dv.Resolve(g, card.ID(), playerID)
						if dmg >= tp.Life() {
							h += 100
						}
					}
				}
			}
		}
		moves = append(moves, Move{
			Type:      interactive.ActionCastSpell,
			CardID:    card.ID(),
			CardName:  card.Name(),
			Targets:   []uuid.UUID{tid},
			heuristic: h,
		})
	}
	return moves
}

// AutoSelectTargetsForSearch uses the same targeting logic as HeuristicStrategy.
func AutoSelectTargetsForSearch(p mage.Player, g *mage.Game, card mage.Card) []uuid.UUID {
	strat := &HeuristicStrategy{Personality: MidrangePersonality}
	return strat.autoSelectTargets(p, g, card)
}
