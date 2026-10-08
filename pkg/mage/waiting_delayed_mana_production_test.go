package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedManaProductionOperands(t *testing.T) {
	for name, body := range map[string]ResolvingEffect{
		"amount":          AddManaAmount(core.Blue, SourceCounters(core.Charge)),
		"arithmetic":      AddManaAmount(core.Blue, AddValues(Fixed(0), PowerOfWithLKI(This))),
		"any-color":       AddManaOfAnyOneColor(SourceCounters(core.Charge)),
		"restricted":      AddRestrictedManaOfAnyOneColor(PowerOfWithLKI(This), core.CreatureSpellsOnly{}),
		"player":          AddManaToPlayer(ControllerOf(This), core.Blue, Fixed(0)),
		"owner":           AddManaToPlayer(OwnerOf(This), core.Blue, Fixed(0)),
		"lki":             AddManaToPlayer(ControllerOfWithLKI(This), core.Blue, Fixed(0)),
		"player-amount":   AddManaToPlayer(You, core.Blue, PowerOfWithLKI(This)),
		"produced-player": AddManaOfProducedTypeToPlayer(ControllerOf(This), BindManaTypes("types"), Fixed(0)),
		"produced-amount": AddManaOfProducedTypeToPlayer(You, BindManaTypes("types"), PowerOfWithLKI(This)),
	} {
		t.Run(name, func(t *testing.T) {
			a := &producedTypeChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: core.Green}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "", 2, 4)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			if err := ApplyEffect(g, AtNextModes(EachEndStep, ModeOf("mana", Sequence(body)), ModeOf("other", AddManaAmount(core.Blue, Fixed(0)))), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			check := func() {
				t.Helper()
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(saved) || a.ManaPool().TotalMana() != 0 || b.ManaPool().TotalMana() != 0 || a.calls != 0 {
					t.Fatal("query lost source or added mana", got)
				}
			}
			check()
			ids := g.PhaseOut(p.ID())
			check()
			g.PhaseIn(ids)
			check()
			g.BouncePermanentToHand(p)
			g.zones.ClearLKI()
			check()
			a.RemoveFromHand(card.ID())
			returned := g.PutOnBattlefield(card, b.PlayerID())
			if g.CaptureDamageSource(returned.ID()).SameObject(saved) {
				t.Fatal("return reused incarnation")
			}
			check()
			clone := g.Clone()
			clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			if len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("shared waiting lifetime")
			}
		})
	}
}

func TestWaitingDelayedManaProductionBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 2, 4), a.PlayerID())
	reads := 0
	if err := ApplyEffect(g, AtNext(EachEndStep, AddManaToPlayer(ControllerOf(This), core.Blue, waitingMultiAmount{&reads})), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 {
			t.Fatal("query evaluated mana amount")
		}
	}
	g.triggers.SetDelayed(nil)
	for _, body := range []ResolvingEffect{
		AddManaAmount(core.Blue, Fixed(1)),
		AddManaToPlayer(ControllerOf(AttachedPermanent), core.Blue, Fixed(1)),
		AddManaOfAnyOneColor(ValueAtResolution(Fixed(1))),
		AtNext(EachEndStep, AddManaToPlayer(ControllerOf(This), core.Blue, Fixed(1))),
	} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("inferred implicit or future source")
	}
	for _, body := range []ResolvingEffect{
		AddManaToPlayer(nil, core.Blue, Fixed(1)), AddManaAmount(core.Blue, nil), AddManaOfAnyOneColor(nil), AddRestrictedManaOfAnyOneColor(Fixed(1), nil), AddManaOfProducedTypeToPlayer(You, BindManaTypes("missing"), Fixed(1)),
		AddManaToPlayer(FromPlayer(BindPlayer("missing")), core.Blue, Fixed(1)),
		AddManaToPlayer(ControllerOf(DeclarePermanentTarget("missing", IsCreature)), core.Blue, Fixed(1)),
	} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid delayed mana accepted")
		}
	}
}

func TestWaitingDelayedManaProductionResolution(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phased", "control"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "", 2, 4)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			body := AddManaToPlayer(ControllerOfWithLKI(This), core.Blue, AddValues(PowerOfWithLKI(This), Fixed(-1)))
			if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			p.AddCounter(core.P1P1, 1)
			switch state {
			case "departed", "returned":
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					returned := g.PutOnBattlefield(card, b.PlayerID())
					returned.AddCounter(core.P1P1, 5)
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if len(g.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("lost mana amount reference")
			}
			clone := g.Clone()
			clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			clone.PutTriggersOnStack()
			clone.ResolveStack()
			wantA, wantB := 2, 0
			if state == "control" {
				wantA, wantB = 0, 2
			}
			if clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != wantA || clone.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != wantB {
				t.Fatal("amount used stale or returned information")
			}
			if a.ManaPool().TotalMana() != 0 || b.ManaPool().TotalMana() != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestWaitingDelayedManaProductionEventTypes(t *testing.T) {
	a := &producedTypeChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: core.Green}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	types := BindManaTypes("event types")
	card := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().WithManaTypes(types),
		AtNext(EachEndStep, AddManaOfProducedTypeToPlayer(ControllerOfWithLKI(This), types, Fixed(2))))))
	card.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	land := g.PutOnBattlefield(NewLand("Land"), b.PlayerID())
	for _, colors := range [][]core.Color{{core.Blue, core.Black}, {core.Red}} {
		g.FireEvent(core.GameEvent{Type: core.EvtTappedForMana, SourceID: land.ID(), SourceIncarnation: land.incarnationID, PlayerID: b.PlayerID(), ManaTypes: colors})
		g.PutTriggersOnStack()
		g.ResolveStack()
		colors[0] = core.White
	}
	g.BouncePermanentToHand(p)
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, b.PlayerID())
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || a.calls != 0 || a.ManaPool().TotalMana() != 0 {
			t.Fatal("query chose a type or lost the departed source")
		}
	}
	clone := g.Clone()
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	pool := clone.GetPlayer(a.PlayerID()).ManaPool()
	if pool.Count(core.Blue) != 2 || pool.Count(core.Red) != 2 || pool.TotalMana() != 4 || clone.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 0 {
		t.Fatalf("delayed types or controller lost: blue=%d black=%d red=%d total=%d other=%d", pool.Count(core.Blue), pool.Count(core.Black), pool.Count(core.Red), pool.TotalMana(), clone.GetPlayer(b.PlayerID()).ManaPool().TotalMana())
	}
	if len(g.WaitingDelayedPermanentSources()) != 1 || a.ManaPool().TotalMana() != 0 || a.calls != 0 {
		t.Fatal("clone changed original event frames")
	}
}

func TestWaitingDelayedManaProductionCapturedTypes(t *testing.T) {
	for _, context := range []string{"step", "modes", "departure"} {
		t.Run(context, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			types := BindManaTypes("types")
			g.resolution.manaTypesBindings = map[*ManaTypesBinding][]core.Color{types: {core.Blue}}
			body := AddManaOfProducedTypeToPlayer(You, types, Fixed(1))
			delay := AtNext(EachEndStep, body)
			switch context {
			case "modes":
				delay = AtNextModes(EachEndStep, ModeOf("mana", body), ModeOf("none", GainLifeFor(You, Fixed(0))))
			case "departure":
				delay = WhenLeavesThisTurn(This, body)
			}
			if err := ApplyEffect(g, delay, p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.resolution.manaTypesBindings[types][0] = core.Red
			clone := g.Clone()
			for _, game := range []*Game{clone, g} {
				if context == "departure" {
					game.BouncePermanentToHand(game.FindPermanent(p.ID()))
				} else {
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				}
				game.PutTriggersOnStack()
				game.ResolveStack()
				if game.GetPlayer(a.PlayerID()).ManaPool().Count(core.Blue) != 1 || game.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 1 {
					t.Fatal("delayed mana types followed a later binding value")
				}
				if game == clone && a.ManaPool().TotalMana() != 0 {
					t.Fatal("clone changed original pool")
				}
			}
		})
	}
}
