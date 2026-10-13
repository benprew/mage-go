package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestEligiblePayerScopeOffersAndRollback(t *testing.T) {
	for _, state := range []string{"first", "second", "none", "short", "rollback"} {
		t.Run(state, func(t *testing.T) {
			var order []string
			a := &eligiblePaymentPlayer{BasePlayer: NewBasePlayer("A"), order: &order, accept: state == "first" || state == "rollback"}
			b := &eligiblePaymentPlayer{BasePlayer: NewBasePlayer("B"), order: &order, accept: state == "second" || state == "short" || state == "rollback"}
			g := NewGame(a, b)
			payer := BindPlayer("payer")
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			for _, p := range []*eligiblePaymentPlayer{a, b} {
				p.hook = func() {
					id, ok := FromPlayer(payer).resolvePlayer(ctx)
					if !ok || id != p.PlayerID() || ctx.Controller != a.PlayerID() {
						t.Fatal("offer lost payer or controller")
					}
					if state == "rollback" && p == a {
						a.SetLife(1)
					}
				}
			}
			if state == "short" {
				a.SetLife(1)
			}
			effect := PayAnyOrElseWithPayer(EachPlayer, Costs(LifePayCost(2)), payer, GainLifeFor(FromPlayer(payer), Fixed(5)), GainLifeFor(You, Fixed(7)))
			card := NewArtifact("Offer", "", WithAbility(Activate(Costs(), effect)))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			wantA, wantB := 20, 20
			switch state {
			case "first":
				wantA = 23
			case "second":
				wantB = 23
			case "short", "rollback":
				wantA, wantB = 1, 23
			case "none":
				wantA = 27
			}
			if g.GetPlayer(a.PlayerID()).Life() != wantA || g.GetPlayer(b.PlayerID()).Life() != wantB {
				t.Fatalf("life=%d/%d want %d/%d", a.Life(), b.Life(), wantA, wantB)
			}
			if _, ok := FromPlayer(payer).resolvePlayer(ctx); ok {
				t.Fatal("payer escaped payment scope")
			}
			if state == "first" && len(order) != 1 {
				t.Fatal("offered after complete payment", order)
			}
			if state == "short" && len(order) != 1 {
				t.Fatal("offered unaffordable payment", order)
			}
		})
	}
}

func TestEligiblePayerScopeNestedAndDelayed(t *testing.T) {
	var order []string
	a := &eligiblePaymentPlayer{BasePlayer: NewBasePlayer("A"), order: &order, accept: true}
	b := &eligiblePaymentPlayer{BasePlayer: NewBasePlayer("B"), order: &order, accept: true}
	g := NewGame(a, b)
	outer, inner := BindPlayer("outer payer"), BindPlayer("inner payer")
	effect := PayAnyOrElseWithPayer(OpponentsOf(You), Costs(LifePayCost(1)), outer,
		Sequence(PayAnyOrElseWithPayer(EachPlayer, Costs(LifePayCost(1)), inner, GainLifeFor(FromPlayer(inner), Fixed(2)), nil),
			AtNext(EachEndStep, GainLifeFor(FromPlayer(outer), Fixed(5))), GainLifeFor(FromPlayer(outer), Fixed(3))), nil)
	card := NewArtifact("Offer", "", WithAbility(Activate(Costs(), effect)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 22 {
		t.Fatal("nested payer overwrote outer payer")
	}
	g.RemoveFromBattlefield(source)
	clone := g.Clone()
	for range 3 {
		g.WaitingDelayedPermanentSources()
	}
	if a.Life() != 21 || b.Life() != 22 || len(order) != 2 {
		t.Fatal("waiting query executed payer body")
	}
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	clone.ResolveStack()
	if clone.GetPlayer(b.PlayerID()).Life() != 27 || b.Life() != 22 {
		t.Fatal("clone lost saved payer or changed original")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.ResolveStack()
	if b.Life() != 27 || a.Life() != 21 {
		t.Fatal("delay did not retain payer after local binding ended")
	}
}

func TestEligiblePayerScopeInvalid(t *testing.T) {
	payer := BindPlayer("payer")
	for _, effect := range []ResolvingEffect{
		PayAnyOrElseWithPayer(EachPlayer, Costs(), nil, GainLifeFor(You, Fixed(1)), nil),
		PayAnyOrElseWithPayer(EachPlayer, Costs(), BindPlayer(""), GainLifeFor(You, Fixed(1)), nil),
		PayAnyOrElseWithPayer(EachPlayer, Costs(), payer, nil, GainLifeFor(FromPlayer(payer), Fixed(1))),
		Sequence(PayAnyOrElseWithPayer(EachPlayer, Costs(), payer, GainLifeFor(You, Fixed(1)), nil), GainLifeFor(FromPlayer(payer), Fixed(1))),
		Sequence(SnapshotPlayer(You, payer), PayAnyOrElseWithPayer(EachPlayer, Costs(), payer, GainLifeFor(FromPlayer(payer), Fixed(1)), nil)),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid payer scope")
		}
	}
	if err := ApplyEffect(NewGame(NewBasePlayer("A"), NewBasePlayer("B")), PayAnyOrElse(EachPlayer, Costs(), GainLifeFor(You, Fixed(1)), nil), uuid.Nil, uuid.Nil, nil); err != nil {
		t.Fatal(err)
	}
}

type payerCheckedCost struct {
	payer   *PlayerBinding
	failFor uuid.UUID
}

func (c *payerCheckedCost) CanPay(source, controller uuid.UUID, g *Game) bool {
	id, ok := FromPlayer(c.payer).resolvePlayer(&EffectContext{Game: g, SourceID: source, Controller: controller})
	return ok && id == controller && LifePayCost(2).CanPay(source, controller, g)
}
func (c *payerCheckedCost) Pay(source, controller uuid.UUID, g *Game) error {
	if !c.CanPay(source, controller, g) {
		return fmt.Errorf("cost lost candidate payer")
	}
	if err := LifePayCost(2).Pay(source, controller, g); err != nil {
		return err
	}
	if controller == c.failFor {
		return fmt.Errorf("payment failed after staged life payment")
	}
	return nil
}
func (*payerCheckedCost) Text() string { return "pay 2 life as the candidate" }
func (c *payerCheckedCost) inspectDefinition() definitionNode {
	return definitionNode{Role: "test-payer-cost", children: []definitionEdge{{"payer", FromPlayer(c.payer)}}}
}

func TestEligiblePayerScopeCostBinding(t *testing.T) {
	var order []string
	a := &eligiblePaymentPlayer{BasePlayer: NewBasePlayer("A"), order: &order, accept: true}
	b := &eligiblePaymentPlayer{BasePlayer: NewBasePlayer("B"), order: &order, accept: true}
	g := NewGame(a, b)
	payer := BindPlayer("candidate")
	cost := &payerCheckedCost{payer: payer, failFor: a.PlayerID()}
	effect := PayAnyOrElseWithPayer(EachPlayer, Costs(cost), payer, GainLifeFor(FromPlayer(payer), Fixed(3)), nil)
	card := NewArtifact("Offer", "", WithAbility(Activate(Costs(), effect)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || b.Life() != 21 || len(order) != 2 {
		t.Fatal("failed staged payment leaked resources or candidate binding", a.Life(), b.Life(), order)
	}
	if _, ok := g.resolution.playerBindings[payer]; ok {
		t.Fatal("candidate binding escaped frame")
	}
	invalid := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), PayAnyOrElse(EachPlayer, Costs(cost), GainLifeFor(You, Fixed(1)), nil))))
	if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
		t.Fatal("unscoped cost read was accepted")
	}
}
