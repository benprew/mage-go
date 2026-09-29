package dsl_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

// --- from any_damage_source_test.go ---
func TestAnyDamageSourcePublicDeclaration(t *testing.T) {
	dealt := dsl.BindNumber("dealt")
	source := dsl.BindDamageSource("source")
	card := dsl.NewCreature("Watcher", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(
		dsl.DealsDamage(dsl.This, dealt).WithSource(source),
		dsl.DealDamageFrom(dsl.FromDamageSource(source), dsl.You, dsl.FromNumber(dealt)))))
	if card == nil {
		t.Fatal("missing card")
	}
}

// --- from artifact_tap_opponent_test.go ---
func TestArtifactTapOpponentDSL(t *testing.T) {
	controller := dsl.BindPlayer("controller")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.ArtifactBecomesTapped().ByOpponent().WithController(controller),
		dsl.GainLife(dsl.FromPlayer(controller), dsl.Fixed(1)))))
	if card == nil || dsl.InspectDefinition(dsl.ArtifactBecomesTapped().ByOpponent()).Role != "event-pattern" {
		t.Fatal("opponent artifact tap pattern is unavailable")
	}
}

// --- from attached_damage_player_test.go ---
func TestAttachedDamagePlayerPublicDeclaration(t *testing.T) {
	player := dsl.BindPlayer("damaged player")
	amount := dsl.BindNumber("damage dealt")
	card := dsl.NewAura("Damage Watcher", "{W}", dsl.WithAbility(
		dsl.WhenEvent(dsl.DealsDamageToPlayer(dsl.AttachedPermanent, player).WithAmount(amount),
			dsl.GainLife(dsl.FromPlayer(player), dsl.FromNumber(amount))),
	))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from attached_damage_test.go ---
func TestAttachedDamagePublicDeclaration(t *testing.T) {
	dealt := dsl.BindNumber("damage dealt")
	card := dsl.NewAura("Linked Aura", "{W}", dsl.WithAbility(
		dsl.WhenEvent(dsl.DealsDamage(dsl.AttachedPermanent, dealt),
			dsl.GainLife(dsl.You, dsl.FromNumber(dealt))),
	))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from attached_land_mana_test.go ---
func TestAttachedLandTappedForManaComposition(t *testing.T) {
	player := dsl.BindPlayer("producer")
	amount := dsl.BindNumber("produced")
	card := dsl.NewAura("Land watcher", "", dsl.WithCastTarget(dsl.TargetLand()), dsl.WithAbility(dsl.WhenEvent(
		dsl.LandTappedForMana().OfAttachedLand().OfManaType(core.Blue).WithPlayer(player).WithAmount(amount),
		dsl.GainLife(dsl.FromPlayer(player), dsl.FromNumber(amount)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from attached_step_test.go ---
func TestAttachedControllerUpkeepComposition(t *testing.T) {
	host, player := dsl.BindPermanent("host"), dsl.BindPlayer("player")
	pattern := dsl.AttachedControllerUpkeep(host, player)
	body := dsl.Sequence(dsl.Tap(dsl.From(host)), dsl.GainLife(dsl.FromPlayer(player), dsl.Fixed(1)))
	for _, ability := range []dsl.TriggeredAbility{
		dsl.WhenEvent(pattern, body),
		dsl.WhenEventModes(pattern, dsl.ModeOf("Now", body), dsl.ModeOf("Later", dsl.AtNext(dsl.EachEndStep, body))),
		dsl.WhenEvent(pattern, dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), dsl.BindPermanent("member"), body)),
	} {
		card := dsl.NewAura("Watcher", "", dsl.WithAbility(ability))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	info := dsl.InspectDefinition(pattern)
	if info.Role != "event-pattern" || len(info.Children) != 2 {
		t.Fatalf("inspection = %+v", info)
	}
}

// --- from attached_tap_test.go ---
func TestAttachedBecomesTappedComposition(t *testing.T) {
	host := dsl.BindPermanent("host")
	pattern := dsl.AttachedBecomesTapped(host)
	body := dsl.Sequence(dsl.Tap(dsl.From(host)), dsl.GainLife(dsl.You, dsl.Fixed(1)))
	for _, ability := range []dsl.TriggeredAbility{
		dsl.WhenEvent(pattern, body),
		dsl.WhenEventModes(pattern, dsl.ModeOf("Now", body), dsl.ModeOf("Later", dsl.AtNext(dsl.EachEndStep, body))),
		dsl.WhenEvent(pattern, dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), dsl.BindPermanent("member"), body)),
	} {
		card := dsl.NewAura("Watcher", "", dsl.WithAbility(ability))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	info := dsl.InspectDefinition(pattern)
	if info.Role != "event-pattern" || len(info.Children) != 1 {
		t.Fatalf("inspection = %+v", info)
	}
}

// --- from card_loop_test.go ---
func TestForEachSavedCardAuthorExample(t *testing.T) {
	cards := dsl.BindCards("drawn")
	member := dsl.BindCard("member")
	card := dsl.NewInstant("Synthetic", "", dsl.Spell(dsl.Sequence(
		dsl.Draw(dsl.You, dsl.Fixed(2), dsl.DrawCardsInto(cards)),
		dsl.ForEachSavedCard(cards, member, dsl.RevealCard(dsl.FromCard(member))),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	for _, ability := range []mage.Ability{
		dsl.Activate(dsl.Costs(), dsl.Sequence(dsl.Draw(dsl.You, dsl.Fixed(1), dsl.DrawCardsInto(cards)), dsl.ForEachSavedCard(cards, member, dsl.RevealCard(dsl.FromCard(member))))),
		dsl.WhenEvent(dsl.Enters(dsl.This), dsl.Sequence(dsl.Draw(dsl.You, dsl.Fixed(1), dsl.DrawCardsInto(cards)), dsl.ForEachSavedCard(cards, member, dsl.RevealCard(dsl.FromCard(member))))),
	} {
		permanent := dsl.NewArtifact("Synthetic permanent", "", dsl.WithAbility(ability))
		if err := mage.ValidateCardDefinition(permanent.Name(), permanent); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from combined_timing_test.go ---
func TestCombinedTimingComposition(t *testing.T) {
	step, turn := dsl.OnlyDuring(core.Upkeep), dsl.OnlyOnYourTurn()
	body := dsl.Regenerate(dsl.This)
	card := dsl.NewArtifact("Timed abilities", "",
		dsl.WithAbility(dsl.Activate(dsl.Costs(), body, step, turn)),
		dsl.WithAbility(dsl.ActivateModes(dsl.Costs(), dsl.ModeOf("one", body), dsl.ModeOf("two", body), turn, step)),
		dsl.WithAbility(dsl.Static(dsl.GrantActivatedModes(dsl.This, dsl.Costs(), dsl.ModeOf("one", body), dsl.ModeOf("two", body), step, turn))),
		dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Until(core.Indefinite, dsl.ActivatedAbilityModification(dsl.This, dsl.Costs(), body, turn, step)))),
	)
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from controlled_creature_entry_test.go ---
func TestControlledCreatureEntryPublicDeclaration(t *testing.T) {
	power := dsl.BindNumber("entry power")
	card := dsl.NewCreature("Watcher", "", 3, 3, dsl.WithAbility(dsl.WhenEvent(
		dsl.CreatureYouControlEnters().HasKeyword(core.Flying).WithPower(power),
		dsl.GainLife(dsl.You, dsl.FromNumber(power)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from controller_damage_source_test.go ---
func TestControllerDamageSourcePublicDeclaration(t *testing.T) {
	amount := dsl.BindNumber("amount")
	source := dsl.BindDamageSource("source")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.ControllerIsDealtDamage(amount).WithSource(source),
		dsl.DealDamageFrom(dsl.FromDamageSource(source), dsl.You, dsl.FromNumber(amount)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from creature_entry_any_test.go ---
func TestAnotherCreatureEntersPublicDeclaration(t *testing.T) {
	power := dsl.BindNumber("entry power")
	card := dsl.NewCreature("Watcher", "", 6, 6, dsl.WithAbility(dsl.WhenEvent(
		dsl.AnotherCreatureEnters().WithPower(power),
		dsl.AddCounters(dsl.This, core.P1P1, dsl.FromNumber(power)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from creature_entry_controller_test.go ---
func TestCreatureEntryControllerPublicDeclaration(t *testing.T) {
	controller := dsl.BindPlayer("entrant controller")
	card := dsl.NewCreature("Watcher", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(
		dsl.AnotherCreatureEnters().WithController(controller),
		dsl.GainLife(dsl.FromPlayer(controller), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from creature_entry_power_test.go ---
func TestCreatureEntryPowerPublicDeclaration(t *testing.T) {
	power := dsl.BindNumber("entering creature power")
	card := dsl.NewCreature("Power watcher", "", 1, 1, dsl.WithAbility(
		dsl.WhenEvent(dsl.AnotherCreatureYouControlEnters().WithPower(power),
			dsl.GainLife(dsl.You, dsl.FromNumber(power)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from damage_history_set_test.go ---
func TestDamageHistorySetDeclaration(t *testing.T) {
	players := dsl.PlayersDealtDamageBy(dsl.This, dsl.OpponentsOf(dsl.You))
	planeswalkers := dsl.PermanentsDealtDamageBy(dsl.This, dsl.MatchingPermanents(dsl.IsPlaneswalker))
	card := dsl.NewCreature("History source", "", 2, 3, dsl.WithAbility(dsl.At(dsl.YourUpkeep,
		dsl.DealDamageEach(dsl.This, dsl.DamageRecipients(players, planeswalkers), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from damage_player_recipient_test.go ---
func TestDamagePlayerRecipientPublicDeclaration(t *testing.T) {
	player := dsl.BindPlayer("recipient")
	_ = dsl.WhenEvent(dsl.DealsDamage(dsl.This, dsl.BindNumber("amount")).WithRecipientPlayer(player),
		dsl.GainLife(dsl.FromPlayer(player), dsl.Fixed(1)))
}

// --- from damage_source_controller_test.go ---
func TestDamageSourceControllerPublicDeclaration(t *testing.T) {
	player := dsl.BindPlayer("damaged player")
	controller := dsl.BindPlayer("source controller")
	amount := dsl.BindNumber("damage dealt")
	card := dsl.NewAura("Damage Watcher", "{U}", dsl.WithAbility(
		dsl.WhenEvent(
			dsl.DealsDamageToPlayer(dsl.AttachedPermanent, player).ToController().WithSourceController(controller).WithAmount(amount),
			dsl.DealDamageFrom(dsl.This, dsl.FromPlayer(controller), dsl.FromNumber(amount)),
		),
	))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from damage_to_controller_test.go ---
func TestDamageToControllerPublicDeclaration(t *testing.T) {
	player := dsl.BindPlayer("damaged player")
	amount := dsl.BindNumber("damage dealt")
	card := dsl.NewAura("Damage Watcher", "{W}", dsl.WithAbility(
		dsl.WhenEvent(dsl.DealsDamageToPlayer(dsl.AttachedPermanent, player).ToController().WithAmount(amount),
			dsl.GainLife(dsl.FromPlayer(player), dsl.FromNumber(amount)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from damage_turn_set_test.go ---
func TestTurnDamagePermanentSetDeclaration(t *testing.T) {
	card := dsl.NewCreature("Turn history", "", 1, 5,
		dsl.WithAbility(dsl.At(dsl.YourUpkeep,
			dsl.TapEach(dsl.PermanentsDealtDamageByThisTurn(dsl.This,
				dsl.MatchingPermanents(dsl.IsCreature))))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from destroy_graveyard_result_test.go ---
func TestPermanentsPutIntoGraveyardThisWayPublicDeclaration(t *testing.T) {
	count := dsl.BindNumber("graveyard count")
	targets := dsl.TargetPermanentsExactlyX("mountains", dsl.And(dsl.IsLand, dsl.HasSubType("Mountain")))
	card := dsl.NewSorcery("Synthetic eruption", "{X}{U}{U}{U}", dsl.WithAbility(dsl.Spell(dsl.Sequence(
		dsl.DestroyEach(targets, dsl.PermanentsPutIntoGraveyardThisWayInto(count)),
		dsl.DealDamageEach(dsl.This, dsl.DamageRecipients(dsl.EachPlayer, dsl.MatchingPermanents(dsl.IsCreature)), dsl.FromNumber(count)),
	), dsl.WithTargets(targets))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from destroy_result_test.go ---
func TestCreaturesDiedThisWayIntoPublicDeclaration(t *testing.T) {
	deaths := dsl.BindNumber("creatures died this way")
	body := dsl.Sequence(
		dsl.DestroyEach(dsl.MatchingPermanents(dsl.IsCreature), dsl.CreaturesDiedThisWayInto(deaths)),
		dsl.DealDamageFrom(dsl.This, dsl.You, dsl.FromNumber(deaths)),
	)
	for _, card := range []mage.Card{
		dsl.NewSorcery("Spell", "", dsl.WithAbility(dsl.Spell(body))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatalf("%s: %v", card.Name(), err)
		}
	}
}

// --- from destroy_singular_result_test.go ---
func TestDestroySingularResultPublicDeclaration(t *testing.T) {
	result := dsl.BindNumber("destroyed")
	card := dsl.NewCreature("Test", "", 2, 2, dsl.WithAbility(dsl.At(dsl.YourUpkeep, dsl.Sequence(
		dsl.DestroyWithResult(dsl.This, dsl.DestroyedThisWayInto(result)),
		dsl.IfPositive(dsl.FromNumber(result), dsl.DealDamageFrom(dsl.This, dsl.You, dsl.Fixed(7))),
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from discard_amount_test.go ---
func TestTypedDiscardAmountComposition(t *testing.T) {
	saved := dsl.BindPermanent("amount")
	discard := dsl.Discard(dsl.You, dsl.MaxValue(dsl.Fixed(0), dsl.PowerOf(dsl.From(saved))))
	body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), discard)
	player := dsl.TargetPlayer("player")
	target := dsl.TargetPermanent("amount", dsl.IsCreature)
	for _, card := range []mage.Card{
		dsl.NewSorcery("Target", "", dsl.WithAbility(dsl.Spell(dsl.Discard(player, dsl.ToughnessOf(target)), dsl.WithTargets(player, target)))),
		dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewSorcery("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), saved, discard)))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Discard", dsl.Discard(dsl.You, dsl.PowerOf(target)), dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from discard_card_result_test.go ---
func TestDiscardCardResultPublicDeclaration(t *testing.T) {
	card := dsl.BindCard("discarded")
	player := dsl.TargetPlayer("player")
	spell := dsl.NewInstant("Synthetic", "", dsl.WithAbility(dsl.Spell(
		dsl.Sequence(
			dsl.Discard(player, dsl.Fixed(1), dsl.DiscardCardInto(card)),
			dsl.ReturnCardToHand(dsl.FromCard(card))),
		dsl.WithTargets(player))))
	if err := mage.ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
}

// --- from discard_cards_result_test.go ---
func TestDiscardCardsResultPublicDeclaration(t *testing.T) {
	discarded := dsl.BindCards("discarded")
	player := dsl.TargetPlayer("player")
	card := dsl.NewInstant("Synthetic", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(
		dsl.Discard(player, dsl.Fixed(2), dsl.DiscardCardsInto(discarded)),
		dsl.GainLife(dsl.You, dsl.CountCards(dsl.FromCards(discarded))),
	), dsl.WithTargets(player))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from discard_each_result_test.go ---
func TestDiscardEachTotalPublicDeclaration(t *testing.T) {
	total := dsl.BindNumber("discarded")
	card := dsl.NewSorcery("Shared discard total", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(
		dsl.DiscardEach(dsl.EachPlayer, dsl.DiscardTotalInto(total)),
		dsl.GainLife(dsl.You, dsl.FromNumber(total)),
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from discard_each_test.go ---
func TestDiscardEachPublicDeclaration(t *testing.T) {
	card := dsl.NewSorcery("Wheel-like", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(
		dsl.DiscardEach(dsl.EachPlayer),
		dsl.DrawEach(dsl.EachPlayer, dsl.Fixed(7)),
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from discard_hand_test.go ---
func TestTypedDiscardEntireHandComposition(t *testing.T) {
	player := dsl.TargetPlayer("player")
	saved := dsl.BindPlayer("damaged")
	permanent := dsl.BindPermanent("permanent")
	for _, card := range []mage.Card{
		dsl.NewSorcery("Target", "", dsl.WithAbility(dsl.Spell(dsl.DiscardEntireHand(player), dsl.WithTargets(player)))),
		dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Sequence(dsl.DiscardEntireHand(dsl.You), dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.DealsDamageToOpponent(dsl.This, saved), dsl.DiscardEntireHand(dsl.FromPlayer(saved))))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Discard", dsl.DiscardEntireHand(player), dsl.WithTargets(player)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewSorcery("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), permanent, dsl.DiscardEntireHand(dsl.ControllerOf(dsl.From(permanent))))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from discard_result_test.go ---
func TestDiscardAmountIntoPublicDeclaration(t *testing.T) {
	discarded := dsl.BindNumber("discarded")
	body := dsl.Sequence(
		dsl.DiscardEntireHand(dsl.You, dsl.DiscardAmountInto(discarded)),
		dsl.Draw(dsl.You, dsl.FromNumber(discarded)),
	)
	for _, card := range []mage.Card{
		dsl.NewSorcery("Spell", "", dsl.WithAbility(dsl.Spell(body))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Dies(dsl.This), body))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatalf("%s: %v", card.Name(), err)
		}
	}
}

// --- from divide_test.go ---
func TestDividePublicDeclaration(t *testing.T) {
	x := dsl.BindNumber("x")
	card := dsl.NewCreature("Half X", "", 1, 1, dsl.WithAbility(dsl.Activate(
		dsl.Costs(), dsl.Sequence(
			dsl.SnapshotNumber(dsl.Fixed(5), x),
			dsl.DealDamageFrom(dsl.This, dsl.You, dsl.DivideRoundUp(dsl.FromNumber(x), dsl.Fixed(2))),
		))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from draw_amount_test.go ---
func TestTypedDrawAmountComposition(t *testing.T) {
	saved := dsl.BindPermanent("amount")
	loss := dsl.DrawEach(dsl.OpponentsOf(dsl.You), dsl.MaxValue(dsl.Fixed(0), dsl.PowerOf(dsl.From(saved))))
	body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), loss)
	player := dsl.TargetPlayer("player")
	target := dsl.TargetPermanent("amount", dsl.IsCreature)
	for _, card := range []mage.Card{
		dsl.NewSorcery("Target", "", dsl.WithAbility(dsl.Spell(dsl.Draw(player, dsl.ToughnessOf(target)), dsl.WithTargets(player, target)))),
		dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewSorcery("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), saved, loss)))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Life", dsl.DrawEach(dsl.EachPlayer, dsl.PowerOf(target)), dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from draw_cards_result_test.go ---
func TestDrawCardsResultDeclaration(t *testing.T) {
	drawn := dsl.BindCards("drawn")
	card := dsl.NewInstant("Synthetic draw", "", dsl.Spell(dsl.Sequence(
		dsl.Draw(dsl.You, dsl.Fixed(2), dsl.DrawCardsInto(drawn)),
		dsl.GainLife(dsl.You, dsl.CountCards(dsl.FromCards(drawn))),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from draw_step_replacement_test.go ---
func TestDrawStepReplacementComposition(t *testing.T) {
	var rule = dsl.SkipDrawStepForLife(2)
	card := dsl.NewEnchantment("Static skip", "", dsl.WithAbility(dsl.Static(rule)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	info := dsl.InspectDefinition(card.Abilities()[0])
	if len(info.Children) != 1 || info.Children[0].Role != "static-draw-step-replacement" {
		t.Fatalf("missing replacement definition: %+v", info)
	}
}

// --- from enchantment_entry_test.go ---
func TestEnchantmentEntryDeclaration(t *testing.T) {
	entered := dsl.BindPermanent("entered enchantment")
	card := dsl.NewCreature("Watcher", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(
		dsl.EnchantmentYouControlEnters().WithPermanent(entered),
		dsl.AddCounters(dsl.From(entered), core.P1P1, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestEnchantmentEntrySubtypeDeclaration(t *testing.T) {
	entered := dsl.BindPermanent("entered Aura")
	card := dsl.NewCreature("Watcher", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(
		dsl.EnchantmentYouControlEnters().OfSubtype("Aura").WithPermanent(entered),
		dsl.AddCounters(dsl.From(entered), core.P1P1, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestEnchantmentEntryResolutionSubtypeDeclaration(t *testing.T) {
	entered := dsl.BindPermanent("entered enchantment")
	card := dsl.NewCreature("Watcher", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(
		dsl.EnchantmentYouControlEnters().WithPermanent(entered),
		dsl.IfPermanentHasEnchantmentSubtype(dsl.From(entered), "Aura",
			dsl.GainLife(dsl.You, dsl.Fixed(1))))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from exile_result_test.go ---
func TestExileResultPublicDeclaration(t *testing.T) {
	result := dsl.BindNumber("exiled")
	target := dsl.TargetPermanent("target", dsl.IsCreature)
	card := dsl.NewInstant("Synthetic exile", "", dsl.Spell(dsl.Sequence(
		dsl.ExileWithResult(target, dsl.ExiledThisWayInto(result)),
		dsl.IfPositive(dsl.FromNumber(result), dsl.GainLife(dsl.You, dsl.Fixed(1))),
	), dsl.WithTargets(target)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from gain_life_each_test.go ---
func TestGainLifeEachPublicDeclaration(t *testing.T) {
	card := dsl.NewSorcery("Shared gain", "", dsl.WithAbility(dsl.Spell(dsl.GainLifeEach(dsl.EachPlayer, dsl.Fixed(3)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from graveyard_entry_tapped_test.go ---
func TestReturnCardToBattlefieldTappedAuthorExample(t *testing.T) {
	discarded := dsl.BindCards("discarded")
	picked := dsl.BindCard("picked")
	player := dsl.TargetPlayer("player")
	card := dsl.NewInstant("Synthetic return", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(
		dsl.Discard(player, dsl.Fixed(2), dsl.DiscardCardsInto(discarded)),
		dsl.ChooseUpToOneCard(dsl.You, dsl.FromCards(discarded), dsl.IsLandCard, picked),
		dsl.ReturnCardToBattlefield(dsl.FromCard(picked), dsl.You, dsl.EnterTapped()),
	), dsl.WithTargets(player))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from graveyard_optional_test.go ---
func TestOptionalGraveyardTargetPublicDeclaration(t *testing.T) {
	target := dsl.TargetGraveyardCard("card",
		dsl.GraveyardUpToOne(),
		dsl.GraveyardCardFilter(dsl.IsCreatureCard),
		dsl.GraveyardCardManaValueAtMost(dsl.ControllerLifeGainedThisTurn()))
	card := dsl.NewCreature("Synthetic watcher", "", 1, 1,
		dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This),
			dsl.ReturnCardToBattlefield(target, dsl.You), dsl.WithTargets(target))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from land_mana_amount_test.go ---
func TestLandManaAmountPublicDeclaration(t *testing.T) {
	amount := dsl.BindNumber("produced")
	player := dsl.BindPlayer("activator")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.LandTappedForMana().WithPlayer(player).WithAmount(amount),
		dsl.GainLife(dsl.FromPlayer(player), dsl.FromNumber(amount)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from land_mana_land_test.go ---
func TestLandManaLandPublicDeclaration(t *testing.T) {
	land := dsl.BindPermanent("producing land")
	player := dsl.BindPlayer("activator")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.LandTappedForMana().WithLand(land).WithPlayer(player),
		dsl.Tap(dsl.From(land)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from life_amount_test.go ---
func TestTypedCollectionLifeAmountComposition(t *testing.T) {
	saved := dsl.BindPermanent("amount")
	loss := dsl.LoseLifeEach(dsl.OpponentsOf(dsl.You), dsl.MaxValue(dsl.Fixed(0), dsl.PowerOf(dsl.From(saved))))
	body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), loss)
	player := dsl.TargetPlayer("player")
	target := dsl.TargetPermanent("amount", dsl.IsCreature)
	for _, card := range []mage.Card{
		dsl.NewSorcery("Target", "", dsl.WithAbility(dsl.Spell(dsl.LoseLifeEach(dsl.OpponentsOf(player), dsl.ToughnessOf(target)), dsl.WithTargets(player, target)))),
		dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewSorcery("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), saved, loss)))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Life", dsl.LoseLifeEach(dsl.EachPlayer, dsl.PowerOf(target)), dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from life_gain_result_test.go ---
func TestLifeGainedIntoPublicDeclaration(t *testing.T) {
	gained := dsl.BindNumber("life gained")
	card := dsl.NewArtifact("Synthetic", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Sequence(
		dsl.GainLifeEach(dsl.EachPlayer, dsl.Fixed(2), dsl.LifeGainedInto(gained)),
		dsl.LoseLife(dsl.You, dsl.FromNumber(gained)),
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from life_loss_result_test.go ---
func TestLifeLostIntoPublicDeclaration(t *testing.T) {
	lost := dsl.BindNumber("life lost")
	body := dsl.Sequence(
		dsl.LoseLifeEach(dsl.OpponentsOf(dsl.You), dsl.Fixed(4), dsl.LifeLostInto(lost)),
		dsl.GainLife(dsl.You, dsl.FromNumber(lost)),
	)
	for _, card := range []mage.Card{
		dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Dies(dsl.This), body))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatalf("%s: %v", card.Name(), err)
		}
	}
}

// --- from life_outcome_test.go ---
func TestTypedLifeGainOutcomeSequence(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	source := dsl.NewCreature("Modifier", "", 1, 1)
	g.PutOnBattlefield(source, a.PlayerID())
	g.AddLifeGainModifier(source.ID(), func(_ *mage.Game, _ uuid.UUID, amount int) int { return amount * 2 })
	body := dsl.Sequence(dsl.GainLife(dsl.You, dsl.Fixed(3)), dsl.GainLife(dsl.You, dsl.Fixed(2)))
	card := dsl.NewSorcery("Life", "", dsl.WithAbility(dsl.Spell(body)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := mage.ApplyEffect(g, body, card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 30 || b.Life() != 20 || g.PlayerLifeGainedThisTurn(a.PlayerID()) != 10 {
		t.Fatal("sequence did not report each actual gain")
	}
}

// --- from main_phase_test.go ---
func TestMainPhaseComposition(t *testing.T) {
	for _, pattern := range []dsl.TurnStepPattern{dsl.YourFirstMainPhase, dsl.YourPostcombatMainPhase} {
		body := dsl.GainLife(dsl.You, dsl.Fixed(1))
		for _, ability := range []mage.TriggeredAbility{
			dsl.At(pattern, body),
			dsl.AtModes(pattern, dsl.ModeOf("One", body), dsl.ModeOf("Two", body)),
			dsl.At(dsl.YourUpkeep, dsl.AtNext(pattern, body)),
		} {
			card := dsl.NewEnchantment("Main phase", "", dsl.WithAbility(ability))
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
		}
		if info := dsl.InspectDefinition(pattern); info.Role != "turn-step-pattern" || info.Text == "" {
			t.Fatalf("inspection = %+v", info)
		}
	}
}

// --- from mana_amount_test.go ---
func TestTypedManaAmountComposition(t *testing.T) {
	target := dsl.TargetPermanent("amount", dsl.IsCreature)
	for _, ability := range []mage.Ability{
		dsl.Spell(dsl.AddManaAmount(core.Black, dsl.PowerOf(target)), dsl.WithTargets(target)),
		dsl.SpellModes(dsl.ModeOf("Mana", dsl.AddManaAmount(core.Black, dsl.Fixed(2))), dsl.ModeOf("Life", dsl.GainLife(dsl.You, dsl.Fixed(2)))),
		dsl.At(dsl.YourFirstMainPhase, dsl.AddManaAmount(core.Black, dsl.Fixed(2))),
		dsl.Activate(dsl.Costs(dsl.TapCost(dsl.This)), dsl.AddManaAmount(core.Black, dsl.Fixed(2))),
	} {
		card := dsl.NewArtifact("Mana", "", dsl.WithAbility(ability))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	source := dsl.NewCreature("Amount", "", 3, 4)
	g.PutOnBattlefield(source, a.PlayerID())
	ability := dsl.Activate(dsl.Costs(), dsl.Sequence(dsl.AddManaAmount(core.Black, dsl.PowerOf(target)), dsl.GainLife(dsl.You, dsl.Fixed(1))), dsl.WithTargets(target))
	card := dsl.NewArtifact("Targeted mana", "", dsl.WithAbility(ability))
	p := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{source.ID()}); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != 1 || a.ManaPool().Count(core.Black) != 0 {
		t.Fatal("targeted mana bypassed stack")
	}
	g.ResolveStack()
	if a.ManaPool().Count(core.Black) != 3 || a.Life() != 21 {
		t.Fatal("targeted amount lost resolution context")
	}

}

// --- from mill_test.go ---
func TestTypedMillComposition(t *testing.T) {
	player := dsl.TargetPlayer("player")
	body := dsl.Sequence(dsl.Mill(dsl.You, dsl.Fixed(2)), dsl.Draw(dsl.You, dsl.Fixed(1)))
	for _, card := range []mage.Card{
		dsl.NewInstant("Target", "", dsl.WithAbility(dsl.Spell(dsl.Mill(player, dsl.Fixed(2)), dsl.WithTargets(player)))),
		dsl.NewSorcery("Sequence", "", dsl.WithAbility(dsl.Spell(body))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Mill", body), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewCreature("Entry target", "", 0, 4, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), dsl.Mill(player, dsl.Fixed(4)), dsl.WithTargets(player)))),
		dsl.NewCreature("Paid activation", "", 1, 4, dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.Mana("{U}"), dsl.TapCost(dsl.This)), dsl.Mill(player, dsl.Fixed(2)), dsl.WithTargets(player)))),
		dsl.NewArtifact("Generic paid activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.Mana("{2}"), dsl.TapCost(dsl.This)), dsl.Mill(player, dsl.Fixed(2)), dsl.WithTargets(player)))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Dies(dsl.This), body))),
		dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTypedMillSequenceAfterSourcePhasesIn(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	source := dsl.NewArtifact("Modifier", "")
	g.PutOnBattlefield(source, b.PlayerID())
	g.AddMillReplacement(source.ID(), dsl.DoubleMillForOpponents(source.ID(), b.PlayerID()))
	a.SetLibrary([]mage.Card{
		dsl.NewArtifact("One", ""), dsl.NewArtifact("Two", ""),
		dsl.NewArtifact("Three", ""), dsl.NewArtifact("Four", ""),
		dsl.NewArtifact("Five", ""), dsl.NewArtifact("Six", ""),
	})
	body := dsl.Sequence(dsl.Mill(dsl.You, dsl.Fixed(1)), dsl.Draw(dsl.You, dsl.Fixed(1)))
	g.PhaseOut(source.ID())
	if err := mage.ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Graveyard()) != 1 || len(a.Hand()) != 1 || a.Hand()[0].Name() != "Two" {
		t.Fatal("phased source changed the mill or the following draw")
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	if err := mage.ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Graveyard()) != 3 || len(a.Hand()) != 2 || a.Hand()[1].Name() != "Five" || len(a.Library()) != 1 {
		t.Fatal("returned modifier did not preserve the mill and draw sequence")
	}
}

func TestTypedMillUsesModifierCurrentController(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	source := dsl.NewArtifact("Modifier", "")
	g.PutOnBattlefield(source, a.PlayerID())
	g.AddMillReplacement(source.ID(), dsl.DoubleMillForOpponents(source.ID(), a.PlayerID()))
	if err := mage.ApplyEffect(g, dsl.GainControl(), b.PlayerID(), b.PlayerID(), []uuid.UUID{source.ID()}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		player mage.Player
		want   int
		drawn  string
	}{{a, 2, "Three"}, {b, 1, "Two"}} {
		tc.player.SetLibrary([]mage.Card{dsl.NewArtifact("One", ""), dsl.NewArtifact("Two", ""), dsl.NewArtifact("Three", ""), dsl.NewArtifact("Four", "")})
		body := dsl.Sequence(dsl.Mill(dsl.You, dsl.Fixed(1)), dsl.Draw(dsl.You, dsl.Fixed(1)))
		if err := mage.ApplyEffect(g, body, source.ID(), tc.player.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(tc.player.Graveyard()) != tc.want || len(tc.player.Hand()) != 1 || tc.player.Hand()[0].Name() != tc.drawn {
			t.Fatalf("wrong mill or following draw for %s: graveyard=%d hand=%v", tc.player.Name(), len(tc.player.Graveyard()), tc.player.Hand())
		}
	}
}

func TestTypedMillStaticReplacementSequence(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	source := dsl.NewCreature("Static mill", "", 1, 4, dsl.WithMillReplacement(dsl.DoubleMillForOpponents))
	if err := mage.ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, b.PlayerID())
	a.SetLibrary([]mage.Card{dsl.NewArtifact("One", ""), dsl.NewArtifact("Two", ""), dsl.NewArtifact("Three", ""), dsl.NewArtifact("Four", "")})
	body := dsl.Sequence(dsl.Mill(dsl.You, dsl.Fixed(1)), dsl.Draw(dsl.You, dsl.Fixed(1)))
	if err := mage.ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Graveyard()) != 2 || len(a.Hand()) != 1 || a.Hand()[0].Name() != "Three" || len(a.Library()) != 1 {
		t.Fatal("static replacement did not preserve the immediate mill and draw sequence")
	}
}

// --- from opponent_damage_history_test.go ---
func TestOpponentDamageHistoryDeclaration(t *testing.T) {
	card := dsl.NewCreature("History watcher", "", 1, 1,
		dsl.WithAbility(dsl.At(dsl.EachEndStep,
			dsl.GainLife(dsl.You, dsl.Fixed(1)),
			dsl.WithInterveningIf(dsl.SourceDamagedOpponentThisTurnCondition()))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from opponent_life_test.go ---
func TestTypedOpponentLifeComposition(t *testing.T) {
	body := dsl.Sequence(dsl.LoseLifeEach(dsl.OpponentsOf(dsl.You), dsl.Fixed(3)), dsl.GainLife(dsl.You, dsl.Fixed(3)))
	player := dsl.TargetPlayer("player")
	for _, card := range []mage.Card{
		dsl.NewSorcery("Life", "", dsl.WithAbility(dsl.Spell(body))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Life", body), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Dies(dsl.This), body))),
		dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewSorcery("Target", "", dsl.WithAbility(dsl.Spell(dsl.LoseLifeEach(dsl.OpponentsOf(player), dsl.Fixed(3)), dsl.WithTargets(player)))),
		dsl.NewSorcery("Draw", "", dsl.WithAbility(dsl.Spell(dsl.DrawEach(dsl.OpponentsOf(dsl.You), dsl.Fixed(1))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from optional_reveal_test.go ---
func TestPublicOptionalRevealDeclaration(t *testing.T) {
	paid := dsl.BindNumber("paid")
	revealed := dsl.BindCard("revealed")
	card := dsl.NewInstant("Optional reveal", "",
		dsl.WithAbility(dsl.Spell(dsl.Sequence(
			dsl.GainLife(dsl.You, dsl.FromNumber(paid)),
			dsl.IfCardHasType(dsl.FromCard(revealed), core.TypeCreature,
				dsl.GainLife(dsl.You, dsl.Fixed(2)))))),
		dsl.WithAdditionalCost(dsl.OptionalRevealFromHandCost(
			dsl.IsCreatureCard, "Reveal a creature card", "Reveal a creature?", paid, revealed)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from owner_test.go ---
func TestOwnerOfPublicDeclaration(t *testing.T) {
	target := dsl.TargetPermanent("permanent", dsl.AnyPermanent)
	owner := dsl.BindPlayer("owner")
	card := dsl.NewInstant("Owner", "", dsl.WithAbility(dsl.Spell(
		dsl.Sequence(
			dsl.SnapshotPlayer(dsl.OwnerOf(target), owner),
			dsl.GainLife(dsl.FromPlayer(owner), dsl.Fixed(1)),
		),
		dsl.WithTargets(target),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from player_damage_source_test.go ---
func TestPlayerDamageSourcePublicDeclaration(t *testing.T) {
	player := dsl.BindPlayer("damaged")
	source := dsl.BindDamageSource("source")
	card := dsl.NewCreature("Synthetic Damage Watcher", "", 1, 3, dsl.WithAbility(dsl.WhenEvent(
		dsl.DealsDamageToPlayer(dsl.This, player).WithSource(source),
		dsl.DealDamageFrom(dsl.FromDamageSource(source), dsl.FromPlayer(player), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from receives_damage_source_test.go ---
func TestReceivesDamageSourcePublicDeclaration(t *testing.T) {
	amount := dsl.BindNumber("amount")
	source := dsl.BindDamageSource("source")
	card := dsl.NewCreature("Synthetic Damage Watcher", "", 1, 5, dsl.WithAbility(dsl.WhenEvent(
		dsl.ReceivesDamage(dsl.This, amount).WithSource(source),
		dsl.DealDamageFrom(dsl.FromDamageSource(source), dsl.You, dsl.FromNumber(amount)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from regeneration_test.go ---
func TestTypedRegenerationComposition(t *testing.T) {
	target := dsl.TargetPermanent("subject", dsl.AnyPermanent)
	saved := dsl.BindPermanent("saved")
	member := dsl.BindPermanent("member")
	for _, card := range []mage.Card{
		dsl.NewInstant("Target", "", dsl.WithAbility(dsl.Spell(dsl.Regenerate(target), dsl.WithTargets(target)))),
		dsl.NewCreature("Activation", "", 2, 2, dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Regenerate(dsl.This)))),
		dsl.NewCreature("Trigger", "", 2, 2, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.Regenerate(dsl.From(saved)))))),
		dsl.NewInstant("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), member, dsl.Regenerate(dsl.From(member)))))),
		dsl.NewInstant("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Shield", dsl.Regenerate(target), dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from return_each_test.go ---
func TestReturnEachToHandPublicDeclaration(t *testing.T) {
	player := dsl.TargetPlayer("player")
	card := dsl.NewInstant("Return owned artifacts", "{1}{U}", dsl.WithAbility(dsl.Spell(
		dsl.ReturnEachToHand(dsl.PermanentsOwnedBy(player, dsl.IsArtifact)),
		dsl.WithTargets(player),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from reverse_damage_turn_set_test.go ---
func TestReverseTurnDamageSetDeclaration(t *testing.T) {
	card := dsl.NewCreature("Watcher", "", 1, 4,
		dsl.WithAbility(dsl.WhenEvent(dsl.Dies(dsl.This),
			dsl.TapEach(dsl.PermanentsThatDealtDamageToThisTurn(dsl.This,
				dsl.MatchingPermanents(dsl.IsCreature))))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from sacrifice_subtype_result_test.go ---
func TestSacrificeSubtypeResultComposition(t *testing.T) {
	land := dsl.BindPermanent("chosen land")
	wasIsland := dsl.BindNumber("was Island")
	body := dsl.Sequence(
		dsl.ChoosePermanent(dsl.You, dsl.ControlledPermanents(dsl.IsLand), land),
		dsl.Sacrifice(dsl.From(land), dsl.SacrificeSubtypeInto("Island", wasIsland)),
		dsl.IfPositive(dsl.FromNumber(wasIsland), dsl.DealDamageFrom(dsl.This, dsl.You, dsl.Fixed(3))),
	)
	card := dsl.NewCreature("Synthetic", "", 2, 2, dsl.WithAbility(dsl.At(dsl.YourUpkeep, body)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from singular_life_amount_test.go ---
func TestTypedSingularLifeAmountComposition(t *testing.T) {
	for _, constructor := range []func(dsl.PlayerRef, dsl.ResolutionValue) dsl.ResolvingEffect{dsl.GainLife, dsl.LoseLife} {
		saved := dsl.BindPermanent("amount")
		effect := constructor(dsl.You, dsl.MaxValue(dsl.Fixed(0), dsl.PowerOf(dsl.From(saved))))
		body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), effect)
		player := dsl.TargetPlayer("player")
		target := dsl.TargetPermanent("amount", dsl.IsCreature)
		for _, card := range []mage.Card{
			dsl.NewSorcery("Target", "", dsl.WithAbility(dsl.Spell(constructor(player, dsl.ToughnessOf(target)), dsl.WithTargets(player, target)))),
			dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
			dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
			dsl.NewCreature("Delayed", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.AtNext(dsl.EachEndStep, effect))))),
			dsl.NewSorcery("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), saved, effect)))),
			dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Life", constructor(dsl.You, dsl.PowerOf(target)), dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		} {
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// --- from singular_life_gain_result_test.go ---
func TestSingularLifeGainResultPublicDeclaration(t *testing.T) {
	gained := dsl.BindNumber("gained")
	card := dsl.NewSorcery("Life result", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(
		dsl.GainLifeForWithResult(dsl.You, dsl.Fixed(2), dsl.LifeGainedInto(gained)),
		dsl.LoseLife(dsl.You, dsl.FromNumber(gained)),
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from singular_life_loss_result_test.go ---
func TestSingularLifeLossResultPublicDeclaration(t *testing.T) {
	lost := dsl.BindNumber("life lost")
	target := dsl.TargetPlayer("opponent", dsl.TargetOpponentOnly())
	body := dsl.Sequence(
		dsl.LoseLifeForWithResult(target, dsl.Fixed(4), dsl.LifeLostInto(lost)),
		dsl.GainLife(dsl.You, dsl.FromNumber(lost)),
	)
	for _, card := range []mage.Card{
		dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target)))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body, dsl.WithTargets(target)))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.CreatureDies(), body, dsl.WithTargets(target)))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatalf("%s: %v", card.Name(), err)
		}
	}
}

// --- from spell_cast_mana_test.go ---
func TestSpellCastManaSpentPublicDeclaration(t *testing.T) {
	spent := dsl.BindNumber("mana spent")
	card := dsl.NewArtifact("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.SpellCast().WithManaSpent(spent),
		dsl.GainLife(dsl.You, dsl.FromNumber(spent)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from spell_cast_owner_test.go ---
func TestSpellCastOwnershipPublicDefinition(t *testing.T) {
	card := dsl.NewArtifact("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.SpellCast().NotOwnedByCaster(), dsl.Draw(dsl.You, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from spell_cast_subtype_test.go ---
func TestSpellCastEnchantmentSubtypeDSL(t *testing.T) {
	pattern := dsl.SpellCast().ByOpponent().OfType(core.TypeEnchantment).OfEnchantmentSubtype("Aura")
	card := dsl.NewArtifact("Watcher", "", dsl.WithAbility(dsl.WhenEvent(pattern, dsl.Draw(dsl.You, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if got := dsl.InspectDefinition(pattern); got.Role != "event-pattern" {
		t.Fatalf("inspection=%+v", got)
	}
}

// --- from spell_cast_x_test.go ---
func TestSpellCastXPublicDeclaration(t *testing.T) {
	card := dsl.NewCreature("X watcher", "{2}{U}", 2, 2, dsl.WithAbility(dsl.WhenEvent(
		dsl.SpellCast().WithXInManaCost(), dsl.Until(core.EndOfTurn, dsl.CantBeBlocked(dsl.This)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from spell_cast_zone_test.go ---
func TestSpellCastZonePublicDefinition(t *testing.T) {
	caster := dsl.BindPlayer("caster")
	card := dsl.NewArtifact("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.SpellCast().FromZone(core.ZoneHand).WithCaster(caster),
		dsl.GainLife(dsl.FromPlayer(caster), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from spell_step_test.go ---
func TestSpellStepComposition(t *testing.T) {
	var timing = dsl.OnlyDuring(core.BeginCombat, core.DeclareAttackers)
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	ability := dsl.Spell(dsl.Until(core.EndOfTurn, dsl.CantBeBlocked(target)), dsl.WithTargets(target), timing)
	card := dsl.NewInstant("Timed spell", "", dsl.WithAbility(ability))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	info := dsl.InspectDefinition(ability)
	if len(info.Children) == 0 || info.Children[0].Role != "spell-step" || info.Children[0].Text != "Begin Combat or Declare Attackers" {
		t.Fatalf("timing inspection = %+v", info)
	}
}

// --- from tap_each_test.go ---
func TestTapEachAuthorExample(t *testing.T) {
	card := dsl.NewInstant("Blue tap", "", dsl.WithAbility(dsl.Spell(
		dsl.TapEach(dsl.MatchingPermanents(dsl.HasColorFilter(dsl.Blue))))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from tap_result_test.go ---
func TestTypedTapResultDeclaration(t *testing.T) {
	done := dsl.BindNumber("tap result")
	card := dsl.NewArtifact("Synthetic tap", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Sequence(
		dsl.Tap(dsl.This, dsl.TapAmountInto(done)),
		dsl.IfPositive(dsl.FromNumber(done), dsl.GainLife(dsl.You, dsl.Fixed(2))),
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from token_result_test.go ---
func TestOneTokenResultPublicDeclaration(t *testing.T) {
	created := dsl.BindPermanent("created Cat")
	cat := dsl.DefineToken("Cat", dsl.TokenTypes(core.TypeCreature), dsl.TokenSubtypes("Cat"), dsl.TokenColors(core.White), dsl.TokenPT(2, 2))
	card := dsl.NewInstant("Cat Maker", "", dsl.Spell(dsl.Sequence(
		dsl.CreateOneToken(dsl.You, cat, created),
		dsl.AddCounters(dsl.From(created), core.P1P1, dsl.Fixed(1)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from token_set_test.go ---
func TestCreatedTokenSetComposition(t *testing.T) {
	created := dsl.BindCreatedTokens("created")
	member := dsl.BindPermanent("member")
	cat := dsl.DefineToken("Cat", dsl.TokenTypes(core.TypeCreature), dsl.TokenPT(1, 1))
	body := dsl.Sequence(
		dsl.CreateTokenCopiesInto(dsl.You, dsl.Fixed(2), cat, created),
		dsl.ForEach(dsl.FromCreatedTokens(created), member, dsl.Tap(dsl.From(member))),
	)
	card := dsl.NewInstant("Maker", "", dsl.WithAbility(dsl.Spell(body)))
	if err := mage.ValidateCardDefinition("Maker", card); err != nil {
		t.Fatal(err)
	}
}

// --- from untap_each_test.go ---
func TestUntapEachAuthorExample(t *testing.T) {
	card := dsl.NewInstant("Land untap", "", dsl.WithAbility(dsl.Spell(
		dsl.UntapEach(dsl.ControlledPermanents(dsl.IsLand)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from untap_result_test.go ---
func TestUntapResultPublicDeclaration(t *testing.T) {
	untapped := dsl.BindNumber("untapped")
	target := dsl.TargetPermanent("target", dsl.IsCreature)
	card := dsl.NewInstant("Synthetic untap", "", dsl.Spell(dsl.Sequence(
		dsl.UntapPermanentWithResult(target, dsl.UntappedThisWayInto(untapped)),
		dsl.IfPositive(dsl.FromNumber(untapped), dsl.GainLife(dsl.You, dsl.Fixed(1))),
	), dsl.WithTargets(target)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
