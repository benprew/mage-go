package gametest

import (
	"sync"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

var predefinedTokensOnce sync.Once

func registerPredefinedTokenTestCards() {
	predefinedTokensOnce.Do(func() {
		if !mage.CardRegistered("Predef Make Treasure") {
			mage.Register("Predef Make Treasure", func() mage.Card {
				return mage.NewSorcery("Predef Make Treasure", "{0}",
					mage.NewSpellAbility(mage.CreateTreasureToken()),
				)
			})
		}
		if !mage.CardRegistered("Predef Make Food") {
			mage.Register("Predef Make Food", func() mage.Card {
				return mage.NewSorcery("Predef Make Food", "{0}",
					mage.NewSpellAbility(mage.CreateFoodToken()),
				)
			})
		}
	})
}

// TestPredefinedToken_Treasure_Activates verifies the Treasure token's
// {T}, Sacrifice: Add one mana of any color activated ability (CR 111.10c).
func TestPredefinedToken_Treasure_Activates(t *testing.T) {
	registerPredefinedTokenTestCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Predef Make Treasure")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Predef Make Treasure")
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "Treasure")
	tg.ChooseManaColor(PlayerA, core.Blue)
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Treasure", 0)
	tg.AssertManaProducedAtLeast(PlayerA, core.Blue, 1)
}

// TestPredefinedToken_Treasure_IsColorlessArtifact verifies the Treasure token
// is a colorless artifact with subtype Treasure when created.
func TestPredefinedToken_Treasure_IsColorlessArtifact(t *testing.T) {
	registerPredefinedTokenTestCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Predef Make Treasure")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Predef Make Treasure")
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Treasure", 1)
	assertTokenShape(t, tg, PlayerA, "Treasure", "Treasure")
}

// TestPredefinedToken_Food_Activates verifies the Food token's
// {2}, {T}, Sacrifice: You gain 3 life activated ability (CR 111.10d).
func TestPredefinedToken_Food_Activates(t *testing.T) {
	registerPredefinedTokenTestCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Predef Make Food")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Plains", 2)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Predef Make Food")
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "Food")
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()

	tg.AssertLife(PlayerA, 23)
	tg.AssertPermanentCount(PlayerA, "Food", 0)
}

// TestPredefinedToken_Food_IsColorlessArtifact verifies the Food token is a
// colorless artifact with subtype Food when created.
func TestPredefinedToken_Food_IsColorlessArtifact(t *testing.T) {
	registerPredefinedTokenTestCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Predef Make Food")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Predef Make Food")
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Food", 1)
	assertTokenShape(t, tg, PlayerA, "Food", "Food")
}

// assertTokenShape walks the controller's battlefield to find a permanent with
// the given name and verifies it is an artifact (not a creature) with the
// requested subtype.
func assertTokenShape(t *testing.T, tg *TestGame, p PlayerRef, name, subtype string) {
	t.Helper()
	playerID := tg.getPlayerID(p)
	for _, perm := range tg.AllBattlefield() {
		if perm.Controller != playerID || perm.Card.Name() != name {
			continue
		}
		if !perm.Card.HasType(core.TypeArtifact) {
			t.Errorf("%s token should be an artifact", name)
		}
		if perm.Card.HasType(core.TypeCreature) {
			t.Errorf("%s token should not be a creature", name)
		}
		for _, st := range perm.Card.SubTypes() {
			if st == subtype {
				return
			}
		}
		t.Errorf("%s token missing subtype %s, got %v", name, subtype, perm.Card.SubTypes())
		return
	}
	t.Fatalf("expected to find a %s permanent on %v's battlefield", name, p)
}
