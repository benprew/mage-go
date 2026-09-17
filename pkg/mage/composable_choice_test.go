package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestTypedPermanentChoice(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	small := NewCreature("Small", "", 1, 1)
	large := NewCreature("Large", "", 4, 4)
	for _, c := range []Card{small, large} {
		c.SetOwner(b.PlayerID())
		g.PutOnBattlefield(c, b.PlayerID())
	}
	selected := BindPermanent("least-power creature")
	effect := Sequence(ChoosePermanent(You, LowestPower(MatchingPermanents(IsCreature)), selected), Destroy(From(selected)))
	if err := ValidateCardDefinition("Choice", NewSorcery("Choice", "", WithAbility(Spell(effect)))); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(small.ID()) != nil || g.FindPermanent(large.ID()) == nil {
		t.Fatal("choice did not use least power")
	}
}

func TestTypedChoiceClearsAbsentResult(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Source", "", 1, 1)
	c.SetOwner(a.PlayerID())
	g.PutOnBattlefield(c, a.PlayerID())
	saved := BindPermanent("chosen")
	effect := Sequence(Snapshot(This, saved), ChoosePermanent(You, MatchingPermanents(IsArtifact), saved), Destroy(From(saved)))
	if err := ApplyEffect(g, effect, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(c.ID()) == nil {
		t.Fatal("empty choice retained an earlier result")
	}
}
