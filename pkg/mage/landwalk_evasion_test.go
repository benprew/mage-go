package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLandwalkEvasionPhasingAndController(t *testing.T) {
	for _, kw := range []core.Attr{core.Plainswalk, core.Islandwalk, core.Swampwalk, core.Mountainwalk, core.Forestwalk, core.Desertwalk, core.LegendaryLandwalk} {
		t.Run(kw.String(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			attacker := g.PutOnBattlefield(NewCreature("Walker", "", 1, 1, WithKeyword(kw)), a.PlayerID())
			card := NewLand("Matching land", WithSubTypes(kw.LandwalkSubtype()))
			if kw == core.LegendaryLandwalk {
				card = NewLand("Matching land", WithSuperTypes(core.SuperLegendary))
			}
			card.SetOwner(a.PlayerID())
			land := g.PutOnBattlefield(card, b.PlayerID())
			if !HasLandwalkEvasion(attacker, b.PlayerID(), g) || HasLandwalkEvasion(attacker, a.PlayerID(), g) {
				t.Fatal("landwalk used owner instead of defender control")
			}
			clone := g.Clone()
			phased := clone.PhaseOut(land.ID())
			clone.ApplyContinuousEffects()
			if HasLandwalkEvasion(clone.FindPermanent(attacker.ID()), b.PlayerID(), clone) {
				t.Fatal("phased land enabled evasion")
			}
			if !HasLandwalkEvasion(g.FindPermanent(attacker.ID()), b.PlayerID(), g) {
				t.Fatal("clone phase-out changed original")
			}
			clone.PhaseIn(phased)
			clone.ApplyContinuousEffects()
			if !HasLandwalkEvasion(clone.FindPermanent(attacker.ID()), b.PlayerID(), clone) {
				t.Fatal("land phase-in did not restore evasion")
			}
			g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: land.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			if HasLandwalkEvasion(g.FindPermanent(attacker.ID()), b.PlayerID(), g) || !HasLandwalkEvasion(g.FindPermanent(attacker.ID()), a.PlayerID(), g) {
				t.Fatal("landwalk did not follow land control")
			}
			g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
			if HasLandwalkEvasion(g.FindPermanent(attacker.ID()), a.PlayerID(), g) {
				t.Fatal("departed land enabled evasion")
			}
			g.PutOnBattlefield(card, b.PlayerID())
			if !HasLandwalkEvasion(g.FindPermanent(attacker.ID()), b.PlayerID(), g) {
				t.Fatal("returned land did not enable evasion")
			}
		})
	}
}
