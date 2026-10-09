package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextAbilityCorrespondenceContracts(t *testing.T) {
	for _, tc := range []struct {
		name              string
		original, changed int
		valid             bool
	}{
		{"same", 1, 1, true}, {"missing", 1, 0, false}, {"extra", 1, 2, false}, {"unkeyed", -1, -1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			build := func(group int) CardFactory {
				return func() Card {
					option := WithAbility(Activate(Costs(), GainLifeFor(You, ValueAtResolution(Fixed(1)))))
					if group >= 0 {
						option = WithDefinitionAbilityGroup(group, option)
					}
					return NewArtifact("source", "", option)
				}
			}
			plan, err := NewCardDefinitionPlan(build(tc.original))
			if err != nil {
				t.Fatal(err)
			}
			card, err := plan.Build()
			if err != nil {
				t.Fatal(err)
			}
			WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(tc.changed), nil })(card.(*BaseCard))
			revision, err := BuildTextDefinitionRevision(card)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v: %v", tc.valid, err)
			}
			if err != nil {
				return
			}
			original := card.Abilities()[0]
			original.SetSource(uuid.New())
			original.SetController(uuid.New())
			replacement, ok := revision.ReplacementAbility(original.AbilityID())
			if !ok || replacement.Source() != uuid.Nil || replacement.Controller() != uuid.Nil {
				t.Fatal("saved context transferred")
			}
			if _, ok := revision.ReplacementAbility(uuid.New()); ok {
				t.Fatal("unknown ability matched")
			}
		})
	}
	if _, err := BuildTextDefinitionRevision(nil); err == nil {
		t.Fatal("nil accepted")
	}
}

func TestTextAbilityCorrespondenceInvalidGroups(t *testing.T) {
	for _, options := range [][]CardOption{
		{WithDefinitionAbilityGroup(-1, WithAbility(Activate(Costs(), DrawFor(You, ValueAtResolution(Fixed(1))))))},
		{WithDefinitionAbilityGroup(0, nil)},
		{WithDefinitionAbilityGroup(0, WithAbility(Activate(Costs(), DrawFor(You, ValueAtResolution(Fixed(1)))))), WithDefinitionAbilityGroup(0, WithAbility(Activate(Costs(), DrawFor(You, ValueAtResolution(Fixed(1))))))},
	} {
		if err := ValidateCardDefinition("invalid", NewArtifact("invalid", "", options...)); err == nil {
			t.Fatal("invalid declaration group accepted")
		}
	}
	var revision *TextDefinitionRevision
	if _, ok := revision.ReplacementAbility(uuid.New()); ok {
		t.Fatal("nil revision matched")
	}
}

func TestTextAbilityCorrespondenceMembersAndCopy(t *testing.T) {
	build := func() Card {
		return NewArtifact("source", "", WithDefinitionAbilityGroup(9, func(card *BaseCard) {
			card.AddAbility(Activate(Costs(), DrawFor(You, ValueAtResolution(Fixed(1)))))
			card.AddAbility(Activate(Costs(), DrawFor(You, ValueAtResolution(Fixed(2)))))
		}))
	}
	plan, err := NewCardDefinitionPlan(build)
	if err != nil {
		t.Fatal(err)
	}
	original, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build, nil })(original.(*BaseCard))
	revision, err := BuildTextDefinitionRevision(original)
	if err != nil {
		t.Fatal(err)
	}
	for i, ability := range original.Abilities() {
		replacement, ok := revision.ReplacementAbility(ability.AbilityID())
		if !ok || replacement != revision.Definition().Abilities()[i] {
			t.Fatal("group members were swapped")
		}
	}
	base := original.(*BaseCard)
	copied := base.Copy().(*BaseCard)
	delete(copied.abilityKeys, copied.Abilities()[0].AbilityID())
	if _, err := BuildTextDefinitionRevision(copied); err == nil {
		t.Fatal("missing key accepted")
	}
	if _, err := BuildTextDefinitionRevision(base); err != nil {
		t.Fatal("copy changed original keys", err)
	}
	base.CloneFrom(NewArtifact("legacy", ""))
	if _, err := BuildTextDefinitionRevision(base); err == nil {
		t.Fatal("legacy replacement retained correspondence")
	}
}

func TestTextAbilityCorrespondenceCopyOmissionsAndTokens(t *testing.T) {
	plan, err := NewCardDefinitionPlan(func() Card {
		return NewCreature("source", "", 1, 1,
			WithDefinitionAbilityGroup(4, func(card *BaseCard) {
				card.AddAbility(StaticAbility(&testCopyColorDefinition{}))
				card.AddAbility(Activate(Costs(), DrawFor(You, ValueAtResolution(Fixed(1)))))
			}))
	})
	if err != nil {
		t.Fatal(err)
	}
	values, err := plan.CopiableValues().WithModifications(CopyRetainColors(core.Blue))
	if err != nil {
		t.Fatal(err)
	}
	changed, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	base := changed.(*BaseCard)
	if len(base.Abilities()) != 1 || base.abilityKeys[base.Abilities()[0].AbilityID()] != (definitionAbilityKey{4, 1}) {
		t.Fatal("copy omission changed remaining declaration position")
	}
	original, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	if len(original.Abilities()) != 2 {
		t.Fatal("copy omission changed original plan")
	}
	token := NewToken("token", 1, 1, []core.CardType{core.TypeCreature}, nil)
	WithDefinitionAbilityGroup(8, WithAbility(Activate(Costs(), DrawFor(You, ValueAtResolution(Fixed(1))))))(token)
	clone := token.Copy().(*BaseCard)
	if clone.Abilities()[0].AbilityID() == token.Abilities()[0].AbilityID() {
		t.Fatal("token ability identity reused")
	}
	if err := ValidateCardDefinition(clone.Name(), clone); err != nil {
		t.Fatal(err)
	}
	if clone.abilityKeys[clone.Abilities()[0].AbilityID()] != (definitionAbilityKey{8, 0}) {
		t.Fatal("token copy lost group")
	}
	token.AddAbility(Activate(Costs(), DrawFor(You, ValueAtResolution(Fixed(2)))))
	if err := ValidateCardDefinition(token.Name(), token); err != nil {
		t.Fatal("existing added-ability caller rejected", err)
	}
}
