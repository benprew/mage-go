package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestColorsSpentToCastThisSpellResolutionAndIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	spell := NewInstant("Converge test", "", Spell(GainLifeFor(You, ColorsSpentToCastThisSpell())))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	original := g.StackObjects()[0]
	original.CastContext.ColorsSpent = map[core.Color]int{core.Red: 2, core.Blue: 1, core.Colorless: 3, core.Green: 0}
	spellCopy := g.CopyStackObjectDirect(original, b.PlayerID(), false)
	spellCopy.CastContext.ColorsSpent[core.Green] = 1
	clone := g.Clone()
	clone.StackObjects()[0].CastContext.ColorsSpent[core.White] = 1

	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 23 {
		t.Fatalf("original life totals = %d, %d", a.Life(), b.Life())
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 23 || clone.GetPlayer(b.PlayerID()).Life() != 23 {
		t.Fatalf("clone life totals = %d, %d", clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life())
	}
}

func TestColorsSpentToCastThisSpellAbsentAndDeclaration(t *testing.T) {
	value := ColorsSpentToCastThisSpell()
	if got := value.resolveValue(&EffectContext{Game: NewGame(NewBasePlayer("A"), NewBasePlayer("B"))}); got != 0 {
		t.Fatalf("absent cast context = %d", got)
	}
	if got := InspectDefinition(value); got.Role != "resolution-value" || !strings.Contains(got.Text, "colors") {
		t.Fatalf("inspection = %#v", got)
	}
	for _, tc := range []struct {
		name string
		card Card
		want string
	}{
		{"spell", NewInstant("Spell", "", Spell(GainLifeFor(You, value))), ""},
		{"activation", NewArtifact("Activation", "", WithAbility(Activate(Costs(), GainLifeFor(You, value)))), "spell ability"},
		{"trigger", NewArtifact("Trigger", "", WithAbility(WhenEvent(Enters(This), GainLifeFor(You, value)))), "spell ability"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCardDefinition(tc.card.Name(), tc.card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
}

func TestColorsSpentToCastThisSpellPaidCastAndFailedPayment(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	spell := NewInstant("Paid converge", "{B}{U}", Spell(GainLifeFor(You, ColorsSpentToCastThisSpell())))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Black, 1)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err == nil {
		t.Fatal("cast without blue mana succeeded")
	}
	if g.StackSize() != 0 || a.Life() != 20 {
		t.Fatal("failed payment changed stack or life")
	}
	a.ManaPool().Add(core.Blue, 1)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("paid colors gave life %d", a.Life())
	}
}
