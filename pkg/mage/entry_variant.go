package mage

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ChooseCopiableVariantAsEnters requires one copiable variant before entry.
// Copies make new choices. Phasing keeps the selected characteristics.
func ChooseCopiableVariantAsEnters(variants ...[]CopyModification) Ability {
	saved := make([][]CopyModification, len(variants))
	for i, variant := range variants {
		saved[i] = slices.Clone(variant)
	}
	return &entryVariantAbility{BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic}, variants: saved}
}

type entryVariantAbility struct {
	BaseAbility
	variants [][]CopyModification
}

func (a *entryVariantAbility) inspectDefinition() definitionNode {
	n := definitionNode{Role: "entry-copiable-variant", Text: "choose copiable entry variant", battlefieldSource: true}
	if len(a.variants) == 0 {
		n.problem = "entry variants require at least one option"
	}
	for _, variant := range a.variants {
		if len(variant) == 0 {
			n.problem = "entry variants require nonempty options"
			break
		}
		if err := validateCopyModifications(variant); err != nil {
			n.problem = err.Error()
			break
		}
	}
	return n
}

type entryVariantReplacement struct {
	replacementBase
	ability *entryVariantAbility
}

func (*entryVariantReplacement) IsActive(GameReader) bool { return true }
func (*entryVariantReplacement) Matches(action Action, _ GameReader) bool {
	_, ok := action.(*battlefieldEntryAction)
	return ok
}
func (r *entryVariantReplacement) Clone() ReplacementEffect { c := *r; return &c }
func (r *entryVariantReplacement) Replace(action Action, g *Game) Action {
	entry := action.(*battlefieldEntryAction)
	p := entry.permanent
	definition := p.Card
	if copied, ok := definition.(permanentSpellCopyCard); ok {
		definition = copied.Card
	}
	base, ok := definition.(*BaseCard)
	if !ok || base.copiableValues == nil {
		return nil
	}
	options := make([]*CopiableValues, len(r.ability.variants))
	labels := make([]string, len(options))
	for i, variant := range r.ability.variants {
		values, err := base.copiableValues.WithModifications(variant...)
		if err != nil {
			return nil
		}
		options[i] = values
		card, err := values.Build()
		if err != nil {
			return nil
		}
		labels[i] = entryVariantLabel(card.(*BaseCard))
	}
	choice := 0
	if player := g.GetPlayer(p.ControllerID()); player != nil {
		choice = player.ChooseMode(labels, "Choose entry characteristics")
	}
	if choice < 0 || choice >= len(labels) {
		choice = 0
	}
	card, err := BuildEntryCopyWithText(p.Card, options[choice], entry.textChanges...)
	if err != nil {
		return nil
	}
	p.replaceEntryDefinition(card)
	g.syncAbilityContext(p)
	return action
}

func entryVariantLabel(card *BaseCard) string {
	parts := make([]string, 0, 1+len(card.Types())+len(card.SubTypes())+len(card.attrSeeds))
	parts = append(parts, fmt.Sprintf("%d/%d", card.Power(), card.Toughness()))
	for _, kind := range card.Types() {
		parts = append(parts, kind.String())
	}
	parts = append(parts, card.SubTypes()...)
	keywords := make([]string, 0, len(card.attrSeeds))
	for keyword, count := range card.attrSeeds {
		if count > 0 {
			keywords = append(keywords, keyword.String())
		}
	}
	slices.Sort(keywords)
	parts = append(parts, keywords...)
	return strings.Join(parts, " ")
}
