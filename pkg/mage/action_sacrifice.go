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
	g.performSacrifice(g.prepareSacrifice(perm))
}

func (g *Game) prepareSacrifice(perm *Permanent) *SacrificePermanentAction {
	if perm == nil || perm.PhasedOut {
		return nil
	}
	current := g.FindPermanent(perm.ID())
	if current == nil || current.incarnationID != perm.incarnationID {
		return nil
	}
	action := &SacrificePermanentAction{permanentID: perm.ID(), incarnation: perm.incarnationID, destination: ZoneGraveyard}
	result, ok := g.effects.ApplyReplacements(action, g).(*SacrificePermanentAction)
	if !ok {
		return nil
	}
	return result
}

func (g *Game) performSacrifice(result *SacrificePermanentAction) {
	if result == nil {
		return
	}
	perm := g.FindPermanent(result.permanentID)
	if perm == nil || perm.PhasedOut || perm.incarnationID != result.incarnation {
		return
	}
	controller := perm.ControllerID()
	owner := perm.Card.Owner()
	if owner == uuid.Nil {
		owner = controller
	}

	isCreature := perm.HasType(TypeCreature)
	isToken := perm.IsToken
	permID := perm.ID()
	incarnation := perm.incarnationID
	card := perm.Card

	g.RemoveFromBattlefield(perm)
	selfTriggers := g.LKIAbilities(permID)

	if result.destination == ZoneExile {
		g.ExileCard(card, result.exileSource)
	} else if !isToken {
		if p := g.GetPlayer(owner); p != nil {
			p.AddToGraveyard(card)
		}
	}

	if isCreature && result.destination == ZoneGraveyard {
		g.trackers.Turn.RecordCreatureDeath()
		g.recordCreatureDeath(controller)
	}

	zoneEvt := GameEvent{
		Type:     EvtZoneChange,
		SourceID: permID,
		PlayerID: controller,
		Flag:     true, // sacrifice path (vs. destroy/SBA)
		FromZone: ZoneBattlefield,
		ToZone:   result.destination,
	}
	g.FireEvent(zoneEvt)
	g.checkAbilitiesForEvent(selfTriggers, &zoneEvt, permID, controller)

	sacEvt := GameEvent{
		Type:              EvtSacrifice,
		SourceID:          permID,
		SourceIncarnation: incarnation,
		PlayerID:          controller,
		Flag:              isCreature, // Flag=true means the sacrificed permanent was a creature
		FromZone:          ZoneBattlefield,
		ToZone:            result.destination,
	}
	g.FireEvent(sacEvt)
	g.checkAbilitiesForEvent(selfTriggers, &sacEvt, permID, controller)
}

// sacrificePermanents sacrifices up to count permanents the player controls.
func (g *Game) sacrificePermanents(playerID uuid.UUID, count int) {
	for range count {
		var target *Permanent
		for _, p := range g.zones.battlefield {
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
	filter            PermanentFilter
	text              string
	count             int
	includeSource     bool
	locked            bool
	selected          []capturedPermanent
	chooser           Player
	paidPower         *NumberBinding
	paidToughness     *NumberBinding
	paidManaValue     *NumberBinding
	paidSubtype       *NumberBinding
	subtype           string
	paidCard          *CardBinding
	paidPermanent     *PermanentBinding
	paidTotalPower    *NumberBinding
	captureStats      bool
	captureManaValue  bool
	captureSubtype    bool
	captureCard       bool
	capturePermanent  bool
	captureTotalPower bool
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

// SacrificeNMatchingIncludingSourceCost permits the source to pay when it matches.
func SacrificeNMatchingIncludingSourceCost(n int, filter PermanentFilter, text string) Cost {
	return &sacrificeMatchingCost{filter: filter, text: text, count: n, includeSource: true}
}

// SacrificeNMatchingCostWithTotalPower saves the chosen permanents' power at payment.
func SacrificeNMatchingCostWithTotalPower(n int, filter PermanentFilter, text string, total *NumberBinding) Cost {
	return &sacrificeMatchingCost{filter: filter, text: text, count: n, paidTotalPower: total, captureTotalPower: true}
}

// SacrificeArtifactCost creates a cost that sacrifices another artifact.
func SacrificeArtifactCost() Cost {
	return SacrificeMatchingCost(IsArtifact, "Sacrifice an artifact")
}

// SacrificeCreatureCost creates a cost that sacrifices another creature.
func SacrificeCreatureCost() Cost {
	return SacrificeMatchingCost(IsCreature, "Sacrifice a creature")
}

// SacrificeCreatureCostWithStats saves the chosen creature's power and
// toughness during payment. The values belong to this stack object.
func SacrificeCreatureCostWithStats(power, toughness *NumberBinding) Cost {
	return &sacrificeMatchingCost{
		filter: IsCreature, text: "Sacrifice a creature", count: 1,
		paidPower: power, paidToughness: toughness, captureStats: true,
	}
}

// SacrificeCreatureIncludingSourceCostWithStats lets a creature source pay its own cost.
func SacrificeCreatureIncludingSourceCostWithStats(power, toughness *NumberBinding) Cost {
	return &sacrificeMatchingCost{
		filter: IsCreature, text: "Sacrifice a creature", count: 1, includeSource: true,
		paidPower: power, paidToughness: toughness, captureStats: true,
	}
}

// SacrificeCreatureCostWithSubtype saves whether the chosen creature had the subtype during payment.
func SacrificeCreatureCostWithSubtype(subtype string, result *NumberBinding) Cost {
	return &sacrificeMatchingCost{
		filter: IsCreature, text: "Sacrifice a creature", count: 1, includeSource: true,
		subtype: subtype, paidSubtype: result, captureSubtype: true,
	}
}

// SacrificeMatchingCostWithManaValue saves the chosen permanent's mana value
// when its controller pays this cost. The source is not a candidate.
func SacrificeMatchingCostWithManaValue(filter PermanentFilter, text string, value *NumberBinding) Cost {
	return &sacrificeMatchingCost{filter: filter, text: text, count: 1, paidManaValue: value, captureManaValue: true}
}

// SacrificeMatchingIncludingSourceCostWithManaValue lets the source pay when it matches.
// The chosen permanent's mana value belongs to this payment instance.
func SacrificeMatchingIncludingSourceCostWithManaValue(filter PermanentFilter, text string, value *NumberBinding) Cost {
	return &sacrificeMatchingCost{filter: filter, text: text, count: 1, includeSource: true, paidManaValue: value, captureManaValue: true}
}

// SacrificeMatchingCostWithCard saves the sacrificed card in its new graveyard visit.
func SacrificeMatchingCostWithCard(filter PermanentFilter, text string, card *CardBinding) Cost {
	return &sacrificeMatchingCost{filter: filter, text: text, count: 1, paidCard: card, captureCard: true}
}

// SacrificeMatchingCostWithPermanent saves the chosen battlefield incarnation.
func SacrificeMatchingCostWithPermanent(filter PermanentFilter, text string, permanent *PermanentBinding) Cost {
	return &sacrificeMatchingCost{filter: filter, text: text, count: 1, paidPermanent: permanent, capturePermanent: true}
}

// SacrificeMatchingIncludingSourceCostWithPermanent saves the paid incarnation.
func SacrificeMatchingIncludingSourceCostWithPermanent(filter PermanentFilter, text string, permanent *PermanentBinding) Cost {
	return &sacrificeMatchingCost{filter: filter, text: text, count: 1, includeSource: true, paidPermanent: permanent, capturePermanent: true}
}

func (c *sacrificeMatchingCost) inspectDefinition() definitionNode {
	n := definitionNode{Role: "sacrifice-cost", Text: c.text}
	if c.count <= 0 {
		n.problem = "supply a positive sacrifice count"
		return n
	}
	if !c.captureStats && !c.captureManaValue && !c.captureSubtype && !c.captureCard && !c.capturePermanent && !c.captureTotalPower {
		return n
	}
	if c.capturePermanent {
		if c.text == "" {
			n.problem = "supply sacrifice text"
		} else if c.paidPermanent == nil {
			n.problem = "supply a non-nil paid permanent binding"
		}
		n.bindingWrites = []savedBinding{c.paidPermanent}
		n.children = []definitionEdge{{"permanent", c.paidPermanent}}
		return n
	}
	if c.captureTotalPower {
		if c.count <= 0 {
			n.problem = "supply a positive sacrifice count"
		} else if c.text == "" {
			n.problem = "supply sacrifice text"
		} else if c.paidTotalPower == nil {
			n.problem = "supply a non-nil paid total power binding"
		}
		n.bindingWrites = []savedBinding{c.paidTotalPower}
		n.children = []definitionEdge{{"total power", c.paidTotalPower}}
		return n
	}
	if c.captureSubtype {
		if c.subtype == "" {
			n.problem = "supply a creature subtype"
		} else if family, known := knownSubtypeFamily(c.subtype); known && family != SubtypeCreature {
			n.problem = "subtype is not a creature subtype"
		} else if c.paidSubtype == nil {
			n.problem = "supply a non-nil paid subtype binding"
		}
		n.bindingWrites = []savedBinding{c.paidSubtype}
		n.children = []definitionEdge{{"subtype result", c.paidSubtype}}
		return n
	}
	if c.captureCard {
		if c.text == "" {
			n.problem = "supply sacrifice text"
		} else if c.paidCard == nil {
			n.problem = "supply a non-nil paid card binding"
		}
		n.bindingWrites = []savedBinding{c.paidCard}
		n.children = []definitionEdge{{"card", c.paidCard}}
		return n
	}
	if c.captureManaValue {
		if c.text == "" {
			n.problem = "supply sacrifice text"
		} else if c.paidManaValue == nil {
			n.problem = "supply a non-nil paid mana value binding"
		}
		n.bindingWrites = []savedBinding{c.paidManaValue}
		n.children = []definitionEdge{{"mana value", c.paidManaValue}}
		return n
	}
	if c.paidPower == nil || c.paidToughness == nil {
		n.problem = "supply non-nil paid power and toughness bindings"
	} else if c.paidPower == c.paidToughness {
		n.problem = "duplicate paid sacrifice binding"
	}
	n.bindingWrites = []savedBinding{c.paidPower, c.paidToughness}
	n.children = []definitionEdge{{"power", c.paidPower}, {"toughness", c.paidToughness}}
	return n
}

func (c *sacrificeMatchingCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	if c.count <= 0 {
		return false
	}
	if c.locked {
		if len(c.selected) != c.count {
			return false
		}
		for _, choice := range c.selected {
			p := g.FindPermanent(choice.id)
			if p == nil || p.PhasedOut || p.incarnationID != choice.incarnation || p.ControllerID() != controller || (!c.includeSource && p.ID() == sourceID) || !c.filter.Match(p, g) {
				return false
			}
		}
		return true
	}
	found := 0
	for _, p := range g.zones.battlefield {
		if !p.PhasedOut && p.ControllerID() == controller && (c.includeSource || p.ID() != sourceID) && c.filter.Match(p, g) {
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
	selected := c.selected
	if !c.locked {
		selected = make([]capturedPermanent, 0, c.count)
		player := g.GetPlayer(controller)
		if c.chooser != nil {
			player = c.chooser
		}
		for range c.count {
			var candidates []*Permanent
			for _, p := range g.zones.battlefield {
				if p.PhasedOut || p.ControllerID() != controller || (!c.includeSource && p.ID() == sourceID) || !c.filter.Match(p, g) {
					continue
				}
				used := false
				for _, choice := range selected {
					if choice.id == p.ID() {
						used = true
						break
					}
				}
				if !used {
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
			valid := false
			for _, candidate := range candidates {
				if chosen.ID() == candidate.ID() && chosen.incarnationID == candidate.incarnationID {
					selected = append(selected, capturedPermanent{candidate.ID(), candidate.incarnationID})
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid permanent chosen for sacrifice")
			}
		}
	}
	// All choices must remain legal before any permanent leaves.
	locked := *c
	locked.locked, locked.selected = true, selected
	if !locked.CanPay(sourceID, controller, g) {
		return fmt.Errorf("cannot pay chosen sacrifice")
	}
	if c.selected != nil {
		c.selected = selected
	}
	g.withZoneChangeBatch(func() {
		actions := make([]*SacrificePermanentAction, len(selected))
		for i, choice := range selected {
			actions[i] = g.prepareSacrifice(g.FindPermanent(choice.id))
		}
		for i, choice := range selected {
			g.CaptureSacrificed(g.FindPermanent(choice.id))
			g.performSacrifice(actions[i])
		}
	})
	return nil
}

func (c *sacrificeMatchingCost) Text() string { return c.text }

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
