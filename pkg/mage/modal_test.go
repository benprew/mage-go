package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// CR 700.2 — Modal spells.
//
// NewModalSpell builds a card whose SpellAbility has multiple Modes, each
// with its own targets and effects. The controller picks the mode at cast
// time; only that mode's targets are gathered, and only that mode's
// effects resolve.

// modalChoosePlayer is a player wrapper that scripts ChooseMode and
// ChooseTargets so we can drive modal-spell tests deterministically.
type modalChoosePlayer struct {
	*BasePlayer
	modes   []int
	targets [][]uuid.UUID
}

func (p *modalChoosePlayer) ChooseMode(_ []string, _ string) int {
	if len(p.modes) == 0 {
		return 0
	}
	m := p.modes[0]
	p.modes = p.modes[1:]
	return m
}

func (p *modalChoosePlayer) ChooseTargets(possible []uuid.UUID, min, max int, _ *Game) []uuid.UUID {
	if len(p.targets) > 0 {
		out := p.targets[0]
		p.targets = p.targets[1:]
		return out
	}
	if min == 0 {
		return nil
	}
	if len(possible) < min {
		return nil
	}
	return possible[:min]
}

func newModalGame() (*Game, *modalChoosePlayer, *modalChoosePlayer) {
	a := &modalChoosePlayer{BasePlayer: NewBasePlayer("A")}
	b := &modalChoosePlayer{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	for _, p := range g.players {
		for range 30 {
			p.AddToLibrary(NewLand("Plains"))
		}
	}
	return g, a, b
}

// TestNewModalSpell_Mode0 builds a "Choose one — Destroy target creature
// you control / Destroy target permanent" modal spell. With mode 0
// scripted, the controller's chosen creature is destroyed.
func TestNewModalSpell_Mode0DestroysOwnCreature(t *testing.T) {
	g, a, b := newModalGame()

	// Two creatures — one A, one B.
	mine := NewCreature("Mine", "{1}", 1, 1)
	mine.SetOwner(a.PlayerID())
	myPerm := g.PutOnBattlefield(mine, a.PlayerID())

	yours := NewCreature("Yours", "{1}", 1, 1)
	yours.SetOwner(b.PlayerID())
	yourPerm := g.PutOnBattlefield(yours, b.PlayerID())

	// Modal spell with two modes.
	spell := NewInstant("Modal Test", "{1}", nil)
	ms := NewModalSpell([]Mode{
		{
			Label:   "Destroy target creature you control",
			Targets: []Target{TargetControlledCreature()},
			Effects: []Effect{DestroyTarget()},
		},
		{
			Label:   "Destroy target permanent",
			Targets: []Target{TargetPermanent()},
			Effects: []Effect{DestroyTargetPermanent()},
		},
	})
	spell.AddAbility(ms)
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	// Mana for {1}: dump one generic into the pool.
	a.ManaPool().Add(Colorless, 1)

	a.modes = []int{0}
	a.targets = [][]uuid.UUID{{myPerm.ID()}}

	if err := g.CastSpellByName(a.PlayerID(), "Modal Test", nil); err != nil {
		t.Fatalf("CastSpellByName: %v", err)
	}
	g.ResolveStack()

	// "Mine" is destroyed; "Yours" remains.
	if g.FindPermanent(myPerm.ID()) != nil {
		t.Errorf("expected Mine destroyed, still on battlefield")
	}
	if g.FindPermanent(yourPerm.ID()) == nil {
		t.Errorf("expected Yours untouched, missing from battlefield")
	}
}

// TestNewModalSpell_Mode1DestroysAnyPermanent: with mode 1 scripted,
// the spell destroys any target permanent — including PlayerB's creature.
func TestNewModalSpell_Mode1DestroysAnyPermanent(t *testing.T) {
	g, a, b := newModalGame()

	mine := NewCreature("Mine", "{1}", 1, 1)
	mine.SetOwner(a.PlayerID())
	g.PutOnBattlefield(mine, a.PlayerID())

	yours := NewCreature("Yours", "{1}", 1, 1)
	yours.SetOwner(b.PlayerID())
	yourPerm := g.PutOnBattlefield(yours, b.PlayerID())

	spell := NewInstant("Modal Test 2", "{1}", nil)
	ms := NewModalSpell([]Mode{
		{
			Label:   "Destroy target creature you control",
			Targets: []Target{TargetControlledCreature()},
			Effects: []Effect{DestroyTarget()},
		},
		{
			Label:   "Destroy target permanent",
			Targets: []Target{TargetPermanent()},
			Effects: []Effect{DestroyTargetPermanent()},
		},
	})
	spell.AddAbility(ms)
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(Colorless, 1)

	a.modes = []int{1}
	a.targets = [][]uuid.UUID{{yourPerm.ID()}}

	if err := g.CastSpellByName(a.PlayerID(), "Modal Test 2", nil); err != nil {
		t.Fatalf("CastSpellByName: %v", err)
	}
	g.ResolveStack()

	if g.FindPermanent(yourPerm.ID()) != nil {
		t.Errorf("expected Yours destroyed, still on battlefield")
	}
}

// TestNewModalSpell_PanicsOnFewerThanTwoModes: NewModalSpell panics if
// fewer than two modes are supplied.
func TestNewModalSpell_PanicsOnFewerThanTwoModes(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for single-mode NewModalSpell")
		}
	}()
	NewModalSpell([]Mode{{Label: "only one"}})
}

// TestModalTriggerEffect_Dispatch: a modal trigger effect prompts ChooseMode
// at resolution and runs the chosen mode's Resolve callback.
func TestModalTriggerEffect_Dispatch(t *testing.T) {
	g, a, _ := newModalGame()

	chose := -1
	eff := ModalTriggerEffect("test trigger", []ModalTriggerMode{
		{Label: "mode A", Resolve: func(_ *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
			chose = 0
			return nil
		}},
		{Label: "mode B", Resolve: func(_ *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
			chose = 1
			return nil
		}},
	})

	a.modes = []int{1}
	if err := ApplyEffect(g, eff, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if chose != 1 {
		t.Errorf("expected mode 1, got %d", chose)
	}
}

// TestModalTriggerEffect_PanicsOnFewerThanTwoModes: ModalTriggerEffect
// panics with fewer than two modes, matching CR 700.2.
func TestModalTriggerEffect_PanicsOnFewerThanTwoModes(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for single-mode ModalTriggerEffect")
		}
	}()
	_ = ModalTriggerEffect("x", []ModalTriggerMode{{Label: "only"}})
}

// TestNewModalSpell_StackObjectRecordsMode: cast pipeline writes the chosen
// mode index and the chosen mode's targets onto the StackObject.
func TestNewModalSpell_StackObjectRecordsMode(t *testing.T) {
	g, a, _ := newModalGame()

	mine := NewCreature("Mine", "{1}", 1, 1)
	mine.SetOwner(a.PlayerID())
	myPerm := g.PutOnBattlefield(mine, a.PlayerID())

	spell := NewInstant("Modal Stack Test", "{1}", nil)
	spell.AddAbility(NewModalSpell([]Mode{
		{
			Label:   "self",
			Targets: []Target{TargetControlledCreature()},
			Effects: []Effect{DestroyTarget()},
		},
		{
			Label:   "any",
			Targets: []Target{TargetPermanent()},
			Effects: []Effect{DestroyTargetPermanent()},
		},
	}))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(Colorless, 1)

	a.modes = []int{1}
	a.targets = [][]uuid.UUID{{myPerm.ID()}}

	if err := g.CastSpellByName(a.PlayerID(), "Modal Stack Test", nil); err != nil {
		t.Fatalf("CastSpellByName: %v", err)
	}
	top := g.stack.Peek()
	if top == nil {
		t.Fatalf("stack empty after cast")
	}
	if top.ModeChoice != 1 {
		t.Errorf("ModeChoice: got %d, want 1", top.ModeChoice)
	}
	if len(top.Targets) != 1 || top.Targets[0] != myPerm.ID() {
		t.Errorf("Targets: got %v, want %v", top.Targets, []uuid.UUID{myPerm.ID()})
	}
	if len(top.ModalTargets) != 2 || len(top.ModalTargets[1]) != 1 {
		t.Errorf("ModalTargets: got %v, want mode 1 populated", top.ModalTargets)
	}
}
