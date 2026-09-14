package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentBindingLifetime(t *testing.T) {
	for _, change := range []string{"source leaves", "host returns", "phasing", "clone"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source, host := NewCreature("Source", "", 1, 1), NewCreature("Host", "", 1, 1)
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutOnBattlefield(host, a.PlayerID())
			g.FindPermanent(source.ID()).AttachedTo = host.ID()
			saved := BindPermanent("host")
			ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}
			if err := Snapshot(AttachedPermanent, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "source leaves":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "host returns":
				g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
				g.PutOnBattlefield(host, a.PlayerID())
			case "phasing":
				ids := g.PhaseOut(host.ID())
				g.PhaseIn(ids)
			case "clone":
				g = g.Clone()
				ctx.Game = g
			}
			if err := Sacrifice(From(saved)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if exists := g.FindPermanent(host.ID()) != nil; exists != (change == "host returns") {
				t.Fatalf("host exists = %v", exists)
			}
		})
	}
}

func TestPermanentBindingValidation(t *testing.T) {
	saved, sameName := BindPermanent("host"), BindPermanent("host")
	tests := []struct {
		name   string
		effect Effect
		want   string
	}{
		{"valid", Sequence(Snapshot(This, saved), Sacrifice(From(saved))), ""},
		{"missing", Sacrifice(From(saved)), "producer"},
		{"before write", Sequence(Sacrifice(From(saved)), Snapshot(This, saved)), "producer"},
		{"branch escape", Sequence(IfState(Controls(IsCreature), Snapshot(This, saved)), Sacrifice(From(saved))), "producer"},
		{"inside branch", IfState(Controls(IsCreature), Sequence(Snapshot(This, saved), Sacrifice(From(saved)))), ""},
		{"duplicate", Sequence(Snapshot(This, saved), Snapshot(This, sameName)), "duplicate"},
		{"delayed escape", Sequence(Snapshot(This, saved), RegisterDelayedTriggerStep(EvtTapped, "", Sacrifice(From(saved)))), "producer"},
		{"empty label", Snapshot(This, BindPermanent("")), "name"},
		{"nil binding", Snapshot(This, nil), "non-nil"},
		{"loop escape", Sequence(ForEachPermanent(IsCreature, Snapshot(This, saved), "save"), Sacrifice(From(saved))), "producer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := NewCreature("Binding", "", 1, 1, WithActivatedAbility(tt.effect, ManaCostOf("{0}")))
			err := ValidateCardDefinition("Binding", card)
			if tt.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), "Binding: abilities[0]") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestPermanentBindingFrames(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Source", "", 1, 1)
	g.PutOnBattlefield(c, a.PlayerID())
	saved := BindPermanent("saved")
	ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID()}
	if err := Snapshot(This, saved).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	delete(clone.resolution.permanentBindings, saved)
	if _, ok := From(saved).resolvePermanent(ctx); !ok {
		t.Fatal("clone changed original frame")
	}
	before := g.FindPermanent(c.ID())
	if err := ApplyEffect(g, Sacrifice(From(saved)), c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(c.ID()) != before {
		t.Fatal("binding leaked into another resolution")
	}
}

func TestPermanentBindingNestedResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Source", "", 1, 1)
	g.PutOnBattlefield(c, a.PlayerID())
	saved := BindPermanent("host")
	effect := Sequence(Snapshot(This, saved), FuncEffect("nested resolution", EffectProperties{}, func(g *Game, source, controller uuid.UUID, _ []uuid.UUID) error {
		if err := ApplyEffect(g, Sacrifice(From(saved)), source, controller, nil); err != nil {
			return err
		}
		if g.FindPermanent(source) == nil {
			t.Fatal("nested resolution read the outer frame")
		}
		return nil
	}), Sacrifice(From(saved)))
	if err := ApplyEffect(g, effect, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(c.ID()) != nil {
		t.Fatal("nested resolution lost the outer frame")
	}
	if len(g.resolution.permanentBindings) != 0 {
		t.Fatal("resolution retained its frame")
	}
}

func TestPermanentBindingAbsentSnapshot(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Source", "", 1, 1)
	g.PutOnBattlefield(c, a.PlayerID())
	saved := BindPermanent("host")
	effect := Sequence(Snapshot(This, saved), Snapshot(AttachedPermanent, saved), Sacrifice(From(saved)), GainLife(2))
	if err := ApplyEffect(g, effect, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(c.ID()) == nil {
		t.Fatal("absent snapshot retained its previous subject")
	}
	if a.Life() != 22 {
		t.Fatal("absent subject stopped the sequence")
	}
}

func TestPermanentBindingSnapshotReadsBeforeWrite(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Source", "", 1, 1)
	g.PutOnBattlefield(c, a.PlayerID())
	saved := BindPermanent("host")
	effect := Sequence(Snapshot(This, saved), Snapshot(From(saved), saved), Sacrifice(From(saved)))
	if err := ApplyEffect(g, effect, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(c.ID()) != nil {
		t.Fatal("snapshot erased its input before reading it")
	}
}
