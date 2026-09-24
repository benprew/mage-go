package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTypedSingularLifeAmountContext(t *testing.T) {
	for _, gain := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		source := NewCreature("Amount", "", 3, 4)
		g.PutOnBattlefield(source, a.PlayerID())
		saved := BindPermanent("amount")
		amount := SubtractValues(PowerOf(From(saved)), Fixed(1))
		effect := LoseLifeFor(ActivePlayer, amount)
		want := 18
		if gain {
			effect = GainLifeFor(ActivePlayer, amount)
			want = 22
		}
		body := Sequence(Snapshot(This, saved), effect)
		clone := g.Clone()
		if err := ApplyEffect(clone, body, source.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if clone.GetPlayer(a.PlayerID()).Life() != want || clone.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatal("saved amount or recipient was lost")
		}
		if a.Life() != 20 || b.Life() != 20 {
			t.Fatal("clone changed original life")
		}
	}
}

func TestTypedSingularLifeTargetCopy(t *testing.T) {
	for _, gain := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		source := NewCreature("Amount", "", 3, 4)
		g.PutOnBattlefield(source, a.PlayerID())
		player := DeclarePlayerTarget("player")
		amount := DeclarePermanentTarget("amount", IsCreature)
		effect := LoseLifeFor(player, ToughnessOf(amount))
		want := 16
		if gain {
			effect = GainLifeFor(player, ToughnessOf(amount))
			want = 24
		}
		ability := Spell(effect, WithTargets(player, amount))
		card := NewSorcery("Life", "", WithAbility(ability))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		original := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID(), source.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
		g.pushStack(original)
		clone := g.Clone()
		copied := clone.CopySpellOnStack(card.ID(), b.PlayerID(), false)
		if copied == nil {
			t.Fatal("copy missing")
		}
		clone.ResolveStackObject(copied)
		if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != want {
			t.Fatal("copy lost amount or recipient target")
		}
		if a.Life() != 20 || b.Life() != 20 {
			t.Fatal("copy changed original game")
		}
		g.ResolveStackObject(original)
		if a.Life() != 20 || b.Life() != want {
			t.Fatal("original lost amount or recipient target")
		}
	}
}

func TestTypedSingularLifeAmountValidationAndCapture(t *testing.T) {
	for _, constructor := range []func(PlayerRef, ResolutionValue) ResolvingEffect{GainLifeFor, LoseLifeFor} {
		for _, amount := range []ResolutionValue{nil, PowerOf(nil), PowerOf(From(BindPermanent("missing"))), PowerOf(DeclarePermanentTarget("undeclared", IsCreature)), ValueAtResolution(nil)} {
			card := NewSorcery("Invalid life", "", WithAbility(Spell(constructor(You, amount))))
			if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "abilities[0].effects[0].amount") {
				t.Fatalf("validation = %v", err)
			}
		}
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		calls := 0
		amount := countedLifeAmount{&calls}
		InspectDefinition(constructor(You, amount))
		if err := ApplyEffect(g, constructor(ControllerOf(This), amount), uuid.New(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if calls != 0 {
			t.Fatal("inspection or absent recipient evaluated amount")
		}
		if err := ApplyEffect(g, constructor(You, amount), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if calls != 1 || a.Life() == 20 || b.Life() != 20 {
			t.Fatal("amount was not captured exactly once")
		}
	}
}

func TestTypedSingularLifeDelayedAmount(t *testing.T) {
	for _, gain := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		source := NewCreature("Amount", "", 3, 4)
		source.SetOwner(a.PlayerID())
		g.PutOnBattlefield(source, a.PlayerID())
		saved := BindPermanent("amount")
		effect := LoseLifeFor(You, PowerOf(From(saved)))
		want := 17
		if gain {
			effect = GainLifeFor(You, PowerOf(From(saved)))
			want = 23
		}
		body := Sequence(Snapshot(This, saved), AtNext(EachEndStep, effect))
		if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		clone := g.Clone()
		clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
		clone.PutTriggersOnStack()
		clone.ResolveStack()
		if clone.GetPlayer(a.PlayerID()).Life() != want || a.Life() != 20 {
			t.Fatal("delayed amount lost its saved frame or clone isolation")
		}
		g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
		g.PutOnBattlefield(source, a.PlayerID())
		g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
		g.PutTriggersOnStack()
		g.ResolveStack()
		if a.Life() != 20 {
			t.Fatal("delayed amount followed a new incarnation")
		}
	}
}
