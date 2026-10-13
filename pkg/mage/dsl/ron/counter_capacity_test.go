package ron

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCounterCapacity(t *testing.T) {
	for _, count := range []int{255, 256, 300, 65536} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", fmt.Sprintf(`AsPermanentEnters(ThisPermanent,[EntersWithNumberCounters(Integer(%d),ChargeCounter)])`, count))
			g, a, _ := ronGame()
			c := factory()
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(c.ID()) != nil {
				t.Fatal("entry before resolution")
			}
			g.ResolveStack()
			p := g.FindPermanent(c.ID())
			if p == nil || p.Counters[core.Charge] != count {
				t.Fatalf("entry count: %v", p)
			}
			clone := g.Clone()
			g.MutablePermanent(p.ID()).RemoveCounter(core.Charge, count-1)
			if clone.FindPermanent(p.ID()).Counters[core.Charge] != count {
				t.Fatal("clone shared counters")
			}
			q := g.PutOnBattlefield(factory(), a.PlayerID())
			if q.Counters[core.Charge] != count {
				t.Fatal("factory shared counters")
			}
		})
	}
}

func TestRONCounterCapacityTriskelion(t *testing.T) {
	g, a, b := ronGame()
	donor := g.PutOnBattlefield(mage.NewArtifact("Doubler", ""), a.PlayerID())
	for range 7 {
		g.AddCounterDoubler(donor.ID(), core.P1P1, mage.IsCreature)
	}
	c := repositoryCard(t, "ATQ", "Triskelion")
	p := g.PutOnBattlefield(c, a.PlayerID())
	if p.Counters[core.P1P1] != 384 || p.CurrentPower(g) != 385 {
		t.Fatal("replacement-expanded entry truncated")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{uuid.New()}); err == nil {
		t.Fatal("illegal target accepted")
	}
	if g.FindPermanent(p.ID()).Counters[core.P1P1] != 384 || g.StackSize() != 0 {
		t.Fatal("illegal announcement spent counters")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(p.ID()).Counters[core.P1P1] != 383 || b.Life() != 19 {
		t.Fatal("large counter payment or damage failed")
	}
}

func TestRONCounterCapacityReplacementOrder(t *testing.T) {
	for _, choice := range []int{0, 1} {
		t.Run(fmt.Sprint(choice), func(t *testing.T) {
			a := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("A"), index: 1 - choice}
			b := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("B"), index: choice}
			g := mage.NewGame(a, b)
			donor := g.PutOnBattlefield(mage.NewArtifact("Donor", ""), a.PlayerID())
			g.AddETBAdditionalCounters(donor.ID(), core.P1P1, 300, mage.IsCreature, false)
			g.AddCounterDoubler(donor.ID(), core.P1P1, mage.IsCreature)
			p := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Triskelion"), b.PlayerID())
			want := 606
			if choice == 1 {
				want = 306
			}
			if p.Counters[core.P1P1] != want || a.choices != 0 || b.choices != 1 {
				t.Fatalf("counters=%d choices=%d/%d, want %d", p.Counters[core.P1P1], a.choices, b.choices, want)
			}
		})
	}
}
