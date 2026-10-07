package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPublicVisibilityLifecycle(t *testing.T) {
	for _, hands := range []bool{false, true} {
		for _, change := range []string{"depart", "phase", "remove", "control", "clone", "multiple"} {
			t.Run(change, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				rule := RevealAllLibraryTops()
				if hands {
					rule = RevealAllHands()
				}
				visible := func(game *Game, id uuid.UUID) bool {
					if hands {
						return game.IsHandRevealed(id)
					}
					return game.IsTopCardRevealed(id)
				}
				card := NewEnchantment("Public hands", "", WithAbility(Static(rule)))
				if err := ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				p := g.PutOnBattlefield(card, a.PlayerID())
				g.ApplyContinuousEffects()
				if !visible(g, a.PlayerID()) || !visible(g, b.PlayerID()) {
					t.Fatal("hands remain private")
				}
				clone := g.Clone()
				if !visible(clone, a.PlayerID()) || !visible(clone, b.PlayerID()) {
					t.Fatal("clone lost visibility before layer pass")
				}
				switch change {
				case "depart":
					g.RemoveFromBattlefield(p)
				case "phase":
					g.PhaseOut(p.ID())
				case "remove":
					if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), p.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "clone":
					clone.RemoveFromBattlefield(clone.FindPermanent(p.ID()))
					clone.ApplyContinuousEffects()
				case "multiple":
					g.PutOnBattlefield(NewEnchantment("Second", "", WithAbility(Static(rule))), b.PlayerID())
					g.RemoveFromBattlefield(p)
				}
				g.ApplyContinuousEffects()
				want := change == "control" || change == "clone" || change == "multiple"
				if visible(g, a.PlayerID()) != want || visible(g, b.PlayerID()) != want {
					t.Fatal("wrong visibility lifetime")
				}
				if visible(clone, a.PlayerID()) != (change != "clone") {
					t.Fatal("visibility crossed clones")
				}
				if change == "phase" {
					g.PhaseIn([]uuid.UUID{p.ID()})
					g.ApplyContinuousEffects()
					if !visible(g, a.PlayerID()) || !visible(g, b.PlayerID()) {
						t.Fatal("phase-in did not restore visibility")
					}
				}
				if change == "remove" {
					g.RunStepWithPriority(core.Cleanup)
					if !visible(g, a.PlayerID()) || !visible(g, b.PlayerID()) {
						t.Fatal("cleanup did not restore visibility")
					}
				}
			})
		}
	}
}
