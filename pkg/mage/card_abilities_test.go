package mage

import (
	"testing"

	"github.com/google/uuid"

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
	if len(abilities) != 5 {
		t.Fatalf("got %d abilities, want one trigger and four attribute abilities", len(abilities))
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
	if counts[Flying] != 2 || counts[FirstStrike] != 1 || counts[AttrDoesNotUntap] != 1 || counts[AttrCanBlock] != 0 {
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
		Keywords: map[Attr]int{Flying: 2, Haste: 1, AttrCanBlock: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	original := def.NewInstance(uuid.New())
	if len(original.Abilities()) != 3 {
		t.Fatalf("got %d abilities, want 3", len(original.Abilities()))
	}
	for _, copy := range []*Card{original.Copy(), original.CloneInstance(), def.NewInstance(uuid.New())} {
		if len(copy.Abilities()) != 3 {
			t.Fatal("copy lost or duplicated abilities")
		}
		copy.Abilities()[0].SetController(uuid.New())
		if original.Abilities()[0].Controller() != uuid.Nil {
			t.Fatal("copies share ability state")
		}
	}
	legacy := NewCreature("Legacy Bird", "{U}", 1, 1, WithKeyword(Flying))
	legacyCopy := legacy.Copy()
	legacyCopy.Abilities()[0].SetController(uuid.New())
	if legacy.Abilities()[0].Controller() != uuid.Nil {
		t.Fatal("legacy copies share attribute ability state")
	}
	for _, target := range []*Card{def.NewInstance(uuid.New()), NewCreature("Copy", "", 1, 1)} {
		target.CloneFrom(original)
		if len(target.Abilities()) != 3 {
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

func TestCardAbilitiesWardDefinition(t *testing.T) {
	for _, seeds := range []map[Attr]int{nil, {Ward: 1}} {
		def, err := NewCardDefinition(CardDefinitionSpec{
			Name: "Defined Ward", TypeLine: TypeLine{CardTypes: []CardType{TypeCreature}},
			Keywords: seeds,
			Rules:    []RuleDefinition{NewRuleDefinition(func() Ability { return WardTrigger(ManaCostOf("{2}")) })},
		})
		if err != nil {
			t.Fatal(err)
		}
		card := def.NewInstance(uuid.New())
		if len(card.Abilities()) != 1 || card.Abilities()[0].Type() != AbilityTriggered {
			t.Fatal("ward rule and attribute must be one ability")
		}
		if NewPermanent(card, uuid.New()).baseAttrs[Ward] != 1 {
			t.Fatal("ward rule must supply exactly one attribute")
		}
	}
}

func TestCardAbilitiesInitializePermanentAttributes(t *testing.T) {
	def, err := NewCardDefinition(CardDefinitionSpec{
		Name: "Mixed Abilities", TypeLine: TypeLine{CardTypes: []CardType{TypeCreature}},
		Keywords: map[Attr]int{Flying: 2, FirstStrike: 1},
		Rules:    []RuleDefinition{NewRuleDefinition(func() Ability { return WardTrigger(ManaCostOf("{2}")) })},
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
			t.Fatal("capability attributes must not add ability objects")
		}
	}
}
