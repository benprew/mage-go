package mage

import (
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// CheckStateBasedActions checks and performs all state-based actions until no further
// state-based actions occur (CR 704.3).
func (g *Game) CheckStateBasedActions() {
	g.checkStateBasedActions()
}

func (g *Game) checkStateBasedActions() bool {
	performed := false
	for {
		actions := false

		// All loss conditions use the state before this batch changes permanents.
		var losing []Player
		for _, p := range g.players {
			lifeLoss := p.Life() <= 0 && !g.PreventsLifeTotalLoss(p.PlayerID())
			if p.IsAlive() && (lifeLoss || p.PoisonCounters() >= 10 || p.DrewFromEmpty()) {
				losing = append(losing, p)
			}
			p.ClearDrewFromEmpty()
		}

		for _, card := range slices.Clone(g.zones.exile) {
			if card.spellCopy {
				g.RemoveFromExile(card.Card.ID())
				actions = true
			}
		}

		var toDestroy, toGraveyard []*Permanent
		for _, p := range g.zones.battlefield {
			if p.HasType(TypeCreature) && p.CurrentToughness(g) <= 0 || p.HasType(TypePlaneswalker) && p.Counters[Loyalty] <= 0 {
				toGraveyard = append(toGraveyard, p)
			} else if p.HasType(TypeCreature) && (p.LethalDamage(g) || (p.DeathtouchDamage && p.Damage > 0)) && !p.HasKeyword(Indestructible) {
				toDestroy = append(toDestroy, p)
			}
		}
		if len(toDestroy)+len(toGraveyard) > 0 {
			actions = true
			g.moveGraveyardBatch(toDestroy, toGraveyard, false)
		}

		// MTG rule 704.5q: +1/+1 and -1/-1 counter annihilation
		for _, p := range g.zones.battlefield {
			plus := p.Counters[P1P1]
			minus := p.Counters[M1M1]
			if plus > 0 && minus > 0 {
				remove := min(minus, plus)
				p = g.MutablePermanent(p.ID())
				if p == nil {
					continue
				}
				p.Counters[P1P1] -= remove
				p.Counters[M1M1] -= remove
				actions = true
			}
		}

		for _, p := range g.zones.battlefield {
			if p.PhasedOut {
				continue
			}
			for counter, count := range p.Counters {
				if count == 0 {
					continue
				}
				if limit, found := sourceCounterLimit(p, CounterType(counter)); found && count > limit {
					g.MutablePermanent(p.ID()).RemoveCounter(CounterType(counter), count-limit)
					actions = true
				}
			}
		}

		// Check for auras attached to nothing or illegal targets (CR 704.5m / 303.4c).
		var aurasToDrop []*Permanent
		for _, p := range g.zones.battlefield {
			if p.PhasedOut {
				continue
			}
			if p.HasSubType("Aura") && p.IsAttached() {
				host := g.FindPermanent(p.AttachedTo)
				if host == nil || host.incarnationID != p.attachedIncarnation {
					aurasToDrop = append(aurasToDrop, p)
					actions = true
				} else if host.HasProtectionFromInGame(p.Card, g) {
					aurasToDrop = append(aurasToDrop, p)
					actions = true
				} else if !auraHostIsLegalForController(p.Card, host, g, p.ControllerID()) {
					aurasToDrop = append(aurasToDrop, p)
					actions = true
				}
			}
		}
		if len(aurasToDrop) > 0 {
			g.withZoneChangeBatch(func() {
				for _, a := range aurasToDrop {
					aura := capturedPermanent{a.ID(), a.incarnationID}
					host := capturedPermanent{a.AttachedTo, a.attachedIncarnation}
					currentHost := g.FindPermanent(host.id)
					absentHost := currentHost == nil || currentHost.incarnationID != host.incarnation
					g.PutPermanentIntoGraveyard(a)
					if absentHost && g.FindPermanent(aura.id) == nil {
						g.captureDepartedAuraCard(aura, host, capturedCard{id: aura.id, visit: g.graveyardVisit(aura.id), zone: ZoneGraveyard})
					}
				}
			})
		}

		// Protection can make an existing Equipment or Fortification attachment illegal.
		for _, p := range g.zones.battlefield {
			if p.PhasedOut {
				continue
			}
			if (p.HasSubType("Equipment") || p.HasSubType("Fortification")) && p.IsAttached() {
				host := g.FindPermanent(p.AttachedTo)
				if host == nil || (p.HasSubType("Equipment") && !host.HasType(TypeCreature)) || (p.HasSubType("Fortification") && !host.HasType(TypeLand)) || host.HasProtectionFromInGame(p.Card, g) {
					p = g.MutablePermanent(p.ID())
					if p == nil {
						continue
					}
					if host != nil {
						host = g.MutablePermanent(host.ID())
						host.Attachments = slices.DeleteFunc(host.Attachments, func(id uuid.UUID) bool { return id == p.ID() })
					}
					p.AttachedTo = uuid.Nil
					p.attachedIncarnation = uuid.Nil
					actions = true
				}
			}
		}

		// MTG rule 704.5j: Legend rule — if a player controls two or more legendary
		// permanents with the same name, they choose one and sacrifice the rest.
		var legendCounts map[uuid.UUID]map[string][]*Permanent // controller -> name -> perms
		for _, p := range g.zones.battlefield {
			if p.Card.HasSuperType(SuperLegendary) {
				if legendCounts == nil {
					legendCounts = make(map[uuid.UUID]map[string][]*Permanent)
				}
				if legendCounts[p.ControllerID()] == nil {
					legendCounts[p.ControllerID()] = make(map[string][]*Permanent)
				}
				legendCounts[p.ControllerID()][p.Name()] = append(legendCounts[p.ControllerID()][p.Name()], p)
			}
		}
		for ctrlID, byName := range legendCounts {
			for _, perms := range byName {
				if len(perms) > 1 {
					player := g.GetPlayer(ctrlID)
					keep := player.ChoosePermanent(perms, "legend rule: keep one", g)
					for _, p := range perms {
						if p != keep {
							g.PutPermanentIntoGraveyard(p)
						}
					}
					actions = true
				}
			}
		}

		// MTG rule 704.5k: World rule — if two or more permanents have the World
		// supertype, all except the most recent one are put into their owners' graveyards.
		var worldPerms []*Permanent
		for _, p := range g.zones.battlefield {
			if p.Card.HasSuperType(SuperWorld) {
				worldPerms = append(worldPerms, p)
			}
		}
		if len(worldPerms) > 1 {
			// Keep the most recently entered one (last in Battlefield slice)
			keep := worldPerms[len(worldPerms)-1]
			for _, p := range worldPerms {
				if p != keep {
					g.PutPermanentIntoGraveyard(p)
				}
			}
			actions = true
		}

		for _, p := range losing {
			p.SetLost()
			actions = true
		}

		// MTG rule 704.5d / CR 111.7: a token in any zone other than the battlefield
		// ceases to exist. Token status lives on Permanent, so when a token permanent
		// leaves the battlefield its underlying Card is just a regular card — it is
		// skipped by DestroyPermanent/bounce rather than cleaned up here.

		if !actions {
			break
		}
		performed = true
	}

	// Put any pending triggers on the stack
	if g.triggers.immediateManaDepth == 0 {
		g.PutTriggersOnStack()
	}
	return performed
}
