package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

// --- from cast_condition_test.go ---
func TestCastConditionPublicDefinition(t *testing.T) {
	card := dsl.NewInstant("Cast condition test", "", dsl.Spell(
		dsl.IfCast(dsl.ControlledCreatureSubtypeAtCast("Dragon"), dsl.GainLife(dsl.You, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestRevealedCastConditionPublicDefinition(t *testing.T) {
	paid := dsl.BindNumber("reveal paid")
	revealed := dsl.BindCard("revealed Dragon")
	card := dsl.NewInstant("Reveal condition test", "", dsl.WithAbility(dsl.Spell(
		dsl.IfCast(dsl.RevealedSubtypeAtCast("Dragon"), dsl.GainLife(dsl.You, dsl.Fixed(1))))),
		dsl.WithAdditionalCost(dsl.OptionalRevealFromHandCost(
			dsl.HasSubTypeCardFilter("Dragon"), "Reveal a Dragon card", "Reveal?", paid, revealed)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestAnyCastPublicDefinition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	controller := dsl.BindPlayer("creature controller")
	card := dsl.NewInstant("Dragon condition test", "", dsl.Spell(dsl.Sequence(
		dsl.SnapshotPlayer(dsl.ControllerOf(target), controller),
		dsl.DealDamageFrom(dsl.This, target, dsl.Fixed(3)),
		dsl.IfCast(dsl.AnyCast(dsl.RevealedSubtypeAtCast("Dragon"),
			dsl.ControlledCreatureSubtypeAtCast("Dragon")),
			dsl.DealDamageFrom(dsl.This, dsl.FromPlayer(controller), dsl.Fixed(3))),
	), dsl.WithTargets(target)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from control_since_turn_test.go ---
func TestControlledSinceTurnStartDeclaration(t *testing.T) {
	card := dsl.NewArtifact("Gate", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(), dsl.GainLife(dsl.You, dsl.Fixed(1)),
		dsl.ActivationIf(dsl.ControlledSinceTurnStart(dsl.This)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from delayed_mode_test.go ---
func TestDelayedModesAuthorSurface(t *testing.T) {
	target := dsl.TargetPlayer("future target")
	options := []dsl.ModalDelayedStepOption{
		dsl.ModeOf("Life", dsl.GainLife(target, dsl.Fixed(2)), dsl.WithTargets(target)),
		dsl.ModeOf("Other", dsl.GainLife(dsl.You, dsl.Fixed(1))),
		dsl.WithInterveningIf(dsl.Controls(dsl.IsArtifact)),
	}
	body := dsl.AtNextModes(dsl.EachEndStep, options...)
	card := dsl.NewSorcery("Schedule", "", dsl.Spell(body))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if mage.InspectDefinition(body).Role != "delayed-step" || body.Properties().Outcome != mage.OutcomeBenefit {
		t.Fatal("missing metadata")
	}
}

// --- from delayed_target_test.go ---
func TestDelayedTargetAuthorSurface(t *testing.T) {
	target := dsl.TargetPlayer("future player")
	options := []dsl.DelayedStepOption{dsl.WithTargets(target), dsl.WithInterveningIf(dsl.Controls(dsl.IsArtifact))}
	body := dsl.AtNext(dsl.EachEndStep, dsl.GainLife(target, dsl.Fixed(2)), options...)
	card := dsl.NewSorcery("Schedule", "", dsl.Spell(body))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if mage.InspectDefinition(body).Role != "delayed-step" {
		t.Fatal("missing delayed descriptor")
	}
}

// --- from hand_size_condition_test.go ---
func TestControllerHandSizeIsPublicDeclaration(t *testing.T) {
	card := dsl.NewLand("Hand gate", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.TapCost(dsl.This)),
		dsl.Draw(dsl.You, dsl.Fixed(1)),
		dsl.ActivationIf(dsl.ControllerHandSizeIs(7)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from life_gain_history_condition_test.go ---
func TestControllerGainedLifeThisTurnPublicDeclaration(t *testing.T) {
	card := dsl.NewInstant("Conditional", "", dsl.WithAbility(dsl.Spell(dsl.IfState(
		dsl.ControllerGainedLifeThisTurn(), dsl.Draw(dsl.You, dsl.Fixed(1))))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from player_controls_test.go ---
func TestPlayerControlsPublicDeclaration(t *testing.T) {
	card := dsl.NewInstant("Watch the active player", "", dsl.WithAbility(dsl.Spell(
		dsl.IfState(dsl.PlayerControls(dsl.ActivePlayer, dsl.IsLand),
			dsl.GainLife(dsl.You, dsl.Fixed(1))))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from random_discard_test.go ---
func TestTypedRandomDiscardComposition(t *testing.T) {
	player := dsl.TargetPlayer("player")
	body := dsl.DiscardAtRandom(player, dsl.ValueAtResolution(dsl.XValue()))
	for _, card := range []mage.Card{
		dsl.NewSorcery("Spell", "{X}{B}", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(player)))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body, dsl.WithTargets(player)))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body, dsl.WithTargets(player)))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Discard", body, dsl.WithTargets(player)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from starting_life_condition_test.go ---
func TestControllerLifeAtLeastStartingLifePublicDeclaration(t *testing.T) {
	card := dsl.NewEnchantment("Life watcher", "", dsl.WithAbility(dsl.Static(dsl.While(
		dsl.ControllerLifeAtLeastStartingLife(),
		dsl.ModifyEachPT(dsl.ControlledPermanents(dsl.IsCreature), dsl.Fixed(1), dsl.Fixed(1)),
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
