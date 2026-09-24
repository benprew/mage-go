package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDelayedModesCaptureCopyAndClone(t *testing.T) {
	a := &modalTriggerPlayer{recordingPlayer: newRecordingPlayer("A")}
	b := newRecordingPlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("recipient")
	missing := DeclarePermanentTarget("missing creature", IsCreature)
	player := BindPlayer("future player")
	amount := BindNumber("saved amount")
	effect := Sequence(SnapshotNumber(Fixed(3), amount), AtNextModes(TurnStepBegins(EachEndStep, player),
		ModeOf("Unavailable", Destroy(missing), WithTargets(missing)),
		ModeOf("Life", Sequence(GainLifeFor(target, FromNumber(amount)), GainLifeFor(FromPlayer(player), Fixed(2))), WithTargets(target))))
	card := NewSorcery("Schedule", "", Spell(effect))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.menus) != 0 {
		t.Fatal("mode chosen at installation")
	}
	registered := g.Clone()
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	if len(a.menus) != 0 || g.StackSize() != 0 || len(g.DelayedTriggers()) != 0 {
		t.Fatal("incorrect detection lifecycle")
	}
	pending := g.Clone()
	a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
	g.PutTriggersOnStack()
	original := g.StackPeek()
	if original == nil || original.ModeChoice != 1 || len(a.menus) != 1 || len(a.menus[0]) != 1 || a.menus[0][0] != "Life" {
		t.Fatal("incorrect legal mode menu")
	}
	b.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}
	cp := g.CopyStackObjectDirect(original, b.PlayerID(), true)
	if cp == nil || cp.ModeChoice != 1 || cp.Targets[0] != b.PlayerID() || original.Targets[0] != a.PlayerID() {
		t.Fatal("copy lost mode or target")
	}
	cloned := g.Clone()
	cloned.ResolveStack()
	if cloned.GetPlayer(a.PlayerID()).Life() != 23 || cloned.GetPlayer(b.PlayerID()).Life() != 27 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("stack clone lost captures or leaked")
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 27 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
	for _, c := range []*Game{registered, pending} {
		if len(c.DelayedTriggers()) > 0 {
			c.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
		}
		c.ResolveStack()
		if c.GetPlayer(a.PlayerID()).Life()+c.GetPlayer(b.PlayerID()).Life() != 45 {
			t.Fatal("clone lost delayed modes or captures")
		}
	}
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 27 {
		t.Fatal("one-shot trigger repeated")
	}
}

func TestDelayedModesNoLegalMode(t *testing.T) {
	a := &modalTriggerPlayer{recordingPlayer: newRecordingPlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	target := DeclarePermanentTarget("creature", IsCreature)
	other := DeclarePermanentTarget("artifact", IsArtifact)
	effect := AtNextModes(EachEndStep, ModeOf("Creature", Destroy(target), WithTargets(target)), ModeOf("Artifact", Destroy(other), WithTargets(other)))
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 || len(g.DelayedTriggers()) != 0 || len(a.menus) != 0 {
		t.Fatal("empty menu did not consume trigger")
	}
}

func TestDelayedModesValidation(t *testing.T) {
	target := DeclarePlayerTarget("first mode target")
	effect := AtNextModes(EachEndStep, ModeOf("First", GainLifeFor(target, Fixed(1)), WithTargets(target)), ModeOf("Second", GainLifeFor(target, Fixed(2))))
	err := ValidateCardDefinition("Scope", NewSorcery("Scope", "", Spell(effect)))
	if err == nil || !strings.Contains(err.Error(), "modes[1]") {
		t.Fatalf("scope diagnostic=%v", err)
	}
	for _, build := range []func(){
		func() { AtNextModes(EachEndStep) },
		func() {
			AtNextModes(nil, ModeOf("One", GainLifeFor(You, Fixed(1))), ModeOf("Two", GainLifeFor(You, Fixed(2))))
		},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("invalid modes accepted")
				}
			}()
			build()
		}()
	}
}

func TestDelayedModesConditionAndLegality(t *testing.T) {
	for _, scenario := range []string{"resolve", "condition false", "target returned", "invalid choice"} {
		t.Run(scenario, func(t *testing.T) {
			a := &modalTriggerPlayer{recordingPlayer: newRecordingPlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			target := DeclarePermanentTarget("creature", IsCreature)
			effect := AtNextModes(EachEndStep,
				ModeOf("Destroy", Sequence(Destroy(target), GainLifeFor(You, Fixed(3))), WithTargets(target)),
				ModeOf("Other", GainLifeFor(You, Fixed(1))),
				WithInterveningIf(Controls(IsArtifact)))
			source := NewEnchantment("Source", "", WithAbility(Activate(Costs(), effect)))
			creature := NewCreature("Creature", "", 1, 1)
			creature.SetOwner(b.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutOnBattlefield(creature, b.PlayerID())
			if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			g.PutOnBattlefield(source, b.PlayerID())
			g.PhaseOut(source.ID())
			g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if len(g.DelayedTriggers()) != 1 || g.StackSize() != 0 {
				t.Fatal("false condition consumed delay")
			}
			artifact := NewArtifact("Artifact", "")
			g.PutOnBattlefield(artifact, a.PlayerID())
			g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
			if scenario == "condition false" {
				g.RemoveFromBattlefield(g.FindPermanent(artifact.ID()))
			}
			if scenario == "invalid choice" {
				a.choice = 99
			}
			a.chooseQueue = [][]uuid.UUID{{creature.ID()}}
			g.PutTriggersOnStack()
			if scenario == "invalid choice" {
				if g.StackSize() != 0 || len(g.DelayedTriggers()) != 0 {
					t.Fatal("invalid choice retained trigger")
				}
				return
			}
			if g.StackSize() != 1 {
				t.Fatal("missing pending trigger")
			}
			if scenario == "target returned" {
				g.RemoveFromBattlefield(g.FindPermanent(creature.ID()))
				g.PutOnBattlefield(creature, b.PlayerID())
			}
			g.ResolveStack()
			want := 20
			if scenario == "resolve" {
				want = 23
				if g.FindPermanent(creature.ID()) != nil {
					t.Fatal("target survived")
				}
			}
			if a.Life() != want || b.Life() != 20 {
				t.Fatal("condition, source controller, or target legality lost")
			}
		})
	}
}

func TestDelayedModesOptionalTarget(t *testing.T) {
	a := &modalTriggerPlayer{recordingPlayer: newRecordingPlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	target := DeclarePermanentTargets("optional", 0, 1, IsCreature)
	effect := AtNextModes(EachEndStep, ModeOf("Optional", Sequence(DestroyEach(target), GainLifeFor(You, Fixed(2))), WithTargets(target)), ModeOf("Other", GainLifeFor(You, Fixed(1))))
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
	g.ResolveStack()
	if a.Life() != 22 || len(a.menus) != 1 || len(a.menus[0]) != 2 {
		t.Fatal("optional target mode was unavailable")
	}
}
