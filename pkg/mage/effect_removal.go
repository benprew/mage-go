package mage

import (
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// destroyTargetEffect destroys the target permanent.
type destroyTargetEffect struct{}

// DestroyTarget creates an effect that destroys the first target permanent.
func DestroyTarget() Effect {
	return &destroyTargetEffect{}
}

func (e *destroyTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for destroy")
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil // target gone, fizzle
	}
	if perm.HasKeyword(Indestructible) {
		return nil
	}
	g.DestroyPermanent(perm)
	return nil
}

func (e *destroyTargetEffect) Text() string { return "destroy target" }
func (e *destroyTargetEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// destroyTargetPermanentEffect destroys a target permanent matching a filter.
type destroyTargetPermanentEffect struct {
	text string
}

// DestroyTargetPermanent creates an effect that destroys a target permanent.
func DestroyTargetPermanent() Effect {
	return &destroyTargetPermanentEffect{text: "destroy target permanent"}
}

// DestroyTargetLand creates an effect that destroys a target land (e.g. Stone Rain, Sinkhole).
func DestroyTargetLand() Effect {
	return &destroyTargetPermanentEffect{text: "destroy target land"}
}

// DestroyTargetArtifact creates an effect that destroys a target artifact (e.g. Shatter).
func DestroyTargetArtifact() Effect {
	return &destroyTargetPermanentEffect{text: "destroy target artifact"}
}

func (e *destroyTargetPermanentEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for destroy")
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	if perm.HasKeyword(Indestructible) {
		return nil
	}
	g.DestroyPermanent(perm)
	return nil
}

func (e *destroyTargetPermanentEffect) Text() string { return e.text }
func (e *destroyTargetPermanentEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// DestroyAllLands destroys all lands (Armageddon).
// This is a convenience alias for DestroyAllMatching with the IsLand filter.
func DestroyAllLands() Effect {
	return DestroyAllMatching(IsLand, "destroy all lands")
}

// DestroyAllEnchantments destroys all enchantments (Tranquility).
// This is a convenience alias for DestroyAllMatching with the IsEnchantment filter.
func DestroyAllEnchantments() Effect {
	return DestroyAllMatching(IsEnchantment, "destroy all enchantments")
}

// destroyAllMatchingEffect destroys all permanents matching a filter.
type destroyAllMatchingEffect struct {
	filter PermanentFilter
	text   string
}

// DestroyAllMatching creates an effect that destroys all permanents matching the filter.
// The text parameter is used for rules text display.
func DestroyAllMatching(filter PermanentFilter, text string) Effect {
	return &destroyAllMatchingEffect{filter: filter, text: text}
}

func (e *destroyAllMatchingEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	toDestroy := g.FilterBattlefield(And(e.filter, Not(HasKeywordFilter(Indestructible))))
	for _, p := range toDestroy {
		g.DestroyPermanent(p)
	}
	return nil
}

func (e *destroyAllMatchingEffect) Text() string { return e.text }
func (e *destroyAllMatchingEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}

// DestroyAllCreatures destroys all creatures (board wipe).
// This is a convenience alias for DestroyAllMatching with the IsCreature filter.
func DestroyAllCreatures() Effect {
	return DestroyAllMatching(IsCreature, "destroy all creatures")
}

// destroyAllMatchingNoRegenEffect destroys all matching permanents with "can't be regenerated".
type destroyAllMatchingNoRegenEffect struct {
	filter PermanentFilter
	text   string
}

// DestroyAllMatchingNoRegen creates an effect that destroys all permanents matching the filter.
// The destroyed permanents can't be regenerated (e.g. Shatterstorm).
func DestroyAllMatchingNoRegen(filter PermanentFilter, text string) Effect {
	return &destroyAllMatchingNoRegenEffect{filter: filter, text: text}
}

func (e *destroyAllMatchingNoRegenEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	toDestroy := g.FilterBattlefield(And(e.filter, Not(HasKeywordFilter(Indestructible))))
	for _, p := range toDestroy {
		p.GrantBaseAttr(CantRegenerate)
		g.DestroyPermanent(p)
	}
	return nil
}

func (e *destroyAllMatchingNoRegenEffect) Text() string { return e.text }
func (e *destroyAllMatchingNoRegenEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}

// destroyTargetNoRegenEffect destroys a target permanent, preventing regeneration.
type destroyTargetNoRegenEffect struct{}

// DestroyTargetNoRegen creates an effect that destroys the first target permanent.
// The destroyed permanent can't be regenerated (e.g. Terror, Tunnel).
func DestroyTargetNoRegen() Effect {
	return &destroyTargetNoRegenEffect{}
}

func (e *destroyTargetNoRegenEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for destroy")
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	if perm.HasKeyword(Indestructible) {
		return nil
	}
	perm.GrantBaseAttr(CantRegenerate)
	g.DestroyPermanent(perm)
	return nil
}

func (e *destroyTargetNoRegenEffect) Text() string { return "destroy target (can't be regenerated)" }
func (e *destroyTargetNoRegenEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// DestroyAllCreaturesNoRegen destroys all creatures; they can't be regenerated (e.g. Wrath of God).
func DestroyAllCreaturesNoRegen() Effect {
	return DestroyAllMatchingNoRegen(IsCreature, "destroy all creatures (can't be regenerated)")
}

// exileTargetEffect exiles a target permanent (removes from game).
type exileTargetEffect struct{}

// ExileTarget creates an effect that exiles a target permanent (e.g. Swords to Plowshares).
func ExileTarget() Effect {
	return &exileTargetEffect{}
}

func (e *exileTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for exile")
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	g.ExilePermanent(perm)
	return nil
}

func (e *exileTargetEffect) Text() string { return "exile target permanent" }
func (e *exileTargetEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// sacrificeSourceEffect sacrifices the source permanent.
type sacrificeSourceEffect struct{}

// SacrificeSource creates an effect that sacrifices the source permanent.
func SacrificeSource() Effect {
	return &sacrificeSourceEffect{}
}

func (e *sacrificeSourceEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	perm := g.FindPermanent(sourceID)
	if perm == nil {
		return nil
	}
	g.Sacrifice(perm)
	return nil
}

func (e *sacrificeSourceEffect) Text() string { return "sacrifice this permanent" }
func (e *sacrificeSourceEffect) Properties() EffectProperties { return EffectProperties{} }

// balanceEffect equalizes lands, creatures, and hand sizes.
type balanceEffect struct{}

// BalanceEffect creates an effect that equalizes lands, creatures, and hand sizes across all
// players by having each player sacrifice/discard down to the minimum (Balance).
func BalanceEffect() Effect { return &balanceEffect{} }

func (e *balanceEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	// Count lands for each player
	landCounts := make(map[uuid.UUID]int)
	creatureCounts := make(map[uuid.UUID]int)
	for _, p := range g.FilterBattlefield(IsLand) {
		landCounts[p.Controller]++
	}
	for _, p := range g.FilterBattlefield(IsCreature) {
		creatureCounts[p.Controller]++
	}

	// Find minimums
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

	// Sacrifice lands down to minimum
	for _, p := range g.AllPlayers() {
		pid := p.PlayerID()
		toSac := landCounts[pid] - minLands
		for toSac > 0 {
			for _, perm := range g.FilterBattlefield(And(ControlledBy(pid), IsLand)) {
				if true {
					g.Sacrifice(perm)
					toSac--
					break
				}
			}
		}
	}

	// Sacrifice creatures down to minimum
	for _, p := range g.AllPlayers() {
		pid := p.PlayerID()
		toSac := creatureCounts[pid] - minCreatures
		for toSac > 0 {
			for _, perm := range g.FilterBattlefield(And(ControlledBy(pid), IsCreature)) {
				if true {
					g.Sacrifice(perm)
					toSac--
					break
				}
			}
		}
	}

	// Discard down to minimum hand size
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
func (e *balanceEffect) Text() string {
	return "Each player sacrifices to match fewest lands, creatures; discards to match smallest hand"
}
func (e *balanceEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}

// destroyRandomNontokenPermanent destroys a random nontoken permanent an
// opponent controls, then destroys the source.
type chaosOrbEffect struct{}

// ChaosOrbEffect creates an effect that destroys a random nontoken permanent an opponent
// controls, then destroys the source (Chaos Orb).
func ChaosOrbEffect() Effect { return &chaosOrbEffect{} }

func (e *chaosOrbEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	var candidates []*Permanent
	for _, p := range g.FilterBattlefield(Not(ControlledBy(controller))) {
		if !p.Card.(*BaseCard).IsToken() {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) > 0 {
		chosen := candidates[rand.Intn(len(candidates))]
		g.DestroyPermanent(chosen)
	}
	// Destroy self
	src := g.FindPermanent(sourceID)
	if src != nil {
		g.DestroyPermanent(src)
	}
	return nil
}
func (e *chaosOrbEffect) Text() string { return "Destroy a random nontoken permanent, then destroy ~" }
func (e *chaosOrbEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
