package mage

import "testing"

func TestRevealTopCardsOfAllLibraries(t *testing.T) {
	a := NewBasePlayer("A")
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewEnchantment("Shared Library Reveal", "{1}{U}",
		WithStaticAbility(RevealTopCardsOfAllLibraries()),
	)
	source.SetOwner(a.PlayerID())
	perm := g.PutOnBattlefield(source, a.PlayerID())
	g.ApplyEffects()

	if !g.IsTopCardRevealed(a.PlayerID()) {
		t.Error("expected the controller's library top to be revealed")
	}
	if !g.IsTopCardRevealed(b.PlayerID()) {
		t.Error("expected the opponent's library top to be revealed")
	}

	g.DestroyPermanent(perm)
	g.ApplyEffects()
	if g.IsTopCardRevealed(a.PlayerID()) || g.IsTopCardRevealed(b.PlayerID()) {
		t.Error("expected library tops to stop being revealed after the source leaves")
	}
}
