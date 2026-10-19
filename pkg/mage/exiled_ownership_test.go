package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestExiledOwnershipIndependentVisits(t *testing.T) {
	for _, state := range []string{"normal", "missing-first", "returned-first", "disabled", "settled"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := anteEffectTestGame(t)
			first, second := ownedTestCard(b.PlayerID(), "first"), ownedTestCard(b.PlayerID(), "second")
			p, q := g.PutOnBattlefield(first, a.PlayerID()), g.PutOnBattlefield(second, b.PlayerID())
			one, two := BindCards("one"), BindCards("two")
			saved := BindPermanent("target")
			g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{saved: {id: q.ID(), incarnation: q.incarnationID}}
			ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
			if err := ExilePermanentGroupsInto(PermanentExileGroup{SingletonPermanent(This), one}, PermanentExileGroup{SingletonPermanent(From(saved)), two}).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "missing-first", "returned-first":
				g.RemoveFromExile(first.ID())
				if state == "returned-first" {
					g.ExileCard(first, p.ID())
				} else {
					b.AddToHand(first)
				}
			case "disabled":
				g.anteEnabled = false
			case "settled":
				g.anteSettled = true
			}
			clone := g.Clone()
			effect := Sequence(AcquireCardOwnership(SingleExiledCardFrom(one), You), AcquireCardOwnership(SingleExiledCardFrom(two), You))
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			acquired := state != "disabled" && state != "settled"
			if (g.FindCardAnywhere(second.ID()).Owner() == a.PlayerID()) != acquired {
				t.Fatal("second assignment depended on first")
			}
			if (g.FindCardAnywhere(first.ID()).Owner() == a.PlayerID()) != (state == "normal") {
				t.Fatal("assignment followed wrong visit")
			}
			if clone.FindCardAnywhere(second.ID()).Owner() != b.PlayerID() || g.CardZone(second.ID()) != core.ZoneExile {
				t.Fatal("assignment changed clone or zone")
			}
		})
	}
}
