package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestMillDrawReplacementBranches(t *testing.T) {
	for _, state := range []string{"empty", "discard", "canceled", "library", "zero", "empty-library"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetTurn(3)
			g.SetStep(core.Upkeep)
			player := BindPlayer("drawing player")
			body := DiscardOneOrElse(FromPlayer(player), DrawFor(FromPlayer(player), Fixed(1)), Mill(FromPlayer(player), Fixed(1)))
			card := NewEnchantment("Draw rule", "", WithAbility(Static(ReplaceDraw(ExceptFirstDrawStepDraw, player, body))))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			if state != "empty-library" {
				b.AddToLibrary(NewLand("Top"))
			}
			if state == "discard" || state == "canceled" || state == "library" {
				b.AddToHand(NewLand("Discard"))
			}
			if state == "canceled" {
				g.effects.AddReplacement(&discardToHandReplacement{})
			}
			if state == "library" {
				g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
			}
			if state == "zero" {
				g.AddMillReplacement(source.ID(), func(_ *Game, _ uuid.UUID, _ int) int { return 0 })
			}
			clone := g.Clone()
			clone.PlayerDrawCard(clone.GetPlayer(b.PlayerID()))
			p := clone.GetPlayer(b.PlayerID())
			wantHand, wantGrave, wantLibrary := 0, 1, 0
			switch state {
			case "discard", "canceled":
				wantHand = 1
			case "library":
				wantHand, wantGrave, wantLibrary = 1, 0, 1
			case "zero":
				wantGrave, wantLibrary = 0, 1
			case "empty-library":
				wantGrave = 0
			}
			if len(p.Hand()) != wantHand || len(p.Graveyard()) != wantGrave || len(p.Library()) != wantLibrary {
				t.Fatalf("hand/graveyard/library=%d/%d/%d want=%d/%d/%d", len(p.Hand()), len(p.Graveyard()), len(p.Library()), wantHand, wantGrave, wantLibrary)
			}
			if len(clone.resolution.playerBindings) != 0 || len(clone.resolution.drawReplacementRules) != 0 {
				t.Fatal("replacement frame escaped")
			}
			if len(b.Graveyard()) != 0 {
				t.Fatal("clone changed original")
			}
			clone.CheckStateBasedActions()
			if clone.IsGameOver() {
				t.Fatal("replaced empty-library draw must not cause a loss")
			}
		})
	}
}
