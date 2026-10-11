package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// SpellFilter keeps immutable spell predicates for targets and cast events.
// Its zero value is invalid so a missing declaration cannot match all spells.
type SpellFilter struct {
	kind          string
	card          CardFilter
	permanents    PermanentFilter
	controlled    bool
	excludeSource bool
	children      []SpellFilter
}

// SpellCardMatches uses current spell characteristics with an existing card filter.
func SpellCardMatches(filter CardFilter) SpellFilter { return SpellFilter{kind: "card", card: filter} }

// SpellControlledByYou compares current spell control with the action controller.
func SpellControlledByYou() SpellFilter { return SpellFilter{kind: "controller"} }

// SpellSharesColorWithPermanents reads current, present battlefield members.
// Colorless objects cannot share a color. Controller selection uses the action.
func SpellSharesColorWithPermanents(filter PermanentFilter, controlled bool) SpellFilter {
	return SpellFilter{kind: "shares-color", permanents: filter, controlled: controlled}
}

// AllSpellFilters requires every operand and retains a separate operand list.
func AllSpellFilters(filters ...SpellFilter) SpellFilter {
	return SpellFilter{kind: "all", children: slices.Clone(filters)}
}

// AnySpellFilter requires one matching operand and retains a separate operand list.
func AnySpellFilter(filters ...SpellFilter) SpellFilter {
	return SpellFilter{kind: "any", children: slices.Clone(filters)}
}

// NotSpellFilter excludes spells that match its operand.
func NotSpellFilter(filter SpellFilter) SpellFilter {
	return SpellFilter{kind: "not", children: []SpellFilter{filter}}
}

func (f SpellFilter) matches(g *Game, controller uuid.UUID, source Card, spell *StackObject) bool {
	if spell == nil || spell.IsAbility || spell.Card == nil {
		return false
	}
	switch f.kind {
	case "card":
		return f.card.Match(spell.Card)
	case "controller":
		return controller != uuid.Nil && spell.Controller == controller
	case "shares-color":
		for _, permanent := range g.AllBattlefield() {
			if permanent.PhasedOut || (f.controlled && permanent.ControllerID() != controller) || !f.permanents.Match(permanent, g) {
				continue
			}
			for _, color := range permanent.Colors() {
				if color >= core.White && color <= core.Green && HasColorCardFilter(color).Match(spell.Card) {
					return true
				}
			}
		}
		return false
	case "ability-target-exclusion":
		return !g.spellTargetedByPermanentAbility(spell.ID, source, f.permanents, f.excludeSource)
	case "all":
		return len(f.children) > 0 && !slices.ContainsFunc(f.children, func(child SpellFilter) bool { return !child.matches(g, controller, source, spell) })
	case "any":
		return slices.ContainsFunc(f.children, func(child SpellFilter) bool { return child.matches(g, controller, source, spell) })
	case "not":
		return len(f.children) == 1 && !f.children[0].matches(g, controller, source, spell)
	default:
		return false
	}
}

func (f SpellFilter) inspectDefinition() definitionNode {
	n := definitionNode{Role: "spell-filter", Text: f.kind}
	switch f.kind {
	case "card", "controller", "shares-color", "ability-target-exclusion":
	case "all", "any":
		if len(f.children) == 0 {
			n.problem = "supply at least one spell filter"
		}
	case "not":
		if len(f.children) != 1 {
			n.problem = "supply one excluded spell filter"
		}
	default:
		n.problem = "supply a spell filter"
	}
	for i, child := range f.children {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("filters[%d]", i), child})
	}
	return n
}

// WithFilter adds a checked contextual filter to the existing spell declaration.
// Target choices remain in each resolution frame.
func (b *SpellTargetBinding) WithFilter(filter SpellFilter) *SpellTargetBinding {
	if spec, ok := b.spec.(*typedSpellTarget); ok {
		spec.filterCount++
		spec.contextualFilter = filter
	} else {
		b.invalidSpellFilter = true
	}
	return b
}

// Matching checks current spell and battlefield facts when a cast event occurs.
// Pending triggers keep the detection result and do not repeat this check.
func (p spellCastPattern) Matching(filter SpellFilter) spellCastPattern {
	p.filterCount++
	p.contextualFilter = &filter
	return p
}

type spellCastMatchesFilter struct{ filter SpellFilter }

func (c spellCastMatchesFilter) CheckTriggerCond(event *core.GameEvent, reader GameReader, sourceID, controller uuid.UUID) bool {
	game, ok := reader.(*Game)
	if !ok || event.TargetID == uuid.Nil {
		return false
	}
	spell := game.stack.FindByID(event.TargetID)
	return spell != nil && spell.SourceID == event.SourceID && c.filter.matches(game, controller, game.FindCardAnywhere(sourceID), spell)
}
