package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCapturedBasePTLayersAndMembership(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	firstCard := NewCreature("First", "", 4, 5)
	first := g.PutOnBattlefield(firstCard, b.PlayerID())
	set := MatchingPermanents(IsCreature)
	apply := func(effect ResolvingEffect) {
		t.Helper()
		if err := ApplyEffect(g, effect, a.PlayerID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	apply(Until(core.Indefinite, SetEachBasePowerToughness(set, Fixed(0), Fixed(2))))
	later := g.PutOnBattlefield(NewCreature("Later", "", 6, 6), b.PlayerID())
	g.AddCountersWithReplacement(first, core.P1P1, 1, a.PlayerID(), false)
	apply(Until(core.EndOfTurn, ModifyEachPT(set, Fixed(2), Fixed(3))))
	if first.CurrentPower(g) != 3 || first.CurrentToughness(g) != 6 || later.CurrentPower(g) != 8 {
		t.Fatal("base values did not precede counters and bonuses")
	}
	apply(Until(core.EndOfTurn, SetEachBasePowerToughness(set, Fixed(7), Fixed(8))))
	if first.CurrentPower(g) != 10 || first.CurrentToughness(g) != 12 {
		t.Fatal("later base values lost timestamp order")
	}
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if first.CurrentPower(g) != 1 || first.CurrentToughness(g) != 3 || later.CurrentPower(g) != 6 {
		t.Fatal("expiry changed the indefinite membership or base values")
	}
	clone := g.Clone()
	clone.BouncePermanentToHand(clone.FindPermanent(first.ID()))
	if g.FindPermanent(first.ID()).CurrentPower(g) != 1 {
		t.Fatal("clone changed original layers")
	}
}

func TestCapturedBasePTDepartedBinding(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	recipientCard := NewCreature("Recipient", "", 1, 5)
	recipient := g.PutOnBattlefield(recipientCard, a.PlayerID())
	dealer := g.PutOnBattlefield(NewCreature("Dealer", "", 4, 5), b.PlayerID())
	binding := BindPermanent("recipient")
	set := PermanentsThatDealtDamageToThisTurn(From(binding), MatchingPermanents(IsCreature))
	body := Until(core.Indefinite, SetEachBasePowerToughness(set, Fixed(0), Fixed(2)))
	if err := ValidateCardDefinition("capture", NewCreature("capture", "", 1, 5, WithAbility(Activate(Costs(), Sequence(Snapshot(This, binding), body))))); err != nil {
		t.Fatal(err)
	}
	ctx := &EffectContext{Game: g, SourceID: recipient.ID(), Controller: a.PlayerID()}
	if err := Snapshot(This, binding).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.DealDamageToPermanent(recipient, 1, dealer.ID())
	g.RemoveFromBattlefield(recipient)
	g.zones.ClearLKI()
	g.PutOnBattlefield(recipientCard, a.PlayerID())
	if err := body.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if dealer.CurrentPower(g) != 0 || dealer.CurrentToughness(g) != 2 {
		t.Fatal("saved departed recipient lost its damage history")
	}
	g.damage.ClearEndOfTurn()
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if dealer.CurrentPower(g) != 0 || dealer.CurrentToughness(g) != 2 {
		t.Fatal("cleanup ended an indefinite captured effect")
	}
}

func TestCapturedBasePTDelayedSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	set := PermanentsThatDealtDamageToThisTurn(This, MatchingPermanents(IsCreature))
	body := Until(core.Indefinite, SetEachBasePowerToughness(set, Fixed(0), Fixed(2)))
	card := NewCreature("Recipient", "", 1, 5, WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))
	card.SetOwner(a.PlayerID())
	recipient := g.PutOnBattlefield(card, a.PlayerID())
	dealer := g.PutOnBattlefield(NewCreature("Dealer", "", 4, 5), b.PlayerID())
	g.DealDamageToPermanent(recipient, 1, dealer.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if sources := g.WaitingDelayedPermanentSources(); len(sources) != 1 || sources[0].ID() != recipient.ID() {
		t.Fatal("reverse history failed to declare its explicit source read")
	}
	g.BouncePermanentToHand(recipient)
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if dealer.CurrentPower(g) != 0 || dealer.CurrentToughness(g) != 2 {
		t.Fatal("delayed reverse history lost the source visit")
	}
}
