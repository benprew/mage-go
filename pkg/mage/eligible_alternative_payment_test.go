package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type alternativeOfferPlayer struct {
	*eligiblePaymentPlayer
	mode  int
	modes int
}

func (p *alternativeOfferPlayer) ChooseMode(options []string, _ string) int {
	p.modes = len(options)
	return p.mode
}

func TestEligibleAlternativePayment(t *testing.T) {
	for _, state := range []string{"damage", "prevent", "sacrifice", "short", "decline", "rollback", "invalid-mode"} {
		t.Run(state, func(t *testing.T) {
			var order []string
			a := &alternativeOfferPlayer{eligiblePaymentPlayer: &eligiblePaymentPlayer{BasePlayer: NewBasePlayer("A"), order: &order, accept: state != "decline"}}
			b := &alternativeOfferPlayer{eligiblePaymentPlayer: &eligiblePaymentPlayer{BasePlayer: NewBasePlayer("B"), order: &order, accept: true}}
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			for range 2 {
				g.PutOnBattlefield(NewLand("Land"), a.PlayerID())
			}
			payer := BindPlayer("payer")
			effect := PayAnyAlternativeOrElseWithPayer(EachPlayer, []ResolutionPaymentOption{
				CostPaymentOption(Costs(SacrificeNMatchingCost(2, IsLand, "sacrifice two lands"))),
				ActionPaymentOption(DamagePayment(This, FromPlayer(payer), Fixed(5))),
			}, payer, GainLifeFor(FromPlayer(payer), Fixed(1)), GainLifeFor(You, Fixed(10)))
			card := NewArtifact("Offer", "", WithAbility(Activate(Costs(), effect)))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			a.mode = 1
			if state == "sacrifice" || state == "rollback" {
				a.mode = 0
			}
			if state == "invalid-mode" {
				a.mode = 9
			}
			if state == "short" {
				g.RemoveFromBattlefield(g.FilterBattlefield(ControlledBy(a.PlayerID()))[1])
			}
			if state == "prevent" {
				g.AddPreventionShield(a.PlayerID(), 5)
			}
			if state == "rollback" {
				a.hook = func() { a.SetLife(1) }
				effect = PayAnyAlternativeOrElseWithPayer(EachPlayer, []ResolutionPaymentOption{CostPaymentOption(Costs(LifePayCost(2), ManaCostOf("{2}")))}, payer, GainLifeFor(FromPlayer(payer), Fixed(1)), nil)
				a.ManaPool().Add(core.Colorless, 2)
				b.ManaPool().Add(core.Colorless, 2)
			}
			if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			wantA, wantB := 16, 20
			switch state {
			case "prevent", "sacrifice":
				wantA = 21
			case "decline", "invalid-mode":
				wantA, wantB = 20, 16
			case "rollback":
				wantA, wantB = 1, 19
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life %d/%d want %d/%d", a.Life(), b.Life(), wantA, wantB)
			}
			if state == "sacrifice" && len(g.FilterBattlefield(ControlledBy(a.PlayerID()))) != 1 {
				t.Fatal("wrong sacrifice batch")
			}
			if state == "rollback" && a.ManaPool().TotalMana() != 2 {
				t.Fatal("partial payment committed")
			}
			if _, ok := FromPlayer(payer).resolvePlayer(&EffectContext{Game: g, Controller: a.PlayerID()}); ok {
				t.Fatal("payer escaped")
			}
		})
	}
}

func TestEligibleAlternativePaymentInvalid(t *testing.T) {
	for _, options := range [][]ResolutionPaymentOption{nil, {{}}, {ActionPaymentOption(nil)}, {CostPaymentOption(CostGroup{})}, {ActionPaymentOption(DamagePayment(This, nil, Fixed(1)))}} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), PayAnyAlternativeOrElseWithPayer(EachPlayer, options, BindPlayer("payer"), GainLifeFor(You, Fixed(1)), nil))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid options accepted")
		}
	}
}

func TestEligibleAlternativePaymentOrderAndRecheck(t *testing.T) {
	for _, active := range []int{0, 1} {
		for _, state := range []string{"live", "phase", "return", "cycle", "zero", "negative"} {
			t.Run(fmt.Sprintf("%d/%s", active, state), func(t *testing.T) {
				var order []string
				a := &eligiblePaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true, order: &order}
				b := &eligiblePaymentPlayer{BasePlayer: NewBasePlayer("B"), accept: true, order: &order}
				g := NewGame(a, b)
				g.SetActivePlayerIndex(active)
				card := NewCreature("Recipient", "", 1, 8)
				card.SetOwner(a.PlayerID())
				recipient := g.PutOnBattlefield(card, a.PlayerID())
				binding := DeclarePermanentTarget("recipient", IsCreature)
				payer := BindPlayer("payer")
				amount := 3
				if state == "zero" {
					amount = 0
				}
				if state == "negative" {
					amount = -3
				}
				effect := PayAnyAlternativeOrElseWithPayer(EachPlayer, []ResolutionPaymentOption{ActionPaymentOption(DamagePayment(This, binding, Fixed(amount)))}, payer, GainLifeFor(FromPlayer(payer), Fixed(1)), GainLifeFor(You, Fixed(7)))
				first := a
				if active == 1 {
					first = b
				}
				first.hook = func() {
					switch state {
					case "phase":
						g.PhaseOut(recipient.ID())
					case "return":
						g.RemoveFromBattlefield(recipient)
						g.PutOnBattlefield(card, a.PlayerID())
					case "cycle":
						g.PhaseIn(g.PhaseOut(recipient.ID()))
					}
				}
				source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(Costs(), effect, WithTargets(binding)))), a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{recipient.ID()}); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				failed := state == "phase" || state == "return"
				if len(order) != 1 || order[0] != first.Name() {
					t.Fatal("wrong offer order", order)
				}
				if failed {
					if a.Life() != 27 || b.Life() != 20 {
						t.Fatal("stale recipient paid")
					}
				} else {
					if first.Life() != 21 {
						t.Fatal("wrong payer")
					}
					if g.FindPermanent(recipient.ID()).Damage != max(0, amount) {
						t.Fatal("wrong damage")
					}
				}
			})
		}
	}
}

func TestEligibleAlternativePaymentDeclarations(t *testing.T) {
	payer := BindPlayer("payer")
	options := []ResolutionPaymentOption{ActionPaymentOption(DamagePayment(This, FromPlayer(payer), Fixed(1)))}
	for _, effect := range []ResolvingEffect{
		PayAnyAlternativeOrElseWithPayer(nil, options, payer, GainLifeFor(You, Fixed(1)), nil),
		PayAnyAlternativeOrElseWithPayer(EachPlayer, options, nil, GainLifeFor(You, Fixed(1)), nil),
		PayAnyAlternativeOrElseWithPayer(EachPlayer, options, payer, nil, nil),
		PayAnyAlternativeOrElseWithPayer(EachPlayer, options, payer, nil, GainLifeFor(FromPlayer(payer), Fixed(1))),
		Sequence(PayAnyAlternativeOrElseWithPayer(EachPlayer, options, payer, GainLifeFor(You, Fixed(1)), nil), GainLifeFor(FromPlayer(payer), Fixed(1))),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid local payer declaration accepted")
		}
	}
}

func TestEligibleAlternativePaymentSacrificeBatch(t *testing.T) {
	var order []string
	a := &alternativeOfferPlayer{eligiblePaymentPlayer: &eligiblePaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true, order: &order}}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewLand("First"), a.PlayerID())
	second := g.PutOnBattlefield(NewLand("Second"), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtZoneChange, false, GainLifeFor(You, Fixed(1))).SetCondition(func(event *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		if event.SourceID != first.ID() && event.SourceID != second.ID() {
			return false
		}
		if reader.FindPermanent(first.ID()) == nil || reader.FindPermanent(second.ID()) == nil {
			t.Fatal("sacrifice was not simultaneous or lost LKI")
		}
		return true
	}))), a.PlayerID())
	payer := BindPlayer("payer")
	effect := PayAnyAlternativeOrElseWithPayer(EachPlayer, []ResolutionPaymentOption{CostPaymentOption(Costs(SacrificeNMatchingIncludingSourceCost(2, IsLand, "sacrifice lands")))}, payer, GainLifeFor(FromPlayer(payer), Fixed(1)), nil)
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 21 || g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil {
		t.Fatal("wrong sacrifice outcome", a.Life())
	}
	g.ResolveStack()
	if a.Life() != 23 {
		t.Fatal("wrong committed sacrifice event count", a.Life())
	}
}
