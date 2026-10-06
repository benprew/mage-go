package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSetObjectColorDefinition(t *testing.T) {
	for _, tc := range []struct {
		name     string
		ref      *SpellOrPermanentTargetBinding
		color    core.Color
		declared bool
		valid    bool
	}{
		{"valid", DeclareSpellOrPermanentTarget("spell"), core.Blue, true, true},
		{"nil", nil, core.Blue, false, false},
		{"unbound", DeclareSpellOrPermanentTarget("spell"), core.Blue, false, false},
		{"empty-name", DeclareSpellOrPermanentTarget(""), core.Blue, true, false},

		{"colorless", DeclareSpellOrPermanentTarget("spell"), core.Colorless, true, false},
		{"any", DeclareSpellOrPermanentTarget("spell"), core.AnyColor, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var options []SpellOption
			if tc.declared {
				options = append(options, WithTargets(tc.ref))
			}
			card := NewInstant("Color", "", Spell(SetObjectColor(tc.ref, tc.color), options...))
			err := ValidateCardDefinition(card.Name(), card)
			if (err == nil) != tc.valid {
				t.Fatalf("validation = %v, valid = %v", err, tc.valid)
			}
		})
	}
}

func TestSetObjectColorIdentity(t *testing.T) {
	for _, scenario := range []string{"live", "gone", "recast", "copy", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := newObjectColorGame()
			g.SetTurn(2)
			g.SetStep(core.PrecombatMain)
			victim := pushCreatureSpell(g, b, "Victim", "{G}")
			target := DeclareSpellOrPermanentTarget("spell")
			card := NewInstant("Color", "", Spell(SetObjectColor(target, core.Blue), WithTargets(target)))
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim.ID}, 0); err != nil {
				t.Fatal(err)
			}
			var replacement *StackObject
			switch scenario {
			case "gone", "recast":
				g.CounterStackObject(victim.ID)
				if scenario == "recast" {
					returned, ok := g.MoveFromGraveyard(b.PlayerID(), victim.SourceID, core.ZoneStack)
					if !ok {
						t.Fatal("missing departed spell")
					}
					replacement = &StackObject{ID: uuid.New(), Card: returned, SourceID: victim.SourceID, Controller: b.PlayerID()}
					g.pushStack(replacement)
					g.stack.Pop()
					color := g.stack.Pop()
					g.stack.Push(replacement)
					g.stack.Push(color)
				}
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveTopOfStack()
			if scenario == "copy" {
				g.ResolveTopOfStack()
			}
			if scenario == "gone" || scenario == "recast" {
				if replacement != nil && !slices.Equal(g.EffectiveColors(replacement.ID), []core.Color{core.Green}) {
					t.Fatal("old reference changed a new spell")
				}
				return
			}
			if !slices.Equal(g.EffectiveColors(victim.ID), []core.Color{core.Blue}) {
				t.Fatal("spell color did not change")
			}
			if !slices.Equal(clone.EffectiveColors(victim.ID), []core.Color{core.Green}) {
				t.Fatal("clone changed")
			}
			clone.ResolveTopOfStack()
			if !slices.Equal(clone.EffectiveColors(victim.ID), []core.Color{core.Blue}) {
				t.Fatal("clone lost binding")
			}
			g.ResolveStack()
			p := g.FindPermanent(victim.SourceID)
			if p == nil || !slices.Equal(p.Colors(), []core.Color{core.Blue}) {
				t.Fatal("spell color did not reach permanent")
			}
		})
	}
}
