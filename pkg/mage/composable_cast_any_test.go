package mage

import (
	"strings"
	"testing"
)

func TestAnyCastConditionOnceForEitherOrBothFacts(t *testing.T) {
	for _, tc := range []struct {
		name       string
		revealed   bool
		controlled bool
		want       int
	}{
		{"neither", false, false, 20},
		{"reveal", true, false, 23},
		{"control", false, true, 23},
		{"both", true, true, 23},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			spell := NewInstant("Union", "", Spell(IfCast(AnyCast(
				RevealedSubtypeAtCast("Dragon"), ControlledCreatureSubtypeAtCast("Dragon")),
				GainLifeFor(You, Fixed(3)))))
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
				t.Fatal(err)
			}
			stack := g.StackObjects()[0]
			stack.CastContext.ControllerCreatureSubtypesAtCast["Dragon"] = tc.controlled
			if tc.revealed {
				stack.CastContext.RevealedAtCast = append(stack.CastContext.RevealedAtCast,
					NewCreature("Dragon", "", 1, 1, WithSubTypes("Dragon")))
			}
			spellCopy := g.CopyStackObjectDirect(stack, b.PlayerID(), false)
			clone := g.Clone()
			spellCopy.CastContext.ControllerCreatureSubtypesAtCast["Dragon"] = false
			spellCopy.CastContext.RevealedAtCast = nil
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			if a.Life() != tc.want || clone.GetPlayer(a.PlayerID()).Life() != tc.want {
				t.Fatalf("life=%d clone=%d want=%d", a.Life(), clone.GetPlayer(a.PlayerID()).Life(), tc.want)
			}
		})
	}
}

func TestAnyCastConditionDeclaration(t *testing.T) {
	good := AnyCast(RevealedSubtypeAtCast("Dragon"), ControlledCreatureSubtypeAtCast("Dragon"))
	if got := InspectDefinition(good); got.Role != "cast-condition" || len(got.Children) != 2 {
		t.Fatalf("inspection=%+v", got)
	}
	for _, tc := range []struct {
		name      string
		condition CastCondition
		problem   string
	}{
		{"empty", AnyCast(), "at least"},
		{"nil", AnyCast(nil, RevealedSubtypeAtCast("Dragon")), "nil"},
		{"invalid child", AnyCast(RevealedSubtypeAtCast("")), "subtype"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewInstant("Invalid", "", Spell(IfCast(tc.condition, GainLifeFor(You, Fixed(1)))))
			if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.problem) {
				t.Fatalf("validation=%v", err)
			}
		})
	}
	card := NewArtifact("Wrong lifecycle", "", WithAbility(Activate(Costs(), IfCast(good, GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "spell ability") {
		t.Fatalf("validation=%v", err)
	}
}

func TestAnyCastCopiesInputConditions(t *testing.T) {
	conditions := []CastCondition{RevealedSubtypeAtCast("Dragon")}
	union := AnyCast(conditions...)
	conditions[0] = ControlledCreatureSubtypeAtCast("Elf")
	ctx := &CastContext{RevealedAtCast: []Card{NewCreature("Dragon", "", 1, 1, WithSubTypes("Dragon"))}}
	if !union.checkCast(ctx) || union.checkCast(nil) {
		t.Fatal("cast condition changed with caller slice or matched without a cast")
	}
}
