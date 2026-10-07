package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTapNMatchingIncludingSourceCost(t *testing.T) {
	for _, count := range []int{1, 2} {
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		source := NewCreature("Source", "{W}", 2, 4)
		p := g.PutOnBattlefield(source, a.PlayerID())
		other := g.PutOnBattlefield(NewCreature("Other", "{W}", 1, 1), a.PlayerID())
		cost := TapNMatchingIncludingSourceCost(count, IsCreature, "Tap creatures")
		if !cost.CanPay(source.ID(), a.PlayerID(), g) || p.Tapped || other.Tapped {
			t.Fatal("pure affordability")
		}
		tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{cost}})
		if err != nil {
			t.Fatal(err)
		}
		if p.Tapped || other.Tapped {
			t.Fatal("preparation changed live objects")
		}
		clone := g.Clone()
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
		if !g.FindPermanent(p.ID()).Tapped || (count == 2 && !g.FindPermanent(other.ID()).Tapped) {
			t.Fatal("source-inclusive payment")
		}
		if clone.FindPermanent(p.ID()).Tapped || clone.FindPermanent(other.ID()).Tapped {
			t.Fatal("clone changed")
		}
	}
}

func TestTapNMatchingIncludingSourceStale(t *testing.T) {
	for _, change := range []string{"tap", "phase", "return", "control", "color"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Source", "{W}", 2, 4)
			g.PutOnBattlefield(source, a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "{W}", 1, 1), a.PlayerID())
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{LifePayCost(2), TapNMatchingIncludingSourceCost(2, And(IsCreature, HasColorFilter(core.White)), "Tap creatures")}})
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "tap":
				g.TapPermanent(g.FindPermanent(source.ID()))
			case "phase":
				g.PhaseOut(source.ID())
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(source, a.PlayerID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "color":
				colors := []core.Color{core.Blue}
				g.MutablePermanent(source.ID()).ColorOverride = &colors
			}
			if err := tx.Commit(); err == nil {
				t.Fatal("stale choice accepted")
			}
			if a.Life() != 20 || g.FindPermanent(other.ID()).Tapped {
				t.Fatal("failed commit spent resources")
			}
		})
	}
}

func TestTapNMatchingIncludingSourceSimultaneous(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	source := NewCreature("Source", "", 2, 4)
	g.PutOnBattlefield(source, a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 1), a.PlayerID())
	events, incomplete := 0, false
	watcher := NewEnchantment("Watcher", "", WithAbility(NewTriggered(core.EvtTapped, false, GainLifeFor(You, Fixed(1))).SetCondition(func(_ *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		events++
		incomplete = incomplete || !reader.FindPermanent(source.ID()).Tapped || !reader.FindPermanent(other.ID()).Tapped
		return false
	})))
	g.PutOnBattlefield(watcher, a.PlayerID())
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{TapNMatchingIncludingSourceCost(2, IsCreature, "Tap creatures")}})
	if err != nil {
		t.Fatal(err)
	}
	events, incomplete = 0, false
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if events != 2 || incomplete {
		t.Fatalf("events=%d incomplete=%v", events, incomplete)
	}
}

func TestTapNMatchingIncludingSourceDefinition(t *testing.T) {
	for _, count := range []int{-1, 0, 1} {
		card := NewArtifact("Test", "", WithAbility(Activate(Costs(TapNMatchingIncludingSourceCost(count, IsCreature, "Tap creatures")), GainLifeFor(You, Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if (err == nil) != (count > 0) {
			t.Fatalf("count=%d error=%v", count, err)
		}
	}
}

type repeatedTapChoicePlayer struct {
	*BasePlayer
	choice *Permanent
	calls  int
}

func (p *repeatedTapChoicePlayer) ChoosePermanent(_ []*Permanent, _ string, _ GameReader) *Permanent {
	p.calls++
	return p.choice
}

func TestTapNMatchingIncludingSourceDistinctAndRollback(t *testing.T) {
	a := &repeatedTapChoicePlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	source := NewCreature("Source", "", 1, 1)
	a.choice = g.PutOnBattlefield(source, a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 1), a.PlayerID())
	cost := TapNMatchingIncludingSourceCost(2, IsCreature, "Tap creatures")
	if !cost.CanPay(source.ID(), a.PlayerID(), g) || a.calls != 0 {
		t.Fatal("query made choices")
	}
	if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{LifePayCost(2), cost}}); err == nil {
		t.Fatal("duplicate payment accepted")
	}
	if a.calls != 2 || a.Life() != 20 || g.FindPermanent(source.ID()).Tapped || g.FindPermanent(other.ID()).Tapped {
		t.Fatal("failed payment changed live resources")
	}
}

func TestTapNMatchingIncludingSourceSeparateSourceTap(t *testing.T) {
	for _, first := range []bool{true, false} {
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		source := NewCreature("Source", "", 1, 1, WithKeyword(core.Haste))
		g.PutOnBattlefield(source, a.PlayerID())
		other := g.PutOnBattlefield(NewCreature("Other", "", 1, 1), a.PlayerID())
		matching := TapNMatchingIncludingSourceCost(2, IsCreature, "Tap creatures")
		costs := []Cost{Tap(), matching}
		if !first {
			costs = []Cost{matching, Tap()}
		}
		if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: costs}); err == nil {
			t.Fatal("source paid two taps")
		}
		if g.FindPermanent(source.ID()).Tapped || g.FindPermanent(other.ID()).Tapped {
			t.Fatal("failed payment tapped live objects")
		}
	}
}
