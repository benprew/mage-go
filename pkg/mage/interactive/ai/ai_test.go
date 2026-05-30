package ai

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

// ── Mulligan ─────────────────────────────────────────────────────────────────

func TestShouldMulligan_ZeroLands(t *testing.T) {
	ai := NewAIPlayer("Bot", &dummyStrategy{})
	for range 7 {
		c := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		c.SetOwner(ai.PlayerID())
		ai.AddToHand(c)
	}
	if !ai.ShouldMulligan(7) {
		t.Error("should mulligan with 0 lands in a 7-card hand")
	}
}

func TestShouldMulligan_SevenLands(t *testing.T) {
	ai := NewAIPlayer("Bot", &dummyStrategy{})
	for range 7 {
		c := mage.NewLand("Forest")
		c.SetOwner(ai.PlayerID())
		ai.AddToHand(c)
	}
	if !ai.ShouldMulligan(7) {
		t.Error("should mulligan with 7 lands in a 7-card hand")
	}
}

func TestShouldMulligan_ThreeLandsFourSpells(t *testing.T) {
	ai := NewAIPlayer("Bot", &dummyStrategy{})
	for range 3 {
		c := mage.NewLand("Forest")
		c.SetOwner(ai.PlayerID())
		ai.AddToHand(c)
	}
	for range 4 {
		c := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		c.SetOwner(ai.PlayerID())
		ai.AddToHand(c)
	}
	if ai.ShouldMulligan(7) {
		t.Error("should keep with 3 lands and 4 castable spells")
	}
}

func TestShouldMulligan_OneLandSixSevenDrops(t *testing.T) {
	ai := NewAIPlayer("Bot", &dummyStrategy{})
	c := mage.NewLand("Forest")
	c.SetOwner(ai.PlayerID())
	ai.AddToHand(c)
	for range 6 {
		s := mage.NewCreature("Wurm", "{5}{G}{G}", 7, 7)
		s.SetOwner(ai.PlayerID())
		ai.AddToHand(s)
	}
	if !ai.ShouldMulligan(7) {
		t.Error("should mulligan with 1 land and no castable spells")
	}
}

func TestShouldMulligan_TwoLandsFourTwoDrops(t *testing.T) {
	ai := NewAIPlayer("Bot", &dummyStrategy{})
	for range 2 {
		c := mage.NewLand("Forest")
		c.SetOwner(ai.PlayerID())
		ai.AddToHand(c)
	}
	for range 4 {
		c := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		c.SetOwner(ai.PlayerID())
		ai.AddToHand(c)
	}
	s := mage.NewCreature("Centaur", "{2}{G}", 3, 3)
	s.SetOwner(ai.PlayerID())
	ai.AddToHand(s)
	if ai.ShouldMulligan(7) {
		t.Error("should keep with 2 lands and castable spells")
	}
}

func TestShouldMulligan_SixCardHandOneLand(t *testing.T) {
	ai := NewAIPlayer("Bot", &dummyStrategy{})
	c := mage.NewLand("Forest")
	c.SetOwner(ai.PlayerID())
	ai.AddToHand(c)
	for range 5 {
		s := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		s.SetOwner(ai.PlayerID())
		ai.AddToHand(s)
	}
	if ai.ShouldMulligan(6) {
		t.Error("should keep 6-card hand with 1 land")
	}
}

func TestShouldMulligan_FiveCardHand(t *testing.T) {
	ai := NewAIPlayer("Bot", &dummyStrategy{})
	for range 5 {
		c := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		c.SetOwner(ai.PlayerID())
		ai.AddToHand(c)
	}
	if ai.ShouldMulligan(5) {
		t.Error("should always keep a 5-card hand")
	}
}

func TestMulligan_ShufflesAndDrawsFewerCards(t *testing.T) {
	ai := NewAIPlayer("Bot", &dummyStrategy{})
	for range 30 {
		c := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		c.SetOwner(ai.PlayerID())
		ai.AddToLibrary(c)
	}
	for range 7 {
		ai.DrawCard()
	}
	if len(ai.Hand()) != 7 {
		t.Fatalf("expected 7 cards in hand, got %d", len(ai.Hand()))
	}
	if len(ai.Library()) != 23 {
		t.Fatalf("expected 23 cards in library, got %d", len(ai.Library()))
	}

	ai.Mulligan()

	if len(ai.Hand()) != 6 {
		t.Errorf("expected 6 cards in hand after mulligan, got %d", len(ai.Hand()))
	}
	if len(ai.Library()) != 24 {
		t.Errorf("expected 24 cards in library after mulligan, got %d", len(ai.Library()))
	}
}

func TestMulliganAI_KeepsGoodHand(t *testing.T) {
	ai := NewAIPlayer("Bot", &dummyStrategy{})
	for range 15 {
		c := mage.NewLand("Forest")
		c.SetOwner(ai.PlayerID())
		ai.AddToLibrary(c)
	}
	for range 15 {
		c := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		c.SetOwner(ai.PlayerID())
		ai.AddToLibrary(c)
	}
	ai.ShuffleLibrary()
	for range 7 {
		ai.DrawCard()
	}

	startHand := len(ai.Hand())
	startLib := len(ai.Library())

	MulliganAI(ai)

	if len(ai.Hand()) < 5 || len(ai.Hand()) > 7 {
		t.Errorf("hand size after mulligan loop should be 5-7, got %d", len(ai.Hand()))
	}
	total := len(ai.Hand()) + len(ai.Library())
	if total != startHand+startLib {
		t.Errorf("total cards changed: started %d, now %d", startHand+startLib, total)
	}
}

// ── ChooseMode ──────────────────────────────────────────────────────────────

func TestAIChooseMode_HealingSalveLowLife(t *testing.T) {
	ai := NewAIPlayer("Bot", &dummyStrategy{})
	ai.SetLife(5)
	got := ai.ChooseMode([]string{"Gain 3 life", "Prevent 3"}, "Healing Salve")
	if got != 0 {
		t.Errorf("ChooseMode(Healing Salve, low life) = %d, want 0", got)
	}
}

func TestAIChooseMode_HealingSalveHighLife(t *testing.T) {
	ai := NewAIPlayer("Bot", &dummyStrategy{})
	ai.SetLife(20)
	got := ai.ChooseMode([]string{"Gain 3 life", "Prevent 3"}, "Healing Salve")
	if got != 1 {
		t.Errorf("ChooseMode(Healing Salve, high life) = %d, want 1", got)
	}
}

func TestAIChooseMode_UnknownCard(t *testing.T) {
	ai := NewAIPlayer("Bot", &dummyStrategy{})
	got := ai.ChooseMode([]string{"A", "B"}, "Unknown Card")
	if got != 0 {
		t.Errorf("ChooseMode(unknown) = %d, want 0", got)
	}
}

// ── NewAIPlayer constructor smoke test ──────────────────────────────────────

func TestNewAIPlayer_StoresStrategy(t *testing.T) {
	s := &dummyStrategy{}
	ai := NewAIPlayer("Bot", s)
	if ai.Name() != "Bot" {
		t.Errorf("name = %q, want Bot", ai.Name())
	}
	if ai.Strategy != s {
		t.Error("AIPlayer should hold the supplied strategy")
	}
}
