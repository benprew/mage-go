package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestComposableReturnToHandSavedLifetime(t *testing.T) {
	for _, change := range []string{"none", "return", "absent", "phase", "phase cycle", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Saved", "", 2, 2)
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			saved := BindPermanent("saved")
			ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: b.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(c, a.PlayerID())
			case "absent":
				g.RemoveFromBattlefield(p)
			case "phase":
				g.PhaseOut(c.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(c.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: c.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			}
			clone := g.Clone()
			ctx.Game = clone
			if err := ReturnToHand(From(saved)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := 0
			if change == "none" || change == "phase cycle" || change == "control" {
				want = 1
			}
			if got := len(clone.GetPlayer(a.PlayerID()).Hand()); got != want {
				t.Fatalf("owner hand = %d, want %d", got, want)
			}
			if len(clone.GetPlayer(b.PlayerID()).Hand()) != 0 {
				t.Fatal("returned card to controller")
			}
			if len(a.Hand()) != 0 || (change != "absent" && change != "phase" && g.FindPermanent(c.ID()) == nil) {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestComposableReturnToHandValidation(t *testing.T) {
	saved := BindPermanent("saved")
	for _, body := range []Effect{
		ReturnToHand(nil), ReturnToHand(From(saved)),
		ReturnToHand(DeclarePermanentTarget("undeclared", IsCreature)),
		Sequence(ReturnToHand(From(saved)), Snapshot(This, saved)),
	} {
		card := NewInstant("Invalid", "", WithAbility(Spell(body)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid subject")
		}
	}
}

func TestComposableReturnToHandLeaveTriggerAndToken(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Leave", "", 2, 2, WithAbility(OnLeaveZone(ZoneBattlefield, ZoneHand, GainLifeFor(You, Fixed(2)), false)))
	c.SetOwner(a.PlayerID())
	g.PutOnBattlefield(c, a.PlayerID())
	if err := ApplyEffect(g, Sequence(ReturnToHand(This), ReturnToHand(This), GainLifeFor(You, Fixed(1))), c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	resolvePendingTriggers(g)
	if len(a.Hand()) != 1 || a.Life() != 23 {
		t.Fatalf("hand = %d, life = %d; want one return, one trigger, and sequence continuation", len(a.Hand()), a.Life())
	}
	token := NewToken("Token", 1, 1, []CardType{TypeCreature}, nil)
	g.PutOnBattlefield(token, a.PlayerID())
	if err := ApplyEffect(g, ReturnToHand(This), token.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(token.ID()) != nil || len(a.Hand()) != 1 {
		t.Fatal("returned token remained on the battlefield or entered the hand")
	}
}

func TestComposableReturnToHandTargets(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "phase cycle", "absent"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
			for _, c := range []Card{first, second} {
				g.PutOnBattlefield(c, b.PlayerID())
				c.SetOwner(b.PlayerID())
			}
			x, y := DeclarePermanentTarget("first", IsCreature), DeclarePermanentTarget("second", IsCreature)
			ability := Spell(Sequence(ReturnToHand(x), ReturnToHand(y)), WithTargets(x, y))
			card := NewInstant("Return", "", WithAbility(ability))
			obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID(), second.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
			g.pushStack(obj)
			switch change {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
				g.PutOnBattlefield(first, b.PlayerID())
			case "phase":
				g.PhaseOut(first.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(first.ID()))
			case "absent":
				g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
			}
			clone := g.Clone()
			clone.ResolveStack()
			if change == "phase" {
				clone.PhaseIn([]uuid.UUID{first.ID()})
			}
			wantHand := 1
			if change == "none" || change == "phase cycle" {
				wantHand = 2
			}
			if got := len(clone.GetPlayer(b.PlayerID()).Hand()); got != wantHand {
				t.Fatalf("hand = %d, want %d", got, wantHand)
			}
			if clone.FindPermanent(second.ID()) != nil {
				t.Fatal("legal second target stayed on battlefield")
			}
			if g.FindPermanent(second.ID()) == nil || len(b.Hand()) != 0 {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestComposableReturnToHandCopyRetarget(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	for _, c := range []Card{first, second} {
		g.PutOnBattlefield(c, b.PlayerID())
		c.SetOwner(b.PlayerID())
	}
	target := DeclarePermanentTarget("creature", IsCreature)
	ability := Spell(ReturnToHand(target), WithTargets(target))
	card := NewInstant("Return", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	a.chooseQueue = [][]uuid.UUID{{second.ID()}}
	copied := g.CopySpellOnStack(card.ID(), a.PlayerID(), true)
	g.ResolveStackObject(copied)
	if g.FindPermanent(second.ID()) != nil || g.FindPermanent(first.ID()) == nil {
		t.Fatal("copy used original binding")
	}
	g.ResolveStackObject(obj)
	if g.FindPermanent(first.ID()) != nil {
		t.Fatal("copy changed original binding")
	}
}
