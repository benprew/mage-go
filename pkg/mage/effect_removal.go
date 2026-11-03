package mage

import (
	"math/rand"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// destroyTargetEffect destroys the target permanent.

// DestroyTarget creates an effect that destroys the first target permanent.
func DestroyTarget() Effect {
	return Destroy(legacyFirstReference{})
}

// DestroyTargetStep returns the Effect for use as a pipeline/ForEach inner step.
func DestroyTargetStep() Effect { return Destroy(legacyFirstReference{}) }

// destroyTargetPermanentEffect destroys a target permanent matching a filter.

// DestroyTargetPermanent creates an effect that destroys a target permanent.
func DestroyTargetPermanent() Effect {
	return Destroy(legacyFirstReference{})
}

// DestroyTargetLand creates an effect that destroys a target land (e.g. Stone Rain, Sinkhole).
func DestroyTargetLand() Effect {
	return Destroy(legacyFirstReference{})
}

// DestroyTargetArtifact creates an effect that destroys a target artifact (e.g. Shatter).
func DestroyTargetArtifact() Effect {
	return Destroy(legacyFirstReference{})
}

// DestroyAllLands destroys all lands (Armageddon).
func DestroyAllLands() Effect {
	return DestroyAllMatching(IsLand, "destroy all lands")
}

// DestroyAllEnchantments destroys all enchantments (Tranquility).
func DestroyAllEnchantments() Effect {
	return DestroyAllMatching(IsEnchantment, "destroy all enchantments")
}

// destroyAllMatchingEffect destroys all permanents matching a filter.

// DestroyAllMatching creates an effect that destroys all permanents matching the filter.
func DestroyAllMatching(filter PermanentFilter, text string) Effect {
	return Destroy(BattlefieldObjects(filter))
}

// DestroyAllCreatures destroys all creatures (board wipe).
func DestroyAllCreatures() Effect {
	return DestroyAllMatching(IsCreature, "destroy all creatures")
}

// destroyAllMatchingNoRegenEffect destroys all matching permanents with "can't be regenerated".

// DestroyAllMatchingNoRegen creates an effect that destroys all permanents matching the filter.
// The destroyed permanents can't be regenerated (e.g. Shatterstorm).
func DestroyAllMatchingNoRegen(filter PermanentFilter, text string) Effect {
	return Destroy(BattlefieldObjects(filter), WithoutRegeneration())
}

// destroyTargetNoRegenEffect destroys a target permanent, preventing regeneration.

// DestroyTargetNoRegen creates an effect that destroys the first target permanent.
// The destroyed permanent can't be regenerated (e.g. Terror, Tunnel).
func DestroyTargetNoRegen() Effect {
	return Destroy(legacyFirstReference{}, WithoutRegeneration())
}

// DestroyTargetNoRegenStep returns the Effect for use in pipelines/ForEach.
func DestroyTargetNoRegenStep() Effect { return Destroy(legacyFirstReference{}, WithoutRegeneration()) }

// DestroyAllCreaturesNoRegen destroys all creatures; they can't be regenerated (e.g. Wrath of God).
func DestroyAllCreaturesNoRegen() Effect {
	return DestroyAllMatchingNoRegen(IsCreature, "destroy all creatures (can't be regenerated)")
}

// ExileTarget creates an effect that exiles a target permanent.
func ExileTarget() Effect {
	return MoveObjects(legacyFirstReference{}, ZoneExile)
}

// ExileTargetStep returns the Effect for use as a pipeline/ForEach inner step.
func ExileTargetStep() Effect { return MoveObjects(legacyFirstReference{}, ZoneExile) }

// balanceEffect equalizes lands, creatures, and hand sizes.
type balanceEffect struct{}

// BalanceEffect creates an effect that equalizes lands, creatures, and hand sizes across all
// players by having each player sacrifice/discard down to the minimum (Balance).
func BalanceEffect() Effect { return &balanceEffect{} }

func (e *balanceEffect) Text() string {
	return "Each player sacrifices to match fewest lands, creatures; discards to match smallest hand"
}
func (e *balanceEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}

// chaosOrbEffect destroys a random nontoken permanent an opponent controls, then destroys self.
type chaosOrbEffect struct{}

// ChaosOrbEffect creates the Chaos Orb effect.
func ChaosOrbEffect() Effect { return &chaosOrbEffect{} }

func (e *chaosOrbEffect) Text() string {
	return "Destroy a random nontoken permanent, then destroy ~"
}
func (e *chaosOrbEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// --- Executor functions (called from executor.go) ---

// ReturnedPermanentModifier mutates a permanent that has just re-entered the
// battlefield from exile (e.g. via ExileTargetReturnAtEndStep). Implementations
// may add counters, attach to something, etc. The Game is provided so the
// modifier can run ApplyContinuousEffects after mutating runtime state.
type ReturnedPermanentModifier func(g *Game, perm *Permanent)

// ExileTargetReturnAtEndStep exiles the resolving target permanent and
// registers a one-shot delayed triggered ability (CR 603.7) that, at the
// beginning of the next end step, returns the exiled card to the battlefield
// under its owner's control. The optional `onReturn` modifier runs once the
// permanent re-enters the battlefield, before the next ApplyContinuousEffects
// is needed. Used by Long Road Home (returns with a +1/+1 counter) and is the
// general primitive behind any "exile, return at end of turn with X" effect.
func ExileTargetReturnAtEndStep(onReturn ReturnedPermanentModifier) Effect {
	return FuncEffect(
		"exile target permanent; return it at the beginning of the next end step",
		EffectProperties{Outcome: OutcomeDetriment},
		func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
			if len(targets) == 0 {
				return nil
			}
			perm := g.FindPermanent(targets[0])
			if perm == nil {
				return nil
			}
			cardID := perm.Card.ID()
			owner := perm.Card.Owner()
			g.ExilePermanent(perm)
			returnEffect := FuncEffect(
				"return exiled card",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
					card, ok := g.RemoveFromExile(cardID)
					if !ok {
						return nil
					}
					newPerm := g.PutOnBattlefield(card, owner)
					if newPerm != nil && onReturn != nil {
						onReturn(g, newPerm)
					}
					return nil
				},
			)
			g.RegisterDelayedTrigger(&DelayedTrigger{
				EventType:  EvtEndStep,
				Effects:    []Effect{returnEffect},
				SourceID:   sourceID,
				Controller: controller,
			})
			return nil
		},
	)
}

// ReturnWithCounter is a ReturnedPermanentModifier that adds `amount` counters
// of `counter` type to the returning permanent (e.g. Long Road Home's +1/+1).
// ApplyContinuousEffects is called so dependent layers see the new counter.
func ReturnWithCounter(counter CounterType, amount int) ReturnedPermanentModifier {
	return func(g *Game, perm *Permanent) {
		if amount <= 0 {
			return
		}
		perm = g.MutablePermanent(perm.ID())
		if perm == nil {
			return
		}
		perm.AddCounter(counter, amount)
		g.ApplyContinuousEffects()
	}
}

func (*balanceEffect) Apply(ctx *EffectContext) error {
	g := ctx.Game
	landCounts := make(map[uuid.UUID]int)
	creatureCounts := make(map[uuid.UUID]int)
	for _, p := range g.FilterBattlefield(IsLand) {
		landCounts[p.ControllerID()]++
	}
	for _, p := range g.FilterBattlefield(IsCreature) {
		creatureCounts[p.ControllerID()]++
	}

	minLands := -1
	minCreatures := -1
	minHand := -1
	for _, p := range g.AllPlayers() {
		pid := p.PlayerID()
		if minLands < 0 || landCounts[pid] < minLands {
			minLands = landCounts[pid]
		}
		if minCreatures < 0 || creatureCounts[pid] < minCreatures {
			minCreatures = creatureCounts[pid]
		}
		if minHand < 0 || len(p.Hand()) < minHand {
			minHand = len(p.Hand())
		}
	}

	for _, p := range g.AllPlayers() {
		pid := p.PlayerID()
		toSac := landCounts[pid] - minLands
		for toSac > 0 {
			for _, perm := range g.FilterBattlefield(And(ControlledBy(pid), IsLand)) {
				if true {
					g.DoSacrifice(perm)
					toSac--
					break
				}
			}
		}
	}

	for _, p := range g.AllPlayers() {
		pid := p.PlayerID()
		toSac := creatureCounts[pid] - minCreatures
		for toSac > 0 {
			for _, perm := range g.FilterBattlefield(And(ControlledBy(pid), IsCreature)) {
				if true {
					g.DoSacrifice(perm)
					toSac--
					break
				}
			}
		}
	}

	for _, p := range g.AllPlayers() {
		for len(p.Hand()) > minHand {
			hand := p.Hand()
			if len(hand) == 0 {
				break
			}
			chosen := p.ChooseCardsFromHand(1, "discard for Balance", g)
			if len(chosen) > 0 {
				p.RemoveFromHand(chosen[0].ID())
				p.AddToGraveyard(chosen[0])
			} else {
				break
			}
		}
	}

	return nil
}

func (*chaosOrbEffect) Apply(ctx *EffectContext) error {
	var candidates []*Permanent
	for _, p := range ctx.Game.FilterBattlefield(Not(ControlledBy(ctx.Controller))) {
		if !p.IsToken {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) > 0 {
		chosen := candidates[rand.Intn(len(candidates))]
		ctx.Game.DestroyPermanent(chosen)
	}
	src := ctx.Game.FindPermanent(ctx.SourceID)
	if src != nil {
		ctx.Game.DestroyPermanent(src)
	}
	return nil
}
