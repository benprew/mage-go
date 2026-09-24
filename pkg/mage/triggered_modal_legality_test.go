package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type modalTriggerPlayer struct {
	*recordingPlayer
	menus  [][]string
	choice int
}

func (p *modalTriggerPlayer) ChooseMode(labels []string, _ string) int {
	p.menus = append(p.menus, append([]string(nil), labels...))
	return p.choice
}

func TestModalTriggerLegalMenu(t *testing.T) {
	for _, scenario := range []string{"empty", "insufficient", "later group", "optional", "legal", "shroud", "phased", "graveyard", "random"} {
		t.Run(scenario, func(t *testing.T) {
			a := &modalTriggerPlayer{recordingPlayer: newRecordingPlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			targets := []Target{TargetCreature()}
			legal := scenario == "optional" || scenario == "legal"
			switch scenario {
			case "insufficient", "legal", "shroud", "phased":
				card := NewCreature("Candidate", "", 1, 1)
				perm := g.PutOnBattlefield(card, b.PlayerID())
				if scenario == "shroud" {
					perm.GrantBaseAttr(core.Shroud)
				}
				if scenario == "phased" {
					perm.PhasedOut = true
				}
				if scenario == "insufficient" {
					targets = []Target{TargetNCreatures(2)}
				}
			case "later group":
				targets = []Target{TargetPlayer(), TargetCreature()}
			case "optional":
				targets = []Target{TargetUpToNCreatures(1)}
			case "graveyard":
				targets = []Target{TargetCardInYourGraveyard(IsCreatureCard)}
			case "random":
				targets = []Target{TargetRandom(TargetCreature())}
			}
			resolved := -1
			effect := func(n int) Effect {
				return FuncEffect("record mode", EffectProperties{}, func(*Game, uuid.UUID, uuid.UUID, []uuid.UUID) error { resolved = n; return nil })
			}
			trigger := EntersBattlefieldTrigger(nil, false).WithModes(
				Mode{Label: "Targeted", Targets: targets, Effects: []Effect{effect(0)}},
				Mode{Label: "Fallback", Effects: []Effect{effect(1)}},
			)
			source := NewEnchantment("Modal source", "", WithAbility(trigger))
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutTriggersOnStack()
			wantMenu := []string{"Fallback"}
			wantMode := 1
			if legal {
				wantMenu = []string{"Targeted", "Fallback"}
				wantMode = 0
			}
			if !reflect.DeepEqual(a.menus, [][]string{wantMenu}) {
				t.Fatalf("menus = %v, want %v", a.menus, wantMenu)
			}
			obj := g.StackPeek()
			if obj == nil || obj.ModeChoice != wantMode {
				t.Fatalf("selected mode = %+v, want %d", obj, wantMode)
			}
			if resolved != -1 {
				t.Fatal("mode resolved during selection")
			}
			if !legal && len(a.calls) != 0 {
				t.Fatal("unavailable mode prompted for targets")
			}
			g.ResolveStack()
			if resolved != wantMode {
				t.Fatalf("resolved mode = %d, want %d", resolved, wantMode)
			}
		})
	}
}

func TestModalTriggerNoMode(t *testing.T) {
	for _, scenario := range []string{"all unavailable", "negative choice", "out of range"} {
		t.Run(scenario, func(t *testing.T) {
			a := &modalTriggerPlayer{recordingPlayer: newRecordingPlayer("A"), choice: -1}
			if scenario == "out of range" {
				a.choice = 2
			}
			g := NewGame(a, NewBasePlayer("B"))
			modes := []Mode{{Label: "First"}, {Label: "Second"}}
			if scenario == "all unavailable" {
				for i := range modes {
					modes[i].Targets = []Target{TargetCreature()}
				}
			}
			source := NewEnchantment("Modal source", "", WithAbility(EntersBattlefieldTrigger(nil, false).WithModes(modes...)))
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("trigger remained without a legal chosen mode")
			}
			if scenario == "all unavailable" && len(a.menus) != 0 {
				t.Fatal("empty mode menu prompted")
			}
			if len(a.calls) != 0 {
				t.Fatal("removed trigger prompted for targets")
			}
		})
	}
}

func TestModalTriggerFilteredIndexAndCopy(t *testing.T) {
	a := &modalTriggerPlayer{recordingPlayer: newRecordingPlayer("A"), choice: 1}
	b := newRecordingPlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("recipient")
	trigger := EntersBattlefieldTrigger(nil, false).WithModes(
		Mode{Label: "Unavailable", Targets: []Target{TargetCreature()}},
		Mode{Label: "First legal", Effects: []Effect{GainLifeFor(You, Fixed(1))}},
		Mode{Label: "Second legal", Targets: []Target{target.targetSpec()}, Effects: []Effect{GainLifeFor(target, Fixed(3))}},
	)
	source := NewEnchantment("Modal source", "", WithAbility(trigger))
	source.SetOwner(a.PlayerID())
	a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutTriggersOnStack()
	original := g.StackPeek()
	if original == nil || original.ModeChoice != 2 {
		t.Fatal("filtered menu index replaced the original mode index")
	}
	if !reflect.DeepEqual(a.menus, [][]string{{"First legal", "Second legal"}}) {
		t.Fatalf("menus = %v", a.menus)
	}
	b.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}
	cp := g.CopyStackObjectDirect(original, b.PlayerID(), true)
	if cp.ModeChoice != 2 || cp.Targets[0] != b.PlayerID() || original.Targets[0] != a.PlayerID() {
		t.Fatal("copy lost mode or independent targets")
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 23 {
		t.Fatalf("life totals = %d, %d", a.Life(), b.Life())
	}
	if len(a.menus) != 1 {
		t.Fatal("copy prompted for another mode")
	}
}
