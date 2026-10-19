package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestSpellPlayerRetargetContract(t *testing.T) {
	for _, state := range []string{"legal", "own-controller", "cancel", "old-choice", "invalid", "no-player", "two-targets", "absent", "copy", "modal", "allocation", "changed"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			spec := TargetPlayer()
			if state == "own-controller" {
				spec = TargetOpponent()
			}
			if state == "no-player" {
				spec = TargetCreature()
			}
			card := NewInstant("Spell", "{R}", NewTargetedSpell(spec, DealDamage(Fixed(3))))
			obj := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID(), Effects: []Effect{DealDamage(Fixed(3))}, Targets: []uuid.UUID{a.PlayerID()}, TargetSpecs: []Target{spec}, XValue: 4}
			if state == "two-targets" {
				obj.Targets = append(obj.Targets, a.PlayerID())
				obj.TargetSpecs = append(obj.TargetSpecs, spec)
			}
			if state == "modal" {
				obj.ModalTargets = [][]uuid.UUID{{a.PlayerID()}}
				obj.ModeChoice = 0
			}
			if state == "allocation" {
				obj.DamageDistribution = map[uuid.UUID]int{a.PlayerID(): 3}
				obj.CounterDistribution = map[uuid.UUID]int{a.PlayerID(): 2}
			}
			g.pushStack(obj)
			if state == "copy" {
				obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
			}
			binding := BindSpell("spell")
			g.resolution.spellBindings = map[*SpellBinding]uuid.UUID{binding: obj.ID}
			effect := IfSpellTargetsPlayer(FromSpell(binding), You, ChangeSingleSpellTargetToPlayer(FromSpell(binding)))
			a.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}
			if state == "cancel" {
				a.chooseQueue = [][]uuid.UUID{nil}
			}
			if state == "old-choice" {
				a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
			}
			if state == "invalid" {
				a.chooseQueue = [][]uuid.UUID{{uuid.New()}}
			}
			if state == "absent" {
				g.stack.RemoveByID(obj.ID)
			}
			if state == "changed" {
				obj.Targets = []uuid.UUID{b.PlayerID()}
			}
			clone := g.Clone()
			if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			change := state == "legal" || state == "copy" || state == "modal" || state == "allocation" || state == "changed"
			want := a.PlayerID()
			if change {
				want = b.PlayerID()
			}
			if obj.Targets[0] != want || obj.Controller != b.PlayerID() || obj.XValue != 4 {
				t.Fatal("retarget contract")
			}
			if state != "absent" && clone.stack.FindByID(obj.ID).Targets[0] != func() uuid.UUID {
				if state == "changed" {
					return b.PlayerID()
				}
				return a.PlayerID()
			}() {
				t.Fatal("clone target mutation")
			}
			if state == "modal" && obj.ModalTargets[0][0] != b.PlayerID() {
				t.Fatal("modal targets")
			}
			if state == "allocation" && (obj.DamageDistribution[b.PlayerID()] != 3 || obj.CounterDistribution[b.PlayerID()] != 2 || clone.stack.FindByID(obj.ID).DamageDistribution[a.PlayerID()] != 3) {
				t.Fatal("allocation changed")
			}
			if state == "copy" && g.stack.Objects()[0].Targets[0] != a.PlayerID() {
				t.Fatal("copy changed original")
			}
			if state == "legal" {
				g.ResolveStack()
				if a.Life() != 20 || b.Life() != 17 {
					t.Fatal("original spell controller lost")
				}
			}
		})
	}
}

func TestSpellPlayerRetargetDefinition(t *testing.T) {
	target := DeclareSpellTarget("spell")
	for _, body := range []ResolvingEffect{
		ChangeSingleSpellTargetToPlayer(nil),
		IfSpellTargetsPlayer(nil, You, GainLifeFor(You, Fixed(1))),
		IfSpellTargetsPlayer(target, nil, GainLifeFor(You, Fixed(1))),
		IfSpellTargetsPlayer(target, You, nil),
	} {
		if err := ValidateCardDefinition("Bad", NewInstant("Bad", "", Spell(body, WithTargets(target)))); err == nil {
			t.Fatal("invalid declaration")
		}
	}
}
