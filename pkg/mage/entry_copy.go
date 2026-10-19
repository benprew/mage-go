package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// MayCopyAsEnters offers a copy choice before other entry operations.
// The copy keeps the physical card's identity and printed departure rules.
func MayCopyAsEnters(filter PermanentFilter) Ability {
	return MayCopyAsEntersWithModifications(filter)
}

// MayCopyAsEntersWithModifications offers an entry copy with copiable exceptions.
// The physical card keeps its identity and printed departure rules.
func MayCopyAsEntersWithModifications(filter PermanentFilter, modifications ...CopyModification) Ability {
	return &entryCopyAbility{BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic}, filter: filter, modifications: slices.Clone(modifications)}
}

// MayCopyAsEntersKeepingColors retains the entering object's copiable colors.
// Later copies keep those colors. Ordinary color changes still apply afterward.
func MayCopyAsEntersKeepingColors(filter PermanentFilter, modifications ...CopyModification) Ability {
	ability := MayCopyAsEntersWithModifications(filter, modifications...).(*entryCopyAbility)
	ability.keepColors = true
	return ability
}

type entryCopyAbility struct {
	BaseAbility
	filter        PermanentFilter
	modifications []CopyModification
	keepColors    bool
}

func (a *entryCopyAbility) inspectDefinition() definitionNode {
	n := definitionNode{Role: "entry-copy", Text: "optional entry copy", battlefieldSource: true}
	if err := validateCopyModifications(a.modifications); err != nil {
		n.problem = err.Error()
	}
	return n
}

type entryCopyKey struct {
	plan  *CardDefinitionPlan
	index int
}

type battlefieldEntryContext struct {
	auraSpell       bool
	graveyardHost   capturedCard
	entry           battlefieldEntryOptions
	actor           uuid.UUID
	announcedX      int
	castColorsSpent int
}

type battlefieldEntryAction struct {
	retained          map[filteredEntryKey]*retainedEntryReplacement
	filtered          map[filteredEntryKey]*filteredEntryReplacement
	context           battlefieldEntryContext
	permanent         *Permanent
	copies            map[entryCopyKey]*entryCopyReplacement
	variants          map[entryCopyKey]*entryVariantReplacement
	sacrifices        map[entryCopyKey]*entrySacrificeReplacement
	lifePayments      map[entryCopyKey]*entryLifeReplacement
	lifeLosses        map[entryCopyKey]*entryLifeLossReplacement
	graveyardPayments map[entryCopyKey]*entryGraveyardPaymentReplacement
	textChanges       []TextChange
}

func (a *battlefieldEntryAction) ActionSource() uuid.UUID { return a.permanent.ID() }
func (a *battlefieldEntryAction) PlayerID() uuid.UUID     { return a.permanent.ControllerID() }

func (a *battlefieldEntryAction) replacementCandidates() []ReplacementEffect {
	definition := a.permanent.Card
	if copied, ok := definition.(permanentSpellCopyCard); ok {
		definition = copied.Card
	}
	base, ok := definition.(*BaseCard)
	if !ok {
		return nil
	}
	var result []ReplacementEffect
	index := 0
	variantIndex := 0
	sacrificeIndex := 0
	lifeIndex := 0
	lossIndex := 0
	graveyardIndex := 0
	for _, ability := range a.permanent.RuntimeAbilities {
		if payment, ok := UnwrapAbility(ability).(*entryGraveyardPaymentAbility); ok {
			plan := base.definitionPlan
			if plan != nil && plan.declarationOrigin != nil {
				plan = plan.declarationOrigin
			}
			key := entryCopyKey{plan, graveyardIndex}
			graveyardIndex++
			if a.graveyardPayments == nil {
				a.graveyardPayments = make(map[entryCopyKey]*entryGraveyardPaymentReplacement)
			}
			replacement := a.graveyardPayments[key]
			if replacement == nil {
				replacement = &entryGraveyardPaymentReplacement{replacementBase: replacementBase{sourceID: a.permanent.ID(), priority: ReplacementPriorityDefault}, ability: payment}
				a.graveyardPayments[key] = replacement
			}
			result = append(result, replacement)
			continue
		}
		if loss, ok := UnwrapAbility(ability).(*entryLifeLossAbility); ok {
			plan := base.definitionPlan
			if plan != nil && plan.declarationOrigin != nil {
				plan = plan.declarationOrigin
			}
			key := entryCopyKey{plan, lossIndex}
			lossIndex++
			if a.lifeLosses == nil {
				a.lifeLosses = make(map[entryCopyKey]*entryLifeLossReplacement)
			}
			replacement := a.lifeLosses[key]
			if replacement == nil {
				replacement = &entryLifeLossReplacement{replacementBase: replacementBase{sourceID: a.permanent.ID(), priority: ReplacementPriorityDefault}, ability: loss}
				a.lifeLosses[key] = replacement
			}
			result = append(result, replacement)
			continue
		}
		if life, ok := UnwrapAbility(ability).(*entryLifeAbility); ok {
			plan := base.definitionPlan
			if plan != nil && plan.declarationOrigin != nil {
				plan = plan.declarationOrigin
			}
			key := entryCopyKey{plan, lifeIndex}
			lifeIndex++
			if a.lifePayments == nil {
				a.lifePayments = make(map[entryCopyKey]*entryLifeReplacement)
			}
			replacement := a.lifePayments[key]
			if replacement == nil {
				replacement = &entryLifeReplacement{replacementBase: replacementBase{sourceID: a.permanent.ID(), priority: ReplacementPriorityDefault}, ability: life}
				a.lifePayments[key] = replacement
			}
			result = append(result, replacement)
			continue
		}
		if sacrifice, ok := UnwrapAbility(ability).(*entrySacrificeAbility); ok {
			plan := base.definitionPlan
			if plan != nil && plan.declarationOrigin != nil {
				plan = plan.declarationOrigin
			}
			key := entryCopyKey{plan, sacrificeIndex}
			sacrificeIndex++
			if a.sacrifices == nil {
				a.sacrifices = make(map[entryCopyKey]*entrySacrificeReplacement)
			}
			replacement := a.sacrifices[key]
			if replacement == nil {
				replacement = &entrySacrificeReplacement{replacementBase: replacementBase{sourceID: a.permanent.ID(), priority: ReplacementPriorityDefault}, ability: sacrifice}
				a.sacrifices[key] = replacement
			}
			result = append(result, replacement)
			continue
		}
		if variant, ok := UnwrapAbility(ability).(*entryVariantAbility); ok {
			plan := base.definitionPlan
			if plan != nil && plan.declarationOrigin != nil {
				plan = plan.declarationOrigin
			}
			key := entryCopyKey{plan, variantIndex}
			variantIndex++
			if a.variants == nil {
				a.variants = make(map[entryCopyKey]*entryVariantReplacement)
			}
			replacement := a.variants[key]
			if replacement == nil {
				replacement = &entryVariantReplacement{replacementBase: replacementBase{sourceID: a.permanent.ID(), priority: ReplacementPriorityDefault}, ability: variant}
				a.variants[key] = replacement
			}
			result = append(result, replacement)
			continue
		}
		copyAbility, ok := UnwrapAbility(ability).(*entryCopyAbility)
		if !ok {
			continue
		}
		plan := base.definitionPlan
		if plan != nil && plan.declarationOrigin != nil {
			plan = plan.declarationOrigin
		}
		// Other ability omissions must not change an entry rule's identity.
		key := entryCopyKey{plan, index}
		index++
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
	modifications := r.ability.modifications
	if r.ability.keepColors {
		modifications = append(slices.Clone(modifications), CopyRetainColors(cardColors(p.Card)...))
	}
	values, err = values.WithModifications(modifications...)
	if err != nil {
		return nil
	}
	card, err := BuildEntryCopyWithText(p.Card, values, entry.textChanges...)
	if err != nil {
		return nil
	}
	p.replaceEntryDefinition(card)
	g.syncAbilityContext(p)
	return action
}
