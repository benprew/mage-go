package mage

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type forcedManaEffect struct {
	player PlayerRef
	filter PermanentFilter
}

// ActivateManaAbilityOfEach requires one mana activation from each selected visit.
// The player chooses the activation order and all choices in each ability.
func ActivateManaAbilityOfEach(player PlayerRef, filter PermanentFilter) ResolvingEffect {
	return &forcedManaEffect{player: player, filter: filter}
}

func (*forcedManaEffect) Text() string                 { return "activate a mana ability of each matching permanent" }
func (*forcedManaEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *forcedManaEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}
func (e *forcedManaEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "forced-mana-activation", Text: e.Text(), children: []definitionEdge{{"player", e.player}}}
	if nilDefinition(e.player) || e.filter.IsZero() {
		n.problem = "supply a player and permanent filter"
	}
	return n
}

type forcedManaOption struct {
	visit   capturedPermanent
	ability uuid.UUID
	index   int
}

func (o forcedManaOption) current(g *Game, player uuid.UUID) *Permanent {
	p := g.FindPermanent(o.visit.id)
	if p == nil || p.incarnationID != o.visit.incarnation || p.ControllerID() != player || o.index >= len(p.RuntimeAbilities) || p.RuntimeAbilities[o.index].AbilityID() != o.ability {
		return nil
	}
	return p
}

func (e *forcedManaEffect) Apply(ctx *EffectContext) error {
	g := ctx.Game
	id, ok := e.player.resolvePlayer(ctx)
	if !ok || g.GetPlayer(id) == nil {
		return nil
	}
	player := g.GetPlayer(id)
	var visits []capturedPermanent
	for _, p := range g.FilterBattlefield(And(ControlledBy(id), e.filter)) {
		visits = append(visits, capturedPermanent{p.ID(), p.incarnationID})
	}
	done := make(map[capturedPermanent]bool)
	failed := make(map[forcedManaOption]bool)
	previousX := g.resolution.X()
	defer g.resolution.SetX(previousX)
	for {
		g.resolution.SetX(0)
		var options []forcedManaOption
		var labels []string
		for _, visit := range visits {
			p := g.FindPermanent(visit.id)
			if done[visit] || p == nil || p.incarnationID != visit.incarnation || p.ControllerID() != id {
				continue
			}
			for index, ability := range p.RuntimeAbilities {
				option := forcedManaOption{visit, ability.AbilityID(), index}
				if failed[option] || !g.canForceManaAbility(id, p, index) {
					continue
				}
				options = append(options, option)
				labels = append(labels, forcedManaLabel(g, p, ability))
			}
		}
		if len(options) == 0 {
			return nil
		}
		choice := 0
		if len(options) > 1 {
			choice = player.ChooseMode(labels, e.Text())
			if choice < 0 || choice >= len(options) {
				choice = 0
			}
		}
		option := options[choice]
		p := option.current(g, id)
		if p == nil || !g.canForceManaAbility(id, p, option.index) {
			failed[option] = true
			continue
		}
		chosenX := false
		if activated, ok := UnwrapAbility(p.RuntimeAbilities[option.index]).(*SimpleActivatedAbility); ok && activated.activationManaX == nil {
			if slices.ContainsFunc(activated.Costs(), forcedManaChoosesX) {
				chosenX = true
				x := chooseNonnegativeNumber(player, "choose X for "+p.Name())
				g.resolution.SetX(max(0, x))
			}
		}
		if option.current(g, id) == nil {
			failed[option] = true
			continue
		}
		err := g.activateForcedManaAbility(id, p.ID(), option.index)
		if err != nil && chosenX && g.resolution.X() > 0 && option.current(g, id) != nil {
			g.resolution.SetX(0)
			err = g.activateForcedManaAbility(id, p.ID(), option.index)
		}
		if err != nil {
			failed[option] = true
			continue
		}
		done[option.visit] = true
		clear(failed)
	}
}

func forcedManaChoosesX(cost Cost) bool {
	switch c := cost.(type) {
	case *ManaCostPayment:
		return c.MC.HasX
	case *xManaCost, *removeXCountersCost, *capturedXCountersCost, *xLifePayCost:
		return true
	case *optionalCost:
		return forcedManaChoosesX(c.inner)
	case *eitherCost:
		if slices.ContainsFunc(c.options, forcedManaChoosesX) {
			return true
		}
	}
	return false
}

func (g *Game) canForceManaAbility(player uuid.UUID, p *Permanent, index int) bool {
	if p.ControllerID() != player || p.HasAttr(AttrCantActivate) || !IsActivatedManaAbility(p.RuntimeAbilities[index]) {
		return false
	}
	if _, ok := UnwrapAbility(p.RuntimeAbilities[index]).(*ManaAbility); ok {
		if p.Tapped || !p.CanTapForEffect(g) {
			return false
		}
		_, err := g.mana.PlanManaForCost(g, player, g.modifyActivationManaCost(p.ID(), ManaCost{}), AutoTapHint{ActivationSource: p.ID(), ActivationTapsSource: true}, nil)
		return err == nil
	}
	return UnwrapAbility(p.RuntimeAbilities[index]).(ActivatedAbility).CanActivate(player, g)
}

func forcedManaLabel(g *Game, p *Permanent, ability Ability) string {
	if mana, ok := UnwrapAbility(ability).(*ManaAbility); ok {
		var parts []string
		for _, production := range mana.currentProductions(g, p.ID()) {
			part := fmt.Sprintf("add %d %s mana", max(1, production.Amount), production.Color)
			if production.Restriction != nil {
				part += "; " + production.Restriction.Description()
			}
			parts = append(parts, part)
		}
		if len(parts) == 0 {
			parts = append(parts, "add no mana")
		}
		return p.Name() + ": {T}: " + strings.Join(parts, ", ")
	}
	activated := UnwrapAbility(ability).(ActivatedAbility)
	var parts []string
	for _, cost := range activated.Costs() {
		parts = append(parts, cost.Text())
	}
	label := p.Name() + ": " + strings.Join(parts, ", ") + ": "
	parts = nil
	for _, effect := range activated.Effects() {
		parts = append(parts, effect.Text())
	}
	return label + strings.Join(parts, ", ")
}

// Immediate mana resolution must not clear the calling instruction's bindings.
func (g *Game) activateForcedManaAbility(player, permanent uuid.UUID, index int) error {
	outer := g.resolution.Clone()
	defer func() { g.resolution = outer }()
	return g.ActivateAbilityByIndex(player, permanent, index, nil)
}
