package gametest

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// registerScryN registers a sorcery whose only effect is Scry n.
func registerScryN(name string, n int) {
	if mage.CardRegistered(name) {
		return
	}
	mage.Register(name, func() mage.Card {
		return mage.NewSorcery(name, "{U}",
			mage.NewSpellAbility(mage.Scry(mage.Fixed(n))),
		)
	})
}

// Scry 1, default test player (no scripted decision) keeps the card on top.
func TestScry1Default(t *testing.T) {
	registerScryN("Scry One Test", 1)

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Scry One Test")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Mountain")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Forest")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Island")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Scry One Test")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertLibraryTop(PlayerA, "Mountain", "Forest", "Island")
}

// Scry 1, scripted to send the seen card to the bottom.
func TestScry1ToBottom(t *testing.T) {
	registerScryN("Scry One Bottom", 1)

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Scry One Bottom")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Mountain") // top
	tg.AddCard(core.ZoneLibrary, PlayerA, "Forest")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Island") // bottom
	tg.ChooseScry(PlayerA, []string{"Mountain"}, nil)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Scry One Bottom")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertLibraryTop(PlayerA, "Forest", "Island", "Mountain")
}

// Scry 2, mixed: keep one on top, send one to the bottom.
func TestScry2Mixed(t *testing.T) {
	registerScryN("Scry Two Mixed", 2)

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Scry Two Mixed")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Mountain") // top, will be seen
	tg.AddCard(core.ZoneLibrary, PlayerA, "Forest")   // also seen
	tg.AddCard(core.ZoneLibrary, PlayerA, "Island")   // remains in middle
	tg.AddCard(core.ZoneLibrary, PlayerA, "Plains")   // remains at bottom
	tg.ChooseScry(PlayerA, []string{"Mountain"}, []string{"Forest"})
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Scry Two Mixed")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertLibraryTop(PlayerA, "Forest", "Island", "Plains", "Mountain")
}

// Scry 2, reorder both kept cards on top.
func TestScry2Reorder(t *testing.T) {
	registerScryN("Scry Two Reorder", 2)

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Scry Two Reorder")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Mountain") // top
	tg.AddCard(core.ZoneLibrary, PlayerA, "Forest")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Island")
	tg.ChooseScry(PlayerA, nil, []string{"Forest", "Mountain"})
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Scry Two Reorder")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertLibraryTop(PlayerA, "Forest", "Mountain", "Island")
}

// Scry 3, send all three seen cards to the bottom in a chosen order.
func TestScry3AllBottom(t *testing.T) {
	registerScryN("Scry Three Bottom", 3)

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Scry Three Bottom")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Mountain") // top, seen
	tg.AddCard(core.ZoneLibrary, PlayerA, "Forest")   // seen
	tg.AddCard(core.ZoneLibrary, PlayerA, "Island")   // seen
	tg.AddCard(core.ZoneLibrary, PlayerA, "Plains")   // remains
	tg.AddCard(core.ZoneLibrary, PlayerA, "Swamp")    // remains
	// Bottom order: Mountain placed first (becomes the third-from-bottom),
	// then Forest, then Island (Island is the new bottom card).
	tg.ChooseScry(PlayerA, []string{"Mountain", "Forest", "Island"}, nil)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Scry Three Bottom")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertLibraryTop(PlayerA, "Plains", "Swamp", "Mountain", "Forest", "Island")
}

// Scry 3 with only 1 card left in the library: scry processes whatever it can.
func TestScryFewerThanN(t *testing.T) {
	registerScryN("Scry More Than Lib", 3)

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Scry More Than Lib")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Mountain") // only card
	tg.ChooseScry(PlayerA, []string{"Mountain"}, nil)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Scry More Than Lib")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertLibraryTop(PlayerA, "Mountain")
}

// Direct engine-level smoke test for Game.PerformScry; exercises the primitive
// without going through spell resolution.
func TestPerformScryPrimitive(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneLibrary, PlayerA, "Mountain")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Forest")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Island")

	tg.ChooseScry(PlayerA, []string{"Mountain"}, []string{"Forest"})
	seen := tg.PerformScry(tg.GetPlayer(PlayerA), 2)
	if seen != 2 {
		t.Fatalf("PerformScry returned %d, want 2", seen)
	}
	tg.AssertLibraryTop(PlayerA, "Forest", "Island", "Mountain")
}
