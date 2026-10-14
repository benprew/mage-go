package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// SacrificeAnyNumberAsEnters saves a count of completed entry sacrifices.
// The entry controller chooses. All choices precede simultaneous movement.
func SacrificeAnyNumberAsEnters(subjects, counted PermanentFilter) Ability {
	return &entrySacrificeAbility{BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic}, subjects: subjects, counted: counted}
}

type entrySacrificeAbility struct {
	BaseAbility
	subjects, counted PermanentFilter
}

func (a *entrySacrificeAbility) inspectDefinition() definitionNode {
	return definitionNode{Role: "entry-sacrifice", Text: "sacrifice any number as enters", battlefieldSource: true}
}
func (a *entrySacrificeAbility) applyEntry(g *Game, p *Permanent) {
	p.EntrySacrificeCount = 0
	player := g.GetPlayer(p.ControllerID())
	if player == nil {
		return
	}
	eligible := func(candidate *Permanent) bool {
		return candidate.ControllerID() == p.ControllerID() && a.subjects.Match(candidate, g)
	}
	candidates := g.FilterBattlefield(And(ControlledBy(p.ControllerID()), a.subjects))
	if len(candidates) == 0 {
		return
	}
	count := validNumericChoice(player, 0, len(candidates))
	selected := choosePermanentSubset(player, candidates, count, "Choose permanents to sacrifice", g)
	p.EntrySacrificeCount = g.controllersSacrificePermanentsCount(selected, eligible, a.counted)
}

type entrySacrificeReplacement struct {
	replacementBase
	ability *entrySacrificeAbility
}

func (*entrySacrificeReplacement) IsActive(GameReader) bool { return true }
func (*entrySacrificeReplacement) Text() string {
	return "sacrifice any number of matching permanents as enters"
}
func (*entrySacrificeReplacement) Matches(action Action, _ GameReader) bool {
	_, ok := action.(*battlefieldEntryAction)
	return ok
}
func (r *entrySacrificeReplacement) Clone() ReplacementEffect { c := *r; return &c }
func (r *entrySacrificeReplacement) Replace(action Action, g *Game) Action {
	r.ability.applyEntry(g, action.(*battlefieldEntryAction).permanent)
	return action
}

// PTEqualsEntrySacrificeCount defines P/T from this visit's saved entry result.
// A card in another zone has no entry result and supplies zero.
func PTEqualsEntrySacrificeCount() ContinuousEffect {
	return &countCharacteristicPT{power: true, toughness: true, entrySacrifice: true}
}

// EntrySacrificeCountOf reads a current or last-known entry result.
// Returned objects cannot supply the saved visit.
func EntrySacrificeCountOf(subject PermanentRef) ResolutionValue {
	return entrySacrificeCountValue{subject}
}

type entrySacrificeCountValue struct{ subject PermanentRef }

func (entrySacrificeCountValue) Text() string { return "completed entry sacrifices" }
func (v entrySacrificeCountValue) inspectDefinition() definitionNode {
	return permanentInformationNode(v.subject, "permanent-value", v.Text())
}
func (v entrySacrificeCountValue) resolveValue(ctx *EffectContext) int {
	p, lki := readPermanentInformation(v.subject, ctx)
	if lki != nil {
		p = lki.Snapshot
	}
	if p == nil {
		return 0
	}
	return p.EntrySacrificeCount
}
