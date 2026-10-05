package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRemoveEachKeywordSelectionAndLifetime(t *testing.T) {
	for _, captured := range []bool{false, true} {
		t.Run(map[bool]string{false: "live", true: "captured"}[captured], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			rule := RemoveEachKeyword(MatchingPermanents(IsCreature), core.Flying)
			opts := []CardOption{}
			if !captured {
				opts = append(opts, WithAbility(Static(While(ControlsNone(Named("Never")), rule))))
			}
			source := g.PutOnBattlefield(NewArtifact("Rule", "", opts...), a.PlayerID())
			card := NewCreature("First", "", 2, 3, WithKeyword(core.Flying), WithKeyword(core.Vigilance))
			card.SetOwner(b.PlayerID())
			first := g.PutOnBattlefield(card, b.PlayerID())
			artifact := g.PutOnBattlefield(NewArtifact("Noncreature", "", WithKeyword(core.Flying)), b.PlayerID())
			if captured {
				if err := ApplyEffect(g, Until(core.Indefinite, rule), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if g.FindPermanent(first.ID()).HasKeyword(core.Flying) || !g.FindPermanent(first.ID()).HasKeyword(core.Vigilance) || !g.FindPermanent(artifact.ID()).HasKeyword(core.Flying) {
				t.Fatal("wrong keyword selection")
			}
			later := g.PutOnBattlefield(NewCreature("Later", "", 1, 1, WithKeyword(core.Flying)), a.PlayerID())
			if g.FindPermanent(later.ID()).HasKeyword(core.Flying) != captured {
				t.Fatal("wrong collection capture")
			}
			clone := g.Clone()
			g.RemoveFromBattlefield(source)
			g.ApplyContinuousEffects()
			if g.FindPermanent(first.ID()).HasKeyword(core.Flying) == captured || clone.FindPermanent(first.ID()).HasKeyword(core.Flying) {
				t.Fatal("wrong source lifetime or clone isolation")
			}
			if captured {
				g.PhaseIn(g.PhaseOut(first.ID()))
				if g.FindPermanent(first.ID()).HasKeyword(core.Flying) {
					t.Fatal("phasing lost captured removal")
				}
				g.BouncePermanentToHand(g.FindPermanent(first.ID()))
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
				if !g.FindPermanent(first.ID()).HasKeyword(core.Flying) {
					t.Fatal("removal followed new incarnation")
				}
			}
		})
	}
}

func TestRemoveEachKeywordSourceAbilityDependency(t *testing.T) {
	for _, collection := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		source := g.PutOnBattlefield(NewArtifact("Rule", "", WithAbility(Static(While(ControlsNone(Named("Never")), RemoveEachKeyword(MatchingPermanents(IsCreature), core.Flying))))), a.PlayerID())
		target := g.PutOnBattlefield(NewCreature("Flyer", "", 2, 3, WithKeyword(core.Flying)), b.PlayerID())
		removal := RemoveAbilities(This)
		if collection {
			removal = RemoveAbilitiesEach(MatchingPermanents(Named("Rule")))
		}
		if err := ApplyEffect(g, Until(core.EndOfTurn, removal), source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if !g.FindPermanent(target.ID()).HasKeyword(core.Flying) {
			t.Fatal("static rule did not depend on source ability removal")
		}
		g.effects.RemoveEndOfTurn()
		g.ApplyContinuousEffects()
		if g.FindPermanent(target.ID()).HasKeyword(core.Flying) {
			t.Fatal("restored source did not remove flying")
		}
	}
}
