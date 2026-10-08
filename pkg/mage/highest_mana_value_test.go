package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestHighestManaValueSelection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	low := g.PutOnBattlefield(NewCreature("Low", "{X}{1}", 1, 1), a.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "{3}", 1, 1), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "{2}{R}", 1, 1), b.PlayerID())
	faceDown := g.PutOnBattlefield(NewCreature("Hidden", "{9}", 1, 1), a.PlayerID())
	g.MutablePermanent(faceDown.ID()).FaceDown = true
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	set := HighestManaValue(MatchingPermanents(IsCreature))
	check := func(want ...*Permanent) {
		t.Helper()
		got := set.resolvePermanents(ctx)
		if len(got) != len(want) {
			t.Fatalf("members=%d want=%d", len(got), len(want))
		}
		for i, p := range got {
			if p.ID() != want[i].ID() {
				t.Fatal("maximum ties lost battlefield order")
			}
		}
	}
	check(first, second)
	g.MutablePermanent(low.ID()).AddCounter(core.P1P1, 4)
	check(first, second)
	g.PhaseOut(first.ID())
	check(second)
	g.PhaseOut(second.ID())
	check(low)
	g.PhaseOut(low.ID())
	check(faceDown)
	g.PhaseOut(faceDown.ID())
	check()
	g.PhaseIn([]uuid.UUID{first.ID(), second.ID()})
	check(first, second)
	clone := g.Clone()
	clone.PhaseOut(first.ID())
	got := set.resolvePermanents(&EffectContext{Game: clone, Controller: a.PlayerID()})
	if len(got) != 1 || got[0].ID() != second.ID() {
		t.Fatal("clone did not read its own membership")
	}
	check(first, second)
}

func TestHighestManaValueDeclaration(t *testing.T) {
	for _, set := range []PermanentSet{nil, DeclarePermanentTargets("undeclared", 1, 1, IsCreature)} {
		card := NewInstant("Invalid", "", WithAbility(Spell(TapEach(HighestManaValue(set)))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid subject declaration accepted")
		}
	}
	target := DeclarePermanentTargets("declared", 1, 2, IsCreature)
	card := NewInstant("Valid", "", WithAbility(Spell(TapEach(HighestManaValue(target)), WithTargets(target))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestHighestManaValueChoiceIdentity(t *testing.T) {
	for _, change := range []string{"phase", "return", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Chosen", "{X}{2}", 1, 1)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			binding := BindPermanent("choice")
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			choice := ChoosePermanent(You, HighestManaValue(MatchingPermanents(IsCreature)), binding)
			if err := choice.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			switch change {
			case "phase":
				g.PhaseOut(p.ID())
			case "return":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if err := TapSubject(From(binding)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if (change == "control") != g.zones.FindPermanentIncludingPhased(p.ID()).Tapped {
				t.Fatal("saved choice followed a new object or lost control-independent identity")
			}
			cloneCtx := &EffectContext{Game: clone, Controller: a.PlayerID()}
			if err := TapSubject(From(binding)).Apply(cloneCtx); err != nil || !clone.FindPermanent(p.ID()).Tapped {
				t.Fatal("clone lost its saved choice", err)
			}
			if change == "phase" {
				g.PhaseIn([]uuid.UUID{p.ID()})
				if err := TapSubject(From(binding)).Apply(ctx); err != nil || !g.FindPermanent(p.ID()).Tapped {
					t.Fatal("phase cycle lost the choice", err)
				}
			}
			if err := ChoosePermanent(You, MatchingPermanents(IsLand), binding).Apply(cloneCtx); err != nil {
				t.Fatal(err)
			}
			if _, ok := From(binding).resolvePermanent(cloneCtx); ok {
				t.Fatal("empty choice kept an earlier result")
			}
		})
	}
}

func TestHighestManaValueWaitingSource(t *testing.T) {
	for _, mode := range []string{"collection", "choice subjects", "choice player"} {
		t.Run(mode, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "{X}{2}", 1, 1)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			original := g.CaptureDamageSource(p.ID())
			reads := 0
			filter := NewPermanentFilter("query must not run", func(*Permanent, *Game) bool { reads++; return true })
			var body = TapEach(HighestManaValue(FilterPermanentSet(SingletonPermanent(This), filter)))
			if mode == "choice subjects" {
				body = ChoosePermanent(You, HighestManaValue(FilterPermanentSet(SingletonPermanent(This), filter)), BindPermanent("choice"))
			}
			if mode == "choice player" {
				body = ChoosePermanent(ControllerOfWithLKI(This), HighestManaValue(MatchingPermanents(filter)), BindPermanent("choice"))
			}
			if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.BouncePermanentToHand(p)
			a.RemoveFromHand(card.ID())
			returned := g.PutOnBattlefield(card, b.PlayerID())
			for range 2 {
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(original) || reads != 0 || returned.Tapped {
					t.Fatal("waiting query evaluated selection or lost its exact source")
				}
			}
			clone := g.Clone()
			clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			clone.PutTriggersOnStack()
			clone.ResolveStack()
			if clone.FindPermanent(returned.ID()).Tapped || len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("delayed body followed a returned source or shared clone lifetime")
			}
		})
	}
}
