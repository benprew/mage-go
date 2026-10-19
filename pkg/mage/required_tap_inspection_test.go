package mage

import (
	"reflect"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRequiredTapInspectionDeclaration(t *testing.T) {
	target := DeclarePermanentTarget("subject", HasCardTypeFilter(core.TypeLand))
	paid, unpaid := AddMana(core.Colorless, 3), AddMana(core.Blue, 1)
	d := RequiredTapInstruction{Player: You, Subject: target, Paid: paid, Unpaid: unpaid}
	effect := d.Effect()
	got, ok := InspectRequiredTapInstruction(effect)
	if effect == nil || !ok || got != d {
		t.Fatal("payment operands lost")
	}
	legacy, ok := InspectRequiredTapInstruction(RequiredTapOrElse(You, target, paid, unpaid))
	if !ok || legacy != d {
		t.Fatal("legacy payment lost")
	}
	var missing *requiredTapPayment
	for _, effect := range []Effect{nil, missing, Sequence(paid), RequiredTapOrElse(nil, target, paid, nil), RequiredTapOrElse(You, nil, paid, nil), RequiredTapOrElse(You, target, nil, nil)} {
		if _, ok := InspectRequiredTapInstruction(effect); ok {
			t.Fatal("malformed payment retained")
		}
	}
	for _, declaration := range []RequiredTapInstruction{{Subject: target, Paid: paid}, {Player: You, Paid: paid}, {Player: You, Subject: target}} {
		if declaration.Effect() != nil {
			t.Fatal("malformed declaration built")
		}
	}
}

func TestRequiredTapInspectionSpell(t *testing.T) {
	for _, state := range []string{"paid", "unpaid", "both", "suffix", "prefix", "tapped", "phase", "return", "unknown paid", "unknown unpaid", "unknown all illegal", "foreign payment", "foreign value", "foreign value suffix", "foreign value all illegal", "opaque value", "power", "foreign payer", "destruction branch", "missing snapshot", "nested"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			g.GetPlayer(obj.Controller).ManaPool().AddRestricted(core.Red, 2, core.ArtifactSpellsOnly{})
			amount := MultiplyValues(Fixed(1), ManaValueOfWithLKI(target))
			subject, payer := PermanentRef(target), PlayerRef(You)
			paid := AddManaAmount(core.Colorless, amount)
			var unpaid ResolvingEffect
			want, known := false, true
			switch state {
			case "unpaid":
				unpaid, paid = paid, nil
			case "both":
				unpaid = AddMana(core.Blue, 1)
			case "unknown paid":
				paid, known = destructionInspectionTrap{}, false
			case "unknown unpaid", "unknown all illegal":
				unpaid, known = destructionInspectionTrap{}, false
			case "foreign payment":
				subject, known = DeclarePermanentTarget("subject", HasCardTypeFilter(core.TypeLand)), false
			case "foreign value", "foreign value suffix", "foreign value all illegal":
				paid, known = AddManaAmount(core.Colorless, ManaValueOfWithLKI(DeclarePermanentTarget("subject", HasCardTypeFilter(core.TypeLand)))), false
			case "opaque value":
				paid, known = AddManaAmount(core.Colorless, damageInspectionValueTrap{}), false
			case "power":
				paid, known = AddManaAmount(core.Colorless, PowerOfWithLKI(target)), false
			case "foreign payer":
				payer, known = DeclarePlayerTarget("foreign"), false
			case "destruction branch":
				paid, known = Destroy(target), false
			case "nested":
				paid = RequiredTapOrElse(You, target, paid, nil)
				known = false
			case "missing snapshot":
				obj.targetIncarnations = nil
				known = false
			}
			payment := RequiredTapOrElse(payer, subject, paid, unpaid)
			obj.Effects = []Effect{payment}
			switch state {
			case "foreign value suffix", "foreign value all illegal":
				obj.Effects = []Effect{Destroy(target), payment}
				if state == "foreign value all illegal" {
					p.GrantBaseAttr(core.Shroud)
				}
			case "suffix":
				obj.Effects = []Effect{Destroy(target), payment}
				want = true
			case "prefix":
				obj.Effects = []Effect{payment, Destroy(target)}
				known = false
			case "tapped":
				g.TapPermanent(p)
			case "phase":
				g.PhaseOut(p.ID())
			case "return":
				g.BouncePermanentToHand(p)
			case "unknown all illegal":
				p.GrantBaseAttr(core.Shroud)
			}
			before := g.Clone()
			for _, game := range []*Game{g, g.Clone()} {
				for range 2 {
					certain, k := game.InspectSpellDestruction(game.StackPeek(), []PermanentVisit{{p.ID(), p.incarnationID}})
					if certain != want || k != known {
						t.Fatalf("got %v/%v, want %v/%v", certain, k, want, known)
					}
				}
			}
			if !reflect.DeepEqual(g.GetPlayer(obj.Controller).ManaPool(), before.GetPlayer(obj.Controller).ManaPool()) || !reflect.DeepEqual(g.random, before.random) || g.StackSize() != before.StackSize() || g.FindPermanent(p.ID()) != nil && g.FindPermanent(p.ID()).Tapped != (state == "tapped") {
				t.Fatal("inspection changed state")
			}
		})
	}
}

func TestRequiredTapInspectionReplacement(t *testing.T) {
	for _, state := range []string{"cancel", "return", "phase", "normal"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			p.Card.SetOwner(p.ControllerID())
			p.Card.(*BaseCard).manaCost = core.ParseManaCost("{X}{3}{U}")
			obj.Effects = []Effect{(RequiredTapInstruction{Player: You, Subject: target, Paid: AddManaAmount(core.Colorless, ManaValueOfWithLKI(target)), Unpaid: AddMana(core.Blue, 1)}).Effect()}
			calls := 0
			g.AddReplacementEffect(&testPrioritizedReplacement{
				replacementBase: replacementBase{duration: core.Indefinite},
				matchFunc:       func(action Action) bool { _, ok := action.(*tapPermanentAction); return ok },
				replaceFunc: func(action Action, game *Game) Action {
					calls++
					switch state {
					case "cancel":
						return nil
					case "phase":
						game.PhaseOut(p.ID())
					case "return":
						game.BouncePermanentToHand(p)
						game.GetPlayer(p.Card.Owner()).RemoveFromHand(p.Card.ID())
						game.PutOnBattlefield(p.Card, p.ControllerID())
					}
					return action
				},
			})
			for range 2 {
				if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known || calls != 0 {
					t.Fatal("query ran replacement")
				}
			}
			g.ResolveStack()
			pool := g.GetPlayer(obj.Controller).ManaPool()
			if pool.Count(core.Colorless) != 4 || pool.Count(core.Blue) != 0 || calls != 2 {
				t.Fatalf("paid replacement: mana %d/%d, calls %d", pool.Count(core.Colorless), pool.Count(core.Blue), calls)
			}
			if state == "return" && g.FindPermanent(p.ID()).Tapped {
				t.Fatal("new visit was tapped")
			}
			if state == "cancel" && g.FindPermanent(p.ID()).Tapped {
				t.Fatal("cancelled tap changed state")
			}
		})
	}
}
