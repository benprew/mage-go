package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTargetManaValueXQueries(t *testing.T) {
	g, a, b := newSpellCopyGame()
	source := NewInstant("Source", "", Spell(GainLifeFor(You, Fixed(1))))
	target := DeclarePermanentTarget("artifact", IsArtifact, TargetManaValueEqualX())
	zero := g.PutOnBattlefield(NewArtifact("Zero", "{X}"), b.PlayerID())
	two := g.PutOnBattlefield(NewArtifact("Two", "{2}"), b.PlayerID())
	for _, x := range []int{0, 2, 3, 0} {
		g.resolution.SetX(x)
		got := target.Possible(a.PlayerID(), source, g)
		for _, p := range []*Permanent{zero, two} {
			want := p.Card.ManaCost().CMC() == x
			if slices.Contains(got, p.ID()) != want {
				t.Fatalf("X=%d targets=%v", x, got)
			}
		}
	}
	g.resolution.SetX(2)
	g.PhaseOut(two.ID())
	if slices.Contains(target.Possible(a.PlayerID(), source, g), two.ID()) {
		t.Fatal("phased target accepted")
	}
	g.PhaseIn([]uuid.UUID{two.ID()})
	g.FindPermanent(two.ID()).GrantBaseAttr(core.Shroud)
	if slices.Contains(target.Possible(a.PlayerID(), source, g), two.ID()) {
		t.Fatal("shroud ignored")
	}
	spell := NewInstant("Variable", "{X}{X}{U}", Spell(GainLifeFor(You, Fixed(1))))
	obj := &StackObject{ID: uuid.New(), Card: spell, Controller: b.PlayerID(), SourceID: spell.ID()}
	g.GetStack().Push(obj)
	obj.XValue = 3
	st := DeclareSpellTargetManaValueX("spell")
	for _, x := range []int{1, 3, 7} {
		g.resolution.SetX(x)
		if slices.Contains(st.Possible(a.PlayerID(), source, g), obj.ID) != (x == 7) {
			t.Fatalf("spell mana value at X=%d", x)
		}
	}
	if err := ValidateCardDefinition("Valid", NewInstant("Valid", "{X}", Spell(Counter(st), WithTargets(st)))); err != nil {
		t.Fatal(err)
	}
}

func TestTargetManaValueXAnnouncementAndCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	target := DeclarePermanentTarget("artifact", IsArtifact, TargetManaValueEqualX())
	card := NewInstant("Variable tap", "{X}", Spell(TapSubject(target), WithTargets(target)))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Colorless, 2)
	wrong := g.PutOnBattlefield(NewArtifact("Wrong", "{1}"), b.PlayerID())
	victim := g.PutOnBattlefield(NewArtifact("Victim", "{2}"), b.PlayerID())
	other := g.PutOnBattlefield(NewArtifact("Other", "{2}"), b.PlayerID())
	g.resolution.SetX(9)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{wrong.ID()}, 2); err == nil {
		t.Fatal("wrong mana value accepted")
	}
	if g.resolution.X() != 9 || a.ManaPool().TotalMana() != 2 {
		t.Fatal("failed announcement changed frame or payment")
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim.ID()}, 2); err != nil {
		t.Fatal(err)
	}
	if g.resolution.X() != 9 {
		t.Fatal("announcement leaked X")
	}
	copyObj := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
	if copyObj == nil || copyObj.XValue != 2 {
		t.Fatal("copy lost X")
	}
	candidates := g.possibleTargetsWithX(target, a.PlayerID(), card, copyObj.XValue)
	if slices.Contains(candidates, wrong.ID()) || !slices.Contains(candidates, other.ID()) || g.resolution.X() != 9 {
		t.Fatal("copy candidate scope")
	}
	clone := g.Clone()
	g.ResolveStack()
	if !g.FindPermanent(victim.ID()).Tapped || clone.FindPermanent(victim.ID()).Tapped {
		t.Fatal("clone isolation")
	}
	clone.ResolveStack()
	if !clone.FindPermanent(victim.ID()).Tapped {
		t.Fatal("clone lost target restriction")
	}
}
