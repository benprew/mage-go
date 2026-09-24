package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTypedModalTriggerEventAndCopy(t *testing.T) {
	for _, dies := range []bool{false, true} {
		t.Run(map[bool]string{false: "enters", true: "dies"}[dies], func(t *testing.T) {
			a := &modalTriggerPlayer{recordingPlayer: newRecordingPlayer("A")}
			b := newRecordingPlayer("B")
			g := NewGame(a, b)
			target := DeclarePlayerTarget("recipient")
			missing := DeclarePermanentTarget("creature", IsCreature)
			pattern := Enters(This)
			if dies {
				pattern = Dies(This)
			}
			trigger := WhenEventModes(pattern,
				ModeOf("Unavailable", Destroy(missing), WithTargets(missing)),
				ModeOf("Life", GainLifeFor(target, Fixed(3)), WithTargets(target)),
			)
			source := NewEnchantment("Modal source", "", WithAbility(trigger))
			source.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
			perm := g.PutOnBattlefield(source, a.PlayerID())
			if dies {
				g.DestroyPermanent(perm)
			}
			g.PutTriggersOnStack()
			original := g.StackPeek()
			if original == nil || original.ModeChoice != 1 || len(a.menus) != 1 || len(a.menus[0]) != 1 || a.menus[0][0] != "Life" {
				t.Fatal("typed trigger lost its legal mode")
			}
			if a.Life() != 20 {
				t.Fatal("mode resolved before priority")
			}
			if !dies {
				g.RemoveFromBattlefield(perm)
			}
			b.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}
			cp := g.CopyStackObjectDirect(original, b.PlayerID(), true)
			if cp == nil || cp.ModeChoice != 1 || cp.Targets[0] != b.PlayerID() || original.Targets[0] != a.PlayerID() {
				t.Fatal("copy lost mode or independent targets")
			}
			g.ResolveStack()
			if a.Life() != 23 || b.Life() != 23 {
				t.Fatalf("life = %d, %d", a.Life(), b.Life())
			}
			if len(a.menus) != 1 {
				t.Fatal("copy chose a new mode")
			}
		})
	}
}

func TestTypedModalStepTrigger(t *testing.T) {
	a := &modalTriggerPlayer{recordingPlayer: newRecordingPlayer("A"), choice: 1}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	trigger := AtModes(YourUpkeep, ModeOf("One", GainLifeFor(You, Fixed(1))), ModeOf("Two", GainLifeFor(You, Fixed(2))))
	source := NewEnchantment("Upkeep source", "", WithAbility(trigger))
	source.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 || len(a.menus) != 0 {
		t.Fatal("trigger matched opponent upkeep")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || a.Life() != 20 {
		t.Fatal("upkeep trigger did not use the stack")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("life = %d", a.Life())
	}
}

func TestTypedModalTriggerValidation(t *testing.T) {
	first := DeclarePlayerTarget("first")
	second := DeclarePlayerTarget("second")
	for _, step := range []bool{false, true} {
		modes := []ModeDefinition{ModeOf("First", GainLifeFor(first, Fixed(1)), WithTargets(first)), ModeOf("Second", GainLifeFor(first, Fixed(1)), WithTargets(second))}
		trigger := WhenEventModes(Enters(This), modes...)
		if step {
			trigger = AtModes(YourUpkeep, modes...)
		}
		err := ValidateCardDefinition("Scope", NewEnchantment("Scope", "", WithAbility(trigger)))
		if err == nil || !strings.Contains(err.Error(), "modes[1]") {
			t.Fatalf("scope diagnostic = %v", err)
		}
	}
	for _, build := range []func(){
		func() {
			WhenEventModes(nil, ModeOf("One", GainLifeFor(You, Fixed(1))), ModeOf("Two", GainLifeFor(You, Fixed(2))))
		},
		func() {
			AtModes(nil, ModeOf("One", GainLifeFor(You, Fixed(1))), ModeOf("Two", GainLifeFor(You, Fixed(2))))
		},
		func() { WhenEventModes(Enters(This)) },
		func() { AtModes(YourUpkeep, ModeOf("One", GainLifeFor(You, Fixed(1)))) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("invalid modal trigger did not panic")
				}
			}()
			build()
		}()
	}
	trigger := WhenEventModes(Enters(This), ModeOf("Missing", nil), ModeOf("Life", GainLifeFor(You, Fixed(1))))
	if err := ValidateCardDefinition("Nil", NewEnchantment("Nil", "", WithAbility(trigger))); err == nil {
		t.Fatal("nil mode body passed validation")
	}
}

func TestTypedModalTriggerClone(t *testing.T) {
	a := &modalTriggerPlayer{recordingPlayer: newRecordingPlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("recipient")
	trigger := WhenEventModes(Enters(This), ModeOf("Life", GainLifeFor(target, Fixed(3)), WithTargets(target)), ModeOf("Other", GainLifeFor(You, Fixed(1))))
	source := NewEnchantment("Clone source", "", WithAbility(trigger))
	source.SetOwner(a.PlayerID())
	a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 23 || a.Life() != 20 || g.StackSize() != 1 {
		t.Fatal("clone changed the original trigger or player")
	}
	g.ResolveStack()
	if a.Life() != 23 || clone.GetPlayer(a.PlayerID()).Life() != 23 {
		t.Fatal("original lost its target binding")
	}
}
