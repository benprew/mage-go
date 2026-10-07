package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestResolutionActionPayment(t *testing.T) {
	for _, kind := range []string{"life", "damage", "destroy"} {
		for _, state := range []string{"accept", "decline", "prevent", "regenerate", "indestructible", "absent", "returned", "phased", "phase-cycle"} {
			t.Run(kind+"/"+state, func(t *testing.T) {
				a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: state != "decline"}
				b := NewBasePlayer("B")
				g := NewGame(a, b)
				target := DeclarePermanentTarget("subject", IsCreature)
				var payment ResolutionActionPayment
				switch kind {
				case "life":
					payment = GainLifePayment(You, PowerOfWithLKI(target))
				case "damage":
					payment = DamagePayment(This, target, Fixed(3))
				case "destroy":
					payment = DestroyPayment(target)
				}
				sourceCard := NewArtifact("Source", "", WithAbility(Activate(Costs(), PayActionOrElse(You, payment, GainLifeFor(You, Fixed(1)), GainLifeFor(You, Fixed(10))), WithTargets(target))))
				if err := ValidateCardDefinition(sourceCard.Name(), sourceCard); err != nil {
					t.Fatal(err)
				}
				source := g.PutOnBattlefield(sourceCard, a.PlayerID())
				card := NewCreature("Subject", "", 4, 8)
				card.SetOwner(b.PlayerID())
				p := g.PutOnBattlefield(card, b.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
					t.Fatal(err)
				}
				switch state {
				case "prevent":
					g.AddPreventionShield(p.ID(), 3)
				case "regenerate":
					g.AddRegenerationShield(p.ID())
				case "indestructible":
					g.MutablePermanent(p.ID()).GrantBaseAttr(core.Indestructible)
				case "absent", "returned":
					g.BouncePermanentToHand(p)
					if state == "returned" {
						b.RemoveFromHand(card.ID())
						g.PutOnBattlefield(card, b.PlayerID())
					}
				case "phased":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				}
				clone := g.Clone()
				g.ResolveStack()
				illegal := state == "absent" || state == "returned" || state == "phased"
				paid := state != "decline" && (kind != "destroy" || state != "indestructible")
				want := 30
				if illegal {
					want = 20
				} else if paid {
					want = 21
					if kind == "life" {
						want += 4
					}
				}
				if a.Life() != want {
					t.Fatalf("life=%d want=%d", a.Life(), want)
				}
				if clone.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("clone changed")
				}
				if !illegal && kind == "damage" {
					wantDamage := 0
					if paid && state != "prevent" {
						wantDamage = 3
					}
					if g.FindPermanent(p.ID()).Damage != wantDamage {
						t.Fatal("damage outcome")
					}
				}
				if !illegal && kind == "destroy" {
					survives := !paid || state == "regenerate"
					if (g.FindPermanent(p.ID()) != nil) != survives {
						t.Fatal("destruction outcome")
					}
				}
			})
		}
	}
}

func TestResolutionActionPaymentInvalid(t *testing.T) {
	for _, payment := range []ResolutionActionPayment{nil, GainLifePayment(nil, Fixed(1)), GainLifePayment(You, nil), DamagePayment(nil, You, Fixed(1)), DamagePayment(This, nil, Fixed(1)), DamagePayment(This, You, nil), DestroyPayment(nil)} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), PayActionOrElse(You, payment, GainLifeFor(You, Fixed(1)), GainLifeFor(You, Fixed(2))))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid payment accepted")
		}
	}
}

func TestResolutionActionPaymentReplacedLifeAndNestedChoice(t *testing.T) {
	for _, amount := range []int{0, -2, 4} {
		a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
		g := NewGame(a, NewBasePlayer("B"))
		watcher := g.PutOnBattlefield(NewArtifact("Life modifier", ""), a.PlayerID())
		g.AddLifeGainModifier(watcher.ID(), func(_ *Game, _ uuid.UUID, _ int) int { return 0 })
		body := PayActionOrElse(You, GainLifePayment(You, Fixed(amount)), Sequence(
			PayActionOrElse(You, DestroyPayment(From(BindPermanent("absent"))), LoseLifeFor(You, Fixed(10)), LoseLifeFor(You, Fixed(2))),
			LoseLifeFor(You, Fixed(1))), LoseLifeFor(You, Fixed(100)))
		if err := ApplyEffect(g, body, watcher.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if a.Life() != 17 {
			t.Fatal("replaced life or nested payment changed the outer result")
		}
	}
}

func TestResolutionActionPaymentSourceLKI(t *testing.T) {
	for _, state := range []string{"left", "returned", "phased", "control"} {
		t.Run(state, func(t *testing.T) {
			a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			target := DeclarePermanentTarget("target", IsCreature)
			card := NewCreature("Source", "", 2, 8, WithKeyword(core.Lifelink), WithAbility(Activate(Costs(), PayActionOrElse(You, DamagePayment(This, target, Fixed(3)), LoseLifeFor(You, Fixed(1)), nil), WithTargets(target))))
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			recipient := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 8), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{recipient.ID()}); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "left", "returned":
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
			}
			clone := g.Clone()
			for _, game := range []*Game{g, clone} {
				if game == clone {
					game.SetPlayerAt(0, &optionalPaymentPlayer{BasePlayer: game.GetPlayer(a.PlayerID()).(*SearchPlayer).BasePlayer, accept: true})
				}
				game.ResolveStack()
				wantA, wantB := 22, 20
				if state == "control" {
					wantA, wantB = 19, 23
				}
				if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB || game.FindPermanent(recipient.ID()).Damage != 3 {
					t.Fatal("source identity or clone payment")
				}
			}
		})
	}
}
