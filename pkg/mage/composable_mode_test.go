package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestTypedModalTargets(t *testing.T) {
	g, a, b := newSpellCopyGame()
	creature := DeclarePermanentTarget("creature", IsCreature)
	player := DeclarePlayerTarget("player")
	ability := SpellModes(ModeOf("Destroy", Destroy(creature), WithTargets(creature)), ModeOf("Draw", DrawFor(player, Fixed(1)), WithTargets(player)))
	card := NewSorcery("Modes", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition("Modes", card); err != nil {
		t.Fatal(err)
	}
	victim := NewCreature("Victim", "", 2, 2)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	obj, err := g.pushCastSpellObject(castStackObjectOptions{Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{victim.ID()}})
	if err != nil {
		t.Fatal(err)
	}
	if len(obj.TargetSpecs) != 1 {
		t.Fatal("modal stack object lost target declaration")
	}
	g.ResolveStackObject(obj)
	if g.FindPermanent(victim.ID()) != nil {
		t.Fatal("modal target binding did not resolve")
	}
}

func TestTypedModalScope(t *testing.T) {
	first := DeclarePlayerTarget("first")
	second := DeclarePlayerTarget("second")
	ability := SpellModes(ModeOf("First", DrawFor(first, Fixed(1)), WithTargets(first)), ModeOf("Second", DrawFor(first, Fixed(1)), WithTargets(second)))
	if err := ValidateCardDefinition("Scope", NewSorcery("Scope", "", WithAbility(ability))); err == nil {
		t.Fatal("target escaped its mode")
	}
}
