package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceCounterManaQueryAndActivation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ability := NewSourceCounterManaAbility(core.Colorless, core.Storage)
	card := NewLand("Source", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	check := func(game *Game, want int) {
		t.Helper()
		got := ManaProductionsForAbilityInGame(ability, game, p.ID())
		if want == 0 {
			if len(got) != 0 {
				t.Fatalf("production = %+v", got)
			}
			return
		}
		if len(got) != 1 || got[0].Amount != want || got[0].Color != core.Colorless {
			t.Fatalf("production = %+v", got)
		}
		got[0].Amount = 99
	}
	check(g, 0)
	if !IsActivatedManaAbility(ability) {
		t.Fatal("zero-count ability lost mana classification")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if !p.Tapped || a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 {
		t.Fatal("zero-count activation")
	}
	p.Tapped = false
	p.Counters[core.Storage] = 3
	p.Counters[core.Charge] = 8
	check(g, 3)
	clone := g.Clone()
	g.MutablePermanent(p.ID()).Counters[core.Storage] = 5
	check(g, 5)
	check(clone, 3)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(core.Colorless) != 5 || g.FindPermanent(p.ID()).Counters[core.Storage] != 5 || g.StackSize() != 0 {
		t.Fatal("live count not produced immediately")
	}
	if clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 || clone.FindPermanent(p.ID()).Tapped {
		t.Fatal("clone changed")
	}
	check(g, 5)
	g.PhaseOut(p.ID())
	check(g, 0)
	g.PhaseIn([]uuid.UUID{p.ID()})
	check(g, 5)
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	check(g, 0)
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, b.PlayerID())
	if returned.Counters[core.Storage] != 0 {
		t.Fatal("return retained counters")
	}
	returned.Counters[core.Storage] = 2
	if err := g.ActivateAbilityByIndex(b.PlayerID(), returned.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Colorless) != 2 {
		t.Fatal("wrong controller or returned count")
	}
	if len(ManaProductionsForAbility(ability)) != 0 {
		t.Fatal("context-free count")
	}
}

func TestSourceCounterManaDefinition(t *testing.T) {
	for _, tc := range []struct {
		color   core.Color
		counter core.CounterType
	}{
		{core.AnyColor, core.Storage}, {core.Color(-1), core.Storage}, {core.Color(99), core.Storage},
		{core.Colorless, core.CounterType(-1)}, {core.Colorless, core.NumCounters},
	} {
		card := NewLand("Invalid", WithAbility(NewSourceCounterManaAbility(tc.color, tc.counter)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid definition accepted", tc)
		}
	}
}

func TestSourceCounterManaPaymentAndEvents(t *testing.T) {
	for _, path := range []string{"manual", "tap", "automatic", "zero"} {
		t.Run(path, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewLand("Source", WithAbility(NewSourceCounterManaAbility(core.White, core.Charge))), a.PlayerID())
			p.Counters[core.Charge] = 3
			if path == "zero" {
				p.Counters[core.Charge] = 0
			}
			g.PutOnBattlefield(NewEnchantment("Bonus", "", WithAbility(NewManaFlareAbility(IsLand))), a.PlayerID())
			if err := ApplyEffect(g, ReplaceTappedManaTypeUntilLeaves(This, core.White, core.Blue), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			var observed []core.GameEvent
			g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(NewTriggered(core.EvtTappedForMana, false, GainLifeFor(You, Fixed(1))).SetCondition(func(evt *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
				if reader == g {
					observed = append(observed, *evt)
				}
				return false
			}))), b.PlayerID())
			for range 3 {
				if g.CanAfford(a.PlayerID(), core.ManaCost{Blue: 4}, nil) != (path != "zero") || g.CanAfford(a.PlayerID(), core.ManaCost{Blue: 5}, nil) {
					t.Fatal("wrong solver quantity")
				}
			}
			if p.Tapped || a.ManaPool().TotalMana() != 0 || len(observed) != 0 {
				t.Fatal("query mutated state")
			}
			var err error
			switch path {
			case "tap":
				err = g.TapForMana(a.PlayerID(), p.ID())
			case "automatic":
				err = g.AutoTapForCost(a.PlayerID(), core.ManaCost{Blue: 4})
			default:
				err = g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil)
			}
			if err != nil {
				t.Fatal(err)
			}
			if path == "zero" {
				if a.ManaPool().TotalMana() != 0 || len(observed) != 0 {
					t.Fatal("zero production emitted mana or event")
				}
				return
			}
			if a.ManaPool().Count(core.Blue) != 4 || len(observed) != 1 || observed[0].Amount != 3 || len(observed[0].ManaTypes) != 1 || observed[0].ManaTypes[0] != core.Blue || observed[0].SourceIncarnation != p.incarnationID {
				t.Fatalf("production or event = %+v, pool=%d", observed, a.ManaPool().TotalMana())
			}
		})
	}
}
