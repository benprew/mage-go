package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// CaptureSacrificed records which permanent is about to be sacrificed as a
// cost. Sacrifice captures that permanent's generic LKI before it leaves the
// battlefield.
func (g *Game) CaptureSacrificed(p *Permanent) {
	if p == nil {
		return
	}
	g.resolution.SetLastSacrificedID(p.ID())
}

// LastSacrificed returns generic last-known information for the permanent most
// recently sacrificed as a cost.
func (g *Game) LastSacrificed() *PermanentLKI {
	if g.resolution.LastSacrificedID() == uuid.Nil {
		return nil
	}
	return g.LKI(g.resolution.LastSacrificedID())
}

// ClearSacrificed forgets which permanent was most recently sacrificed as a cost.
func (g *Game) ClearSacrificed() { g.resolution.SetLastSacrificedID(uuid.Nil) }

// Sacrifice sacrifices a permanent (like destroy but doesn't check
// indestructible). Self-referential triggers fire from the LKI snapshot's
// captured abilities (CR 700.4 / 603.6c: any battlefield → graveyard
// transition is "put into a graveyard," including sacrifice).
func (g *Game) DoSacrifice(perm *Permanent) {
	controller := perm.ControllerID()
	owner := perm.Card.Owner()
	if owner == uuid.Nil {
		owner = controller
	}

	isCreature := perm.HasType(TypeCreature)
	isToken := perm.IsToken
	permID := perm.ID()
	card := perm.Card

	g.RemoveFromBattlefield(perm)
	selfTriggers := g.LKIAbilities(permID)

	if !isToken {
		p := g.GetPlayer(owner)
		if p != nil {
			p.AddToGraveyard(card)
		}
	}

	zoneEvt := GameEvent{
		Type:     EvtZoneChange,
		SourceID: permID,
		PlayerID: controller,
		Flag:     true, // sacrifice path (vs. destroy/SBA)
		FromZone: ZoneBattlefield,
		ToZone:   ZoneGraveyard,
	}
	g.FireEvent(zoneEvt)
	g.checkAbilitiesForEvent(selfTriggers, &zoneEvt, permID, controller)

	sacEvt := GameEvent{
		Type:     EvtSacrifice,
		SourceID: permID,
		PlayerID: controller,
		Flag:     isCreature, // Flag=true means the sacrificed permanent was a creature
		FromZone: ZoneBattlefield,
		ToZone:   ZoneGraveyard,
	}
	g.FireEvent(sacEvt)
	g.checkAbilitiesForEvent(selfTriggers, &sacEvt, permID, controller)

	if isCreature {
		g.creatureDeathsThisTurn++
		g.recordCreatureDeath(controller)
	}
}

// sacrificePermanents sacrifices up to count permanents the player controls.
func (g *Game) sacrificePermanents(playerID uuid.UUID, count int) {
	for range count {
		var target *Permanent
		for _, p := range g.battlefield {
			if p.ControllerID() == playerID {
				target = p
				break
			}
		}
		if target == nil {
			return
		}
		g.DoSacrifice(target)
	}
}

type sacrificeSourceCost struct{}

// SacrificeSourceCost creates a cost that sacrifices the source permanent.
func SacrificeSourceCost() Cost { return &sacrificeSourceCost{} }

func (*sacrificeSourceCost) CanPay(sourceID, _ uuid.UUID, g *Game) bool {
	return g.FindPermanent(sourceID) != nil
}

func (*sacrificeSourceCost) Pay(sourceID, _ uuid.UUID, g *Game) error {
	p := g.FindPermanent(sourceID)
	if p == nil {
		return ErrSourceNotFound
	}
	g.CaptureSacrificed(p)
	g.DoSacrifice(p)
	return nil
}

func (*sacrificeSourceCost) Text() string { return "Sacrifice ~" }

type sacrificeMatchingCost struct {
	filter        PermanentFilter
	text          string
	count         int
	includeSource bool
}

// SacrificeMatchingCost creates a cost that sacrifices another permanent the
// controller controls which matches the filter.
func SacrificeMatchingCost(filter PermanentFilter, text string) Cost {
	return &sacrificeMatchingCost{filter: filter, text: text, count: 1}
}

// SacrificeMatchingIncludingSourceCost creates a cost that sacrifices a
// matching permanent the controller controls, including the ability's source
// when it matches.
func SacrificeMatchingIncludingSourceCost(filter PermanentFilter, text string) Cost {
	return &sacrificeMatchingCost{filter: filter, text: text, count: 1, includeSource: true}
}

// SacrificeNMatchingCost creates a cost that sacrifices n other matching
// permanents the controller controls.
func SacrificeNMatchingCost(n int, filter PermanentFilter, text string) Cost {
	return &sacrificeMatchingCost{filter: filter, text: text, count: n}
}

// SacrificeArtifactCost creates a cost that sacrifices another artifact.
func SacrificeArtifactCost() Cost {
	return SacrificeMatchingCost(IsArtifact, "Sacrifice an artifact")
}

// SacrificeCreatureCost creates a cost that sacrifices another creature.
func SacrificeCreatureCost() Cost {
	return SacrificeMatchingCost(IsCreature, "Sacrifice a creature")
}

func (c *sacrificeMatchingCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	found := 0
	for _, p := range g.battlefield {
		if p.ControllerID() == controller && (c.includeSource || p.ID() != sourceID) && c.filter.Match(p, g) {
			found++
			if found >= c.count {
				return true
			}
		}
	}
	return false
}

func (c *sacrificeMatchingCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	if !c.CanPay(sourceID, controller, g) {
		return fmt.Errorf("not enough permanents to sacrifice")
	}
	player := g.GetPlayer(controller)
	for i := 0; i < c.count; i++ {
		var candidates []*Permanent
		for _, p := range g.battlefield {
			if p.ControllerID() == controller && (c.includeSource || p.ID() != sourceID) && c.filter.Match(p, g) {
				candidates = append(candidates, p)
			}
		}
		if len(candidates) == 0 {
			return fmt.Errorf("no permanent to sacrifice")
		}
		chosen := player.ChoosePermanent(candidates, c.text, g)
		if chosen == nil {
			return fmt.Errorf("no permanent chosen")
		}
		g.CaptureSacrificed(chosen)
		g.DoSacrifice(chosen)
	}
	return nil
}

func (c *sacrificeMatchingCost) Text() string { return c.text }

type sacrificeSourceEffect struct{}

// SacrificeSource creates an effect that sacrifices the source permanent.
func SacrificeSource() Effect { return &sacrificeSourceEffect{} }

func (*sacrificeSourceEffect) Text() string                 { return "sacrifice this permanent" }
func (*sacrificeSourceEffect) Properties() EffectProperties { return EffectProperties{} }
func (*sacrificeSourceEffect) Apply(ctx *EffectContext) error {
	if perm := ctx.Game.FindPermanent(ctx.SourceID); perm != nil {
		ctx.Game.DoSacrifice(perm)
	}
	return nil
}

type sacrificeTargetEffect struct{}

// SacrificeTarget creates an effect that sacrifices the first target permanent.
func SacrificeTarget() Effect { return &sacrificeTargetEffect{} }

// SacrificeTargetStep returns the target-sacrifice effect for a pipeline.
func SacrificeTargetStep() Effect { return &sacrificeTargetEffect{} }

func (*sacrificeTargetEffect) Text() string                 { return "sacrifice target permanent" }
func (*sacrificeTargetEffect) Properties() EffectProperties { return EffectProperties{} }
func (*sacrificeTargetEffect) Apply(ctx *EffectContext) error {
	if len(ctx.Targets) > 0 {
		if perm := ctx.Game.FindPermanent(ctx.Targets[0]); perm != nil {
			ctx.Game.DoSacrifice(perm)
		}
	}
	return nil
}

// SacrificeGatheredData sacrifices the permanent stored in a context variable.
type SacrificeGatheredData struct{ VarName string }

func SacrificeGathered(varName string) Effect               { return &SacrificeGatheredData{VarName: varName} }
func (*SacrificeGatheredData) Text() string                 { return "sacrifice" }
func (*SacrificeGatheredData) Properties() EffectProperties { return EffectProperties{} }
func (e *SacrificeGatheredData) Apply(ctx *EffectContext) error {
	if id := ctx.TryGetUUID(e.VarName); id != uuid.Nil {
		if perm := ctx.Game.FindPermanent(id); perm != nil {
			ctx.Game.DoSacrifice(perm)
		}
	}
	return nil
}

// SacrificeSourceData sacrifices the source permanent in a pipeline.
type SacrificeSourceData struct{}

func SacrificeSourceStep() Effect                         { return &SacrificeSourceData{} }
func (*SacrificeSourceData) Text() string                 { return "sacrifice" }
func (*SacrificeSourceData) Properties() EffectProperties { return EffectProperties{} }
func (*SacrificeSourceData) Apply(ctx *EffectContext) error {
	if perm := ctx.Game.FindPermanent(ctx.SourceID); perm != nil {
		ctx.Game.DoSacrifice(perm)
	}
	return nil
}
