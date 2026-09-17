package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedResolutionComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	member := dsl.BindPermanent("member")
	effect := dsl.FlipCoin(dsl.You,
		dsl.OnWin(dsl.ForEach(dsl.ControlledPermanents(dsl.IsCreature), member,
			dsl.Until(core.EndOfTurn, dsl.ModifyPT(dsl.From(member), dsl.Fixed(1), dsl.Fixed(1))),
		)),
		dsl.OnLoss(dsl.UnlessPaid(dsl.ControllerOf(target), dsl.Costs(dsl.Mana("{1}")), dsl.Destroy(target))),
	)
	card := mage.NewCreature("Composition", "", 1, 1, mage.WithAbility(dsl.Activate(
		dsl.Costs(), effect, dsl.WithTargets(target), dsl.OnlyAsSorcery(), dsl.ActivationLimit(1),
	)))
	if err := mage.ValidateCardDefinition("Composition", card); err != nil {
		t.Fatal(err)
	}
	metadata := dsl.InspectDefinition(effect)
	if metadata.Role != "coin-flip" || !metadata.Properties.Destroys || !metadata.Properties.Mass {
		t.Fatalf("composition metadata lost: %+v", metadata)
	}
}

func TestTypedLayerComposition(t *testing.T) {
	target := dsl.TargetPermanent("permanent", dsl.IsCreature)
	effect := dsl.Until(core.EndOfTurn,
		dsl.SetColors(target, core.Blue),
		dsl.SetSubtypeFamily(target, dsl.SubtypeCreature, "Human"),
		dsl.AddSubtypeFamily(target, dsl.SubtypeCreature, "Wizard"),
	)
	card := mage.NewCreature("Layers", "", 1, 1,
		mage.WithAbility(dsl.Activate(dsl.Costs(), effect, dsl.WithTargets(target))),
		mage.WithAbility(dsl.Static(dsl.While(dsl.Controls(dsl.IsCreature),
			dsl.SetEachColors(dsl.ControlledPermanents(dsl.IsCreature), core.Green),
			dsl.GrantEachKeyword(dsl.ControlledPermanents(dsl.IsCreature), core.Flying),
		))),
	)
	if err := mage.ValidateCardDefinition("Layers", card); err != nil {
		t.Fatal(err)
	}
}

func TestTypedSourceDurationComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	saved := dsl.BindPermanent("saved")
	effect := dsl.Sequence(dsl.Snapshot(target, saved),
		dsl.UntilSourceLeaves(dsl.This,
			dsl.SetColors(dsl.From(saved), core.Blue),
			dsl.ModifyPT(dsl.From(saved), dsl.Fixed(2), dsl.Fixed(3)),
			dsl.GrantKeyword(dsl.From(saved), core.Flying),
		),
	)
	cards := []mage.Card{
		mage.NewCreature("Activation", "", 1, 1, mage.WithAbility(dsl.Activate(dsl.Costs(), effect, dsl.WithTargets(target)))),
		mage.NewCreature("Trigger", "", 1, 1, mage.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), effect, dsl.WithTargets(target)))),
		mage.NewCreature("Modes", "", 1, 1, mage.WithAbility(dsl.ActivateModes(dsl.Costs(), dsl.ModeOf("Modify", effect, dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		mage.NewCreature("Loop", "", 1, 1, mage.WithAbility(dsl.Activate(dsl.Costs(), dsl.ForEach(dsl.ControlledPermanents(dsl.IsCreature), saved,
			dsl.UntilSourceLeaves(dsl.This, dsl.ModifyPT(dsl.From(saved), dsl.Fixed(1), dsl.Fixed(1))),
		)))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition("Source duration", card); err != nil {
			t.Fatal(err)
		}
	}
	info := dsl.InspectDefinition(effect).Children[1]
	if info.Role != "until-source-leaves" || len(info.Children) != 4 || info.Properties.PowerBoost != 2 || info.Properties.ToughnessBoost != 3 || info.Properties.GrantedKeyword != core.Flying {
		t.Fatalf("duration metadata = %+v", info)
	}
}

func TestTypedManaComposition(t *testing.T) {
	ability := dsl.Activate(dsl.Costs(dsl.TapCost(dsl.This)),
		dsl.Sequence(dsl.AddMana(core.Blue, 1), dsl.Sequence(dsl.AddMana(core.Green, 1))),
	)
	card := mage.NewLand("Typed mana", mage.WithAbility(ability))
	if err := mage.ValidateCardDefinition("Typed mana", card); err != nil {
		t.Fatal(err)
	}
	productions := mage.ManaProductionsForAbility(ability)
	if len(productions) != 2 || productions[0].Color != core.Blue || productions[1].Color != core.Green {
		t.Fatalf("productions = %+v", productions)
	}
}

func TestTypedDelayedComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	saved := dsl.BindPermanent("saved")
	effect := dsl.Sequence(dsl.Snapshot(target, saved),
		dsl.AtNext(dsl.EachEndStep, dsl.Destroy(dsl.From(saved))),
	)
	card := mage.NewInstant("Delayed removal", "", mage.WithAbility(dsl.Spell(effect, dsl.WithTargets(target))))
	if err := mage.ValidateCardDefinition("Delayed removal", card); err != nil {
		t.Fatal(err)
	}
	metadata := dsl.InspectDefinition(effect)
	if !metadata.Properties.Destroys || metadata.Children[1].Role != "delayed-step" {
		t.Fatal("delayed composition lost metadata")
	}
}

func TestTypedLifeComposition(t *testing.T) {
	gain := dsl.GainLife(dsl.You, dsl.Fixed(2))
	member := dsl.BindPermanent("member")
	cards := []mage.Card{
		mage.NewSorcery("Direct", "", mage.WithAbility(dsl.Spell(gain))),
		mage.NewSorcery("Sequence", "", mage.WithAbility(dsl.Spell(dsl.Sequence(gain, dsl.LoseLife(dsl.You, dsl.Fixed(1)))))),
		mage.NewSorcery("Loop", "", mage.WithAbility(dsl.Spell(dsl.ForEach(dsl.ControlledPermanents(dsl.IsCreature), member, gain)))),
		mage.NewSorcery("Modes", "", mage.WithAbility(dsl.SpellModes(dsl.ModeOf("Life", gain), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		mage.NewCreature("Trigger", "", 1, 1, mage.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), gain))),
		mage.NewCreature("Activation", "", 1, 1, mage.WithAbility(dsl.Activate(dsl.Costs(), gain))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition("Life composition", card); err != nil {
			t.Fatal(err)
		}
	}
	info := dsl.InspectDefinition(dsl.Sequence(gain, dsl.LoseLife(dsl.You, dsl.Fixed(1))))
	if len(info.Children) != 2 || info.Children[0].Properties.LifeGain != 2 || info.Children[0].Text != "gain 2 life for you" {
		t.Fatalf("life metadata = %+v", info)
	}
}
