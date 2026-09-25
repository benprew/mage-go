package dsl_test

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestGlobalCombatPreventionComposition(t *testing.T) {
	body := dsl.PreventAllCombatDamage()
	cards := []mage.Card{
		dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewInstant("Sequence", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(body, dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewInstant("Modes", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Prevent", body), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	metadata := dsl.InspectDefinition(body)
	if metadata.Role != "install-combat-prevention" || metadata.Text != "prevent all combat damage that would be dealt this turn" || metadata.Properties.Outcome != mage.OutcomeBenefit {
		t.Fatalf("prevention metadata lost: %+v", metadata)
	}
}

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

func TestTypedTemporaryKeywordComposition(t *testing.T) {
	for _, keyword := range []core.Attr{core.Flying, core.Banding, core.AttrMustBeBlockedIfAble} {
		target := dsl.TargetPermanent("creature", dsl.IsCreature)
		body := dsl.Until(core.EndOfTurn, dsl.GrantKeyword(target, keyword))
		saved := dsl.BindPermanent("chosen creature")
		cards := []mage.Card{
			dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.TapCost(dsl.This)), body, dsl.WithTargets(target)))),
			dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target)))),
			dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body, dsl.WithTargets(target)))),
			dsl.NewInstant("Modes", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Grant", body, dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
			dsl.NewInstant("Random", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(
				dsl.ChooseRandomPermanent(dsl.MatchingPermanents(dsl.IsCreature), saved),
				dsl.Until(core.EndOfTurn, dsl.GrantKeyword(dsl.From(saved), keyword)),
			)))),
		}
		for _, card := range cards {
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
		}
		info := dsl.InspectDefinition(body)
		outcome := mage.OutcomeBenefit
		if keyword == core.AttrMustBeBlockedIfAble {
			outcome = mage.OutcomeDetriment
		}
		if info.Role != "until" || info.Properties.GrantedKeyword != keyword || info.Properties.Outcome != outcome {
			t.Fatalf("keyword metadata = %+v", info)
		}
		if !strings.Contains(info.Text, keyword.String()) || !strings.Contains(info.Text, "until end of turn") {
			t.Fatalf("missing keyword or duration in diagnostics: %q", info.Text)
		}
	}
}

func TestTypedTemporaryPTComposition(t *testing.T) {
	for _, delta := range [][2]int{{3, 3}, {0, 3}, {-2, 0}} {
		target := dsl.TargetPermanent("creature", dsl.IsCreature)
		body := dsl.Until(core.EndOfTurn, dsl.ModifyPT(target, dsl.Fixed(delta[0]), dsl.Fixed(delta[1])))
		saved := dsl.BindPermanent("chosen creature")
		cards := []mage.Card{
			dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.TapCost(dsl.This)), body, dsl.WithTargets(target)))),
			dsl.NewSorcery("Pump and required block", "", dsl.Spell(dsl.Sequence(
				dsl.Until(core.EndOfTurn,
					dsl.ModifyPT(target, dsl.Fixed(delta[0]), dsl.Fixed(delta[1])),
					dsl.GrantKeyword(target, core.Trample)),
				dsl.Until(core.EndOfTurn, dsl.GrantKeyword(target, core.AttrMustBeBlockedIfAble)),
			), dsl.WithTargets(target))),
			dsl.NewInstant("Keyword cantrip", "", dsl.Spell(dsl.Sequence(
				dsl.Until(core.EndOfTurn,
					dsl.ModifyPT(target, dsl.Fixed(delta[0]), dsl.Fixed(delta[1])),
					dsl.GrantKeyword(target, core.Trample),
				),
				dsl.Draw(dsl.You, dsl.Fixed(1)),
			), dsl.WithTargets(target))),
			dsl.NewInstant("Cantrip", "", dsl.Spell(dsl.Sequence(body, dsl.Draw(dsl.You, dsl.Fixed(1))), dsl.WithTargets(target))),
			dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target)))),
			dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body, dsl.WithTargets(target)))),
			dsl.NewInstant("Modes", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Grant", body, dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
			dsl.NewInstant("Random", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(
				dsl.ChooseRandomPermanent(dsl.MatchingPermanents(dsl.IsCreature), saved),
				dsl.Until(core.EndOfTurn, dsl.ModifyPT(dsl.From(saved), dsl.Fixed(delta[0]), dsl.Fixed(delta[1]))),
			)))),
		}
		for _, card := range cards {
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
		}
		info := dsl.InspectDefinition(body)
		wantOutcome := mage.OutcomeBenefit
		if delta[0] < 0 || delta[1] < 0 {
			wantOutcome = mage.OutcomeDetriment
		}
		if info.Role != "until" || info.Properties.PowerBoost != delta[0] || info.Properties.ToughnessBoost != delta[1] || info.Properties.Outcome != wantOutcome {
			t.Fatalf("power/toughness metadata = %+v", info)
		}
		if !strings.Contains(info.Text, "modify power and toughness") || !strings.Contains(info.Text, "until end of turn") {
			t.Fatalf("missing modification or duration in diagnostics: %q", info.Text)
		}
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
		dsl.Sequence(dsl.AddManaAmount(core.Blue, dsl.Fixed(1)), dsl.Sequence(dsl.AddManaAmount(core.Green, dsl.Fixed(1)))),
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

func TestTypedSourceUntapComposition(t *testing.T) {
	saved := dsl.BindPermanent("source")
	body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.Untap(dsl.From(saved)))
	cards := []mage.Card{
		dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Untap(dsl.This)))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.At(dsl.YourUpkeep, body))),
		dsl.NewCreature("Modes", "", 1, 1, dsl.WithAbility(dsl.ActivateModes(dsl.Costs(),
			dsl.ModeOf("Untap", body), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	info := dsl.InspectDefinition(body)
	if info.Role != "sequence" || info.Properties.Outcome != mage.OutcomeBenefit {
		t.Fatalf("untap metadata = %+v", info)
	}
}

func TestTypedTargetUntapComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	saved := dsl.BindPermanent("random creature")
	random := dsl.Sequence(dsl.ChooseRandomPermanent(dsl.MatchingPermanents(dsl.IsCreature), saved), dsl.Untap(dsl.From(saved)))
	cards := []mage.Card{
		dsl.NewInstant("Spell", "", dsl.Spell(dsl.Untap(target), dsl.WithTargets(target))),
		dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.TapCost(dsl.This)), dsl.Untap(target), dsl.WithTargets(target)))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.At(dsl.YourUpkeep, dsl.Untap(target), dsl.WithTargets(target)))),
		dsl.NewInstant("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Untap", dsl.Untap(target), dsl.WithTargets(target)), dsl.ModeOf("Random", random)))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	info := dsl.InspectDefinition(random)
	if info.Properties.Outcome != mage.OutcomeBenefit || len(info.Children) != 2 || info.Children[0].Role != "random-permanent-choice" {
		t.Fatalf("metadata = %+v", info)
	}
}

func TestTypedCombatPreventionComposition(t *testing.T) {
	target := dsl.TargetPermanent("attacker", dsl.And(dsl.IsCreature, dsl.IsAttacking), dsl.TargetYouControl())
	member := dsl.BindPermanent("member")
	group := dsl.TargetPermanents("creatures", 0, 2, dsl.IsCreature, dsl.TargetYouControl())
	cards := []mage.Card{
		dsl.NewInstant("Target", "", dsl.Spell(dsl.Sequence(dsl.Untap(target), dsl.PreventCombatDamageToAndBy(target)), dsl.WithTargets(target))),
		dsl.NewInstant("Collection", "", dsl.Spell(dsl.ForEach(group, member, dsl.PreventCombatDamageToAndBy(dsl.From(member))), dsl.WithTargets(group))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.At(dsl.YourUpkeep, dsl.PreventCombatDamageToAndBy(dsl.This)))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	for _, subject := range []dsl.PermanentRef{nil, dsl.TargetPermanent("undeclared", dsl.IsCreature)} {
		card := dsl.NewInstant("Invalid", "", dsl.Spell(dsl.PreventCombatDamageToAndBy(subject)))
		if err := mage.ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid prevention subject accepted")
		}
	}
	info := dsl.InspectDefinition(dsl.PreventCombatDamageToAndBy(target))
	if info.Role != "install-combat-prevention" || info.Properties.Outcome != mage.OutcomeBenefit || len(info.Children) != 1 {
		t.Fatalf("metadata=%+v", info)
	}
}

func TestTypedTapChoiceComposition(t *testing.T) {
	target := dsl.TargetPermanent("artifact", dsl.IsArtifact, dsl.TargetOpponentControls())
	member := dsl.BindPermanent("member")
	body := dsl.MayTapOrUntap(target)
	cards := []mage.Card{
		dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target)))),
		dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body, dsl.WithTargets(target)))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body, dsl.WithTargets(target)))),
		dsl.NewInstant("Modes", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Choice", body, dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewInstant("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsArtifact), member, dsl.MayTapOrUntap(dsl.From(member)))))),
	}
	for _, c := range cards {
		if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
			t.Fatal(err)
		}
	}
	metadata := dsl.InspectDefinition(body)
	if metadata.Role != "may-tap-or-untap" || len(metadata.Children) != 3 {
		t.Fatalf("missing choice structure: %+v", metadata)
	}
}

func TestTypedGatheredTapComposition(t *testing.T) {
	target := dsl.TargetPermanent("opponent creature", dsl.IsCreature, dsl.TargetOpponentControls(), dsl.TargetChosenByOpponent())
	saved := dsl.BindPermanent("saved")
	body := dsl.Sequence(dsl.Snapshot(target, saved), dsl.Tap(dsl.From(saved)))
	ability := dsl.Activate(dsl.Costs(dsl.TapCost(dsl.This)), body, dsl.WithTargets(target))
	card := dsl.NewArtifact("Typed tap", "", dsl.WithAbility(ability))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	info := dsl.InspectDefinition(body)
	if !info.Properties.Taps || info.Properties.Outcome != mage.OutcomeDetriment {
		t.Fatalf("tap metadata = %+v", info)
	}
	var chooser bool
	var visit func(mage.DefinitionMetadata)
	visit = func(node mage.DefinitionMetadata) {
		if node.Role == "target-chooser" && node.Text == "opponent" {
			chooser = true
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(dsl.InspectDefinition(ability))
	if !chooser {
		t.Fatal("inspection lost opponent chooser")
	}
}

func TestTypedDestructionComposition(t *testing.T) {
	for _, destroy := range []func(dsl.PermanentRef) dsl.ResolvingEffect{dsl.Destroy, dsl.DestroyWithoutRegeneration} {
		target := dsl.TargetPermanent("permanent", dsl.Or(dsl.IsArtifact, dsl.IsEnchantment))
		saved := dsl.BindPermanent("chosen permanent")
		body := destroy(target)
		cards := []mage.Card{
			dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target)))),
			dsl.NewSorcery("Sacrifice spell", "{B}", dsl.Spell(body, dsl.WithTargets(target)),
				dsl.WithAdditionalCost(dsl.SacrificeCreatureCost())),
			dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body, dsl.WithTargets(target)))),
			dsl.NewCreature("Sacrifice activation", "", 2, 1, dsl.WithAbility(dsl.Activate(
				dsl.Costs(dsl.Mana("{R}"), dsl.SacrificeSourceCost()), body, dsl.WithTargets(target)))),
			dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body, dsl.WithTargets(target)))),
			dsl.NewInstant("Random", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(
				dsl.ChooseRandomPermanent(dsl.MatchingPermanents(dsl.IsCreature), saved), destroy(dsl.From(saved)),
			)))),
		}
		for _, card := range cards {
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
		}
		info := dsl.InspectDefinition(body)
		if !info.Properties.Destroys || info.Properties.Outcome != mage.OutcomeDetriment || len(info.Children) != 1 || info.Children[0].Role != "permanent-ref" {
			t.Fatalf("destruction metadata = %+v", info)
		}
	}
}

func TestOpponentNonlandEntryDestructionComposition(t *testing.T) {
	target := dsl.TargetPermanent("nonland permanent", dsl.Not(dsl.IsLand), dsl.TargetOpponentControls())
	card := dsl.NewCreature("Entry destruction", "{7}", 3, 3,
		dsl.WithCardType(core.TypeArtifact), dsl.WithSubTypes("Golem"),
		dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), dsl.Destroy(target), dsl.WithTargets(target))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestTypedReturnToHandComposition(t *testing.T) {
	target := dsl.TargetPermanent("permanent", dsl.Or(dsl.IsCreature, dsl.IsLand))
	body := dsl.ReturnToHand(target)
	saved := dsl.BindPermanent("saved")
	opponent := dsl.TargetPermanent("creature", dsl.IsCreature, dsl.TargetOpponentControls())
	wizard := dsl.TargetPermanent("wizard", dsl.HasSubType("Wizard"), dsl.TargetYouControl())
	cards := []mage.Card{
		dsl.NewArtifact("Sacrifice return and draw", "{1}",
			dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.Mana("{U}"), dsl.SacrificeSourceCost()), body, dsl.WithTargets(target))),
			dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.Mana("{1}"), dsl.SacrificeSourceCost()), dsl.Draw(dsl.You, dsl.Fixed(1))))),
		dsl.NewLand("Wizard return", dsl.WithAbility(dsl.Activate(
			dsl.Costs(dsl.Mana("{1}{U}"), dsl.TapCost(dsl.This)),
			dsl.ReturnToHand(wizard), dsl.WithTargets(wizard)))),
		dsl.NewCreature("Chosen entry return", "", 3, 3, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), dsl.Sequence(
			dsl.ChoosePermanent(dsl.You, dsl.ControlledPermanents(dsl.AnyPermanent), saved),
			dsl.ReturnToHand(dsl.From(saved)),
		)))),
		dsl.NewCreature("Opponent entry return", "", 2, 2, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), dsl.ReturnToHand(opponent), dsl.WithTargets(opponent)))),
		dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target)))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body, dsl.WithTargets(target)))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body, dsl.WithTargets(target)))),
		dsl.NewInstant("Modes", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Return", body, dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewInstant("Saved", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(dsl.ChooseRandomPermanent(dsl.MatchingPermanents(dsl.IsCreature), saved), dsl.ReturnToHand(dsl.From(saved)))))),
		dsl.NewInstant("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), saved, dsl.ReturnToHand(dsl.From(saved)))))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	info := dsl.InspectDefinition(body)
	if !info.Properties.IsBounce || info.Properties.Outcome != mage.OutcomeDetriment || !strings.Contains(info.Text, "owner's hand") {
		t.Fatalf("return metadata = %+v", info)
	}
}

func TestTypedExileComposition(t *testing.T) {
	target := dsl.TargetPermanent("permanent", dsl.Or(dsl.IsCreature, dsl.IsLand))
	body := dsl.Exile(target)
	saved := dsl.BindPermanent("saved")
	cards := []mage.Card{
		dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target)))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body, dsl.WithTargets(target)))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body, dsl.WithTargets(target)))),
		dsl.NewInstant("Modes", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Exile", body, dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewInstant("Saved", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(dsl.ChooseRandomPermanent(dsl.MatchingPermanents(dsl.IsCreature), saved), dsl.Exile(dsl.From(saved)))))),
		dsl.NewInstant("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), saved, dsl.Exile(dsl.From(saved)))))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	info := dsl.InspectDefinition(body)
	if info.Properties.Outcome != mage.OutcomeDetriment || !strings.Contains(info.Text, "exile") {
		t.Fatalf("exile metadata = %+v", info)
	}
}

func TestDrawInLegacyContainers(t *testing.T) {
	draw := dsl.Draw(dsl.You, dsl.Fixed(3))
	cards := []mage.Card{
		dsl.NewInstant("Draw list", "", dsl.NewSpellAbility(draw)),
		dsl.NewInstant("Draw composite", "", dsl.NewSpellAbility(dsl.CompositeEffects("draw", draw))),
		dsl.NewInstant("Draw payment", "", dsl.NewSpellAbility(dsl.UnlessTargetPays(dsl.SelectController(), dsl.ManaCostOf("{1}"), "Pay?", draw))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		spell := mage.UnwrapAbility(card.Abilities()[0]).(*mage.SpellAbility)
		if got := spell.Effects()[0].Properties().DrawCount; got != 3 {
			t.Fatalf("%s draw metadata = %d", card.Name(), got)
		}
	}
}

func TestWitchHunterReturnComposition(t *testing.T) {
	creature := dsl.TargetPermanent("creature", dsl.IsCreature, dsl.TargetOpponentControls())
	ability := dsl.Activate(dsl.Costs(dsl.Mana("{1}{W}{W}"), dsl.TapCost(dsl.This)), dsl.ReturnToHand(creature), dsl.WithTargets(creature))
	card := dsl.NewCreature("Return fixture", "{2}{W}{W}", 1, 1, dsl.WithAbility(ability))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if !dsl.InspectDefinition(dsl.ReturnToHand(creature)).Properties.IsBounce {
		t.Fatal("return effect lost bounce metadata")
	}
}

func TestManaValueComposition(t *testing.T) {
	target := dsl.TargetPermanent("artifact", dsl.IsArtifact)
	number := dsl.BindNumber("mana value")
	body := dsl.Sequence(dsl.SnapshotNumber(dsl.ManaValueOf(target), number), dsl.Destroy(target), dsl.GainLife(dsl.You, dsl.FromNumber(number)))
	member := dsl.BindPermanent("member")
	cards := []mage.Card{
		dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target)))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body, dsl.WithTargets(target)))),
		dsl.NewArtifact("Trigger", "", dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body, dsl.WithTargets(target)))),
		dsl.NewInstant("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Offering", body, dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewInstant("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsArtifact), member, dsl.GainLife(dsl.You, dsl.ManaValueOf(dsl.From(member))))))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	info := dsl.InspectDefinition(body)
	if !info.Properties.Destroys || info.Children[0].Children[0].Text != "mana value of subject" {
		t.Fatalf("metadata = %+v", info)
	}
}

func TestCollectionNoRegenerationComposition(t *testing.T) {
	body := dsl.DestroyEachWithoutRegeneration(dsl.MatchingPermanents(dsl.IsCreature))
	cards := []mage.Card{
		dsl.NewSorcery("Spell", "", dsl.Spell(body)),
		dsl.NewSorcery("Sequence", "", dsl.Spell(dsl.Sequence(body, dsl.Draw(dsl.You, dsl.Fixed(1))))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewSorcery("Modes", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Destroy", body), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	invalid := dsl.NewSorcery("Missing set", "", dsl.Spell(dsl.DestroyEachWithoutRegeneration(nil)))
	if err := mage.ValidateCardDefinition(invalid.Name(), invalid); err == nil {
		t.Fatal("accepted nil collection")
	}
}

func TestBlockingRestrictionComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	body := dsl.Until(dsl.EndOfTurn, dsl.CantBeBlocked(target))
	card := dsl.NewInstant("Blocking restriction", "", dsl.Spell(body, dsl.WithTargets(target)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	info := dsl.InspectDefinition(body)
	if info.Properties.Outcome != mage.OutcomeBenefit || len(info.Children) != 1 || info.Children[0].Role != "continuous-modification" {
		t.Fatalf("metadata = %+v", info)
	}
	for _, subject := range []dsl.PermanentRef{nil, target} {
		invalid := dsl.NewInstant("Invalid", "", dsl.Spell(dsl.Until(dsl.EndOfTurn, dsl.CantBeBlocked(subject))))
		if err := mage.ValidateCardDefinition(invalid.Name(), invalid); err == nil {
			t.Fatal("invalid subject accepted")
		}
	}
}

func TestBasePowerToughnessComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature, dsl.TargetOtherThanSource())
	body := dsl.Until(core.EndOfTurn, dsl.SetBasePowerToughness(target, dsl.Fixed(0), dsl.Fixed(2)))
	for _, card := range []mage.Card{
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.TapCost(dsl.This)), body, dsl.WithTargets(target)))),
		dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target)))),
		dsl.NewCreature("Static", "", 1, 1, dsl.WithAbility(dsl.Static(dsl.While(dsl.Controls(dsl.IsCreature), dsl.SetBasePowerToughness(dsl.This, dsl.Fixed(0), dsl.Fixed(2)))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	info := dsl.InspectDefinition(body)
	if info.Properties.Outcome != mage.OutcomeDetriment || info.Properties.PowerBoost != 0 || info.Properties.ToughnessBoost != 0 {
		t.Fatalf("metadata = %+v", info)
	}
}

func TestBasePowerComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature, dsl.TargetOtherThanSource())
	body := dsl.Until(core.EndOfTurn, dsl.SetBasePower(target, dsl.Fixed(0)))
	for _, card := range []mage.Card{
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.TapCost(dsl.This)), body, dsl.WithTargets(target)))),
		dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target)))),
		dsl.NewCreature("Static", "", 1, 1, dsl.WithAbility(dsl.Static(dsl.While(dsl.Controls(dsl.IsCreature), dsl.SetBasePower(dsl.This, dsl.Fixed(0)))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	info := dsl.InspectDefinition(body)
	if info.Properties.Outcome != mage.OutcomeDetriment || info.Properties.PowerBoost != 0 || info.Properties.ToughnessBoost != 0 {
		t.Fatalf("metadata = %+v", info)
	}
}

func TestLandPlayAllowanceComposition(t *testing.T) {
	body := dsl.GrantLandPlays(dsl.You, dsl.Fixed(1))
	member := dsl.BindPermanent("member")
	cards := []mage.Card{
		dsl.NewSorcery("Spell", "", dsl.WithAbility(dsl.Spell(body))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewSorcery("Sequence", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(body, dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Land plays", body), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewSorcery("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), member, dsl.GrantLandPlays(dsl.ControllerOf(dsl.From(member)), dsl.Fixed(1)))))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTeamBoostComposition(t *testing.T) {
	body := dsl.Until(core.EndOfTurn,
		dsl.ModifyEachPT(dsl.ControlledPermanents(dsl.IsCreature), dsl.Fixed(2), dsl.Fixed(1)))
	cards := []mage.Card{
		dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body))),
		dsl.NewEnchantment("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.Mana("{2}{G}")), body))),
		dsl.NewInstant("Modes", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Boost", body), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	props := dsl.InspectDefinition(body).Properties
	if props.Outcome != mage.OutcomeBenefit || !props.Mass || props.PowerBoost != 2 || props.ToughnessBoost != 1 {
		t.Fatalf("team boost metadata lost: %+v", props)
	}
}

func TestSelfPumpActivationComposition(t *testing.T) {
	for _, delta := range [][2]int{{1, 1}, {0, 1}, {1, 0}} {
		body := dsl.Until(core.EndOfTurn, dsl.ModifyPT(dsl.This, dsl.Fixed(delta[0]), dsl.Fixed(delta[1])))
		card := dsl.NewCreature("Self pump", "", 1, 1,
			dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.Mana("{R}")), body)))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		props := dsl.InspectDefinition(body).Properties
		if props.Outcome != mage.OutcomeBenefit || props.Mass || props.PowerBoost != delta[0] || props.ToughnessBoost != delta[1] {
			t.Fatalf("self pump metadata lost: %+v", props)
		}
	}
}

func TestTargetedBoostComposition(t *testing.T) {
	creature := dsl.TargetPermanent("creature", dsl.IsCreature)
	body := dsl.Until(core.EndOfTurn, dsl.ModifyPT(creature, dsl.ValueAtResolution(dsl.XValue()), dsl.Fixed(0)))
	cards := []mage.Card{
		dsl.NewInstant("Spell", "{X}{B}", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(creature)))),
		dsl.NewInstant("Sequence", "{X}{B}", dsl.WithAbility(dsl.Spell(dsl.Sequence(body, dsl.Draw(dsl.You, dsl.Fixed(1))), dsl.WithTargets(creature)))),
		dsl.NewInstant("Mode", "{X}{B}", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Boost", body, dsl.WithTargets(creature)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	props := dsl.InspectDefinition(body).Properties
	if props.Outcome != mage.OutcomeBenefit || props.Mass {
		t.Fatalf("targeted boost metadata = %+v", props)
	}
}
