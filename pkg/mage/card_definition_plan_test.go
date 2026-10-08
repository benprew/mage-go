package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestCardDefinitionPlanFreshDeclarations(t *testing.T) {
	plan, err := NewCardDefinitionPlan(func() Card {
		target := DeclarePermanentTarget("victim", IsCreature)
		chosen := BindPlayer("chosen")
		return NewCreature("Plan fixture", "{U}", 1, 2, WithAbility(Activate(Costs(Tap()), Sequence(ChoosePlayer(You, EachPlayer, chosen), GainLifeFor(FromPlayer(chosen), Fixed(2)), TapSubject(target)), WithTargets(target))))
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	first.SetOwner(uuid.New())
	first.Abilities()[0].SetSource(uuid.New())
	second, err := RebuildCardDefinition(first.Copy())
	if err != nil {
		t.Fatal(err)
	}
	if second.ID() == first.ID() || second.Owner() != uuid.Nil || second.Abilities()[0].Source() != uuid.Nil {
		t.Fatal("reconstruction retained instance state")
	}
	if err := independentCardDefinitions(first, second); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCardDefinition(second.Name(), second); err != nil {
		t.Fatal(err)
	}
	first.CloneFrom(NewCreature("Legacy fixture", "", 3, 3))
	if _, err := RebuildCardDefinition(first); err == nil {
		t.Fatal("legacy rule replacement retained the old plan")
	}
}

func TestCardDefinitionPlanInvalid(t *testing.T) {
	card := NewCreature("Shared card", "", 1, 1)
	ability := NewAction(ActionActivated, WithEffects(TapSubject(This)))
	target := DeclarePermanentTarget("shared target", IsCreature)
	binding := BindPlayer("shared player")
	spec := TargetPermanent(IsCreature)
	for _, tc := range []struct {
		name    string
		build   CardFactory
		message string
	}{
		{"nil factory", nil, "factory"},
		{"nil card", func() Card { return nil }, "card"},
		{"panic", func() Card { panic("bad builder") }, "bad builder"},
		{"shared card", func() Card { return card }, "identity"},
		{"shared ability", func() Card { return NewCreature("Shared ability", "", 1, 1, WithAbility(ability)) }, "declaration"},
		{"shared target", func() Card {
			return NewCreature("Shared target", "", 1, 1, WithAbility(Activate(Costs(), TapSubject(target), WithTargets(target))))
		}, "declaration"},
		{"shared binding", func() Card {
			return NewCreature("Shared binding", "", 1, 1, WithAbility(NewAction(ActionActivated, WithEffects(Sequence(ChoosePlayer(You, EachPlayer, binding), GainLifeFor(FromPlayer(binding), Fixed(1)))))))
		}, "declaration"},
		{"shared target specification", func() Card {
			target := &PermanentTargetBinding{&targetDeclaration{name: "fresh wrapper", spec: spec}}
			return NewCreature("Shared specification", "", 1, 1, WithAbility(Activate(Costs(), TapSubject(target), WithTargets(target))))
		}, "declaration"},
		{"saved activation targets", func() Card {
			target := DeclarePermanentTarget("selected", IsCreature)
			target.spec.(*PermanentTarget).chosen = []uuid.UUID{uuid.New()}
			return NewCreature("Saved target", "", 1, 1, WithAbility(Activate(Costs(), TapSubject(target), WithTargets(target))))
		}, "choices"},
		{"unbound read", func() Card {
			return NewCreature("Invalid binding", "", 1, 1, WithAbility(NewAction(ActionActivated, WithEffects(GainLifeFor(FromPlayer(BindPlayer("missing")), Fixed(1))))))
		}, "producer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewCardDefinitionPlan(tc.build); err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("error=%v, want %s", err, tc.message)
			}
		})
	}
	for _, card := range []Card{nil, NewCreature("Legacy", "", 1, 1)} {
		if _, err := RebuildCardDefinition(card); err == nil {
			t.Fatal("reconstructed a card without a plan")
		}
	}
	var plan *CardDefinitionPlan
	if _, err := plan.Build(); err == nil {
		t.Fatal("built a nil plan")
	}
}

func TestCardDefinitionPlanBuildErrors(t *testing.T) {
	invalid := false
	plan, err := NewCardDefinitionPlan(func() Card {
		card := NewCreature("Build fixture", "", 1, 1)
		if invalid {
			card.SetOwner(uuid.New())
		}
		return card
	})
	if err != nil {
		t.Fatal(err)
	}
	invalid = true
	if _, err := plan.Build(); err == nil || !strings.Contains(err.Error(), "owner") {
		t.Fatalf("invalid build: %v", err)
	}
}
