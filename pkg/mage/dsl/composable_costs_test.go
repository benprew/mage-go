package dsl_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

// --- from ability_activation_filter_test.go ---
func TestAbilityActivationFilterPublicDeclaration(t *testing.T) {
	actor := dsl.BindPlayer("activator")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.ActivatedAbilityEvent().FromArtifact().ByOpponent().WithActivator(actor),
		dsl.GainLife(dsl.FromPlayer(actor), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from activation_capture_test.go ---
func TestActivationCaptureComposition(t *testing.T) {
	n := dsl.BindNumber("credits")
	var capture = dsl.CaptureNumberBeforePayment(dsl.CountersOn(dsl.This, core.Credit), n)
	var _ dsl.ActivationOption = capture
	var _ dsl.GrantedActivationOption = capture
	var _ dsl.ModalActivationOption = capture
	c := dsl.NewArtifact("Capture", "", dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.SacrificeSourceCost()),
		dsl.GainLife(dsl.You, dsl.FromNumber(n)), capture)))
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	descriptor := dsl.InspectDefinition(capture)
	if descriptor.Role != "activation-number-capture" || len(descriptor.Children) != 2 {
		t.Fatalf("descriptor = %+v", descriptor)
	}
}

// --- from activation_no_tap_test.go ---
func TestActivatedAbilityWithoutTapCostPublicDeclaration(t *testing.T) {
	actor := dsl.BindPlayer("actor")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.ActivatedAbilityEvent().FromArtifact().ByOpponent().WithoutTapCost().WithActivator(actor),
		dsl.GainLife(dsl.FromPlayer(actor), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from activation_permission_test.go ---
func TestAnyPlayerMayActivatePaymentAndController(t *testing.T) {
	for _, permission := range []bool{false, true} {
		t.Run(map[bool]string{false: "controller only", true: "any player"}[permission], func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			var options []dsl.ActivationOption
			if permission {
				options = append(options, dsl.AnyPlayerMayActivate())
			}
			card := dsl.NewArtifact("Source", "", dsl.WithAbility(dsl.Activate(
				dsl.Costs(dsl.Mana("{3}")), dsl.GainLife(dsl.You, dsl.Fixed(1)), options...)))
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(card, a.PlayerID())
			a.ManaPool().Add(core.Green, 3)
			if err := g.ActivateAbilityByIndex(b.PlayerID(), card.ID(), 0, nil); err == nil || a.ManaPool().TotalMana() != 3 || g.StackSize() != 0 {
				t.Fatal("the opponent cannot spend the source controller's mana")
			}
			b.ManaPool().Add(core.Blue, 6)
			err := g.ActivateAbilityByIndex(b.PlayerID(), card.ID(), 0, nil)
			if !permission {
				if err == nil || b.ManaPool().TotalMana() != 6 || g.StackSize() != 0 {
					t.Fatal("the default permission must reject the opponent before payment")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if b.ManaPool().TotalMana() != 3 || g.StackSize() != 1 || g.StackPeek().Controller != b.PlayerID() {
					t.Fatal("the activator must pay and control the stacked ability")
				}
				clone := g.Clone()
				clone.ResolveStack()
				if clone.GetPlayer(b.PlayerID()).Life() != 21 || b.Life() != 20 || a.Life() != 20 {
					t.Fatal("You must refer to the activator without changing the original game")
				}
				if err := g.ActivateAbilityByIndex(b.PlayerID(), card.ID(), 0, nil); err != nil {
					t.Fatal("permission must not impose an activation limit", err)
				}
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
				t.Fatal("the source controller must retain permission", err)
			}
			g.ResolveStack()
			want := 20
			if permission {
				want = 22
			}
			if a.Life() != 21 || b.Life() != want || a.ManaPool().TotalMana() != 0 {
				t.Fatal("resolution must use each ability's stored activator")
			}
		})
	}
}

func TestAnyPlayerMayActivateRespectsRestrictions(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	card := dsl.NewArtifact("Source", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.Mana("{1}")), dsl.GainLife(dsl.You, dsl.Fixed(1)),
		dsl.AnyPlayerMayActivate(), dsl.OnlyDuring(core.Upkeep))))
	g.PutOnBattlefield(card, a.PlayerID())
	b.ManaPool().Add(core.Blue, 1)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), card.ID(), 0, nil); err == nil || b.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
		t.Fatal("permission must not bypass the timing restriction")
	}
}

// --- from composable_paid_fixed_counters_test.go ---
func TestFixedCounterPaymentPublicDeclaration(t *testing.T) {
	paid := dsl.BindNumber("paid")
	card := dsl.NewArtifact("Battery", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.RemoveCountersCostInto(core.Charge, 2, paid)),
		dsl.GainLife(dsl.You, dsl.FromNumber(paid)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from composable_paid_x_counters_test.go ---
func TestPaidXCountersPublicDeclaration(t *testing.T) {
	paid := dsl.BindNumber("removed")
	card := dsl.NewArtifact("Counter source", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.RemoveXCountersFromSourceCostInto(core.Charge, paid)),
		dsl.GainLife(dsl.You, dsl.FromNumber(paid)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from granted_activation_test.go ---
func TestGrantedActivationComposition(t *testing.T) {
	saved := dsl.BindPermanent("recipient")
	card := dsl.NewEnchantment("Grant", "", dsl.WithAbility(dsl.Static(dsl.GrantEachActivatedAbility(
		dsl.WithoutPermanent(dsl.ControlledPermanents(dsl.IsCreature), dsl.This),
		dsl.Costs(dsl.Mana("{B}")),
		dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.Regenerate(dsl.From(saved))),
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestGrantedTargetActivationComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature, dsl.TargetYouControl())
	var option dsl.GrantedActivationOption = dsl.WithTargets(target)
	card := dsl.NewEnchantment("Target grant", "", dsl.WithAbility(dsl.Static(dsl.GrantEachActivatedAbility(
		dsl.MatchingPermanents(dsl.IsCreature), dsl.Costs(dsl.Mana("{B}")),
		dsl.Regenerate(target), option,
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestGrantedTimingComposition(t *testing.T) {
	for _, option := range []dsl.ActivationTimingOption{dsl.OnlyAsSorcery(), dsl.OnlyDuring(core.Upkeep), dsl.OnlyOnYourTurn()} {
		target := dsl.TargetPermanent("creature", dsl.IsCreature)
		card := dsl.NewEnchantment("Timed grant", "", dsl.WithAbility(dsl.Static(dsl.GrantEachActivatedAbility(
			dsl.MatchingPermanents(dsl.IsArtifact), dsl.Costs(), dsl.Regenerate(target), dsl.WithTargets(target), option,
		))), dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Regenerate(dsl.This), option)))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGrantedConditionComposition(t *testing.T) {
	var option = dsl.ActivationIf(dsl.SourceHasCounters(core.Charge, 1))
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	card := dsl.NewEnchantment("Conditional grant", "", dsl.WithAbility(dsl.Static(dsl.GrantEachActivatedAbility(
		dsl.MatchingPermanents(dsl.IsArtifact), dsl.Costs(), dsl.Regenerate(target),
		dsl.WithTargets(target), dsl.OnlyAsSorcery(), option,
	))), dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Regenerate(dsl.This), option)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestGrantedLimitComposition(t *testing.T) {
	var option = dsl.ActivationLimit(2)
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	card := dsl.NewEnchantment("Limited grant", "", dsl.WithAbility(dsl.Static(dsl.GrantEachActivatedAbility(
		dsl.MatchingPermanents(dsl.IsArtifact), dsl.Costs(), dsl.Regenerate(target),
		dsl.WithTargets(target), dsl.OnlyAsSorcery(), dsl.ActivationIf(dsl.Controls(dsl.IsCreature)), option,
	))), dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Regenerate(dsl.This), option)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestGrantedModesComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	saved := dsl.BindPermanent("source")
	card := dsl.NewEnchantment("Modal grant", "", dsl.WithAbility(dsl.Static(dsl.GrantEachActivatedModes(
		dsl.WithoutPermanent(dsl.MatchingPermanents(dsl.IsCreature), dsl.This),
		dsl.Costs(dsl.Mana("{1}"), dsl.TapCost(dsl.This)),
		dsl.ModeOf("regenerate", dsl.Regenerate(target), dsl.WithTargets(target)),
		dsl.ModeOf("counter", dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.AddCounters(dsl.From(saved), core.Charge, dsl.Fixed(1)))),
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestModalRestrictionsComposition(t *testing.T) {
	options := []dsl.ModalActivationOption{
		dsl.ModeOf("one", dsl.GainLife(dsl.You, dsl.Fixed(1))),
		dsl.OnlyAsSorcery(), dsl.ActivationIf(dsl.SourceHasCounters(core.Charge, 1)), dsl.ActivationLimit(1),
		dsl.ModeOf("two", dsl.GainLife(dsl.You, dsl.Fixed(2))),
	}
	card := dsl.NewEnchantment("Modal restrictions", "",
		dsl.WithAbility(dsl.ActivateModes(dsl.Costs(), options...)),
		dsl.WithAbility(dsl.Static(dsl.GrantEachActivatedModes(dsl.MatchingPermanents(dsl.IsArtifact), dsl.Costs(), options...))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestGrantedSubjectComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature, dsl.TargetYouControl())
	card := dsl.NewAura("Subject grant", "", dsl.WithAbility(dsl.Static(
		dsl.GrantActivatedAbility(dsl.AttachedPermanent, dsl.Costs(dsl.Mana("{B}")), dsl.Regenerate(target), dsl.WithTargets(target), dsl.OnlyAsSorcery(), dsl.ActivationLimit(1)),
		dsl.GrantActivatedModes(dsl.This, dsl.Costs(), dsl.ActivationIf(dsl.Controls(dsl.IsCreature)),
			dsl.ModeOf("one", dsl.GainLife(dsl.You, dsl.Fixed(1))),
			dsl.ModeOf("two", dsl.GainLife(dsl.You, dsl.Fixed(2)))),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestGrantedModificationComposition(t *testing.T) {
	target := dsl.TargetPermanent("recipient", dsl.IsCreature)
	inner := dsl.TargetPermanent("regenerate", dsl.IsCreature, dsl.TargetYouControl())
	card := dsl.NewInstant("Grant", "", dsl.WithAbility(dsl.Spell(
		dsl.Until(core.Indefinite, dsl.ActivatedAbilityModification(target, dsl.Costs(dsl.RemoveCountersCost(core.Matrix, 1)), dsl.Regenerate(inner), dsl.WithTargets(inner), dsl.OnlyAsSorcery(), dsl.ActivationLimit(1))), dsl.WithTargets(target))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	invalid := dsl.NewInstant("Invalid grant", "", dsl.WithAbility(dsl.Spell(
		dsl.Until(core.Indefinite, dsl.ActivatedAbilityModification(target, dsl.Costs(), dsl.Regenerate(target))), dsl.WithTargets(target))))
	if err := mage.ValidateCardDefinition(invalid.Name(), invalid); err == nil {
		t.Fatal("grant body read outer target")
	}
}

func TestGrantedModalModificationComposition(t *testing.T) {
	outer := dsl.TargetPermanent("recipient", dsl.IsCreature)
	inner := dsl.TargetPermanent("return", dsl.IsCreature)
	modification := dsl.ActivatedModesModification(outer, dsl.Costs(),
		dsl.ModeOf("return", dsl.ReturnToHand(inner), dsl.WithTargets(inner)),
		dsl.ModeOf("gain", dsl.GainLife(dsl.You, dsl.Fixed(2))),
		dsl.OnlyAsSorcery(), dsl.ActivationLimit(1))
	card := dsl.NewInstant("Modal grant", "", dsl.WithAbility(dsl.Spell(dsl.Until(core.EndOfTurn, modification), dsl.WithTargets(outer))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []dsl.ContinuousModification{
		dsl.ActivatedModesModification(outer, dsl.Costs(),
			dsl.ModeOf("outer read", dsl.ReturnToHand(outer)),
			dsl.ModeOf("gain", dsl.GainLife(dsl.You, dsl.Fixed(2)))),
		dsl.ActivatedModesModification(outer, dsl.Costs(),
			dsl.ModeOf("declared", dsl.ReturnToHand(inner), dsl.WithTargets(inner)),
			dsl.ModeOf("other mode read", dsl.ReturnToHand(inner))),
	} {
		card := dsl.NewInstant("Invalid modal grant", "", dsl.WithAbility(dsl.Spell(dsl.Until(core.Indefinite, invalid), dsl.WithTargets(outer))))
		if err := mage.ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("mode read a target from another scope")
		}
	}
}

// --- from land_mana_type_test.go ---
func TestLandManaTypePublicDeclaration(t *testing.T) {
	player := dsl.BindPlayer("activator")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.LandTappedForMana().OfManaType(core.Blue).WithPlayer(player),
		dsl.GainLife(dsl.FromPlayer(player), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from legacy_payment_test.go ---
func TestLegacyPaymentTypedChildren(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	for _, body := range []dsl.ResolvingEffect{
		dsl.IfPlayerPays(dsl.SelectController(), dsl.ManaCostOf("{1}"), "Pay?", dsl.Destroy(target), dsl.Draw(dsl.You, dsl.Fixed(1))),
		dsl.UnlessTargetPays(dsl.SelectController(), dsl.ManaCostOf("{1}"), "Pay?", dsl.Destroy(target)),
	} {
		info := dsl.InspectDefinition(body)
		if info.Role != "legacy-payment-branch" || len(info.Children) < 3 || !info.Children[2].Properties.Destroys {
			t.Fatalf("payment inspection = %+v", info)
		}
		valid := dsl.NewInstant("Payment", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target))))
		if err := mage.ValidateCardDefinition("Payment", valid); err != nil {
			t.Fatal(err)
		}
		invalid := dsl.NewInstant("Payment", "", dsl.WithAbility(dsl.Spell(body)))
		if err := mage.ValidateCardDefinition("Payment", invalid); err == nil || !strings.Contains(err.Error(), "target binding requires") {
			t.Fatalf("target diagnostic = %v", err)
		}
	}
}

// --- from life_payment_test.go ---
type lifePaymentPlayer struct {
	*mage.BasePlayer
	accept bool
}

func (p *lifePaymentPlayer) ChooseMayAbility(string) bool { return p.accept }

func TestTypedLifePaymentBranchesAtBoundary(t *testing.T) {
	for _, accept := range []bool{false, true} {
		for _, total := range []int{3, 4} {
			for _, ifPaid := range []bool{false, true} {
				t.Run(fmt.Sprintf("accept=%v/total=%d/ifPaid=%v", accept, total, ifPaid), func(t *testing.T) {
					a := &lifePaymentPlayer{mage.NewBasePlayer("A"), accept}
					b := mage.NewBasePlayer("B")
					g := mage.NewGame(a, b)
					a.SetLife(3)
					player, amount := dsl.BindPlayer("payer"), dsl.BindNumber("amount")
					watcher := dsl.NewEnchantment("Payment watcher", "", dsl.WithAbility(dsl.WhenEvent(
						dsl.OpponentLosesLife(player).WithAmount(amount), dsl.GainLife(dsl.You, dsl.FromNumber(amount)),
					)))
					g.PutOnBattlefield(watcher, b.PlayerID())
					costs := dsl.Costs(dsl.LifePayCost(1), dsl.LifePayCost(total-1))
					body := dsl.GainLife(dsl.You, dsl.Fixed(5))
					var effect dsl.ResolvingEffect
					if ifPaid {
						effect = dsl.IfPaid(dsl.You, costs, body)
					} else {
						effect = dsl.UnlessPaid(dsl.You, costs, body)
					}
					if err := mage.ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					paid := accept && total == 3
					want := 3
					if paid {
						want = 0
					}
					if paid == ifPaid {
						want += 5
					}
					if a.Life() != want {
						t.Fatalf("life = %d, want %d", a.Life(), want)
					}
					g.PutTriggersOnStack()
					wantTriggers := 0
					if paid {
						wantTriggers = 2
					}
					if g.StackSize() != wantTriggers {
						t.Fatalf("triggers = %d, want %d", g.StackSize(), wantTriggers)
					}
					g.ResolveStack()
					wantGain := 20
					if paid {
						wantGain += 3
					}
					if b.Life() != wantGain {
						t.Fatalf("watcher life = %d, want %d", b.Life(), wantGain)
					}
				})
			}
		}
	}
}

func TestTypedLifePaymentCastAndActivationBoundary(t *testing.T) {
	for _, spell := range []bool{false, true} {
		for _, life := range []int{2, 3} {
			t.Run(fmt.Sprintf("spell=%v/life=%d", spell, life), func(t *testing.T) {
				a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				a.SetLife(life)
				land := dsl.NewLand("Payment land", dsl.WithManaAbility(core.Green))
				g.PutOnBattlefield(land, a.PlayerID())
				var err error
				if spell {
					card := dsl.NewSorcery("Life spell", "{G}", dsl.WithAbility(dsl.Spell(dsl.GainLife(dsl.You, dsl.Fixed(5)))), dsl.WithAdditionalCost(dsl.LifePayCost(3)))
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					err = g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0)
					if life == 2 && len(a.Hand()) != 1 {
						t.Fatal("failed cast removed the card")
					}
				} else {
					card := dsl.NewArtifact("Life device", "", dsl.WithAbility(dsl.Activate(
						dsl.Costs(dsl.Mana("{G}"), dsl.LifePayCost(1), dsl.LifePayCost(2)), dsl.GainLife(dsl.You, dsl.Fixed(5)),
					)))
					g.PutOnBattlefield(card, a.PlayerID())
					err = g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil)
				}
				if (err == nil) != (life == 3) {
					t.Fatalf("payment error = %v", err)
				}
				wantLife, wantStack := life, 0
				if life == 3 {
					wantLife, wantStack = 0, 1
				}
				if a.Life() != wantLife || g.StackSize() != wantStack {
					t.Fatalf("life/stack = %d/%d", a.Life(), g.StackSize())
				}
				if g.FindPermanent(land.ID()).Tapped != (life == 3) || a.ManaPool().TotalMana() != 0 {
					t.Fatal("incorrect mana payment or rollback")
				}
			})
		}
	}
}

// --- from mana_recipient_test.go ---
func TestAddManaToPlayerDeclaration(t *testing.T) {
	producer := dsl.BindPlayer("producer")
	card := dsl.NewEnchantment("Mana watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.LandTappedForMana().WithPlayer(producer),
		dsl.AddManaToPlayer(dsl.FromPlayer(producer), core.Green, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from mill_cost_test.go ---
func TestMillCostPublicDeclaration(t *testing.T) {
	card := dsl.NewCreature("Upkeep", "", 1, 1,
		dsl.WithAbility(dsl.At(dsl.YourUpkeep,
			dsl.UnlessPaid(dsl.You, dsl.Costs(dsl.MillCost(2)), dsl.Sacrifice(dsl.This)))))
	if err := mage.ValidateCardDefinition("Upkeep", card); err != nil {
		t.Fatal(err)
	}
}

// --- from optional_payment_availability_test.go ---
func TestPerDrawnCardOptionalSacrificeAuthorExample(t *testing.T) {
	cards, member := dsl.BindCards("drawn"), dsl.BindCard("member")
	card := dsl.NewInstant("Synthetic draw", "{X}{U}", dsl.Spell(dsl.Sequence(
		dsl.Draw(dsl.You, dsl.Fixed(2), dsl.DrawCardsInto(cards)),
		dsl.ForEachSavedCard(cards, member, dsl.UnlessPaid(dsl.You,
			dsl.Costs(dsl.SacrificeMatchingIncludingSourceCost(dsl.AnyPermanent, "Sacrifice a permanent")),
			dsl.Discard(dsl.You, dsl.Fixed(1)))),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from optional_payment_result_test.go ---
func TestOptionalPaymentResultPublicDeclaration(t *testing.T) {
	paid := dsl.BindNumber("paid")
	card := dsl.NewArtifact("Synthetic", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.OptionalCostInto(dsl.LifePayCost(2), "Pay two life?", paid)),
		dsl.GainLife(dsl.You, dsl.FromNumber(paid)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from paid_card_test.go ---
func TestPaidCardPublicDeclaration(t *testing.T) {
	paid := dsl.BindCard("paid creature")
	spell := dsl.NewInstant("Offering", "",
		dsl.WithAbility(dsl.Spell(
			dsl.IfCardHasType(dsl.FromCard(paid), core.TypeCreature,
				dsl.GainLife(dsl.You, dsl.Fixed(2))))),
		dsl.WithAdditionalCost(dsl.SacrificeMatchingCostWithCard(
			dsl.IsCreature, "Sacrifice a creature", paid)))
	if err := mage.ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
}

// --- from paid_discard_test.go ---
func TestPaidDiscardPublicDeclaration(t *testing.T) {
	paid := dsl.BindCard("discarded card")
	card := dsl.NewArtifact("Discard capture", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.DiscardCardCostInto(paid)),
		dsl.IfCardHasType(dsl.FromCard(paid), core.TypeLand,
			dsl.GainLife(dsl.You, dsl.Fixed(2))),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from paid_exile_card_test.go ---
func TestPaidExileCardPublicDeclaration(t *testing.T) {
	paid := dsl.BindCard("exiled card")
	card := dsl.NewArtifact("Synthetic", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.ExileMatchingCardFromGraveyardCostWithCard(
			dsl.IsCreatureCard, "Exile a creature card", paid)),
		dsl.IfCardHasType(dsl.FromCard(paid), dsl.TypeCreature,
			dsl.GainLife(dsl.You, dsl.Fixed(1))),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from paid_exile_cards_test.go ---
func TestPaidExileCardsPublicDeclaration(t *testing.T) {
	paid := dsl.BindCards("exiled cards")
	card := dsl.NewArtifact("Synthetic", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.ExileFromGraveyardCostWithCards(2, paid)),
		dsl.GainLife(dsl.You, dsl.CountCards(dsl.FromCards(paid))),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from paid_exile_mana_value_test.go ---
func TestPaidGraveyardExileManaValueDeclaration(t *testing.T) {
	paid := dsl.BindNumber("exiled mana value")
	card := dsl.NewArtifact("Synthetic", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.ExileMatchingCardFromGraveyardCostInto(dsl.IsCreatureCard, "Exile a creature card", paid)),
		dsl.GainLife(dsl.You, dsl.FromNumber(paid)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from paid_mill_test.go ---
func TestPaidMillCountDeclaration(t *testing.T) {
	paid := dsl.BindNumber("milled")
	card := dsl.NewArtifact("Mill source", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.MillCostInto(2, paid)),
		dsl.GainLife(dsl.You, dsl.FromNumber(paid)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from paid_permanent_test.go ---
func TestPaidPermanentAuthorSurface(t *testing.T) {
	paid := dsl.BindPermanent("paid")
	card := dsl.NewInstant("Offering", "", dsl.WithAbility(dsl.Spell(
		dsl.DealDamageFrom(dsl.DamageSourceOf(dsl.From(paid)), dsl.You, dsl.Fixed(1)))),
		dsl.WithAdditionalCost(dsl.SacrificeMatchingCostWithPermanent(dsl.IsCreature, "Sacrifice a creature", paid)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from paid_reveal_test.go ---
func TestPaidRevealPublicDeclaration(t *testing.T) {
	paid := dsl.BindCard("revealed card")
	card := dsl.NewInstant("Synthetic", "", dsl.WithAbility(dsl.Spell(
		dsl.IfCardHasType(dsl.FromCard(paid), core.TypeCreature,
			dsl.GainLife(dsl.You, dsl.Fixed(1))))),
		dsl.WithAdditionalCost(dsl.RevealFromHandCostWithCard(
			dsl.IsCreatureCard, "Reveal a creature card", paid)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from paid_sacrifice_total_power_test.go ---
func TestPaidSacrificeTotalPowerPublicDeclaration(t *testing.T) {
	total := dsl.BindNumber("total")
	card := dsl.NewArtifact("Offering", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.SacrificeNMatchingCostWithTotalPower(2, dsl.IsCreature, "Sacrifice two creatures", total)),
		dsl.GainLife(dsl.You, dsl.FromNumber(total)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from paid_source_mana_value_test.go ---
func TestPaidSourceManaValuePublicDeclaration(t *testing.T) {
	paid := dsl.BindNumber("paid mana value")
	card := dsl.NewCreature("Synthetic", "{2}{U}", 1, 1,
		dsl.WithAbility(dsl.Activate(
			dsl.Costs(dsl.SacrificeMatchingIncludingSourceCostWithManaValue(
				dsl.And(dsl.IsCreature, dsl.HasColorFilter(dsl.Blue)), "Sacrifice a blue creature", paid)),
			dsl.GainLife(dsl.You, dsl.FromNumber(paid)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from paid_source_permanent_test.go ---
func TestPaidSourcePermanentPublicDeclaration(t *testing.T) {
	paid := dsl.BindPermanent("paid")
	card := dsl.NewCreature("Test", "", 2, 2, dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.SacrificeMatchingIncludingSourceCostWithPermanent(dsl.IsCreature, "Sacrifice a creature", paid)),
		dsl.DealDamageFrom(dsl.DamageSourceOf(dsl.From(paid)), dsl.You, dsl.Fixed(1)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from paid_tap_permanent_test.go ---
func TestPaidTapPermanentDeclaration(t *testing.T) {
	paid := dsl.BindPermanent("paid Merfolk")
	card := dsl.NewArtifact("Test", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.TapNOtherMatchingCostWithPermanent(dsl.IsCreature,
			"Tap another creature", paid)),
		dsl.Untap(dsl.From(paid)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from permanent_capture_test.go ---
func TestPermanentCaptureComposition(t *testing.T) {
	host := dsl.BindPermanent("host")
	capture := dsl.CapturePermanentBeforePayment(dsl.AttachedPermanent, host)
	var _ dsl.ActivationOption = capture
	var _ dsl.GrantedActivationOption = capture
	var _ dsl.ModalActivationOption = capture
	c := dsl.NewAura("Capture", "", dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.SacrificeSourceCost()), dsl.Regenerate(dsl.From(host)), capture)))
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	descriptor := dsl.InspectDefinition(capture)
	if descriptor.Role != "activation-permanent-capture" || len(descriptor.Children) != 2 {
		t.Fatalf("descriptor = %+v", descriptor)
	}
}

// --- from sacrifice_capture_test.go ---
func TestSacrificePermanentCapturePublicDeclaration(t *testing.T) {
	paid := dsl.BindPermanent("paid creature")
	card := dsl.NewCreature("Sacrifice watcher", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(
		dsl.CreatureYouSacrifice().WithPermanent(paid),
		dsl.DealDamageFrom(dsl.DamageSourceOf(dsl.From(paid)), dsl.You, dsl.Fixed(1)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from spell_cast_x_capture_test.go ---
func TestSpellCastAnnouncedXPublicDeclaration(t *testing.T) {
	x := dsl.BindNumber("cast X")
	card := dsl.NewArtifact("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.SpellCast().WithXInManaCost().WithAnnouncedX(x),
		dsl.GainLife(dsl.You, dsl.FromNumber(x)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from tap_attached_cost_test.go ---
func TestTapAttachedPermanentCostDeclaration(t *testing.T) {
	card := dsl.NewAura("Synthetic gate", "{1}{B}{B}",
		dsl.WithCastTarget(dsl.TargetLand()),
		dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.TapAttachedPermanentCost()),
			dsl.GainLife(dsl.You, dsl.Fixed(1)),
			dsl.ActivationIf(dsl.PermanentIsUntapped(dsl.AttachedPermanent)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from tap_matching_cost_test.go ---
func TestTapNOtherMatchingCostDeclaration(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	ability := dsl.Activate(
		dsl.Costs(dsl.TapCost(dsl.This), dsl.TapNOtherMatchingCost(3,
			dsl.And(dsl.IsCreature, dsl.HasColorFilter(core.White)),
			"Tap three untapped white creatures you control")),
		dsl.Destroy(target), dsl.WithTargets(target))
	card := dsl.NewCreature("Synthetic", "{5}{W}", 2, 6, dsl.WithAbility(ability))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestFilteredDiscardPublicDeclaration(t *testing.T) {
	cost := dsl.DiscardMatchingCardCost(dsl.IsArtifactCard)
	card := dsl.NewInstant("Synthetic filtered discard", "{2}{U}", dsl.WithAbility(dsl.Spell(dsl.Sequence(
		dsl.Draw(dsl.You, dsl.Fixed(3)),
		dsl.UnlessPaid(dsl.You, dsl.Costs(cost), dsl.Discard(dsl.You, dsl.Fixed(2))),
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	descriptor := dsl.InspectDefinition(cost)
	if descriptor.Role != "discard-cost" || descriptor.Text != "Discard one artifact card" {
		t.Fatalf("descriptor = %+v", descriptor)
	}
}

func TestPaidFilteredDiscardPublicDeclaration(t *testing.T) {
	paid := dsl.BindCard("paid")
	cost := dsl.DiscardMatchingCardCostInto(dsl.IsArtifactCard, paid)
	card := dsl.NewArtifact("Synthetic", "", dsl.WithAbility(dsl.Activate(dsl.Costs(cost), dsl.ReturnCardToHand(dsl.FromCard(paid)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if got := dsl.InspectDefinition(cost); got.Role != "discard-cost" || got.Text != "Discard one artifact card" {
		t.Fatalf("descriptor = %+v", got)
	}
}
