package dsl_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

// --- from ability_activation_event_test.go ---
func TestAbilityActivatedPublicDeclaration(t *testing.T) {
	actor := dsl.BindPlayer("actor")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.ActivatedAbilityEvent().WithActivator(actor), dsl.GainLife(dsl.FromPlayer(actor), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestAbilityActivatedSourcePublicDeclaration(t *testing.T) {
	actor := dsl.BindPlayer("actor")
	source := dsl.BindDamageSource("source")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.ActivatedAbilityEvent().WithActivator(actor).WithSource(source),
		dsl.DealDamageFrom(dsl.FromDamageSource(source), dsl.FromPlayer(actor), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from any_draw_card_event_test.go ---
func TestAnyDrawCardEventPublicDeclaration(t *testing.T) {
	drawer, drawn := dsl.BindPlayer("drawer"), dsl.BindCard("drawn")
	for _, pattern := range []dsl.EventPattern{
		dsl.PlayerDrawsCard(drawer).WithCard(drawn),
		dsl.OpponentDrawsCard(drawer).WithCard(drawn),
	} {
		card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(pattern,
			dsl.IfCardHasType(dsl.FromCard(drawn), core.TypeLand,
				dsl.GainLife(dsl.FromPlayer(drawer), dsl.Fixed(1))))))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from any_draw_event_test.go ---
func TestPlayerDrawsCardPublicDeclaration(t *testing.T) {
	drawer := dsl.BindPlayer("drawer")
	card := dsl.NewEnchantment("Draw watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.PlayerDrawsCard(drawer), dsl.LoseLife(dsl.FromPlayer(drawer), dsl.Fixed(2)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from any_life_gain_event_test.go ---
func TestPlayerGainsLifePublicDeclaration(t *testing.T) {
	player, amount := dsl.BindPlayer("gainer"), dsl.BindNumber("gained")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.PlayerGainsLife(player).WithAmount(amount),
		dsl.LoseLife(dsl.FromPlayer(player), dsl.FromNumber(amount)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from any_life_loss_event_test.go ---
func TestPlayerLosesLifePublicDeclaration(t *testing.T) {
	player, amount := dsl.BindPlayer("loser"), dsl.BindNumber("lost")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.PlayerLosesLife(player).WithAmount(amount),
		dsl.GainLife(dsl.FromPlayer(player), dsl.FromNumber(amount)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from artifact_activation_event_test.go ---
func TestArtifactActivationEventDSL(t *testing.T) {
	controller := dsl.BindPlayer("controller")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.ArtifactAbilityActivatedWithoutTapCost().WithController(controller),
		dsl.GainLife(dsl.FromPlayer(controller), dsl.Fixed(1)))))
	if card == nil || dsl.InspectDefinition(dsl.ArtifactAbilityActivatedWithoutTapCost()).Role != "event-pattern" {
		t.Fatal("artifact activation pattern is not available")
	}
}

// --- from creature_dies_test.go ---
func TestCreatureDiesPattern(t *testing.T) {
	var pattern = dsl.CreatureDies()
	target := dsl.TargetPlayer("player")
	for _, ability := range []mage.TriggeredAbility{
		dsl.WhenEvent(pattern, dsl.Mill(target, dsl.Fixed(1)), dsl.WithTargets(target)),
		dsl.WhenEventModes(pattern,
			dsl.ModeOf("Mill", dsl.Mill(target, dsl.Fixed(1)), dsl.WithTargets(target)),
			dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))),
	} {
		card := dsl.NewCreature("Watcher", "", 1, 1, dsl.WithAbility(ability))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		var found bool
		var visit func(mage.DefinitionMetadata)
		visit = func(n mage.DefinitionMetadata) {
			found = found || n.Role == "event-pattern" && n.Text == "a creature dies"
			for _, child := range n.Children {
				visit(child)
			}
		}
		visit(mage.InspectDefinition(ability))
		if !found {
			t.Fatal("missing creature death pattern metadata")
		}
	}
}
func TestCreatureYouControlDiesPattern(t *testing.T) {
	var pattern = dsl.CreatureYouControlDies()
	target := dsl.TargetPlayer("player")
	for _, ability := range []mage.TriggeredAbility{
		dsl.WhenEvent(pattern, dsl.Mill(target, dsl.Fixed(1)), dsl.WithTargets(target)),
		dsl.WhenEventModes(pattern,
			dsl.ModeOf("Mill", dsl.Mill(target, dsl.Fixed(1)), dsl.WithTargets(target)),
			dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))),
	} {
		card := dsl.NewCreature("Watcher", "", 1, 1, dsl.WithAbility(ability))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		var found bool
		var visit func(mage.DefinitionMetadata)
		visit = func(n mage.DefinitionMetadata) {
			found = found || n.Role == "event-pattern" && n.Text == "a creature you control dies"
			for _, child := range n.Children {
				visit(child)
			}
		}
		visit(mage.InspectDefinition(ability))
		if !found {
			t.Fatal("missing creature death pattern metadata")
		}
	}
}

// --- from damage_recipient_event_test.go ---
func TestDamagePermanentRecipientPublicDeclaration(t *testing.T) {
	recipient := dsl.BindPermanent("recipient")
	card := dsl.NewCreature("Damage watcher", "", 1, 1, dsl.WithAbility(
		dsl.WhenEvent(dsl.DealsDamage(dsl.This, dsl.BindNumber("dealt")).WithPermanent(recipient),
			dsl.AddCounters(dsl.From(recipient), dsl.P1P1, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from damaged_creature_death_test.go ---
func TestDamagedCreatureDeathDeclaration(t *testing.T) {
	card := dsl.NewCreature("Synthetic watcher", "", 1, 5,
		dsl.WithAbility(dsl.WhenEvent(
			dsl.CreatureDies().DamagedByThisTurn(),
			dsl.GainLife(dsl.You, dsl.Fixed(1)),
		)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from death_count_test.go ---
func TestCreatureDeathCountComposition(t *testing.T) {
	var value = dsl.CreaturesDiedThisTurn()
	member := dsl.BindPermanent("member")
	bodies := []dsl.ResolvingEffect{
		dsl.GainLife(dsl.You, value),
		dsl.Sequence(dsl.GainLife(dsl.You, value), dsl.Draw(dsl.You, dsl.Fixed(1))),
		dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), member, dsl.AddCounters(dsl.From(member), core.Corpse, value)),
	}
	for _, body := range bodies {
		cards := []dsl.Card{
			dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body))),
			dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.At(dsl.EachEndStep, body))),
			dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
			dsl.NewInstant("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Count", body), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		}
		for _, card := range cards {
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
		}
	}
	info := dsl.InspectDefinition(value)
	if info.Role != "creature-death-count" || info.Text != value.Text() || len(info.Children) != 0 {
		t.Fatalf("metadata = %+v", info)
	}
}

// --- from death_outcome_test.go ---
func TestPermanentDeathOutcomeDSL(t *testing.T) {
	former := dsl.BindPlayer("former controller")
	card := dsl.NewArtifact("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.PermanentDies().OfType(core.TypeArtifact).YouControlled().NotSacrificed().WithController(former),
		dsl.GainLife(dsl.FromPlayer(former), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from death_permanent_test.go ---
func TestDeathPermanentPublicDeclaration(t *testing.T) {
	dead := dsl.BindPermanent("departing creature")
	card := dsl.NewCreature("Watcher", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(
		dsl.AnotherCreatureDies().WithPermanent(dead),
		dsl.DealDamageFrom(dsl.DamageSourceOf(dsl.From(dead)), dsl.You, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from death_power_test.go ---
func TestCreatureDeathPowerPublicDeclaration(t *testing.T) {
	power := dsl.BindNumber("dying creature power")
	card := dsl.NewCreature("Power watcher", "", 1, 1, dsl.WithAbility(
		dsl.WhenEvent(dsl.AnotherCreatureYouControlDies().WithPower(power),
			dsl.GainLife(dsl.You, dsl.FromNumber(power)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from death_pt_filter_test.go ---
func TestDeathPowerOrToughnessFilterPublic(t *testing.T) {
	opponent := dsl.TargetPlayer("opponent", dsl.TargetOpponentOnly())
	card := dsl.NewCreature("Watcher", "", 2, 2, dsl.WithAbility(dsl.WhenEvent(
		dsl.CreatureYouControlDies().PowerOrToughnessAtMost(1),
		dsl.Sequence(dsl.LoseLife(opponent, dsl.Fixed(2)), dsl.GainLife(dsl.You, dsl.Fixed(2))),
		dsl.WithTargets(opponent))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from death_subtype_test.go ---
func TestCreatureDeathSubtypePublicDeclaration(t *testing.T) {
	card := dsl.NewCreature("Goblin watcher", "", 1, 1, dsl.WithAbility(
		dsl.WhenEvent(dsl.AnotherCreatureDies().OfSubtype("Goblin"),
			dsl.GainLife(dsl.You, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from delayed_death_test.go ---
func TestDelayedDeathAuthorSurface(t *testing.T) {
	target := dsl.TargetPermanent("watched", dsl.IsCreature)
	body := dsl.WhenDiesThisTurn(target, dsl.GainLife(dsl.You, dsl.Fixed(3)))
	card := dsl.NewSorcery("Watch", "", dsl.Spell(body, dsl.WithTargets(target)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if mage.InspectDefinition(body).Role != "delayed-event" {
		t.Fatal("missing delayed event descriptor")
	}
}

func TestDelayedLeaveAuthorSurface(t *testing.T) {
	watched := dsl.TargetPermanent("watched", dsl.IsCreature)
	body := dsl.WhenLeavesThisTurn(watched, dsl.GainLife(dsl.You, dsl.Fixed(3)))
	card := dsl.NewSorcery("Watch", "", dsl.Spell(body, dsl.WithTargets(watched)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if meta := mage.InspectDefinition(body); meta.Role != "delayed-event" || meta.Text != "when that permanent leaves the battlefield this turn" {
		t.Fatalf("unexpected descriptor: %+v", meta)
	}
}

// --- from discard_card_event_test.go ---
func TestDiscardCardEventPublicDeclaration(t *testing.T) {
	discarder := dsl.BindPlayer("discarder")
	discarded := dsl.BindCard("discarded")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.OpponentDiscardsCard(discarder).WithCard(discarded),
		dsl.IfCardHasType(dsl.FromCard(discarded), core.TypeLand,
			dsl.LoseLife(dsl.FromPlayer(discarder), dsl.Fixed(2))))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from draw_card_event_capture_test.go ---
func TestControllerDrawCardCaptureDeclaration(t *testing.T) {
	drawer, drawn := dsl.BindPlayer("drawer"), dsl.BindCard("drawn")
	card := dsl.NewEnchantment("Draw Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.ControllerDrawsCardInto(drawer, drawn),
		dsl.IfCardHasType(dsl.FromCard(drawn), core.TypeLand, dsl.GainLife(dsl.FromPlayer(drawer), dsl.Fixed(1))))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from draw_event_test.go ---
func TestControllerDrawEventComposition(t *testing.T) {
	player := dsl.BindPlayer("drawer")
	var pattern = dsl.ControllerDrawsCard(player)
	body := dsl.GainLife(dsl.FromPlayer(player), dsl.Fixed(1))
	for _, card := range []mage.Card{
		dsl.NewEnchantment("Draw", "", dsl.WithAbility(dsl.WhenEvent(pattern, body))),
		dsl.NewEnchantment("Modes", "", dsl.WithAbility(dsl.WhenEventModes(pattern,
			dsl.ModeOf("Gain", body), dsl.ModeOf("Later", dsl.AtNext(dsl.EachEndStep, body))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	if info := mage.InspectDefinition(body); info.Properties.LifeGain != 1 {
		t.Fatalf("metadata = %+v", info)
	}
}

// --- from event_amount_test.go ---
func TestTypedDamageEventAmount(t *testing.T) {
	for _, variant := range []int{0, 1, 2} {
		player, amount := dsl.BindPlayer("damaged player"), dsl.BindNumber("damage")
		pattern := dsl.DealsDamageToPlayer(dsl.This, player).WithAmount(amount)
		if variant == 1 {
			pattern = dsl.DealsDamageToOpponent(dsl.This, player).WithAmount(amount)
		}
		if variant == 2 {
			pattern = dsl.DealsDamage(dsl.This, amount)
		}
		target := dsl.TargetPlayer("mill player")
		body := dsl.Mill(target, dsl.FromNumber(amount))
		card := dsl.NewCreature("Source", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(pattern, body, dsl.WithTargets(target))))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		var capture, value bool
		var visit func(mage.DefinitionMetadata)
		visit = func(n mage.DefinitionMetadata) {
			capture = capture || n.Role == "event-amount-capture"
			value = value || n.Role == "resolution-value" && n.Text == "damage"
			for _, child := range n.Children {
				visit(child)
			}
		}
		visit(mage.InspectDefinition(card.Abilities()[0]))
		if !capture || !value {
			t.Fatal("missing amount definition metadata")
		}
	}
}

// --- from event_player_test.go ---
func TestTypedDamageEventPlayer(t *testing.T) {
	player := dsl.BindPlayer("damaged player")
	card := dsl.NewCreature("Source", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(
		dsl.DealsDamageToPlayer(dsl.This, player), dsl.DiscardAtRandom(dsl.FromPlayer(player), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if mage.InspectDefinition(card.Abilities()[0]).Role != "event-trigger" {
		t.Fatal("missing event metadata")
	}
}

func TestTypedDamageEventOpponent(t *testing.T) {
	for _, modal := range []bool{false, true} {
		player := dsl.BindPlayer("damaged opponent")
		pattern := dsl.DealsDamageToOpponent(dsl.This, player)
		body := dsl.DiscardAtRandom(dsl.FromPlayer(player), dsl.Fixed(1))
		trigger := dsl.WhenEvent(pattern, body)
		if modal {
			trigger = dsl.WhenEventModes(pattern, dsl.ModeOf("Discard", body), dsl.ModeOf("Draw", dsl.Draw(dsl.FromPlayer(player), dsl.Fixed(1))))
		}
		card := dsl.NewCreature("Source", "", 1, 1, dsl.WithAbility(trigger))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		var opponent, discard bool
		var visit func(mage.DefinitionMetadata)
		visit = func(n mage.DefinitionMetadata) {
			opponent = opponent || n.Role == "event-pattern" && n.Text == "source deals damage to an opponent"
			discard = discard || n.Text == "discard at random for damaged opponent" && n.Properties.Outcome == mage.OutcomeDetriment
			for _, child := range n.Children {
				visit(child)
			}
		}
		visit(mage.InspectDefinition(card.Abilities()[0]))
		if !opponent || !discard {
			t.Fatalf("modal=%v opponent=%v discard=%v", modal, opponent, discard)
		}
	}
}

// --- from graveyard_leave_event_test.go ---
func TestGraveyardLeaveEventPublicDeclaration(t *testing.T) {
	count := dsl.BindNumber("left")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.OneOrMoreCardsLeaveYourGraveyard().WithCount(count),
		dsl.GainLife(dsl.You, dsl.FromNumber(count)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from land_entry_event_test.go ---
func TestLandEntryEventDSL(t *testing.T) {
	controller := dsl.BindPlayer("land controller")
	card := dsl.NewArtifact("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.LandEnters().WithController(controller),
		dsl.DealDamageFrom(dsl.This, dsl.FromPlayer(controller), dsl.Fixed(2)))))
	if card == nil || dsl.InspectDefinition(dsl.LandEnters()).Role != "event-pattern" {
		t.Fatal("land entry pattern is not available")
	}
}

func TestLandEntrySubtypeDSL(t *testing.T) {
	controller := dsl.BindPlayer("controller")
	card := dsl.NewArtifact("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.LandEnters().OfSubtype("Swamp").WithController(controller),
		dsl.DealDamageFrom(dsl.This, dsl.FromPlayer(controller), dsl.Fixed(3)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if got := dsl.InspectDefinition(dsl.LandEnters().OfSubtype("Swamp")); got.Role != "event-pattern" {
		t.Fatalf("inspection: %+v", got)
	}
}

func TestLandEntryActorDSL(t *testing.T) {
	actor := dsl.BindPlayer("actor")
	controller := dsl.BindPlayer("controller")
	pattern := dsl.LandEnters().OfSubtype("Swamp").WithActor(actor).WithController(controller)
	card := dsl.NewArtifact("Watcher", "", dsl.WithAbility(dsl.WhenEvent(pattern,
		dsl.DealDamageFrom(dsl.This, dsl.FromPlayer(actor), dsl.Fixed(3)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if got := dsl.InspectDefinition(pattern); got.Role != "event-pattern" {
		t.Fatalf("inspection: %+v", got)
	}
}

// --- from library_choice_event_test.go ---
func TestLibraryChoiceCompletionEvents(t *testing.T) {
	for _, surveil := range []bool{false, true} {
		for _, adapter := range []string{"primitive", "legacy", "typed"} {
			for _, size := range []int{0, 1, 3} {
				for _, amount := range []int{-1, 0, 2} {
					t.Run(fmt.Sprintf("surveil=%t/%s/size=%d/amount=%d", surveil, adapter, size, amount), func(t *testing.T) {
						a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
						g := mage.NewGame(a, b)
						for i := range size {
							b.AddToLibrary(mage.NewLand(fmt.Sprintf("Card %d", i)))
						}
						original := append([]mage.Card(nil), b.Library()...)
						eventType := dsl.EvtScry
						if surveil {
							eventType = dsl.EvtSurveil
						}
						var events []core.GameEvent
						trigger := mage.NewTriggered(eventType, false, dsl.Sequence(dsl.GainLife(dsl.You, dsl.Fixed(1)), dsl.GainLife(dsl.You, dsl.ValueAtResolution(dsl.EventAmountValue()))))
						trigger.SetCondition(func(evt *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
							events = append(events, *evt)
							if amount > 0 && size > 0 {
								if surveil {
									if len(b.Graveyard()) != 1 || b.Graveyard()[0].ID() != original[0].ID() {
										t.Error("event preceded graveyard placement")
									}
								} else if b.Library()[size-1].ID() != original[0].ID() {
									t.Error("event preceded bottom placement")
								}
							}
							return evt.PlayerID == b.PlayerID()
						})
						g.PutOnBattlefield(mage.NewEnchantment("Watcher", "", mage.WithAbility(trigger)), a.PlayerID())
						if size > 0 && amount > 0 {
							if surveil {
								b.AddSurveilDecision([]string{"Card 0"}, nil)
							} else {
								b.AddScryDecision([]string{"Card 0"}, nil)
							}
						}
						var effect mage.Effect
						switch adapter {
						case "primitive":
							var got int
							if surveil {
								got = g.PerformSurveil(b, amount)
							} else {
								got = g.PerformScry(b, amount)
							}
							if want := max(0, min(size, amount)); got != want {
								t.Errorf("examined = %d, want %d", got, want)
							}
						case "legacy":
							effect = mage.Scry(mage.Fixed(amount))
							if surveil {
								effect = mage.Surveil(mage.Fixed(amount))
							}
						case "typed":
							effect = dsl.Scry(dsl.Fixed(amount))
							if surveil {
								effect = dsl.Surveil(dsl.Fixed(amount))
							}
						}
						if effect != nil {
							if err := mage.ApplyEffect(g, effect, uuid.Nil, b.PlayerID(), nil); err != nil {
								t.Fatal(err)
							}
						}
						wantEvents := 0
						if amount > 0 {
							wantEvents = 1
						}
						if len(events) != wantEvents {
							t.Fatalf("events = %d, want %d", len(events), wantEvents)
						}
						if len(events) == 1 && (events[0].Amount != min(size, amount) || events[0].PlayerID != b.PlayerID()) {
							t.Fatalf("event = %+v", events[0])
						}
						if a.Life() != 20 {
							t.Fatal("trigger resolved during the action")
						}
						g.PutTriggersOnStack()
						if g.StackSize() != wantEvents {
							t.Fatalf("stack = %d, want %d", g.StackSize(), wantEvents)
						}
						if wantEvents > 0 {
							g.ResolveTopOfStack()
						}
						if a.Life() != 20+wantEvents+max(0, min(size, amount)) {
							t.Fatalf("watcher life = %d", a.Life())
						}
					})
				}
			}
		}
	}
}

// --- from life_event_test.go ---
func TestTypedLifeEventComposition(t *testing.T) {
	player, amount := dsl.BindPlayer("loser"), dsl.BindNumber("lost")
	pattern := dsl.OpponentLosesLife(player).WithAmount(amount)
	body := dsl.GainLife(dsl.FromPlayer(player), dsl.FromNumber(amount))
	for _, card := range []mage.Card{
		dsl.NewEnchantment("Life", "", dsl.WithAbility(dsl.WhenEvent(pattern, body))),
		dsl.NewEnchantment("Modes", "", dsl.WithAbility(dsl.WhenEventModes(pattern,
			dsl.ModeOf("Gain", body), dsl.ModeOf("Later", dsl.AtNext(dsl.EachEndStep, body))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	info := mage.InspectDefinition(body)
	if info.Role == "" {
		t.Fatal("missing definition metadata")
	}
}

// --- from life_gain_event_test.go ---
func TestControllerLifeGainEventComposition(t *testing.T) {
	player := dsl.BindPlayer("gainer")
	var pattern = dsl.ControllerGainsLife(player)
	body := dsl.GainLife(dsl.FromPlayer(player), dsl.Fixed(1))
	for _, card := range []mage.Card{
		dsl.NewEnchantment("Life", "", dsl.WithAbility(dsl.WhenEvent(pattern, body))),
		dsl.NewEnchantment("Modes", "", dsl.WithAbility(dsl.WhenEventModes(pattern,
			dsl.ModeOf("Gain", body), dsl.ModeOf("Later", dsl.AtNext(dsl.EachEndStep, body))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	if info := mage.InspectDefinition(body); info.Properties.LifeGain != 1 {
		t.Fatalf("metadata = %+v", info)
	}
}

func TestControllerLifeGainAmountComposition(t *testing.T) {
	player := dsl.BindPlayer("gainer")
	amount := dsl.BindNumber("gained")
	card := dsl.NewEnchantment("Life Amount", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.ControllerGainsLife(player).WithAmount(amount),
		dsl.GainLife(dsl.FromPlayer(player), dsl.FromNumber(amount)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from modal_trigger_test.go ---
func TestTypedModalTriggerComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	modes := []dsl.ModeDefinition{
		dsl.ModeOf("Return", dsl.ReturnToHand(target), dsl.WithTargets(target)),
		dsl.ModeOf("Draw", dsl.Sequence(dsl.Draw(dsl.You, dsl.Fixed(1)), dsl.GainLife(dsl.You, dsl.Fixed(1)))),
	}
	for _, tc := range []struct {
		name, role string
		ability    mage.TriggeredAbility
	}{
		{"Entry", "event-trigger", dsl.WhenEventModes(dsl.Enters(dsl.This), modes...)},
		{"Death", "event-trigger", dsl.WhenEventModes(dsl.Dies(dsl.This), modes...)},
		{"Upkeep", "step-trigger", dsl.AtModes(dsl.YourUpkeep, modes...)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := dsl.NewCreature(tc.name, "", 1, 1, dsl.WithAbility(tc.ability))
			if err := mage.ValidateCardDefinition(tc.name, card); err != nil {
				t.Fatal(err)
			}
			info := dsl.InspectDefinition(tc.ability)
			if info.Role != tc.role || len(info.Children) != 3 || info.Children[1].Role != "mode" || info.Children[1].Text != "Return" || info.Children[2].Text != "Draw" {
				t.Fatalf("metadata = %+v", info)
			}
		})
	}
}

// --- from opponent_creature_death_test.go ---
func TestOpponentCreatureDiesAuthorExample(t *testing.T) {
	former := dsl.BindPlayer("former controller")
	ability := dsl.WhenEvent(
		dsl.OpponentCreatureDies().WithController(former),
		dsl.GainLife(dsl.FromPlayer(former), dsl.Fixed(1)),
	)
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(ability))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from permanent_tap_event_test.go ---
func TestPermanentTapEventDeclaration(t *testing.T) {
	permanent := dsl.BindPermanent("tapped permanent")
	controller := dsl.BindPlayer("controller")
	card := dsl.NewEnchantment("Observer", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.PermanentBecomesTapped().OfType(dsl.TypeArtifact).ByOpponent().WithPermanent(permanent).WithController(controller),
		dsl.GainLife(dsl.FromPlayer(controller), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from player_discard_event_test.go ---
func TestPlayerDiscardsCardAuthorExample(t *testing.T) {
	discarder := dsl.BindPlayer("discarder")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.PlayerDiscardsCard(discarder),
		dsl.GainLife(dsl.FromPlayer(discarder), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from prevention_test.go ---
func TestTypedPreventionAmountComposition(t *testing.T) {
	saved := dsl.BindPermanent("amount source")
	body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.PreventNextDamage(dsl.You, dsl.MaxValue(dsl.Fixed(0), dsl.PowerOf(dsl.From(saved)))))
	recipient := dsl.TargetDamageRecipient("recipient")
	target := dsl.TargetPermanent("amount", dsl.IsCreature)
	for _, card := range []mage.Card{
		dsl.NewInstant("Target", "", dsl.WithAbility(dsl.Spell(dsl.PreventNextDamage(recipient, dsl.ToughnessOf(target)), dsl.WithTargets(recipient, target)))),
		dsl.NewCreature("Activation", "", 2, 3, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 2, 3, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewInstant("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Shield", dsl.PreventNextDamage(dsl.You, dsl.Fixed(2))), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from step_player_test.go ---
func TestTurnStepPlayerComposition(t *testing.T) {
	player := dsl.BindPlayer("step player")
	pattern := dsl.TurnStepBegins(dsl.EachUpkeep, player)
	body := dsl.Sequence(dsl.GainLife(dsl.FromPlayer(player), dsl.Fixed(1)),
		dsl.AtNext(dsl.EachEndStep, dsl.PreventNextDamage(dsl.FromPlayer(player), dsl.Fixed(2))))
	for _, ability := range []dsl.TriggeredAbility{
		dsl.WhenEvent(pattern, body),
		dsl.WhenEventModes(pattern, dsl.ModeOf("Life", body), dsl.ModeOf("Draw", dsl.Draw(dsl.FromPlayer(player), dsl.Fixed(1)))),
		dsl.WhenEvent(pattern, dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), dsl.BindPermanent("member"), body)),
	} {
		card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(ability))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	info := dsl.InspectDefinition(pattern)
	if info.Role != "event-pattern" || info.Text == "" || len(info.Children) != 2 {
		t.Fatalf("inspection = %+v", info)
	}
}

func TestDelayedStepPlayerComposition(t *testing.T) {
	player := dsl.BindPlayer("future player")
	var pattern = dsl.TurnStepBegins(dsl.EachUpkeep, player)
	var delayed dsl.DelayedStepPattern = pattern
	body := dsl.AtNext(delayed, dsl.GainLife(dsl.FromPlayer(player), dsl.Fixed(2)))
	for _, ability := range []mage.Ability{
		dsl.Spell(body),
		dsl.Activate(dsl.Costs(), body),
		dsl.At(dsl.EachEndStep, body),
		dsl.ActivateModes(dsl.Costs(), dsl.ModeOf("Delay", body), dsl.ModeOf("Life", dsl.GainLife(dsl.You, dsl.Fixed(1)))),
	} {
		card := dsl.NewEnchantment("Schedule", "", dsl.WithAbility(ability))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	info := dsl.InspectDefinition(body)
	if info.Role != "delayed-step" || len(info.Children) != 2 {
		t.Fatalf("inspection = %+v; properties = %+v", info, body.Properties())
	}
}

// --- from targeted_event_test.go ---
func TestBecomesTargetedPublicDeclaration(t *testing.T) {
	actor := dsl.BindPlayer("actor")
	card := dsl.NewCreature("Watcher", "", 2, 2,
		dsl.WithAbility(dsl.WhenEvent(
			dsl.BecomesTargeted(dsl.This).ByOpponent().WithController(actor),
			dsl.GainLife(dsl.FromPlayer(actor), dsl.Fixed(1)))))
	if card == nil || dsl.InspectDefinition(dsl.BecomesTargeted(dsl.This)).Role != "event-pattern" {
		t.Fatal("targeting pattern is not available")
	}
}
