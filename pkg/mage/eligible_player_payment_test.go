package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type eligiblePaymentPlayer struct {
	*BasePlayer
	accept bool
	order  *[]string
	hook   func()
}

func TestEligiblePlayerPaymentRecheckAndNested(t *testing.T) {
	var order []string
	a := &eligiblePaymentPlayer{BasePlayer: NewBasePlayer("A"), order: &order, accept: true}
	b := &eligiblePaymentPlayer{BasePlayer: NewBasePlayer("B"), order: &order, accept: true}
	g := NewGame(a, b)
	a.ManaPool().Add(core.Colorless, 2)
	b.ManaPool().Add(core.Colorless, 2)
	a.hook = func() { a.ManaPool().Clear() }
	inner := PayAnyOrElse(OpponentsOf(You), Costs(LifePayCost(50)), GainLifeFor(You, Fixed(100)), GainLifeFor(You, Fixed(3)))
	effect := PayAnyOrElse(EachPlayer, Costs(ManaCostOf("{2}")), inner, GainLifeFor(You, Fixed(7)))
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 23 || b.ManaPool().TotalMana() != 0 || len(order) != 2 || order[0] != "A" || order[1] != "B" {
		t.Fatal("failed offer stopped later payment or nested result changed outer result", order)
	}
}

func TestEligiblePlayerPaymentDelayedIdentity(t *testing.T) {
	for _, state := range []string{"live", "return", "phase", "control"} {
		t.Run(state, func(t *testing.T) {
			var order []string
			a := &eligiblePaymentPlayer{BasePlayer: NewBasePlayer("A"), order: &order, accept: true}
			b := &eligiblePaymentPlayer{BasePlayer: NewBasePlayer("B"), order: &order, accept: true}
			g := NewGame(a, b)
			card := NewArtifact("Source", "")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(source.ID())
			body := PayAnyOrElse(OpponentsOf(ControllerOfWithLKI(This)), Costs(LifePayCost(2)), GainLifeFor(You, Fixed(3)), TapSubject(This))
			declaration := NewArtifact("Delay", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))
			if err := ValidateCardDefinition(declaration.Name(), declaration); err != nil {
				t.Fatal(err)
			}
			if err := ApplyEffect(g, AtNext(EachEndStep, body), source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "return":
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(card, b.PlayerID())
			case "phase":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			for range 3 {
				refs := g.WaitingDelayedPermanentSources()
				if len(refs) != 1 || !refs[0].SameObject(saved) || len(order) != 0 || a.Life() != 20 || b.Life() != 20 {
					t.Fatal("waiting query executed payment or lost source")
				}
			}
			clone := g.Clone()
			clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 || len(g.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("clone changed original")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.ResolveStack()
			wantA, wantB := 23, 18
			if state == "control" {
				wantA, wantB = 21, 20
			}
			if g.GetPlayer(a.PlayerID()).Life() != wantA || g.GetPlayer(b.PlayerID()).Life() != wantB || clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatalf("wrong saved visit/controller/payment result: original %d/%d clone %d/%d", g.GetPlayer(a.PlayerID()).Life(), g.GetPlayer(b.PlayerID()).Life(), clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life())
			}
		})
	}
}

func (p *eligiblePaymentPlayer) ChooseMayAbility(string) bool {
	*p.order = append(*p.order, p.Name())
	if p.hook != nil {
		p.hook()
	}
	return p.accept
}

func TestEligiblePlayerPaymentOrderAndResult(t *testing.T) {
	for _, active := range []int{0, 1} {
		for _, state := range []string{"first", "second", "none", "short", "rollback", "opponent", "empty", "zero"} {
			t.Run(state+string(rune('0'+active)), func(t *testing.T) {
				var order []string
				a := &eligiblePaymentPlayer{BasePlayer: NewBasePlayer("A"), order: &order}
				b := &eligiblePaymentPlayer{BasePlayer: NewBasePlayer("B"), order: &order}
				g := NewGame(a, b)
				g.SetActivePlayerIndex(active)
				first, second := a, b
				if active == 1 {
					first, second = b, a
				}
				first.accept = state == "first" || state == "rollback" || state == "zero"
				second.accept = state == "second" || state == "short" || state == "opponent"
				players := PlayerSet(EachPlayer)
				costs := Costs(LifePayCost(2))
				if state == "short" {
					first.SetLife(1)
				}
				if state == "rollback" {
					costs = Costs(LifePayCost(15), LifePayCost(15))
				}
				if state == "opponent" {
					players = OpponentsOf(You)
					a.accept = true
					b.accept = true
				}
				if state == "empty" {
					players = OpponentsOf(FromPlayer(BindPlayer("missing")))
				}
				if state == "zero" {
					costs = Costs()
				}
				effect := PayAnyOrElse(players, costs, GainLifeFor(You, Fixed(3)), GainLifeFor(You, Fixed(7)))
				beforeA, beforeB := a.Life(), b.Life()
				if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				wantA, wantB := beforeA, beforeB
				success := state == "first" || state == "second" || state == "short" || state == "opponent" || state == "zero"
				if success {
					wantA += 3
					payer := first
					if state == "second" || state == "short" {
						payer = second
					}
					if state == "opponent" {
						payer = b
					}
					if state != "zero" {
						if payer == a {
							wantA -= 2
						} else {
							wantB -= 2
						}
					}
				} else {
					wantA += 7
				}
				if g.GetPlayer(a.PlayerID()).Life() != wantA || g.GetPlayer(b.PlayerID()).Life() != wantB {
					t.Fatalf("life A=%d B=%d, want %d %d", g.GetPlayer(a.PlayerID()).Life(), g.GetPlayer(b.PlayerID()).Life(), wantA, wantB)
				}
				if state == "first" || state == "zero" {
					if len(order) != 1 || order[0] != first.Name() {
						t.Fatal("did not stop at first payment", order)
					}
				}
				if state == "second" || state == "none" || state == "rollback" {
					if len(order) != 2 || order[0] != first.Name() || order[1] != second.Name() {
						t.Fatal("wrong offer order", order)
					}
				}
				if state == "short" {
					if len(order) != 1 || order[0] != second.Name() {
						t.Fatal("offered impossible payment", order)
					}
				}
				if state == "empty" {
					if len(order) != 0 {
						t.Fatal("offered absent players", order)
					}
				}
				if state == "opponent" {
					if len(order) != 1 || order[0] != "B" {
						t.Fatal("offered wrong opponent", order)
					}
				}
			})
		}
	}
}

func TestEligiblePlayerPaymentInvalid(t *testing.T) {
	saved := BindPermanent("local")
	for _, effect := range []ResolvingEffect{
		PayAnyOrElse(nil, Costs(), GainLifeFor(You, Fixed(1)), nil),
		PayAnyOrElse(EachPlayer, CostGroup{}, GainLifeFor(You, Fixed(1)), nil),
		PayAnyOrElse(EachPlayer, Costs(), nil, nil),
		PayAnyOrElse(EachPlayer, Costs(nil), GainLifeFor(You, Fixed(1)), nil),
		PayAnyOrElse(EachPlayer, Costs(), Snapshot(This, saved), TapSubject(From(saved))),
		Sequence(PayAnyOrElse(EachPlayer, Costs(), Snapshot(This, saved), nil), TapSubject(From(saved))),
	} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid declaration")
		}
	}
}
