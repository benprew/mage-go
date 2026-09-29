package dsl_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

// --- from announced_target_count_test.go ---
func TestAnnouncedTargetCountDeclaration(t *testing.T) {
	targets := dsl.TargetPermanentsUpToX("creatures", dsl.IsCreature)
	card := dsl.NewInstant("Test", "{X}", dsl.WithAbility(dsl.Spell(
		dsl.GainLife(dsl.You, dsl.AnnouncedTargetCount(targets)), dsl.WithTargets(targets))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from attach_choice_test.go ---
func TestMayAttachPublicDeclaration(t *testing.T) {
	entered := dsl.BindPermanent("entered enchantment")
	created := dsl.BindPermanent("created token")
	cat := dsl.DefineToken("Cat", dsl.TokenTypes(core.TypeCreature), dsl.TokenSubtypes("Cat"), dsl.TokenColors(core.White), dsl.TokenPT(2, 2))
	body := dsl.Sequence(dsl.CreateOneToken(dsl.You, cat, created),
		dsl.IfPermanentHasEnchantmentSubtype(dsl.From(entered), "Aura",
			dsl.MayAttach(dsl.From(entered), dsl.From(created))))
	card := dsl.NewCreature("Watcher", "", 3, 3, dsl.WithAbility(dsl.WhenEvent(
		dsl.EnchantmentYouControlEnters().WithPermanent(entered), body)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from attachment_target_test.go ---
func TestAttachmentTargetComposition(t *testing.T) {
	target := dsl.TargetPermanent("aura", dsl.And(dsl.IsEnchantment, dsl.HasSubType("Aura"), dsl.IsAttachedTo(dsl.IsCreature)), dsl.TargetAttachedToYouControl())
	ability := dsl.Activate(dsl.Costs(dsl.TapCost(dsl.This)), dsl.Destroy(target), dsl.WithTargets(target))
	card := dsl.NewCreature("Worker", "{W}", 1, 1, dsl.WithAbility(ability))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	found := false
	var visit func(mage.DefinitionMetadata)
	visit = func(n mage.DefinitionMetadata) {
		found = found || n.Role == "target-attachment-controller" && n.Text == "you control"
		for _, child := range n.Children {
			visit(child)
		}
	}
	visit(dsl.InspectDefinition(ability))
	if !found {
		t.Fatal("attachment controller metadata is absent")
	}
	invalid := dsl.NewInstant("Undeclared", "", dsl.WithAbility(dsl.Spell(dsl.Destroy(target))))
	if mage.ValidateCardDefinition(invalid.Name(), invalid) == nil {
		t.Fatal("undeclared target was accepted")
	}
}

// --- from composable_attachments_set_test.go ---
func TestAttachedAurasPublicDeclaration(t *testing.T) {
	bonus := dsl.MultiplyValues(dsl.Fixed(2), dsl.CountPermanents(dsl.AttachedAuras(dsl.This)))
	card := dsl.NewCreature("Synthetic wombat", "", 0, 1,
		dsl.WithAbility(dsl.Static(dsl.While(dsl.Controls(dsl.IsCreature),
			dsl.ModifyPT(dsl.This, bonus, bonus)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from composable_card_set_choice_test.go ---
func TestPublicChooseUpToOneCard(t *testing.T) {
	discarded := dsl.BindCards("discarded")
	chosen := dsl.BindCard("chosen")
	player := dsl.TargetPlayer("player")
	card := dsl.NewSorcery("Choice", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(
		dsl.Discard(player, dsl.Fixed(2), dsl.DiscardCardsInto(discarded)),
		dsl.ChooseUpToOneCard(dsl.You, dsl.FromCards(discarded), dsl.IsLandCard, chosen),
		dsl.ReturnCardToHand(dsl.FromCard(chosen)),
	), dsl.WithTargets(player))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from composable_spell_cast_binding_test.go ---
func TestSpellCastBindingPublicDeclaration(t *testing.T) {
	spell := dsl.BindSpell("cast spell")
	card := dsl.NewArtifact("Cast Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.SpellCast().WithSpell(spell), dsl.Counter(dsl.FromSpell(spell)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from controlled_by_set_test.go ---
func TestControlledByPermanentSetDeclaration(t *testing.T) {
	player := dsl.TargetPlayer("player")
	card := dsl.NewInstant("Controlled artifacts", "", dsl.WithAbility(dsl.Spell(
		dsl.GainLife(dsl.You, dsl.CountPermanents(dsl.PermanentsControlledBy(player, dsl.IsArtifact))),
		dsl.WithTargets(player))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from counter_amount_test.go ---
func TestTypedCounterAmountComposition(t *testing.T) {
	saved := dsl.BindPermanent("amount")
	counters := dsl.AddCounters(dsl.From(saved), dsl.P1P1, dsl.MaxValue(dsl.Fixed(0), dsl.PowerOf(dsl.From(saved))))
	body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), counters)
	target := dsl.TargetPermanent("amount", dsl.IsCreature)
	for _, card := range []mage.Card{
		dsl.NewSorcery("Target", "", dsl.WithAbility(dsl.Spell(dsl.AddCounters(target, dsl.P1P1, dsl.ToughnessOf(target)), dsl.WithTargets(target)))),
		dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewSorcery("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), saved, counters)))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Counters", dsl.AddCounters(target, dsl.P1P1, dsl.PowerOf(target)), dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from counter_each_test.go ---
func TestAddCountersEachAuthorExample(t *testing.T) {
	card := dsl.NewSorcery("Counter batch", "", dsl.WithAbility(dsl.Spell(
		dsl.AddCountersEach(dsl.ControlledPermanents(dsl.IsCreature), dsl.P1P1, dsl.Fixed(2)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from counter_removal_test.go ---
func TestCounterRemovalComposition(t *testing.T) {
	saved := dsl.BindPermanent("subject")
	target := dsl.TargetPermanent("subject", dsl.IsCreature)
	body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.RemoveCountersFrom(dsl.From(saved), dsl.Tide, dsl.Fixed(2)))
	for _, card := range []mage.Card{
		dsl.NewCreature("State", "", 1, 1, dsl.WithAbility(dsl.WhenState(dsl.SourceHasCounters(dsl.Tide, 4), body))),
		dsl.NewCreature("Activate", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.At(dsl.YourUpkeep, body))),
		dsl.NewSorcery("Target", "", dsl.WithAbility(dsl.Spell(dsl.RemoveCountersFrom(target, dsl.Tide, dsl.PowerOf(target)), dsl.WithTargets(target)))),
		dsl.NewSorcery("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), saved, dsl.RemoveCountersFrom(dsl.From(saved), dsl.Tide, dsl.Fixed(2)))))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Remove", dsl.RemoveCountersFrom(target, dsl.Tide, dsl.PowerOf(target)), dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from counter_reset_test.go ---
func TestCounterResetComposition(t *testing.T) {
	saved := dsl.BindPermanent("subject")
	target := dsl.TargetPermanent("subject", dsl.IsCreature)
	body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.RemoveAllCounters(dsl.From(saved), dsl.Tide))
	for _, card := range []mage.Card{
		dsl.NewCreature("State", "", 1, 1, dsl.WithAbility(dsl.WhenState(dsl.SourceHasCounters(dsl.Tide, 4), body))),
		dsl.NewCreature("Activate", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.At(dsl.YourUpkeep, body))),
		dsl.NewSorcery("Target", "", dsl.WithAbility(dsl.Spell(dsl.RemoveAllCounters(target, dsl.Tide), dsl.WithTargets(target)))),
		dsl.NewSorcery("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), saved, dsl.RemoveAllCounters(dsl.From(saved), dsl.Tide))))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Remove", dsl.RemoveAllCounters(target, dsl.Tide), dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from counter_value_test.go ---
func TestCounterValueComposition(t *testing.T) {
	target := dsl.TargetPermanent("subject", dsl.IsCreature)
	saved := dsl.BindPermanent("saved")
	value := dsl.CountersOn(dsl.This, core.Charge)
	body := dsl.GainLife(dsl.You, value)
	cards := []dsl.Card{
		dsl.NewInstant("Target", "", dsl.WithAbility(dsl.Spell(dsl.GainLife(dsl.You, dsl.CountersOn(target, core.Charge)), dsl.WithTargets(target)))),
		dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Upkeep", "", 1, 1, dsl.WithAbility(dsl.At(dsl.YourUpkeep, body))),
		dsl.NewCreature("Event", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewCreature("Sequence", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.GainLife(dsl.You, dsl.CountersOn(dsl.From(saved), core.Charge)))))),
		dsl.NewInstant("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), saved, dsl.GainLife(dsl.You, dsl.CountersOn(dsl.From(saved), core.Charge)))))),
		dsl.NewCreature("Modes", "", 1, 1, dsl.WithAbility(dsl.AtModes(dsl.YourUpkeep, dsl.ModeOf("Life", body), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewCreature("Static", "", 1, 1, dsl.WithAbility(dsl.Static(dsl.While(dsl.Controls(dsl.IsCreature), dsl.ModifyPT(dsl.This, value, dsl.Fixed(0)))))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from exact_stats_filter_test.go ---
func TestExactStatsFilters(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	c := dsl.NewCreature("Recipient", "", 1, 1)
	p := g.PutOnBattlefield(c, b.PlayerID())
	filter := dsl.And(dsl.IsCreature, dsl.HasPowerEqual(1), dsl.HasToughnessEqual(1))
	for _, step := range []struct {
		power, toughness int
		matches          bool
	}{
		{0, 0, true}, {1, 0, false}, {-1, 1, false}, {0, -1, true}, {-2, -2, false},
	} {
		body := dsl.Until(core.EndOfTurn, dsl.ModifyPT(dsl.This, dsl.Fixed(step.power), dsl.Fixed(step.toughness)))
		if err := mage.ApplyEffect(g, body, c.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if got := filter.Match(p, g); got != step.matches {
			t.Fatalf("current %d/%d: match = %v, want %v", p.CurrentPower(g), p.CurrentToughness(g), got, step.matches)
		}
	}
	if !dsl.HasPowerEqual(-1).Match(p, g) || !dsl.HasToughnessEqual(-1).Match(p, g) {
		t.Fatal("negative current characteristics did not match")
	}
	target := dsl.TargetPermanent("creature", filter)
	card := dsl.NewLand("Booster", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.TapCost(dsl.This)),
		dsl.Until(core.EndOfTurn, dsl.ModifyPT(target, dsl.Fixed(1), dsl.Fixed(2))),
		dsl.WithTargets(target),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from graveyard_mana_value_test.go ---
func TestGraveyardManaValueBoundPublicDeclaration(t *testing.T) {
	target := dsl.TargetGraveyardCard("card",
		dsl.GraveyardCardFilter(dsl.IsCreatureCard),
		dsl.GraveyardCardManaValueAtMost(dsl.ControllerLifeGainedThisTurn()))
	card := dsl.NewCreature("Synthetic watcher", "", 1, 1,
		dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This),
			dsl.ReturnCardToBattlefield(target, dsl.You), dsl.WithTargets(target))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from hand_size_value_test.go ---
func TestHandSizeOfPublicDeclaration(t *testing.T) {
	opponent := dsl.TargetPlayer("opponent", dsl.TargetOpponentOnly())
	card := dsl.NewSorcery("Synthetic hand spell", "", dsl.WithAbility(dsl.Spell(
		dsl.Sequence(
			dsl.DiscardEntireHand(dsl.You),
			dsl.Draw(dsl.You, dsl.HandSizeOf(opponent)),
		),
		dsl.WithTargets(opponent),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from highest_life_player_test.go ---
func TestPlayerWithUniqueHighestLifePublicDeclaration(t *testing.T) {
	card := dsl.NewArtifact("Synthetic life selector", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(),
		dsl.GainLife(dsl.PlayerWithUniqueHighestLife(), dsl.Fixed(1)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from library_choice_test.go ---
func TestTypedLibraryChoiceResolution(t *testing.T) {
	for _, surveil := range []bool{false, true} {
		for _, amount := range []int{-1, 0, 2, 5} {
			a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
			g := mage.NewGame(a, b)
			top, middle, bottom := mage.NewLand("Top"), mage.NewLand("Middle"), mage.NewLand("Bottom")
			b.SetLibrary([]mage.Card{top, middle, bottom})
			source := mage.NewCreature("Amount", "", amount, 6)
			g.PutOnBattlefield(source, a.PlayerID())
			saved := dsl.BindPermanent("saved")
			value := dsl.PowerOf(dsl.From(saved))
			choice := dsl.Scry(value)
			if surveil {
				choice = dsl.Surveil(value)
			}
			body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), choice)
			if amount > 0 {
				if surveil {
					b.AddSurveilDecision([]string{"Top"}, nil)
				} else {
					b.AddScryDecision([]string{"Top"}, nil)
				}
			}
			if err := mage.ApplyEffect(g, body, source.ID(), b.PlayerID(), []uuid.UUID{a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if amount <= 0 {
				if len(b.Library()) != 3 || b.Library()[0].ID() != top.ID() || len(b.Graveyard()) != 0 {
					t.Fatal("nonpositive amount changed zones")
				}
			} else if surveil {
				if len(b.Library()) != 2 || b.Library()[0].ID() != middle.ID() || len(b.Graveyard()) != 1 || b.Graveyard()[0].ID() != top.ID() {
					t.Fatal("surveil lost binding, recipient, or placement")
				}
			} else if len(b.Library()) != 3 || b.Library()[0].ID() != middle.ID() || b.Library()[2].ID() != top.ID() {
				t.Fatal("scry lost binding, recipient, or placement")
			}
			if len(a.Library()) != 0 || len(a.Graveyard()) != 0 {
				t.Fatal("changed source controller zones")
			}
		}
	}
}

func TestTypedLibraryChoiceComposition(t *testing.T) {
	for _, build := range []func(dsl.ResolutionValue) dsl.ResolvingEffect{dsl.Scry, dsl.Surveil} {
		target := dsl.TargetPermanent("creature", dsl.IsCreature)
		body := build(dsl.MaxValue(dsl.Fixed(1), dsl.PowerOf(target)))
		for _, card := range []mage.Card{
			dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target)))),
			dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Choice", body, dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
			dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body, dsl.WithTargets(target)))),
			dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body, dsl.WithTargets(target)))),
		} {
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// --- from life_gain_history_value_test.go ---
func TestLifeGainedThisTurnValuePublicDeclaration(t *testing.T) {
	card := dsl.NewCreature("Synthetic life history", "", 1, 1,
		dsl.WithAbility(dsl.At(dsl.YourEndStep,
			dsl.GainLife(dsl.You, dsl.LifeGainedThisTurn(dsl.You)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from life_total_value_test.go ---
func TestLifeTotalOfPublicDeclaration(t *testing.T) {
	opponent := dsl.TargetPlayer("opponent", dsl.TargetOpponentOnly())
	card := dsl.NewSorcery("Synthetic life total", "", dsl.WithAbility(dsl.Spell(
		dsl.Sequence(dsl.LoseLife(opponent, dsl.Fixed(3)), dsl.GainLife(dsl.You, dsl.LifeTotalOf(opponent))),
		dsl.WithTargets(opponent),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from opponent_target_test.go ---
func TestTypedOpponentTargetComposition(t *testing.T) {
	var option = dsl.TargetOpponentOnly()
	target := dsl.TargetPlayer("opponent", option)
	body := dsl.Sequence(dsl.LoseLife(target, dsl.Fixed(1)), dsl.GainLife(dsl.You, dsl.Fixed(1)))
	cards := []mage.Card{
		dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target)))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body, dsl.WithTargets(target)))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.CreatureDies(), body, dsl.WithTargets(target)))),
		dsl.NewInstant("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Drain", body, dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		found := false
		var visit func(mage.DefinitionMetadata)
		visit = func(n mage.DefinitionMetadata) {
			found = found || n.Role == "target-player" && n.Text == "opponent"
			for _, child := range n.Children {
				visit(child)
			}
		}
		for _, ability := range card.Abilities() {
			visit(dsl.InspectDefinition(ability))
		}
		if !found {
			t.Fatalf("%s lacks opponent restriction metadata", card.Name())
		}
	}
	card := dsl.NewInstant("Undeclared", "", dsl.WithAbility(dsl.Spell(body)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("accepted undeclared opponent target")
	}
}

func TestTypedOpponentTargetNilOption(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("accepted nil player target option")
		}
	}()
	dsl.TargetPlayer("opponent", nil)
}

// --- from owned_set_test.go ---
func TestOwnedPermanentSetDeclaration(t *testing.T) {
	player := dsl.TargetPlayer("player")
	card := dsl.NewInstant("Owned artifacts", "", dsl.WithAbility(dsl.Spell(
		dsl.GainLife(dsl.You, dsl.CountPermanents(dsl.PermanentsOwnedBy(player, dsl.IsArtifact))),
		dsl.WithTargets(player))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from permanent_count_test.go ---
func TestPermanentCountComposition(t *testing.T) {
	value := dsl.CountPermanents(dsl.ControlledPermanents(dsl.IsCreature))
	card := dsl.NewInstant("Count", "", dsl.Spell(dsl.GainLife(dsl.You, value)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	info := dsl.InspectDefinition(value)
	if info.Role != "permanent-count" || len(info.Children) != 1 {
		t.Fatalf("metadata = %+v", info)
	}
}

// --- from player_binding_test.go ---
func TestTypedRandomPlayerComposition(t *testing.T) {
	var saved = dsl.BindPlayer("recipient")
	var player = dsl.FromPlayer(saved)
	body := dsl.Sequence(dsl.ChooseRandomPlayer(dsl.EachPlayer, saved), dsl.Mill(player, dsl.Fixed(2)))
	for _, card := range []mage.Card{
		dsl.NewSorcery("Draw", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(dsl.ChooseRandomPlayer(dsl.EachPlayer, saved), dsl.Draw(player, dsl.Fixed(3)))))),
		dsl.NewSorcery("GainLife", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(dsl.ChooseRandomPlayer(dsl.EachPlayer, saved), dsl.GainLife(player, dsl.Fixed(3)))))),
		dsl.NewSorcery("Discard", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(dsl.ChooseRandomPlayer(dsl.EachPlayer, saved), dsl.Discard(player, dsl.Fixed(1)))))),
		dsl.NewSorcery("Spell", "", dsl.WithAbility(dsl.Spell(body))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Random", body), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewSorcery("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), dsl.BindPermanent("member"), body)))),
		dsl.NewSorcery("Delayed", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(dsl.ChooseRandomPlayer(dsl.OpponentsOf(dsl.You), saved), dsl.AtNext(dsl.EachEndStep, dsl.PreventNextDamage(player, dsl.Fixed(2))))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from player_snapshot_test.go ---
func TestPlayerSnapshotComposition(t *testing.T) {
	saved := dsl.BindPlayer("player")
	body := dsl.Sequence(dsl.SnapshotPlayer(dsl.You, saved), dsl.PreventNextDamage(dsl.FromPlayer(saved), dsl.Fixed(2)))
	for _, card := range []mage.Card{
		dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewInstant("Modes", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Shield", body), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewInstant("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), dsl.BindPermanent("member"), body)))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from source_counters_test.go ---
func TestSourceCountersComposition(t *testing.T) {
	value := dsl.SourceCounters(core.Charge)
	n := dsl.BindNumber("amount")
	body := dsl.Sequence(dsl.SnapshotNumber(value, n), dsl.GainLife(dsl.You, dsl.FromNumber(n)))
	cards := []dsl.Card{
		dsl.NewEnchantment("Main phase", "", dsl.WithAbility(dsl.At(dsl.YourFirstMainPhase, dsl.AddManaAmount(core.Black, value)))),
		dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Event", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Dies(dsl.This), body))),
		dsl.NewCreature("Delayed", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.AtNext(dsl.EachEndStep, body)))),
		dsl.NewCreature("Modes", "", 1, 1, dsl.WithAbility(dsl.ActivateModes(dsl.Costs(), dsl.ModeOf("Life", body), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from spell_cast_creature_target_test.go ---
func TestSpellCastCreatureTargetPublicDeclaration(t *testing.T) {
	card := dsl.NewArtifact("Repartee source", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.SpellCast().ByController().OfInstantOrSorcery().TargetsCreature(),
		dsl.Draw(dsl.You, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from target_owner_test.go ---
func TestTypedTargetOwnerComposition(t *testing.T) {
	target := dsl.TargetPermanent("owned", dsl.AnyPermanent, dsl.TargetYouOwn(), dsl.TargetYouControl())
	group := dsl.TargetPermanents("owned group", 1, 2, dsl.IsCreature, dsl.TargetYouOwn())
	saved := dsl.BindPermanent("member")
	body := dsl.ReturnToHand(target)
	cards := []mage.Card{
		dsl.NewArtifact("Obelisk of Undoing composition", "{1}", dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.Mana("{6}"), dsl.TapCost(dsl.This)), body, dsl.WithTargets(target)))),
		dsl.NewInstant("Owned spell", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target)))),
		dsl.NewCreature("Owned trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body, dsl.WithTargets(target)))),
		dsl.NewInstant("Owned mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Return", body, dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewInstant("Owned group", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(group, saved, dsl.ReturnToHand(dsl.From(saved))), dsl.WithTargets(group)))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		var owner bool
		var visit func(mage.DefinitionMetadata)
		visit = func(n mage.DefinitionMetadata) {
			owner = owner || n.Role == "target-owner" && n.Text == "you own"
			for _, child := range n.Children {
				visit(child)
			}
		}
		for _, ability := range card.Abilities() {
			visit(dsl.InspectDefinition(ability))
		}
		if !owner {
			t.Fatalf("%s does not expose ownership metadata", card.Name())
		}
	}
	undeclared := dsl.NewInstant("Undeclared", "", dsl.WithAbility(dsl.Spell(body)))
	if err := mage.ValidateCardDefinition(undeclared.Name(), undeclared); err == nil {
		t.Fatal("accepted undeclared owned target")
	}
}

// --- from toughness_filter_test.go ---
func TestHasToughnessGTE(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	c := dsl.NewCreature("Recipient", "", 2, 3)
	g.PutOnBattlefield(c, b.PlayerID())
	p := g.FindPermanent(c.ID())
	filter := dsl.And(dsl.IsCreature, dsl.HasToughnessGTE(4))
	if filter.Match(p, g) {
		t.Fatal("toughness 3 matched threshold 4")
	}
	for _, delta := range []int{1, 1, -2} {
		body := dsl.Until(core.EndOfTurn, dsl.ModifyPT(dsl.This, dsl.Fixed(0), dsl.Fixed(delta)))
		if err := mage.ApplyEffect(g, body, c.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := p.CurrentToughness(g) >= 4
		if filter.Match(p, g) != want {
			t.Fatalf("filter did not use current toughness %d", p.CurrentToughness(g))
		}
	}
	if !dsl.HasToughnessGTE(-1).Match(p, g) {
		t.Fatal("negative threshold rejected")
	}
	target := dsl.TargetPermanent("tough creature", filter)
	spell := dsl.NewInstant("Filtered mode", "", dsl.WithAbility(dsl.SpellModes(
		dsl.ModeOf("Destroy", dsl.Destroy(target), dsl.WithTargets(target)),
		dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1))),
	)))
	if err := mage.ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
}

// --- from value_test.go ---
func TestTypedPermanentValueComposition(t *testing.T) {
	target := dsl.TargetPermanent("subject", dsl.IsCreature)
	saved := dsl.BindPermanent("saved")
	body := dsl.Sequence(dsl.Snapshot(target, saved), dsl.Until(core.EndOfTurn, dsl.ModifyPT(target, dsl.PowerOf(dsl.From(saved)), dsl.ToughnessOf(target))))
	card := dsl.NewInstant("Values", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	metadata := dsl.InspectDefinition(dsl.PowerOf(target))
	if metadata.Role != "permanent-value" || len(metadata.Children) != 1 {
		t.Fatalf("metadata = %+v", metadata)
	}
	var fixed dsl.ResolutionValue = dsl.Fixed(2)
	var legacy dsl.ValueSource = dsl.Fixed(2)
	if fixed.Text() != dsl.ValueAtResolution(legacy).Text() {
		t.Fatal("legacy adapter lost text")
	}
}

func TestTypedArithmeticComposition(t *testing.T) {
	target := dsl.TargetPermanent("subject", dsl.IsCreature)
	value := dsl.MinValue(dsl.Fixed(0), dsl.MaxValue(dsl.Fixed(-4), dsl.SubtractValues(dsl.Fixed(1), dsl.ToughnessOf(target))))
	body := dsl.Until(core.EndOfTurn, dsl.ModifyPT(target, dsl.Fixed(4), value))
	card := dsl.NewInstant("Arithmetic", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	metadata := dsl.InspectDefinition(value)
	if metadata.Role != "arithmetic-value" || metadata.Text != "minimum" || len(metadata.Children) != 2 {
		t.Fatalf("metadata = %+v", metadata)
	}
	if value.Text() != "minimum(0, maximum(-4, subtract(1, toughness of subject)))" {
		t.Fatalf("text = %q", value.Text())
	}
}

func TestTypedMultiplyComposition(t *testing.T) {
	target := dsl.TargetPermanent("subject", dsl.IsCreature)
	value := dsl.MultiplyValues(dsl.Fixed(2), dsl.ValueAtResolution(dsl.XValue()))
	card := dsl.NewSorcery("Multiply", "{X}{U}", dsl.WithAbility(dsl.Spell(dsl.Sequence(dsl.Tap(target), dsl.AddCounters(target, core.Stun, value)), dsl.WithTargets(target))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	metadata := dsl.InspectDefinition(value)
	if metadata.Role != "arithmetic-value" || metadata.Text != "multiply" || len(metadata.Children) != 2 {
		t.Fatalf("metadata = %+v", metadata)
	}
	if got := dsl.MultiplyValues(dsl.Fixed(2), dsl.Fixed(3)).Text(); got != "multiply(2, 3)" {
		t.Fatalf("text = %q", got)
	}
}

// --- from x_targets_test.go ---
func TestXTargetPermanentDeclarations(t *testing.T) {
	lands := dsl.TargetPermanentsExactlyX("lands", dsl.IsLand)
	member := dsl.BindPermanent("land")
	ability := dsl.Activate(dsl.Costs(dsl.Mana("{X}"), dsl.TapCost(dsl.This)),
		dsl.ForEach(lands, member, dsl.Untap(dsl.From(member))),
		dsl.WithTargets(lands))
	card := dsl.NewArtifact("Example", "", dsl.WithAbility(ability))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}

	creatures := dsl.TargetPermanentsUpToX("creatures", dsl.IsCreature)
	spell := dsl.NewInstant("Example spell", "{X}", dsl.WithAbility(dsl.Spell(
		dsl.DestroyEach(creatures), dsl.WithTargets(creatures))))
	if err := mage.ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
}
