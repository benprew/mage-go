package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDelayedInterveningIfTiming(t *testing.T) {
	for _, keep := range []bool{false, true} {
		t.Run(map[bool]string{false: "false at resolution", true: "true at resolution"}[keep], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			effect := AtNext(YourEndStep, GainLifeFor(You, Fixed(3)), WithInterveningIf(Controls(IsArtifact)))
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 || len(g.DelayedTriggers()) != 1 {
				t.Fatal("false condition consumed the delay")
			}
			artifact := NewArtifact("Artifact", "")
			g.PutOnBattlefield(artifact, a.PlayerID())
			g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 || len(g.DelayedTriggers()) != 1 {
				t.Fatal("ignored step filter")
			}
			g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
			if g.StackSize() != 0 || len(g.DelayedTriggers()) != 0 {
				t.Fatal("incorrect pending lifecycle")
			}
			g.RemoveFromBattlefield(g.FindPermanent(artifact.ID()))
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("condition was checked again at placement")
			}
			if keep {
				g.PutOnBattlefield(artifact, a.PlayerID())
			}
			g.ResolveStack()
			want := 20
			if keep {
				want = 23
			}
			if a.Life() != want {
				t.Fatalf("life=%d, want %d", a.Life(), want)
			}
			if !keep {
				g.PutOnBattlefield(artifact, a.PlayerID())
			}
			g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
			g.ResolveStack()
			if a.Life() != want {
				t.Fatal("consumed trigger retried")
			}
		})
	}
}

func TestDelayedInterveningIfCopyAndClone(t *testing.T) {
	a, b := newRecordingPlayer("A"), newRecordingPlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("future player")
	amount := BindNumber("saved amount")
	target := DeclarePlayerTarget("future target")
	effect := Sequence(SnapshotNumber(Fixed(3), amount), AtNext(TurnStepBegins(EachEndStep, player),
		Sequence(GainLifeFor(target, FromNumber(amount)), GainLifeFor(FromPlayer(player), Fixed(2))),
		WithTargets(target), WithInterveningIf(Controls(IsArtifact))))
	source := NewCreature("Source", "", 1, 1, WithAbility(Activate(Costs(), effect)))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	artifact := NewArtifact("Artifact", "")
	g.PutOnBattlefield(artifact, a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.PhaseOut(source.ID())
	registered := g.Clone()
	registered.RemoveFromBattlefield(registered.FindPermanent(artifact.ID()))
	registered.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	registered.PutTriggersOnStack()
	if registered.StackSize() != 0 || len(registered.DelayedTriggers()) != 1 {
		t.Fatal("registered clone ignored its condition")
	}
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	pending := g.Clone()
	pending.RemoveFromBattlefield(pending.FindPermanent(artifact.ID()))
	pending.ResolveStack()
	if pending.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("pending clone ignored resolution condition")
	}
	a.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("missing delayed trigger")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("missing copy")
	}
	stackClone := g.Clone()
	stackClone.PutOnBattlefield(NewArtifact("Copy controller artifact", ""), b.PlayerID())
	stackClone.ResolveStack()
	if stackClone.GetPlayer(b.PlayerID()).Life() != 30 {
		t.Fatal("copy lost condition, target, or saved bindings")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 25 {
		t.Fatal("copy used original controller or clone leaked")
	}
}

func TestDelayedInterveningIfInvalidOptions(t *testing.T) {
	for _, options := range [][]DelayedStepOption{
		{WithInterveningIf(nil)},
		{WithInterveningIf(Controls(IsArtifact)), WithInterveningIf(Controls(IsCreature))},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("invalid condition accepted")
				}
			}()
			AtNext(EachEndStep, GainLifeFor(You, Fixed(1)), options...)
		}()
	}
}

func TestDelayedInterveningIfValidation(t *testing.T) {
	effect := AtNext(EachEndStep, GainLifeFor(You, Fixed(1)), WithInterveningIf(AllConditions()))
	card := NewSorcery("Invalid delayed condition", "", Spell(effect))
	err := ValidateCardDefinition(card.Name(), card)
	if err == nil || !strings.Contains(err.Error(), "condition") || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("condition validation=%v", err)
	}
}
