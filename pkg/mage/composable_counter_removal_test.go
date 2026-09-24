package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTypedCounterRemovalAmounts(t *testing.T) {
	for _, amount := range []int{-2, 0, 2, 5, 6, 300} {
		t.Run(Fixed(amount).Text(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Source", "", 2, 3)
			g.PutOnBattlefield(source, a.PlayerID())
			g.MutablePermanent(source.ID()).AddCounter(core.Charge, 5)
			g.MutablePermanent(source.ID()).AddCounter(core.Tide, 2)
			body := Sequence(RemoveCountersFrom(This, core.Charge, Fixed(amount)), GainLifeFor(You, Fixed(1)))
			if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			p := g.FindPermanent(source.ID())
			if got, want := int(p.Counters[core.Charge]), 5-min(max(amount, 0), 5); got != want {
				t.Fatalf("counters = %d, want %d", got, want)
			}
			if p.Counters[core.Tide] != 2 || a.Life() != 21 {
				t.Fatal("other counters or following instruction changed")
			}
		})
	}
}

func TestTypedCounterRemovalContextAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 2, 3)
	outer := NewCreature("Outer target", "{3}", 1, 1)
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(outer, b.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(core.Charge, 5)
	clone := g.Clone()
	saved := BindPermanent("subject")
	body := Sequence(Snapshot(This, saved), RemoveCountersFrom(From(saved), core.Charge, ValueAtResolution(TargetPermanentManaValue())))
	if err := ApplyEffect(clone, body, source.ID(), a.PlayerID(), []uuid.UUID{outer.ID()}); err != nil {
		t.Fatal(err)
	}
	if clone.FindPermanent(source.ID()).Counters[core.Charge] != 2 || g.FindPermanent(source.ID()).Counters[core.Charge] != 5 {
		t.Fatal("outer amount context or clone isolation changed")
	}
}

func TestTypedCounterRemovalIdentity(t *testing.T) {
	for _, scenario := range []string{"present", "absent", "returned", "phased", "phase cycle"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Source", "", 2, 3)
			g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(c.ID()).AddCounter(core.Charge, 5)
			saved := BindPermanent("subject")
			ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "absent", "returned":
				g.ExilePermanent(g.FindPermanent(c.ID()))
				if scenario == "returned" {
					g.RemoveFromExile(c.ID())
					g.PutOnBattlefield(c, a.PlayerID())
					g.MutablePermanent(c.ID()).AddCounter(core.Charge, 5)
				}
			case "phased":
				g.PhaseOut(c.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(c.ID()))
			}
			calls := 0
			if err := RemoveCountersFrom(From(saved), core.Charge, countedLifeAmount{&calls}).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			wantCalls := 0
			if scenario == "present" || scenario == "phase cycle" {
				wantCalls = 1
			}
			if calls != wantCalls {
				t.Fatalf("amount evaluations = %d, want %d", calls, wantCalls)
			}
			if p := g.FindPermanentIncludingPhased(c.ID()); p != nil && wantCalls == 0 && p.Counters[core.Charge] != 5 {
				t.Fatal("changed unavailable subject")
			}
		})
	}
}

func TestTypedCounterRemovalValidation(t *testing.T) {
	for _, body := range []ResolvingEffect{
		RemoveCountersFrom(nil, core.Charge, Fixed(1)),
		RemoveCountersFrom(From(BindPermanent("missing")), core.Charge, Fixed(1)),
		RemoveCountersFrom(DeclarePermanentTarget("missing", IsCreature), core.Charge, Fixed(1)),
		RemoveCountersFrom(This, core.Charge, nil),
		RemoveCountersFrom(This, core.Charge, PowerOf(From(BindPermanent("missing")))),
	} {
		c := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), body)))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid removal accepted")
		}
	}
	calls := 0
	info := InspectDefinition(RemoveCountersFrom(This, core.Charge, countedLifeAmount{&calls}))
	if calls != 0 || !reflect.DeepEqual(info.Properties, EffectProperties{}) {
		t.Fatal("inspection evaluated amount or changed legacy metadata")
	}
}

func TestTypedCounterRemovalCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	recipient := NewCreature("Recipient", "", 2, 8)
	g.PutOnBattlefield(recipient, a.PlayerID())
	g.MutablePermanent(recipient.ID()).AddCounter(core.Charge, 5)
	target := DeclarePermanentTarget("recipient", IsCreature)
	ability := Spell(RemoveCountersFrom(target, core.Charge, PowerOf(target)), WithTargets(target))
	card := NewSorcery("Counters", "", WithAbility(ability))
	original := &StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{recipient.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(original)
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	g.ResolveStackObject(copied)
	if got := g.FindPermanent(recipient.ID()).Counters[core.Charge]; got != 3 {
		t.Fatalf("copy counters = %d, want 3", got)
	}
	g.ExilePermanent(g.FindPermanent(recipient.ID()))
	g.RemoveFromExile(recipient.ID())
	g.PutOnBattlefield(recipient, a.PlayerID())
	g.MutablePermanent(recipient.ID()).AddCounter(core.Charge, 5)
	g.ResolveStackObject(original)
	if got := g.FindPermanent(recipient.ID()).Counters[core.Charge]; got != 5 {
		t.Fatalf("returned target counters = %d, want 5", got)
	}
}
