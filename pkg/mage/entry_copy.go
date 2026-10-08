package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// MayCopyAsEnters offers a copy choice before other entry operations.
// The copy keeps the physical card's identity and printed departure rules.
func MayCopyAsEnters(filter PermanentFilter) Ability {
	return &entryCopyAbility{BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic}, filter: filter}
}

type entryCopyAbility struct {
	BaseAbility
	filter PermanentFilter
}

func (*entryCopyAbility) inspectDefinition() definitionNode {
	return definitionNode{Role: "entry-copy", Text: "optional entry copy", battlefieldSource: true}
}

type entryCopyKey struct {
	plan  *CardDefinitionPlan
	index int
}

type battlefieldEntryAction struct {
	permanent *Permanent
	copies    map[entryCopyKey]*entryCopyReplacement
}

func (a *battlefieldEntryAction) ActionSource() uuid.UUID { return a.permanent.ID() }
func (a *battlefieldEntryAction) PlayerID() uuid.UUID     { return a.permanent.ControllerID() }

func (a *battlefieldEntryAction) replacementCandidates() []ReplacementEffect {
	base, ok := a.permanent.Card.(*BaseCard)
	if !ok {
		return nil
	}
	var result []ReplacementEffect
	for i, ability := range a.permanent.RuntimeAbilities {
		copyAbility, ok := UnwrapAbility(ability).(*entryCopyAbility)
		if !ok {
			continue
		}
		key := entryCopyKey{base.definitionPlan, i}
		replacement := a.copies[key]
		if replacement == nil {
			replacement = &entryCopyReplacement{replacementBase: replacementBase{sourceID: a.permanent.ID(), priority: ReplacementPriorityCopy}, ability: copyAbility}
			a.copies[key] = replacement
		}
		result = append(result, replacement)
	}
	return result
}

type entryCopyReplacement struct {
	replacementBase
	ability *entryCopyAbility
}

func (*entryCopyReplacement) IsActive(GameReader) bool { return true }
func (*entryCopyReplacement) Matches(a Action, _ GameReader) bool {
	_, ok := a.(*battlefieldEntryAction)
	return ok
}
func (r *entryCopyReplacement) Clone() ReplacementEffect { c := *r; return &c }
func (r *entryCopyReplacement) Replace(action Action, g *Game) Action {
	entry := action.(*battlefieldEntryAction)
	p := entry.permanent
	player := g.GetPlayer(p.ControllerID())
	var candidates []*Permanent
	for _, candidate := range g.AllBattlefield() {
		if !candidate.PhasedOut && r.ability.filter.Match(candidate, g) {
			candidates = append(candidates, candidate)
		}
	}
	if player == nil || len(candidates) == 0 || !player.ChooseMayAbility("enter as a copy of a permanent") {
		return action
	}
	selected := player.ChoosePermanent(candidates, "Choose a permanent to copy", g)
	if selected == nil {
		selected = candidates[0]
	}
	var original *Permanent
	for _, candidate := range candidates {
		if candidate.ID() == selected.ID() && candidate.incarnationID == selected.incarnationID {
			original = candidate
			break
		}
	}
	if original == nil {
		original = candidates[0]
	}
	live := g.FindPermanent(original.ID())
	if live == nil || live.incarnationID != original.incarnationID || !r.ability.filter.Match(live, g) {
		return action
	}
	values, err := g.CaptureCopiableValues(live)
	if err != nil {
		return nil
	}
	card, err := BuildEntryCopy(p.Card, values)
	if err != nil {
		return nil
	}
	p.replaceEntryDefinition(card)
	g.syncAbilityContext(p)
	return action
}
