package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type antePaymentPlayer struct {
	*BasePlayer
	accept  bool
	offers  int
	onOffer func()
}

func (p *antePaymentPlayer) ChooseMayAbility(string) bool {
	p.offers++
	if p.onOffer != nil {
		p.onOffer()
	}
	return p.accept
}

func TestAntePaymentPermanent(t *testing.T) {
	for _, required := range []bool{false, true} {
		for _, state := range []string{"pay", "decline", "foreign-owner", "foreign-control", "phased", "departed", "returned", "token", "disabled", "settled", "offer-return"} {
			t.Run(state+map[bool]string{false: "/optional", true: "/required"}[required], func(t *testing.T) {
				a := &antePaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: state != "decline"}
				b := NewBasePlayer("B")
				g, _ := NewGameWithAnte(a, b, nil, nil)
				card := NewArtifact("Payment", "")
				if state == "token" {
					card = NewToken("Payment", 1, 1, []core.CardType{core.TypeCreature}, nil)
				}
				card.SetOwner(a.PlayerID())
				if state == "foreign-owner" {
					card.SetOwner(b.PlayerID())
				}
				controller := a.PlayerID()
				if state == "foreign-control" {
					controller = b.PlayerID()
				}
				p := g.PutOnBattlefield(card, controller)
				ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID(), sourceIncarnation: p.incarnationID, sourceCaptured: true}
				if state == "phased" {
					g.PhaseOut(p.ID())
				}
				if state == "departed" || state == "returned" {
					g.BouncePermanentToHand(p)
				}
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
				if state == "disabled" {
					g.anteEnabled = false
				}
				if state == "settled" {
					g.anteSettled = true
				}
				if state == "offer-return" {
					a.onOffer = func() {
						g.BouncePermanentToHand(p)
						a.RemoveFromHand(card.ID())
						g.PutOnBattlefield(card, a.PlayerID())
					}
				}
				paid, unpaid := GainLifeFor(You, Fixed(3)), LoseLifeFor(You, Fixed(2))
				var effect = PayPermanentAnteOrElse(You, This, paid, unpaid)
				if required {
					effect = AntePermanentOrElse(You, This, paid, unpaid)
				}
				if err := ValidateCardDefinition("Definition", NewArtifact("Definition", "", WithAbility(Activate(Costs(), effect)))); err != nil {
					t.Fatal(err)
				}
				if err := effect.Apply(ctx); err != nil {
					t.Fatal(err)
				}
				wantPaid := state == "pay" || state == "foreign-control" || required && (state == "decline" || state == "offer-return")
				wantLife := 18
				if wantPaid {
					wantLife = 23
				}
				if a.Life() != wantLife || (len(a.Ante()) == 1) != wantPaid {
					t.Fatalf("life=%d ante=%d", a.Life(), len(a.Ante()))
				}
				if required && a.offers != 0 {
					t.Fatal("mandatory payment offered a choice")
				}
				if wantPaid && a.Ante()[0].ID() != card.ID() {
					t.Fatal("physical identity changed")
				}
			})
		}
	}
}

func TestAntePaymentTopTransaction(t *testing.T) {
	g, a, _ := anteEffectTestGame(t)
	first, second := ownedTestCard(a.PlayerID(), "first"), ownedTestCard(a.PlayerID(), "second")
	a.SetLibrary([]Card{first, second})
	a.ManaPool().Add(core.Red, 1)
	cost := AnteTopCost()
	if !cost.CanPay(uuid.Nil, a.PlayerID(), g) || len(a.Ante()) != 0 {
		t.Fatal("impure affordability")
	}
	tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{Controller: a.PlayerID(), Mana: core.ParseManaCost("{R}"), Costs: []Cost{cost, cost, cost}}, nil)
	if err == nil {
		t.Fatal("accepted incomplete payment", tx)
	}
	if len(a.Library()) != 2 || len(a.Ante()) != 0 || a.ManaPool().TotalMana() != 1 {
		t.Fatal("failed payment changed state")
	}
	tx, err = g.prepareCostPaymentTransaction(costPaymentSpec{Controller: a.PlayerID(), Mana: core.ParseManaCost("{R}"), Costs: []Cost{cost, cost}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if len(clone.GetPlayer(a.PlayerID()).Ante()) != 0 {
		t.Fatal("preparation moved cards")
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(a.Ante()) != 2 || a.Ante()[0].ID() != first.ID() || a.Ante()[1].ID() != second.ID() || a.ManaPool().TotalMana() != 0 {
		t.Fatal("wrong locked payment")
	}
}

func TestAntePaymentStaleTop(t *testing.T) {
	g, a, _ := anteEffectTestGame(t)
	first, second := ownedTestCard(a.PlayerID(), "first"), ownedTestCard(a.PlayerID(), "second")
	a.SetLibrary([]Card{first, second})
	a.ManaPool().Add(core.Red, 1)
	tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{Controller: a.PlayerID(), Mana: core.ParseManaCost("{R}"), Costs: []Cost{AnteTopCost()}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	a.SetLibrary([]Card{second, first})
	if err := tx.Commit(); err == nil {
		t.Fatal("accepted stale top")
	}
	if a.ManaPool().TotalMana() != 1 || len(a.Ante()) != 0 {
		t.Fatal("stale payment spent resources")
	}
}

func TestAntePaymentDefinition(t *testing.T) {
	for _, effect := range []ResolvingEffect{PayPermanentAnteOrElse(nil, This, GainLifeFor(You, Fixed(1)), nil), AntePermanentOrElse(You, nil, GainLifeFor(You, Fixed(1)), nil), AntePermanentOrElse(You, This, nil, nil)} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))); err == nil {
			t.Fatal("invalid payment accepted")
		}
	}
}

func TestAntePaymentPermanentTransaction(t *testing.T) {
	g, a, _ := anteEffectTestGame(t)
	card := ownedTestCard(a.PlayerID(), "payment")
	p := g.PutOnBattlefield(card, a.PlayerID())
	a.ManaPool().Add(core.Red, 1)
	cost := &anteCost{locked: true, permanent: capturedPermanent{id: p.ID(), incarnation: p.incarnationID}}
	_, err := g.prepareCostPaymentTransaction(costPaymentSpec{Controller: a.PlayerID(), SourceID: p.ID(), Mana: core.ParseManaCost("{R}"), Costs: []Cost{Tap(), cost, LifePayCost(21)}}, nil)
	if err == nil || len(a.Ante()) != 0 || a.ManaPool().TotalMana() != 1 || g.FindPermanent(p.ID()).Tapped {
		t.Fatal("failed combined payment changed state")
	}
	tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{Controller: a.PlayerID(), SourceID: p.ID(), Mana: core.ParseManaCost("{R}"), Costs: []Cost{Tap(), cost}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(p)
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if err := tx.Commit(); err == nil {
		t.Fatal("stale permanent paid")
	}
	if a.ManaPool().TotalMana() != 1 || g.FindPermanent(returned.ID()).Tapped || len(a.Ante()) != 0 {
		t.Fatal("stale combined payment spent resources")
	}
}

func TestAntePaymentTopLegality(t *testing.T) {
	for _, state := range []string{"empty", "foreign-owner", "token", "disabled", "settled", "decline", "pay"} {
		t.Run(state, func(t *testing.T) {
			a := &antePaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: state != "decline"}
			b := NewBasePlayer("B")
			g, _ := NewGameWithAnte(a, b, nil, nil)
			card := ownedTestCard(a.PlayerID(), "top")
			if state == "foreign-owner" {
				card.SetOwner(b.PlayerID())
			}
			if state == "token" {
				card = NewToken("top", 1, 1, []core.CardType{core.TypeCreature}, nil)
				card.SetOwner(a.PlayerID())
			}
			if state != "empty" {
				a.SetLibrary([]Card{card})
			}
			if state == "disabled" {
				g.anteEnabled = false
			}
			if state == "settled" {
				g.anteSettled = true
			}
			effect := PayOrElse(You, Costs(AnteTopCost()), GainLifeFor(You, Fixed(3)), LoseLifeFor(You, Fixed(2)))
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			want := 18
			if state == "pay" {
				want = 23
			}
			if a.Life() != want || (len(a.Ante()) == 1) != (state == "pay") {
				t.Fatal("legality or branch")
			}
			if state != "pay" && state != "decline" && a.offers != 0 {
				t.Fatal("offered impossible payment")
			}
		})
	}
}
