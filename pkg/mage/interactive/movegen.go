package interactive

import (
	"sort"

	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

// Move represents a single action the AI can take during a priority window
// or attacker declaration. Used by the search engine to enumerate legal actions.
type Move struct {
	Type         ActionType
	CardID       uuid.UUID
	CardName     string
	Targets      []uuid.UUID
	PermanentID  uuid.UUID
	AbilityIndex int
	Attackers    []uuid.UUID

	// IsCreature marks this move as a creature cast (always worth taking).
	IsCreature bool
	// heuristic is a pre-computed score used for move ordering (higher = search first).
	heuristic int
}

// GeneratePriorityMoves enumerates legal moves for playerID at the current
// priority window. Moves are sorted by heuristic value (descending) to
// improve alpha-beta pruning cut-off rates.
func GeneratePriorityMoves(g *mage.Game, p mage.Player, landsPlayed int, mainPhase bool) []Move {
	playerID := p.PlayerID()
	var moves []Move

	if mainPhase {
		// Land plays
		if landsPlayed < 1 {
			for _, c := range p.Hand() {
				if c.HasType(core.TypeLand) {
					moves = append(moves, Move{
						Type:      ActionPlayLand,
						CardID:    c.ID(),
						CardName:  c.Name(),
						heuristic: 10, // lands are high priority
					})
					break // only need one land play option
				}
			}
		}

		// Castable spells (non-instant during main phase)
		for _, card := range g.GetCastableSpells(playerID) {
			if card.HasType(core.TypeInstant) {
				continue
			}
			if spellIsWorthless(card, p, g) {
				continue
			}
			moves = append(moves, expandSpellMoves(p, g, card)...)
		}
	}

	// Instants (available in any phase)
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
		if spellIsWorthless(card, p, g) {
			continue
		}
		moves = append(moves, expandSpellMoves(p, g, card)...)
	}

	// Activated abilities (only those scoring >= 3)
	for _, info := range g.GetActivatableAbilities(playerID) {
		q := abilityQualityFromInfo(g, info)
		if q < 3 {
			continue
		}
		moves = append(moves, Move{
			Type:         ActionActivateAbility,
			PermanentID:  info.PermanentID,
			CardName:     info.PermanentName,
			AbilityIndex: info.AbilityIndex,
			heuristic:    q,
		})
	}

	// Pass is always an option
	moves = append(moves, Move{
		Type:      ActionPass,
		heuristic: 0,
	})

	// Sort by heuristic descending for better pruning
	sort.Slice(moves, func(i, j int) bool {
		return moves[i].heuristic > moves[j].heuristic
	})

	return moves
}

// GenerateAttackerSets produces a pruned set of attacker combinations.
// Instead of enumerating all 2^N subsets, it produces:
//   - Attack with all eligible
//   - Attack with none
//   - Each individual creature solo
//   - All evasive creatures (flying, unblockable, landwalk)
func GenerateAttackerSets(g *mage.Game, playerID uuid.UUID) [][]uuid.UUID {
	eligible := getEligibleAttackers(g, playerID)
	if len(eligible) == 0 {
		return [][]uuid.UUID{nil}
	}

	var sets [][]uuid.UUID

	// Attack with all
	all := make([]uuid.UUID, len(eligible))
	for i, p := range eligible {
		all[i] = p.ID()
	}
	sets = append(sets, all)

	// Attack with none
	sets = append(sets, nil)

	// Each individual creature solo
	if len(eligible) > 1 {
		for _, p := range eligible {
			sets = append(sets, []uuid.UUID{p.ID()})
		}
	}

	// Evasive creatures only
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

// abilityQualityFromInfo scores an activated ability for move-generation filtering.
// Returns a heuristic quality score; abilities scoring < 3 are pruned from search.
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
	return abilityQuality(aa)
}

// expandSpellMoves generates Move entries for a castable spell. For targeted
// spells, it creates one Move per possible target so the search can evaluate
// different targeting choices. For untargeted spells, it creates a single Move.
func expandSpellMoves(p mage.Player, g *mage.Game, card mage.Card) []Move {
	playerID := p.PlayerID()
	sv := spellValue(card, p, g)

	// Check if the spell has targets
	var possibleTargets []uuid.UUID
	for _, a := range card.Abilities() {
		sa, ok := a.(*mage.SpellAbility)
		if !ok {
			continue
		}
		for _, t := range sa.Targets() {
			possibleTargets = t.Possible(playerID, card, g)
			break // use first target requirement
		}
	}

	// Untargeted spell: one move
	if len(possibleTargets) == 0 {
		return []Move{{
			Type:       ActionCastSpell,
			CardID:     card.ID(),
			CardName:   card.Name(),
			IsCreature: card.HasType(core.TypeCreature),
			heuristic:  sv,
		}}
	}

	// Targeted spell: one move per target
	var moves []Move
	for _, tid := range possibleTargets {
		h := sv
		// Boost heuristic for player targets (lethal check)
		if tp := g.GetPlayer(tid); tp != nil && tp.PlayerID() != playerID {
			// Check if this could be lethal
			for _, a := range card.Abilities() {
				sa, ok := a.(*mage.SpellAbility)
				if !ok {
					continue
				}
				for _, e := range sa.Effects() {
					if dv := e.Properties().DamageValue; dv != nil {
						dmg := dv.Resolve(g, card.ID(), playerID)
						if dmg >= tp.Life() {
							h += 100 // massive bonus for lethal
						}
					}
				}
			}
		}
		moves = append(moves, Move{
			Type:      ActionCastSpell,
			CardID:    card.ID(),
			CardName:  card.Name(),
			Targets:   []uuid.UUID{tid},
			heuristic: h,
		})
	}
	return moves
}

// autoSelectTargetsForSearch uses the same targeting logic as HeuristicStrategy
// but is callable without a strategy instance.
func autoSelectTargetsForSearch(p mage.Player, g *mage.Game, card mage.Card) []uuid.UUID {
	strat := &HeuristicStrategy{Personality: MidrangePersonality}
	return strat.autoSelectTargets(p, g, card)
}
