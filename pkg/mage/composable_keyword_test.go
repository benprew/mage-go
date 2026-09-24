package mage

import (
	"slices"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestComposableKeywordSourceExit(t *testing.T) {
	for _, keyword := range []Attr{Flying, Banding} {
		t.Run(keyword.String(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source, recipient := NewArtifact("Source", ""), NewCreature("Recipient", "", 2, 2)
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutOnBattlefield(recipient, a.PlayerID())
			subjects := ControlledPermanents(IsCreature)
			body := Until(EndOfTurn, GrantEachKeyword(subjects, keyword),
				ModifyEachPT(subjects, Fixed(1), Fixed(0)), SetEachColors(subjects, Green))
			if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			g.ApplyContinuousEffects()
			p := g.FindPermanent(recipient.ID())
			if !p.HasKeyword(keyword) || p.CurrentPower(g) != 3 || !slices.Contains(p.Colors(), Green) {
				t.Fatal("fixed-duration grant ended when its source left")
			}
			g.PutOnBattlefield(source, b.PlayerID())
			g.ApplyContinuousEffects()
			clone := g.Clone()
			clone.effects.RemoveEndOfTurn()
			clone.ApplyContinuousEffects()
			p = clone.FindPermanent(recipient.ID())
			if p.HasKeyword(keyword) || p.CurrentPower(clone) != 2 || slices.Contains(p.Colors(), Green) || !g.FindPermanent(recipient.ID()).HasKeyword(keyword) {
				t.Fatal("expiry did not stay in the cloned game")
			}
		})
	}
}

func TestComposableKeywordLayerOrder(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	creature := NewCreature("Creature", "", 2, 2)
	g.PutOnBattlefield(creature, a.PlayerID())
	body := Until(EndOfTurn, GrantKeywordModification(This, Flying))
	if err := ApplyEffect(g, body, creature.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.AddContinuousEffect(RemoveAllAbilities(creature.ID(), EndOfTurn))
	g.ApplyContinuousEffects()
	if g.FindPermanent(creature.ID()).HasKeyword(Flying) {
		t.Fatal("later ability removal did not remove flying")
	}
	if err := ApplyEffect(g, body, creature.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(creature.ID()).HasKeyword(Flying) {
		t.Fatal("later grant did not restore flying")
	}
}

func TestComposableRequiredBlockMetadata(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	body := Until(EndOfTurn, GrantKeywordModification(target, AttrMustBeBlockedIfAble))
	info := InspectDefinition(body)
	if info.Properties.Outcome != OutcomeDetriment || info.Properties.GrantedKeyword != AttrMustBeBlockedIfAble {
		t.Fatalf("required block metadata = %+v", info.Properties)
	}
}
