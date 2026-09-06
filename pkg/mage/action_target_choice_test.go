package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestActivateAbility_OpponentChoosesOnlyMarkedTarget(t *testing.T) {
	g, a, b := newTriggerTargetGame()

	source := NewArtifact("Choice Source", "{0}",
		WithActivatedAbility(
			FuncEffect("record targets", EffectProperties{}, func(_ *Game, _, _ uuid.UUID, _ []uuid.UUID) error { return nil }),
			Tap(),
			WithTarget(TargetCreatureYouControl()),
			WithTarget(TargetOpponentChoice(TargetCreatureOpponentControls())),
		),
	)
	source.SetOwner(a.PlayerID())
	sourcePerm := g.PutOnBattlefield(source, a.PlayerID())

	mine := NewCreature("Mine", "{1}", 1, 1)
	mine.SetOwner(a.PlayerID())
	minePerm := g.PutOnBattlefield(mine, a.PlayerID())
	first := NewCreature("Opponent First", "{1}", 1, 1)
	first.SetOwner(b.PlayerID())
	firstPerm := g.PutOnBattlefield(first, b.PlayerID())
	second := NewCreature("Opponent Choice", "{1}", 2, 2)
	second.SetOwner(b.PlayerID())
	secondPerm := g.PutOnBattlefield(second, b.PlayerID())

	b.chooseQueue = [][]uuid.UUID{{secondPerm.ID()}}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), sourcePerm.ID(), 0, []uuid.UUID{minePerm.ID(), firstPerm.ID()}); err != nil {
		t.Fatalf("ActivateAbilityByIndex: %v", err)
	}

	obj := g.Stack().Peek()
	if obj == nil {
		t.Fatal("expected ability on stack")
	}
	if got, want := obj.Targets, []uuid.UUID{minePerm.ID(), secondPerm.ID()}; !slices.Equal(got, want) {
		t.Fatalf("targets = %v, want %v", got, want)
	}
	if len(a.calls) != 0 {
		t.Fatalf("controller received %d target prompts, want 0", len(a.calls))
	}
	if len(b.calls) != 1 {
		t.Fatalf("opponent received %d target prompts, want 1", len(b.calls))
	}
	if !slices.Equal(b.calls[0].possible, []uuid.UUID{firstPerm.ID(), secondPerm.ID()}) {
		t.Fatalf("opponent candidates = %v, want only their creatures", b.calls[0].possible)
	}
}

func TestPromptTargetsForList_RoutesEachTargetToItsChooser(t *testing.T) {
	g, a, b := newTriggerTargetGame()
	source := NewInstant("Prompt Source", "{0}")
	source.SetOwner(a.PlayerID())
	mine := NewCreature("Mine", "{1}", 1, 1)
	mine.SetOwner(a.PlayerID())
	minePerm := g.PutOnBattlefield(mine, a.PlayerID())
	theirs := NewCreature("Theirs", "{1}", 1, 1)
	theirs.SetOwner(b.PlayerID())
	theirsPerm := g.PutOnBattlefield(theirs, b.PlayerID())

	a.chooseQueue = [][]uuid.UUID{{minePerm.ID()}}
	b.chooseQueue = [][]uuid.UUID{{theirsPerm.ID()}}
	got := g.promptTargetsForList(a.PlayerID(), source, []Target{
		TargetCreatureYouControl(),
		TargetOpponentChoice(TargetCreatureOpponentControls()),
	})
	if want := []uuid.UUID{minePerm.ID(), theirsPerm.ID()}; !slices.Equal(got, want) {
		t.Fatalf("targets = %v, want %v", got, want)
	}
	if len(a.calls) != 1 || len(b.calls) != 1 {
		t.Fatalf("prompt counts = controller %d, opponent %d; want 1 each", len(a.calls), len(b.calls))
	}
}

func TestCastSpell_OpponentChoiceCannotSelectIllegalTarget(t *testing.T) {
	g, a, b := newTriggerTargetGame()

	spell := NewInstant("Choice Spell", "{0}", NewMultiTargetSpell(
		[]Target{
			TargetCreatureYouControl(),
			TargetOpponentChoice(TargetCreatureOpponentControls()),
		},
		FuncEffect("record targets", EffectProperties{}, func(_ *Game, _, _ uuid.UUID, _ []uuid.UUID) error { return nil }),
	))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)

	mine := NewCreature("Mine", "{1}", 1, 1)
	mine.SetOwner(a.PlayerID())
	minePerm := g.PutOnBattlefield(mine, a.PlayerID())
	theirs := NewCreature("Theirs", "{1}", 1, 1)
	theirs.SetOwner(b.PlayerID())
	theirsPerm := g.PutOnBattlefield(theirs, b.PlayerID())

	// A malformed chooser response cannot escape the wrapped target's legal
	// candidate set. The required target falls back to the sole legal choice.
	b.chooseQueue = [][]uuid.UUID{{minePerm.ID()}}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{minePerm.ID(), theirsPerm.ID()}); err != nil {
		t.Fatalf("CastSpellByName: %v", err)
	}

	obj := g.Stack().Peek()
	if obj == nil {
		t.Fatal("expected spell on stack")
	}
	if got, want := obj.Targets, []uuid.UUID{minePerm.ID(), theirsPerm.ID()}; !slices.Equal(got, want) {
		t.Fatalf("targets = %v, want %v", got, want)
	}
}

func TestOpponentChosenTarget_RechecksControlRestrictionAtResolution(t *testing.T) {
	g, a, b := newTriggerTargetGame()
	source := NewArtifact("Arena Analog", "{0}",
		WithActivatedAbility(
			tapThenFightPipeline(),
			Tap(),
			WithTarget(TargetCreatureYouControl()),
			WithTarget(TargetOpponentChoice(TargetCreatureOpponentControls())),
		),
	)
	source.SetOwner(a.PlayerID())
	sourcePerm := g.PutOnBattlefield(source, a.PlayerID())
	first := NewCreature("First", "{1}", 2, 3)
	first.SetOwner(a.PlayerID())
	firstPerm := g.PutOnBattlefield(first, a.PlayerID())
	second := NewCreature("Second", "{1}", 3, 4)
	second.SetOwner(b.PlayerID())
	secondPerm := g.PutOnBattlefield(second, b.PlayerID())

	b.chooseQueue = [][]uuid.UUID{{secondPerm.ID()}}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), sourcePerm.ID(), 0, []uuid.UUID{firstPerm.ID(), secondPerm.ID()}); err != nil {
		t.Fatalf("ActivateAbilityByIndex: %v", err)
	}
	g.AddControlEffect(ControlEffectSpec{
		SourceID: a.PlayerID(), TargetID: secondPerm.ID(), ControllerID: a.PlayerID(), Duration: Indefinite,
	})
	g.ResolveTopOfStack()

	if !firstPerm.Tapped {
		t.Fatal("still-legal first target was not tapped")
	}
	if secondPerm.Tapped {
		t.Fatal("second target was tapped after becoming illegal")
	}
	if firstPerm.Damage != 0 || secondPerm.Damage != 0 {
		t.Fatalf("damage = (%d, %d), want no fight", firstPerm.Damage, secondPerm.Damage)
	}
}

func TestTargetAuraAttachedToCreatureYouControl(t *testing.T) {
	g, a, b := newTriggerTargetGame()

	myCreature := NewCreature("My Creature", "{1}", 1, 1)
	myCreature.SetOwner(a.PlayerID())
	myCreaturePerm := g.PutOnBattlefield(myCreature, a.PlayerID())

	oppCreature := NewCreature("Opp Creature", "{1}", 1, 1)
	oppCreature.SetOwner(b.PlayerID())
	oppCreaturePerm := g.PutOnBattlefield(oppCreature, b.PlayerID())

	auraOnMine := NewAura("Aura On Mine", "{W}")
	auraOnMine.SetOwner(a.PlayerID())
	auraOnMinePerm := g.PutOnBattlefield(auraOnMine, a.PlayerID())
	auraOnMinePerm.AttachedTo = myCreaturePerm.ID()

	auraOnOpp := NewAura("Aura On Opp", "{W}")
	auraOnOpp.SetOwner(a.PlayerID())
	auraOnOppPerm := g.PutOnBattlefield(auraOnOpp, a.PlayerID())
	auraOnOppPerm.AttachedTo = oppCreaturePerm.ID()

	targetSpec := TargetAuraAttachedToCreatureYouControl()
	possible := targetSpec.Possible(a.PlayerID(), nil, g)

	if !slices.Contains(possible, auraOnMinePerm.ID()) {
		t.Fatalf("expected possible targets to contain auraOnMine, got %v", possible)
	}
	if slices.Contains(possible, auraOnOppPerm.ID()) {
		t.Fatalf("expected possible targets not to contain auraOnOpp, got %v", possible)
	}
}
