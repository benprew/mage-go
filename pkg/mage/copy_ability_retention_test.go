package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func retainedCopyTrigger() *GenericTriggered {
	return BindTriggeredAbility(func(self *TriggeredAbilityBinding) *GenericTriggered {
		return WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("upkeep")),
			MayCopyPermanent(You, This, This, true, RetainCopyAbility(self))).(*GenericTriggered)
	})
}

func TestCopyAbilityRetention(t *testing.T) {
	plan, err := NewCardDefinitionPlan(func() Card {
		return NewCreature("Retainer", "{U}", 2, 2, WithAbility(retainedCopyTrigger()))
	})
	if err != nil {
		t.Fatal(err)
	}
	a := &optionalCopyPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	for _, want := range []int{1, 2, 4} {
		g.SetStep(core.Upkeep)
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
		g.PutTriggersOnStack()
		if g.StackSize() != want {
			t.Fatalf("got %d, want %d independent triggers", g.StackSize(), want)
		}
		g.ResolveStack()
		p = g.FindPermanent(p.ID())
		if len(p.RuntimeAbilities) != want*2 {
			t.Fatalf("self-copy abilities=%d, want %d", len(p.RuntimeAbilities), want*2)
		}
	}
	snapshot, err := g.CaptureCopiableValues(p)
	if err != nil {
		t.Fatal(err)
	}
	one, err := snapshot.Build()
	if err != nil {
		t.Fatal(err)
	}
	two, err := snapshot.Build()
	if err != nil {
		t.Fatal(err)
	}
	if err := independentCardDefinitions(one, two); err != nil {
		t.Fatal(err)
	}
}

func TestCopyAbilityRetentionInvalid(t *testing.T) {
	var escaped *TriggeredAbilityBinding
	trigger := BindTriggeredAbility(func(self *TriggeredAbilityBinding) *GenericTriggered {
		escaped = self
		return WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("upkeep")), MayCopyPermanent(You, This, This, true, RetainCopyAbility(self))).(*GenericTriggered)
	})
	valid := NewCreature("Valid", "", 1, 1, WithAbility(trigger))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []Effect{
		CopyPermanent(This, This, true, nil),
		CopyPermanent(This, This, true, RetainCopyAbility(&TriggeredAbilityBinding{})),
		CopyPermanent(This, This, true, RetainCopyAbility(nil)),
		CopyPermanent(This, This, true, RetainCopyAbility(escaped)),
		CopyPermanent(This, This, true, RetainCopyAbility(escaped), RetainCopyAbility(escaped)),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid retained declaration accepted")
		}
	}
}
