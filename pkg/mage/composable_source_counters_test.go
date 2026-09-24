package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceCountersLifetime(t *testing.T) {
	for _, change := range []string{"current", "leave", "sacrifice", "return", "return twice", "cleanup", "phase", "phase cycle", "control", "clone before", "clone after"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewEnchantment("Counter mana", "", WithAbility(At(YourFirstMainPhase, AddManaAmount(Black, SourceCounters(Charge)))))
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.MutablePermanent(source.ID()).AddCounter(Charge, 2)
			g.MutablePermanent(source.ID()).AddCounter(Tide, 9)
			g.FireEvent(GameEvent{Type: EvtMainPhase, PlayerID: a.PlayerID(), Flag: true})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || a.ManaPool().Count(Black) != 0 {
				t.Fatal("first-main trigger must use the stack")
			}
			g.MutablePermanent(source.ID()).AddCounter(Charge, 3)
			original := g
			if change == "clone before" {
				g = g.Clone()
				g.MutablePermanent(source.ID()).AddCounter(Charge, 1)
			}
			switch change {
			case "sacrifice":
				g.DoSacrifice(g.FindPermanent(source.ID()))
			case "leave", "return", "return twice", "cleanup", "clone before", "clone after":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "return" || change == "return twice" || change == "clone after" {
					if change == "clone after" {
						g = g.Clone()
					}
					g.PutOnBattlefield(source, a.PlayerID())
					g.MutablePermanent(source.ID()).AddCounter(Charge, 17)
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					if change == "return" {
						g.PutOnBattlefield(source, a.PlayerID())
					}
				}
				if change == "cleanup" {
					g.zones.ClearEndOfTurn()
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
				g.MutablePermanent(source.ID()).AddCounter(Charge, 1)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
				g.ApplyContinuousEffects()
			}
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy missing")
			}
			g.ResolveStack()
			want := 5
			if change == "clone before" || change == "phase cycle" {
				want = 6
			}
			for _, player := range []*BasePlayer{a, b} {
				if got := g.GetPlayer(player.PlayerID()).ManaPool().Count(Black); got != want {
					t.Fatalf("mana = %d, want %d", got, want)
				}
			}
			if g != original {
				if a.ManaPool().Count(Black) != 0 || b.ManaPool().Count(Black) != 0 {
					t.Fatal("clone changed original mana")
				}
				original.ResolveStack()
				if a.ManaPool().Count(Black) != 5 {
					t.Fatal("clone changed original counter history")
				}
			}
		})
	}
}

func TestSourceCountersDelayedResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	body := AtNext(EachEndStep, GainLifeFor(You, SourceCounters(Charge)))
	source := NewCreature("Delayed counters", "", 2, 2, WithAbility(Activate(Costs(), body)))
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Charge, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.MutablePermanent(source.ID()).AddCounter(Charge, 3)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.zones.ClearEndOfTurn()
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Charge, 11)
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 25 {
		t.Fatalf("delayed gain = %d, want 5", a.Life()-20)
	}
}

func TestSourceCountersValidation(t *testing.T) {
	value := SourceCounters(Charge)
	card := NewSorcery("Invalid source", "", WithAbility(Spell(GainLifeFor(You, value))))
	if ValidateCardDefinition(card.Name(), card) == nil {
		t.Fatal("spell accepted battlefield source value")
	}
	for _, kind := range []CounterType{-1, NumCounters} {
		t.Run(kind.String(), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid counter accepted")
				}
			}()
			SourceCounters(kind)
		})
	}
	info := InspectDefinition(value)
	if info.Role != "source-counter-value" || info.Text != "Charge counters on source, with last-known information" {
		t.Fatalf("metadata = %+v", info)
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if got := value.resolveValue(&EffectContext{Game: g}); got != 0 {
		t.Fatalf("missing source count = %d", got)
	}
}

func TestSourceCountersDeathAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	body := Sequence(GainLifeFor(You, SourceCounters(Charge)), GainLifeFor(You, SourceCounters(Tide)))
	source := NewCreature("Death counters", "", 2, 2, WithAbility(WhenEvent(Dies(This), body)))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Charge, 2)
	g.DoSacrifice(g.FindPermanent(source.ID()))
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("death life = %d, want 22", a.Life())
	}
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Charge, 3)
	clone := g.Clone()
	clone.MutablePermanent(source.ID()).AddCounter(Charge, 4)
	clone.DoSacrifice(clone.FindPermanent(source.ID()))
	g.DoSacrifice(g.FindPermanent(source.ID()))
	for _, game := range []*Game{g, clone} {
		game.PutTriggersOnStack()
		game.ResolveStack()
	}
	if a.Life() != 25 || clone.GetPlayer(a.PlayerID()).Life() != 29 {
		t.Fatal("games shared counter history for the same source incarnation")
	}
	if got := SourceCounters(Charge).resolveValue(&EffectContext{Game: g, SourceID: source.ID()}); got != 0 {
		t.Fatalf("uncaptured source read history: %d", got)
	}
}
