package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCopyTriggeredAbilitySnapshots(t *testing.T) {
	build := func() *GenericTriggered {
		target := DeclarePermanentTarget("target", IsCreature)
		trigger := WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("upkeep")), TapSubject(target)).(*GenericTriggered)
		trigger.AddTarget(target.targetSpec())
		return trigger
	}
	values, err := entryCopyPlan(t, "Donor", func() []CardOption { return nil }).CopiableValues().WithModifications(CopyAddTriggeredAbility(build))
	if err != nil {
		t.Fatal(err)
	}
	first, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	second, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	if err := independentCardDefinitions(first, second); err != nil {
		t.Fatal(err)
	}
	if len(first.Abilities()) != 1 || len(second.Abilities()) != 1 {
		t.Fatal("lost copy ability")
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(first, a.PlayerID())
	captured, err := g.CaptureCopiableValues(p)
	if err != nil {
		t.Fatal(err)
	}
	later, err := captured.Build()
	if err != nil || len(later.Abilities()) != 1 {
		t.Fatal("ability not copiable", err)
	}
	g.PhaseIn(g.PhaseOut(p.ID()))
	if len(g.FindPermanent(p.ID()).RuntimeAbilities) != 1 {
		t.Fatal("phase changed ability count")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	captured, err = g.CaptureCopiableValues(g.FindPermanent(p.ID()))
	if err != nil {
		t.Fatal(err)
	}
	later, err = captured.Build()
	if err != nil || len(later.Abilities()) != 1 {
		t.Fatal("ordinary removal changed copy values", err)
	}
}

func TestCopyTriggeredAbilityInvalid(t *testing.T) {
	shared := WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("upkeep")), GainLifeFor(You, Fixed(1))).(*GenericTriggered)
	sharedTarget := DeclarePermanentTarget("shared target", IsCreature)
	invalid := []func() *GenericTriggered{
		func() *GenericTriggered {
			trigger := WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("upkeep")), TapSubject(sharedTarget)).(*GenericTriggered)
			trigger.AddTarget(sharedTarget.targetSpec())
			return trigger
		},
		nil,
		func() *GenericTriggered { return nil },
		func() *GenericTriggered { return shared },
		func() *GenericTriggered {
			return WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("upkeep")), TapSubject(From(BindPermanent("missing")))).(*GenericTriggered)
		},
		func() *GenericTriggered { panic("invalid builder") },
	}
	for i, build := range invalid {
		modification := CopyAddTriggeredAbility(build)
		if _, err := entryCopyPlan(t, "Donor", func() []CardOption { return nil }).CopiableValues().WithModifications(modification); err == nil {
			t.Fatal("invalid addition accepted", i)
		}
		if err := ValidateCardDefinition("Invalid", NewCreature("Invalid", "", 1, 1, WithAbility(MayCopyAsEntersWithModifications(IsCreature, modification)))); err == nil {
			t.Fatal("invalid entry accepted", i)
		}
	}
}

func TestCopyTriggeredAbilityEntryOrder(t *testing.T) {
	a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	donor, err := entryCopyPlan(t, "Donor", func() []CardOption { return nil }).Build()
	if err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(donor, b.PlayerID())
	modification := CopyAddTriggeredAbility(func() *GenericTriggered {
		return WhenEvent(Enters(This), GainLifeFor(You, Fixed(1))).(*GenericTriggered)
	})
	plan := entryCopyPlan(t, "Physical", func() []CardOption {
		return []CardOption{WithAbility(MayCopyAsEntersWithModifications(IsCreature, modification))}
	})
	cards := make([]Card, 0, 2)
	for range 2 {
		card, err := plan.Build()
		if err != nil {
			t.Fatal(err)
		}
		cards = append(cards, card)
	}
	seen := 0
	g.effects.AddReplacement(&testPrioritizedReplacement{replacementBase: replacementBase{priority: ReplacementPriorityDefault}, matchFunc: func(action Action) bool { _, ok := action.(*battlefieldEntryAction); return ok }, replaceFunc: func(action Action, _ *Game) Action {
		p := action.(*battlefieldEntryAction).permanent
		if p.Name() != "Donor" || len(p.RuntimeAbilities) != 1 {
			t.Fatal("copy grant missing before ordinary replacement")
		}
		seen++
		return action
	}})
	entries := g.putTokensOnBattlefieldTogether(cards, a.PlayerID())
	if len(entries) != 2 || seen != 2 {
		t.Fatal("batch entry")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("entry observation lost added ability")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatal("entry source or controller")
	}
}
