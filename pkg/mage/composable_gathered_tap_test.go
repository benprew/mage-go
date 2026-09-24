package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestComposableOpponentChosenTap(t *testing.T) {
	for _, many := range []bool{false, true} {
		var binding TargetBinding
		if many {
			binding = DeclarePermanentTargets("opponent", 1, 2, IsCreature, TargetOpponentControls(), TargetChosenByOpponent())
		} else {
			binding = DeclarePermanentTarget("opponent", IsCreature, TargetOpponentControls(), TargetChosenByOpponent())
		}
		spec := binding.targetSpec()
		if !isOpponentChosenTarget(spec) {
			t.Fatal("typed declaration lost the opponent chooser")
		}
	}
}

func TestComposableSavedTapLifetime(t *testing.T) {
	for _, change := range []string{"none", "return", "absent", "phase", "phase cycle", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Saved", "", 2, 2)
			p := g.PutOnBattlefield(c, a.PlayerID())
			saved := BindPermanent("saved")
			ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(c, a.PlayerID())
			case "absent":
				g.RemoveFromBattlefield(p)
			case "phase":
				g.PhaseOut(c.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(c.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: c.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			}
			clone := g.Clone()
			ctx.Game = clone
			if err := TapSubject(From(saved)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if change == "phase" {
				clone.PhaseIn([]uuid.UUID{c.ID()})
			}
			want := change == "none" || change == "phase cycle" || change == "control"
			if got := clone.FindPermanent(c.ID()); got != nil && got.Tapped != want {
				t.Fatalf("tapped = %v, want %v", got.Tapped, want)
			}
			if got := g.FindPermanent(c.ID()); got != nil && got.Tapped {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestComposableTapBindingValidation(t *testing.T) {
	saved := BindPermanent("saved")
	for _, body := range []Effect{
		TapSubject(nil),
		TapSubject(From(saved)),
		TapSubject(DeclarePermanentTarget("undeclared", IsCreature)),
		Sequence(TapSubject(From(saved)), Snapshot(This, saved)),
	} {
		card := NewCreature("Invalid tap", "", 1, 1, WithAbility(Activate(Costs(), body)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid tap subject")
		}
	}
}
