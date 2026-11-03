package mage

import (
	"testing"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func countCardAbility(card *Card, attr Attr) int {
	count := 0
	for _, ability := range card.Abilities() {
		if AbilityAttribute(ability) == attr {
			count++
		}
	}
	return count
}

func TestCardAbilitiesIncludesAttributes(t *testing.T) {
	trigger := EntersBattlefieldTrigger(GainLife(1), false)
	card := NewCreature("Unified Bear", "{2}{W}", 2, 2,
		WithKeyword(Flying), WithAbility(trigger), WithKeyword(FirstStrike),
		WithKeyword(Flying), WithKeyword(AttrDoesNotUntap), WithKeyword(AttrCanBlock))
	abilities := card.Abilities()
	if len(abilities) != 6 {
		t.Fatalf("got %d abilities, want one trigger and five attribute abilities", len(abilities))
	}
	counts := map[Attr]int{}
	for i, a := range abilities {
		counts[AbilityAttribute(a)]++
		if a != card.Abilities()[i] || a.AbilityID() != card.Abilities()[i].AbilityID() {
			t.Fatal("ability identity changed between reads")
		}
		if AbilityAttribute(a) != 0 && a.Type() != AbilityStatic {
			t.Fatal("simple attribute ability must be static")
		}
	}
	if counts[Flying] != 2 || counts[FirstStrike] != 1 || counts[AttrDoesNotUntap] != 1 || counts[AttrCanBlock] != 1 {
		t.Fatalf("incorrect ability attributes: %v", counts)
	}
	abilities[0] = nil
	if card.Abilities()[0] != trigger {
		t.Fatal("returned slice changed the card's ability list")
	}
}

func TestCardAbilitiesDefinitionAndCopies(t *testing.T) {
	def, err := NewCardDefinition(CardDefinitionSpec{
		Name: "Defined Bird", TypeLine: TypeLine{CardTypes: []CardType{TypeCreature}},
		Abilities: []Ability{StaticKeyword(Flying), StaticKeyword(Flying), StaticKeyword(Haste), StaticCapability(AttrCanBlock, 1)},
	})
	if err != nil {
		t.Fatal(err)
	}
	original := def.NewInstance(uuid.New())
	if len(original.Abilities()) != 4 {
		t.Fatalf("got %d abilities, want 3", len(original.Abilities()))
	}
	for _, copy := range []*Card{original.Copy(), original.CloneInstance(), def.NewInstance(uuid.New())} {
		if len(copy.Abilities()) != 4 {
			t.Fatal("copy lost or duplicated abilities")
		}

	}
	legacy := NewCreature("Legacy Bird", "{U}", 1, 1, WithKeyword(Flying))
	if len(legacy.Copy().Abilities()) != 1 {
		t.Fatal("copy lost keyword")
	}

	for _, target := range []*Card{def.NewInstance(uuid.New()), NewCreature("Copy", "", 1, 1)} {
		target.CloneFrom(original)
		if len(target.Abilities()) != 4 {
			t.Fatal("CloneFrom lost or duplicated abilities")
		}
		p := NewPermanent(target, uuid.New())
		if !p.HasAttr(Flying) || !p.HasAttr(Haste) {
			t.Fatal("copied abilities lost their attributes")
		}
	}
	token := NewToken("Bird", 1, 1, []CardType{TypeCreature}, nil, nil, Flying)
	if len(token.Abilities()) != 1 || AbilityAttribute(token.Abilities()[0]) != Flying {
		t.Fatal("token keyword is absent from Abilities")
	}
}

func TestCardAbilitiesWardIsOneTriggeredAbility(t *testing.T) {
	card := NewCreature("Warded Bear", "{G}", 2, 2, WithWard(ManaCostOf("{2}")))
	if len(card.Abilities()) != 1 {
		t.Fatalf("got %d abilities, want one ward ability", len(card.Abilities()))
	}
	ward := card.Abilities()[0]
	if AbilityAttribute(ward) != Ward || ward.Type() != AbilityTriggered {
		t.Fatal("ward must expose its keyword and triggered behavior together")
	}
	if _, ok := ward.(TriggeredAbility); !ok {
		t.Fatal("ward lost its executable trigger")
	}
	if !NewPermanent(card, uuid.New()).HasAttr(Ward) {
		t.Fatal("ward attribute is absent")
	}
}

func TestCardAbilitiesCompiledDefinitions(t *testing.T) {
	def, err := NewCardDefinition(CardDefinitionSpec{
		Name: "Compiled Bird", TypeLine: TypeLine{CardTypes: []CardType{TypeCreature}},
		Abilities: []Ability{keywordRule(Flying), keywordRule(FirstStrike)},
	})
	if err != nil {
		t.Fatal(err)
	}
	card := def.NewInstance(uuid.New())
	counts := map[Attr]int{}
	for _, ability := range card.Abilities() {
		counts[AbilityAttribute(ability)]++
	}
	if len(card.Abilities()) != 2 || counts[Flying] != 1 || counts[FirstStrike] != 1 {
		t.Fatalf("compiled abilities are absent or duplicated: %v", counts)
	}
}

func TestCardAbilitiesWardDefinition(t *testing.T) {
	definition, err := NewCardDefinition(CardDefinitionSpec{Name: "Ward", TypeLine: TypeLine{CardTypes: []CardType{TypeCreature}}, Abilities: []Ability{WardTrigger(ManaCostOf("{2}"))}})
	if err != nil {
		t.Fatal(err)
	}
	card := NewCard(definition)
	if len(card.Abilities()) != 1 || NewPermanent(card, uuid.New()).baseAttrs[Ward] != 1 {
		t.Fatal("ward must supply one ability and one attribute")
	}
}

func TestCardAbilitiesInitializePermanentAttributes(t *testing.T) {
	def, err := NewCardDefinition(CardDefinitionSpec{
		Name: "Mixed Abilities", TypeLine: TypeLine{CardTypes: []CardType{TypeCreature}},
		Abilities: []Ability{keywordRule(Flying), keywordRule(Flying), keywordRule(FirstStrike), WardTrigger(ManaCostOf("{2}"))},
	})
	if err != nil {
		t.Fatal(err)
	}
	card := NewCard(def, WithKeyword(AttrCanBlock))
	originalCount := len(card.Abilities())
	for _, source := range []*Card{card, card.Copy(), card.CloneInstance()} {
		p := NewPermanent(source, uuid.New())
		if p.baseAttrs[Flying] != 2 || p.printedAbilityAttrs[Flying] != 2 || p.baseAttrs[FirstStrike] != 1 || p.baseAttrs[Ward] != 1 {
			t.Fatal("permanent attributes do not match its card abilities")
		}
		if !p.HasAttr(AttrCanBlock) || len(p.RuntimeAbilities) != originalCount {
			t.Fatal("capability ability was lost")
		}
	}
}

func TestLegacyCopyOwnsWardState(t *testing.T) {
	original := NewCreature("Ward", "{G}", 2, 2, WithWard(ManaCostOf("{2}")))
	copied := original.Copy()
	a, b := original.abilities[0], copied.abilities[0]
	if a.AbilityID() != b.AbilityID() {
		t.Fatal("Copy must create a new ability identity")
	}

}

func TestLegacyCloneFromDefinitionOwnsState(t *testing.T) {
	def, err := NewCardDefinition(CardDefinitionSpec{
		Name: "Defined Ward", TypeLine: TypeLine{CardTypes: []CardType{TypeCreature}}, Colors: []Color{Blue},
		Abilities: []Ability{WardTrigger(ManaCostOf("{2}"))},
	})
	if err != nil {
		t.Fatal(err)
	}
	original := NewCard(def)
	cloned := NewCreature("Clone", "{3}{U}", 0, 0)
	cloned.CloneFrom(original)
	if colors := cloned.Colors(); len(colors) != 1 || colors[0] != Blue {
		t.Fatalf("lost color indicator: %v", colors)
	}

}

func TestDefinitionRepeatedKeywordAbilities(t *testing.T) {
	definition, err := NewCardDefinition(CardDefinitionSpec{Name: "Repeated", TypeLine: TypeLine{CardTypes: []CardType{TypeCreature}}, Abilities: []Ability{StaticKeyword(Flying), StaticKeyword(Flying)}})
	if err != nil {
		t.Fatal(err)
	}
	if countCardAbility(NewCard(definition), Flying) != 2 {
		t.Fatal("definition lost a printed ability")
	}
}
