package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type counterPlacementClonePayer struct{ Player }

func (*counterPlacementClonePayer) ChooseMayAbility(string) bool { return true }

type counterPlacementPayer struct {
	*BasePlayer
	choice *Permanent
	calls  int
	accept bool
}

func (p *counterPlacementPayer) ChooseMayAbility(string) bool { return p.accept }
func (p *counterPlacementPayer) ChoosePermanent(cs []*Permanent, _ string, _ GameReader) *Permanent {
	p.calls++
	if p.choice != nil {
		return p.choice
	}
	return cs[0]
}

func TestPutCounterCostTransaction(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "phase-cycle", "control", "filter", "absent", "invalid", "later-failure", "doubled"} {
		t.Run(change, func(t *testing.T) {
			a := &counterPlacementPayer{BasePlayer: NewBasePlayer("A"), accept: true}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Payment", "", 2, 4)
			card.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			if change == "invalid" {
				a.choice = source
			}
			if change == "doubled" {
				g.AddCounterDoubler(source.ID(), core.M1M1, IsCreature)
			}
			g.MutablePermanent(p.ID()).AddCounter(core.Charge, 1)
			cost := PutCounterOnMatchingCost(core.M1M1, And(IsCreature, HasCounter(core.Charge)))
			for range 3 {
				if !cost.CanPay(source.ID(), a.PlayerID(), g) {
					t.Fatal("pure availability")
				}
			}
			if a.calls != 0 || g.FindPermanent(p.ID()).Counters[core.M1M1] != 0 {
				t.Fatal("query changed state")
			}
			a.ManaPool().Add(core.Red, 1)
			costs := []Cost{LifePayCost(2), cost}
			if change == "later-failure" {
				costs = append(costs, SacrificeMatchingCost(IsLand, "land"))
			}
			tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Mana: core.ParseManaCost("{R}"), Costs: costs}, nil)
			if change == "invalid" || change == "later-failure" {
				if err == nil {
					t.Fatal("invalid payment accepted")
				}
				if a.Life() != 20 || a.ManaPool().Count(core.Red) != 1 || g.FindPermanent(p.ID()).Counters[core.M1M1] != 0 {
					t.Fatal("failed preparation changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if a.calls != 1 || g.FindPermanent(p.ID()).Counters[core.M1M1] != 0 {
				t.Fatal("choice not locked or early placement")
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, a.PlayerID())
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "filter":
				g.MutablePermanent(p.ID()).RemoveCounter(core.Charge, 1)
			case "absent":
				g.RemoveFromBattlefield(p)
			}
			err = tx.Commit()
			paid := change == "none" || change == "phase-cycle" || change == "doubled"
			if (err == nil) != paid {
				t.Fatalf("commit=%v paid=%t", err, paid)
			}
			if a.calls != 1 {
				t.Fatal("commit repeated choice")
			}
			if !paid {
				if a.Life() != 20 || a.ManaPool().Count(core.Red) != 1 {
					t.Fatal("failed commit spent resources")
				}
			} else {
				want := 1
				if change == "doubled" {
					want = 2
				}
				if a.Life() != 18 || a.ManaPool().Count(core.Red) != 0 || int(g.FindPermanent(p.ID()).Counters[core.M1M1]) != want {
					t.Fatal("wrong payment")
				}
			}
		})
	}
}

func TestPutCounterCostResolutionAndDeclaration(t *testing.T) {
	for _, state := range []string{"pay", "decline", "opponent", "phase", "missing", "source", "clone"} {
		t.Run(state, func(t *testing.T) {
			a := &counterPlacementPayer{BasePlayer: NewBasePlayer("A"), accept: state != "decline"}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			owner := a.PlayerID()
			if state == "opponent" {
				owner = b.PlayerID()
			}
			p := g.PutOnBattlefield(NewCreature("Payment", "", 2, 4), owner)
			if state == "phase" {
				g.PhaseOut(p.ID())
			}
			if state == "missing" {
				g.RemoveFromBattlefield(p)
			}
			effect := IfPaid(You, Costs(PutCounterOnMatchingCost(core.M1M1, IsCreature)), GainLifeFor(You, Fixed(3)))
			c := NewArtifact("Declaration", "", WithAbility(Activate(Costs(), effect)))
			if err := ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			game := g
			if state == "clone" {
				game = g.Clone()
				game.SetPlayerAt(0, &counterPlacementClonePayer{game.GetPlayer(a.PlayerID())})
			}
			src := uuid.Nil
			if state == "source" {
				src = p.ID()
			}
			if err := ApplyEffect(game, effect, src, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			paid := state == "pay" || state == "source" || state == "clone"
			want := 20
			if paid {
				want = 23
			}
			if game.GetPlayer(a.PlayerID()).Life() != want {
				t.Fatal("wrong paid branch")
			}
			if state == "clone" && (a.Life() != 20 || g.FindPermanent(p.ID()).Counters[core.M1M1] != 0) {
				t.Fatal("clone shared payment")
			}
		})
	}
	bad := NewArtifact("Invalid", "", WithAbility(Activate(Costs(PutCounterOnMatchingCost(core.NumCounters, IsCreature)), GainLifeFor(You, Fixed(1)))))
	if ValidateCardDefinition(bad.Name(), bad) == nil {
		t.Fatal("invalid counter accepted")
	}
}

func TestPutCounterCostReplacementPayment(t *testing.T) {
	for _, mode := range []string{"prevent", "change-type", "double"} {
		t.Run(mode, func(t *testing.T) {
			a := &counterPlacementPayer{BasePlayer: NewBasePlayer("A"), accept: true}
			g := NewGame(a, NewBasePlayer("B"))
			p := g.PutOnBattlefield(NewCreature("Payment", "", 2, 4, WithKeyword(core.Shroud)), a.PlayerID())
			g.AddReplacementEffect(&testPrioritizedReplacement{
				replacementBase: replacementBase{duration: core.Indefinite},
				matchFunc:       func(a Action) bool { _, ok := a.(*AddCountersAction); return ok },
				replaceFunc: func(a Action, _ *Game) Action {
					if mode == "prevent" {
						return nil
					}
					if mode == "change-type" {
						changed := *a.(*AddCountersAction)
						changed.counterType = core.P1P1
						return &changed
					}
					return a.(*AddCountersAction).WithAmount(2)
				},
			})
			body := IfPaid(You, Costs(PutCounterOnMatchingCost(core.M1M1, IsCreature)), GainLifeFor(You, Fixed(3)))
			if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 23 || a.calls != 1 {
				t.Fatal("replacement changed paid result or repeated choice")
			}
			p = g.FindPermanent(p.ID())
			wantMinus, wantPlus := 0, 0
			if mode == "double" {
				wantMinus = 2
			}
			if mode == "change-type" {
				wantPlus = 1
			}
			if int(p.Counters[core.M1M1]) != wantMinus || int(p.Counters[core.P1P1]) != wantPlus {
				t.Fatal("replacement bypassed")
			}
		})
	}
}

func TestPutCounterCostStaleChoice(t *testing.T) {
	a := &counterPlacementPayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	card := NewCreature("Payment", "", 2, 4)
	p := g.PutOnBattlefield(card, a.PlayerID())
	a.choice = p
	g.RemoveFromBattlefield(p)
	returned := g.PutOnBattlefield(card, a.PlayerID())
	effect := IfPaid(You, Costs(PutCounterOnMatchingCost(core.M1M1, IsCreature)), GainLifeFor(You, Fixed(3)))
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || g.FindPermanent(returned.ID()).Counters[core.M1M1] != 0 {
		t.Fatal("stale incarnation paid")
	}
}

func TestPutCounterCostAlternativeTransaction(t *testing.T) {
	a := &counterPlacementPayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	p := g.PutOnBattlefield(NewCreature("Payment", "", 2, 4), a.PlayerID())
	tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{EitherCost(EitherCost(PutCounterOnMatchingCost(core.M1M1, IsCreature)))}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.calls != 1 {
		t.Fatal("selection not locked during preparation")
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if a.calls != 1 || g.FindPermanent(p.ID()).Counters[core.M1M1] != 1 {
		t.Fatal("alternative repeated selection")
	}
}

type counterPlacementOrderPayer struct {
	*counterPlacementPayer
	index, orders int
}

func (p *counterPlacementOrderPayer) ChooseMode(_ []string, reason string) int {
	if reason == "Choose a replacement or prevention effect" {
		p.orders++
		return p.index
	}
	return 0
}
func TestPutCounterCostReplacementOrder(t *testing.T) {
	for _, index := range []int{0, 1} {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			a := &counterPlacementOrderPayer{counterPlacementPayer: &counterPlacementPayer{BasePlayer: NewBasePlayer("A"), accept: true}, index: index}
			g := NewGame(a, NewBasePlayer("B"))
			p := g.PutOnBattlefield(NewCreature("Payment", "", 2, 8), a.PlayerID())
			for _, double := range []bool{false, true} {
				g.AddReplacementEffect(&testPrioritizedReplacement{replacementBase: replacementBase{duration: core.Indefinite}, matchFunc: func(action Action) bool { _, ok := action.(*AddCountersAction); return ok }, replaceFunc: func(action Action, _ *Game) Action {
					c := action.(*AddCountersAction)
					if double {
						return c.WithAmount(c.Amount() * 2)
					}
					return c.WithAmount(c.Amount() + 1)
				}})
			}
			if err := ApplyEffect(g, IfPaid(You, Costs(PutCounterOnMatchingCost(core.M1M1, IsCreature)), GainLifeFor(You, Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			want := 4
			if index == 1 {
				want = 3
			}
			if int(g.FindPermanent(p.ID()).Counters[core.M1M1]) != want || a.orders != 1 || a.calls != 1 || a.Life() != 21 {
				t.Fatal("affected payer order or locked payment")
			}
		})
	}
}
