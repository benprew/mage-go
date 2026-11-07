package mage

import (
	"testing"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCopySpellAndMayChooseNewTargetsWithEffects(t *testing.T) {
	for _, keepTarget := range []bool{false, true} {
		t.Run(map[bool]string{false: "new_target", true: "same_target"}[keepTarget], func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			spell := NewInstant("Blue Bolt", "{U}", NewTargetedSpell(TargetDamageAnyTarget(), DealDamage(Fixed(3))))
			original := &StackObject{
				ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: b.PlayerID(),
				Effects: []Effect{DealDamage(Fixed(3))}, Targets: []uuid.UUID{b.PlayerID()}, XValue: 4,
			}
			g.pushStack(original)
			chosen := a.PlayerID()
			if keepTarget {
				chosen = b.PlayerID()
			}
			a.chooseQueue = [][]uuid.UUID{{chosen}}
			effect := CopySpellAndMayChooseNewTargetsWithEffects(ChangeColorEffect(Red))
			if err := effect.Apply(NewEffectContext(g, uuid.New(), a.PlayerID(), []uuid.UUID{spell.ID()})); err != nil {
				t.Fatal(err)
			}
			cp := g.Stack().Peek()
			if !cp.IsCopy || cp.Controller != a.PlayerID() || cp.XValue != 4 {
				t.Fatalf("copy state: %+v", cp)
			}
			if len(cp.Targets) != 1 || cp.Targets[0] != chosen {
				t.Fatalf("copy targets: %v, want %v", cp.Targets, chosen)
			}
			if colors := g.EffectiveColors(cp.SourceID); len(colors) != 1 || colors[0] != Red {
				t.Fatalf("copy colors: %v, want red", colors)
			}
			if colors := g.EffectiveColors(spell.ID()); len(colors) != 1 || colors[0] != Blue {
				t.Fatalf("original colors: %v, want blue", colors)
			}
			g.ResolveStack()
			wantA, wantB := 17, 17
			if keepTarget {
				wantA, wantB = 20, 14
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life: %d, %d; want %d, %d", a.Life(), b.Life(), wantA, wantB)
			}
		})
	}
}
