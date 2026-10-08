package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedCounterCollectionOperands(t *testing.T) {
	for name, build := range map[string]func(PermanentSet) ResolvingEffect{
		"placement subjects": func(s PermanentSet) ResolvingEffect { return AddCountersEach(s, core.Charge, Fixed(0)) },
		"placement amount": func(s PermanentSet) ResolvingEffect {
			return AddCountersEach(MatchingPermanents(IsCreature), core.Charge, MultiplyValues(Fixed(0), CountPermanents(s)))
		},
	} {
		for shape, set := range map[string]PermanentSet{
			"singleton": SingletonPermanent(This), "auras": AttachedAuras(This),
			"attachments": AttachmentsOn(SingletonPermanent(This)), "blockers": BlockersOf(This),
			"blocked": BlockedAttackersOf(This), "combat": BlockingOrBlockedBy(This),
			"history": BlockedByThisTurn(This), "excluded": WithoutPermanent(MatchingPermanents(IsCreature), This),
			"lowest":     LowestPower(SingletonPermanent(This)),
			"controller": PermanentsControlledBy(ControllerOfWithLKI(This), AnyPermanent),
			"owner":      PermanentsOwnedBy(OwnerOf(This), AnyPermanent),
			"opponents":  PermanentsControlledByPlayers(OpponentsOf(ControllerOfWithLKI(This)), AnyPermanent),
		} {
			t.Run(name+"/"+shape, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewCreature("Source", "", 1, 4)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				original := g.CaptureDamageSource(p.ID())
				body := build(set)
				if err := ValidateCardDefinition("Source", NewArtifact("Source", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))); err != nil {
					t.Fatal(err)
				}
				if err := ApplyEffect(g, AtNext(EachEndStep, Sequence(body)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				returned := g.PutOnBattlefield(card, b.PlayerID())
				g.ChangeObjectColor(returned.ID(), core.Red)
				for range 3 {
					got := g.WaitingDelayedPermanentSources()
					if len(got) != 1 || !got[0].SameObject(original) || got[0].HasColor(core.Red) {
						t.Fatal("collection action lost exact source declaration")
					}
					got[0].card.AddType(core.TypeEnchantment)
					if g.WaitingDelayedPermanentSources()[0].HasType(core.TypeEnchantment) {
						t.Fatal("query shares mutable facts")
					}
				}
				clone := g.Clone()
				clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				if len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
					t.Fatal("clone shares delayed lifetime")
				}
			})
		}
	}
}

func TestWaitingDelayedCounterCollectionTotal(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 1, 4)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	p.AddCounter(core.Charge, 3)
	saved := g.CaptureDamageSource(p.ID())
	reads := 0
	filter := NewPermanentFilter("must not run in query", func(*Permanent, *Game) bool { reads++; return true })
	set := FilterPermanentSet(UnionPermanents(SingletonPermanent(This), SingletonPermanent(This)), filter)
	body := GainLifeFor(You, CountCountersAmong(set, core.Charge))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		refs := g.WaitingDelayedPermanentSources()
		if len(refs) != 1 || !refs[0].SameObject(saved) || reads != 0 || a.Life() != 20 {
			t.Fatal("counter total query evaluated collection or lost source")
		}
	}
	clone := g.Clone()
	clone.MutablePermanent(p.ID()).Counters[core.Charge] = 5
	for _, game := range []*Game{clone, g} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		if len(game.WaitingDelayedPermanentSources()) != 0 {
			t.Fatal("detected total still waits")
		}
		game.PutTriggersOnStack()
		game.ResolveStack()
	}
	if a.Life() != 23 || clone.GetPlayer(a.PlayerID()).Life() != 25 {
		t.Fatal("duplicate total or clone shared counters")
	}
	for _, value := range []ResolutionValue{CountCountersAmong(nil, core.Charge), CountCountersAmong(set, core.NumCounters), CountCountersAmong(SingletonPermanent(nil), core.Charge)} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), GainLifeFor(You, value))))) == nil {
			t.Fatal("invalid counter total accepted")
		}
	}
}

func TestCounterCollectionCurrentMembers(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("First", "", 1, 4)
	c.SetOwner(a.PlayerID())
	old := g.PutOnBattlefield(c, a.PlayerID())
	old.AddCounter(core.Charge, 7)
	g.BouncePermanentToHand(old)
	a.RemoveFromHand(c.ID())
	current := g.PutOnBattlefield(c, a.PlayerID())
	current.AddCounter(core.Charge, 3)
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 4), a.PlayerID())
	other.AddCounter(core.Charge, 4)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	value := CountCountersAmong(fixedTapSet{[]*Permanent{nil, old, current, current, other}}, core.Charge)
	if value.resolveValue(ctx) != 7 {
		t.Fatal("total includes stale or duplicate member")
	}
	g.PhaseOut(other.ID())
	if value.resolveValue(ctx) != 3 {
		t.Fatal("total includes phased member")
	}
	g.PhaseIn(g.PhaseOut(current.ID()))
	if value.resolveValue(ctx) != 3 {
		t.Fatal("phase cycle lost current counters")
	}
}

func TestWaitingDelayedCounterCollectionPurity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 4), a.PlayerID())
	reads := 0
	filter := NewPermanentFilter("query must not run", func(*Permanent, *Game) bool { reads++; return true })
	set := FilterPermanentSet(SingletonPermanent(This), filter)
	body := AddCountersEach(set, core.Charge, MultiplyValues(Fixed(0), CountCountersAmong(set, core.Charge)))
	if err := ApplyEffect(g, AtNextModes(EachEndStep, ModeOf("First", body), ModeOf("Second", body)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || p.Counters[core.Charge] != 0 {
		t.Fatal("query ran collection or lost zero/mode read")
	}
	g.triggers.SetDelayed(nil)
	for _, set := range []PermanentSet{SingletonPermanent(AttachedPermanent), MatchingPermanents(filter), waitingMultiEmptySet{&reads}} {
		if err := ApplyEffect(g, AtNext(EachEndStep, AddCountersEach(set, core.Charge, CountCountersAmong(set, core.Charge))), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 || reads != 0 {
		t.Fatal("query inferred opaque or implicit source")
	}
	if err := ApplyEffect(g, AtNext(EachEndStep, AtNext(EachEndStep, body)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("future body supplied source before creation")
	}
}
