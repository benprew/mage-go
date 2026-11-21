package gametest

import (
	"errors"
	"slices"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// CastAsThoughHadFlash grants the controller permission to cast cards
// matching the filter at instant speed. Used by Rattlechains for Spirit
// spells. Without the grant, casting a sorcery-speed creature during the
// opponent's turn would fail with ErrSorcerySpeed.
func TestCastAsThoughHadFlash_LiftsSorcerySpeedGate(t *testing.T) {
	const cardName = "CASTPERM Spirit Granter"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() *mage.Card {
			return mage.NewEnchantment(cardName, "{0}",
				mage.WithStaticAbility(mage.CastAsThoughHadFlash(
					mage.ObjectAnd(mage.ObjectHasType(core.TypeCreature), mage.ObjectHasSubtype("Spirit")),
				)),
			)
		})
	}

	const spiritName = "CASTPERM Spirit Token Card"
	if !mage.CardRegistered(spiritName) {
		mage.Register(spiritName, func() *mage.Card {
			return mage.NewCreature(spiritName, "{0}", 1, 1, mage.WithSubTypes("Spirit"))
		})
	}

	// PlayerA owns the granter. PlayerB casts a creature during A's end step.
	// Without the grant, casting the Spirit at instant speed would fail; we
	// assert the spell resolved by checking the permanent count.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerB, cardName)
	tg.AddCard(core.ZoneHand, PlayerB, spiritName)
	tg.CastSpell(1, core.EndStep, PlayerB, spiritName)
	tg.StopAt(2, core.Untap)
	tg.Execute()

	tg.AssertPermanentCount(PlayerB, spiritName, 1)
}

// CastAsThoughHadFlash does NOT lift the gate for cards outside the filter.
func TestCastAsThoughHadFlash_FilterRestricts(t *testing.T) {
	const cardName = "CASTPERM Spirit-only Granter"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() *mage.Card {
			return mage.NewEnchantment(cardName, "{0}",
				mage.WithStaticAbility(mage.CastAsThoughHadFlash(
					mage.ObjectAnd(mage.ObjectHasType(core.TypeCreature), mage.ObjectHasSubtype("Spirit")),
				)),
			)
		})
	}

	const goblinName = "CASTPERM Goblin"
	if !mage.CardRegistered(goblinName) {
		mage.Register(goblinName, func() *mage.Card {
			return mage.NewCreature(goblinName, "{0}", 1, 1, mage.WithSubTypes("Goblin"))
		})
	}

	// PlayerB tries to cast a Goblin (not a Spirit) during A's end step.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerB, cardName)
	tg.AddCard(core.ZoneHand, PlayerB, goblinName)
	tg.CastSpell(1, core.EndStep, PlayerB, goblinName)
	tg.StopAt(2, core.Untap)
	tg.Execute()

	// Cast attempt failed — the goblin is still in hand and there is no
	// goblin permanent on the battlefield.
	tg.AssertPermanentCount(PlayerB, goblinName, 0)
}

func TestCastAsThoughHadFlashUsesCurrentGame(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.Upkeep)
	g.PutOnBattlefield(mage.NewEnchantment("Flash Grant", "{0}", mage.WithStaticAbility(
		mage.CastAsThoughHadFlash(mage.ObjectOwnedBy(mage.SelectActivePlayer())),
	)), a.PlayerID())
	card := mage.NewCreature("Creature", "{0}", 1, 1)
	a.AddToHand(card)

	g.SetActivePlayerIndex(1)
	if slices.Contains(g.GetCastableSpells(a.PlayerID()), card) {
		t.Fatal("grant admitted a card owned by the nonactive player")
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); !errors.Is(err, mage.ErrSorcerySpeed) {
		t.Fatalf("cast with the grant inactive for this card: %v", err)
	}

	g.SetActivePlayerIndex(0)
	if !slices.Contains(g.GetCastableSpells(a.PlayerID()), card) {
		t.Fatal("grant excluded a card owned by the active player")
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatalf("cast with the grant active for this card: %v", err)
	}
}

func TestGlobalFlashGrantBindsSourceController(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.Upkeep)
	g.PutOnBattlefield(mage.NewEnchantment("Global Flash Grant", "{0}", mage.WithStaticAbility(
		mage.CastAsThoughHadFlashGlobal(mage.ObjectOwnedBy(mage.SelectController())),
	)), a.PlayerID())
	a.AddToHand(mage.NewCreature("First Creature", "{0}", 1, 1))
	b.AddToHand(mage.NewCreature("Second Creature", "{0}", 1, 1))

	if got := g.GetCastableSpells(a.PlayerID()); len(got) != 1 {
		t.Errorf("grant excluded a card owned by its controller: %v", got)
	}
	if got := g.GetCastableSpells(b.PlayerID()); len(got) != 0 {
		t.Errorf("grant admitted a card owned by its controller's opponent: %v", got)
	}
}
