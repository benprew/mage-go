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
	return DataEffect(&destroyTargetEffect{})
}

// DestroyTargetStep returns the EffectData for use as a pipeline/ForEach inner step.
func DestroyTargetStep() EffectData { return &destroyTargetEffect{} }

func (e *destroyTargetEffect) EffectText() string { return "destroy target" }
func (e *destroyTargetEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// destroyTargetPermanentEffect destroys a target permanent matching a filter.
type destroyTargetPermanentEffect struct {
	text string
}

// DestroyTargetPermanent creates an effect that destroys a target permanent.
func DestroyTargetPermanent() Effect {
	return DataEffect(&destroyTargetPermanentEffect{text: "destroy target permanent"})
}

// DestroyTargetLand creates an effect that destroys a target land (e.g. Stone Rain, Sinkhole).
func DestroyTargetLand() Effect {
	return DataEffect(&destroyTargetPermanentEffect{text: "destroy target land"})
}

// DestroyTargetArtifact creates an effect that destroys a target artifact (e.g. Shatter).
func DestroyTargetArtifact() Effect {
	return DataEffect(&destroyTargetPermanentEffect{text: "destroy target artifact"})
}

func (e *destroyTargetPermanentEffect) EffectText() string { return e.text }
func (e *destroyTargetPermanentEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
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
type destroyAllMatchingEffect struct {
	filter PermanentFilter
	text   string
}

// DestroyAllMatching creates an effect that destroys all permanents matching the filter.
func DestroyAllMatching(filter PermanentFilter, text string) Effect {
	return DataEffect(&destroyAllMatchingEffect{filter: filter, text: text})
}

func (e *destroyAllMatchingEffect) EffectText() string { return e.text }
func (e *destroyAllMatchingEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}

// DestroyAllCreatures destroys all creatures (board wipe).
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
	return DataEffect(&destroyAllMatchingNoRegenEffect{filter: filter, text: text})
}

func (e *destroyAllMatchingNoRegenEffect) EffectText() string { return e.text }
func (e *destroyAllMatchingNoRegenEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}

// destroyTargetNoRegenEffect destroys a target permanent, preventing regeneration.
type destroyTargetNoRegenEffect struct{}

// DestroyTargetNoRegen creates an effect that destroys the first target permanent.
// The destroyed permanent can't be regenerated (e.g. Terror, Tunnel).
func DestroyTargetNoRegen() Effect {
	return DataEffect(&destroyTargetNoRegenEffect{})
}

// DestroyTargetNoRegenStep returns the EffectData for use in pipelines/ForEach.
func DestroyTargetNoRegenStep() EffectData { return &destroyTargetNoRegenEffect{} }

// DestroyAllCreaturesNoRegen destroys all creatures; they can't be regenerated (e.g. Wrath of God).
func DestroyAllCreaturesNoRegen() Effect {
	return DestroyAllMatchingNoRegen(IsCreature, "destroy all creatures (can't be regenerated)")
}

func (e *destroyTargetNoRegenEffect) EffectText() string {
	return "destroy target (can't be regenerated)"
}
func (e *destroyTargetNoRegenEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// exileTargetEffect exiles a target permanent (removes from game).
type exileTargetEffect struct{}

// ExileTarget creates an effect that exiles a target permanent.
func ExileTarget() Effect {
	return DataEffect(&exileTargetEffect{})
}

// ExileTargetStep returns the EffectData for use as a pipeline/ForEach inner step.
func ExileTargetStep() EffectData { return &exileTargetEffect{} }

func (e *exileTargetEffect) EffectText() string { return "exile target permanent" }
func (e *exileTargetEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// sacrificeSourceEffect sacrifices the source permanent.
type sacrificeSourceEffect struct{}

// SacrificeSource creates an effect that sacrifices the source permanent.
func SacrificeSource() Effect {
	return DataEffect(&sacrificeSourceEffect{})
}

func (e *sacrificeSourceEffect) EffectText() string        { return "sacrifice this permanent" }
func (e *sacrificeSourceEffect) EffectProps() EffectProperties { return EffectProperties{} }

// sacrificeTargetEffect sacrifices the target permanent (targets[0]).
type sacrificeTargetEffect struct{}

// SacrificeTarget creates an effect that sacrifices the first target permanent.
func SacrificeTarget() Effect {
	return DataEffect(&sacrificeTargetEffect{})
}

// SacrificeTargetStep returns the EffectData for use in pipelines/ForEach.
func SacrificeTargetStep() EffectData { return &sacrificeTargetEffect{} }

func (e *sacrificeTargetEffect) EffectText() string        { return "sacrifice target permanent" }
func (e *sacrificeTargetEffect) EffectProps() EffectProperties { return EffectProperties{} }

// balanceEffect equalizes lands, creatures, and hand sizes.
type balanceEffect struct{}

// BalanceEffect creates an effect that equalizes lands, creatures, and hand sizes across all
// players by having each player sacrifice/discard down to the minimum (Balance).
func BalanceEffect() Effect { return DataEffect(&balanceEffect{}) }

func (e *balanceEffect) EffectText() string {
	return "Each player sacrifices to match fewest lands, creatures; discards to match smallest hand"
}
func (e *balanceEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}

// chaosOrbEffect destroys a random nontoken permanent an opponent controls, then destroys self.
type chaosOrbEffect struct{}

// ChaosOrbEffect creates the Chaos Orb effect.
func ChaosOrbEffect() Effect { return DataEffect(&chaosOrbEffect{}) }

func (e *chaosOrbEffect) EffectText() string {
	return "Destroy a random nontoken permanent, then destroy ~"
}
func (e *chaosOrbEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// --- Executor functions (called from executor.go) ---

func execSacrificeTarget(ctx *EffectContext, _ *sacrificeTargetEffect) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	ctx.Game.Sacrifice(perm)
	return nil
}

func execDestroyTarget(ctx *EffectContext, _ *destroyTargetEffect) error {
	if len(ctx.Targets) == 0 {
		return fmt.Errorf("no target for destroy")
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	if perm.HasKeyword(Indestructible) {
		return nil
	}
	ctx.Game.DestroyPermanent(perm)
	return nil
}

func execDestroyTargetPermanent(ctx *EffectContext, _ *destroyTargetPermanentEffect) error {
	if len(ctx.Targets) == 0 {
		return fmt.Errorf("no target for destroy")
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	if perm.HasKeyword(Indestructible) {
		return nil
	}
	ctx.Game.DestroyPermanent(perm)
	return nil
}

func execDestroyAllMatching(ctx *EffectContext, e *destroyAllMatchingEffect) error {
	toDestroy := ctx.Game.FilterBattlefield(And(e.filter, Not(HasKeywordFilter(Indestructible))))
	for _, p := range toDestroy {
		ctx.Game.DestroyPermanent(p)
	}
	return nil
}

func execDestroyAllMatchingNoRegen(ctx *EffectContext, e *destroyAllMatchingNoRegenEffect) error {
	toDestroy := ctx.Game.FilterBattlefield(And(e.filter, Not(HasKeywordFilter(Indestructible))))
	for _, p := range toDestroy {
		p.GrantBaseAttr(CantRegenerate)
		ctx.Game.DestroyPermanent(p)
	}
	return nil
}

func execDestroyTargetNoRegen(ctx *EffectContext, _ *destroyTargetNoRegenEffect) error {
	if len(ctx.Targets) == 0 {
		return fmt.Errorf("no target for destroy")
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	if perm.HasKeyword(Indestructible) {
		return nil
	}
	perm.GrantBaseAttr(CantRegenerate)
	ctx.Game.DestroyPermanent(perm)
	return nil
}

func execExileTarget(ctx *EffectContext, _ *exileTargetEffect) error {
	if len(ctx.Targets) == 0 {
		return fmt.Errorf("no target for exile")
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	ctx.Game.ExilePermanent(perm)
	return nil
}

func execSacrificeSource(ctx *EffectContext, _ *sacrificeSourceEffect) error {
	perm := ctx.Game.FindPermanent(ctx.SourceID)
	if perm == nil {
		return nil
	}
	ctx.Game.Sacrifice(perm)
	return nil
}

func execBalance(ctx *EffectContext, _ *balanceEffect) error {
	g := ctx.Game
	landCounts := make(map[uuid.UUID]int)
	creatureCounts := make(map[uuid.UUID]int)
	for _, p := range g.FilterBattlefield(IsLand) {
		landCounts[p.Controller]++
	}
	for _, p := range g.FilterBattlefield(IsCreature) {
		creatureCounts[p.Controller]++
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
					g.Sacrifice(perm)
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
					g.Sacrifice(perm)
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

func execChaosOrb(ctx *EffectContext, _ *chaosOrbEffect) error {
	var candidates []*Permanent
	for _, p := range ctx.Game.FilterBattlefield(Not(ControlledBy(ctx.Controller))) {
		if !p.Card.(*BaseCard).IsToken() {
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
