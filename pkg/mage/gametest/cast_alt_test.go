package gametest

import (
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

var castAltRegisterOnce sync.Once

func registerCastAltTestCards() {
	castAltRegisterOnce.Do(func() {
		// A simple sorcery that gains 5 life — we use it as the spell that
		// is cast from a non-hand zone.
		if !mage.CardRegistered("Cast-Alt Heal") {
			mage.Register("Cast-Alt Heal", func() mage.Card {
				return mage.NewSorcery("Cast-Alt Heal", "{2}{W}{W}",
					mage.NewSpellAbility(mage.GainLife(5)),
				)
			})
		}
		// A creature spell with a printed mana cost — used to verify free
		// casting from graveyard puts a permanent onto the battlefield.
		if !mage.CardRegistered("Cast-Alt Bear") {
			mage.Register("Cast-Alt Bear", func() mage.Card {
				return mage.NewCreature("Cast-Alt Bear", "{1}{G}", 2, 2,
					mage.WithSubTypes("Bear"),
				)
			})
		}
		// A sorcery with the flashback keyword — used to verify CR 702.34
		// exile-after-resolution semantics.
		if !mage.CardRegistered("Cast-Alt Flashback Heal") {
			mage.Register("Cast-Alt Flashback Heal", func() mage.Card {
				return mage.NewSorcery("Cast-Alt Flashback Heal", "{2}{W}",
					mage.NewSpellAbility(mage.GainLife(3)),
					mage.WithFlashback(core.ParseManaCost("{4}{W}")),
				)
			})
		}
		if !mage.CardRegistered("Cast-Alt Atomic Failure") {
			mage.Register("Cast-Alt Atomic Failure", func() mage.Card {
				return mage.NewSorcery("Cast-Alt Atomic Failure", "{2}{R}",
					mage.NewSpellAbility(mage.GainLife(1)),
					mage.WithAdditionalCost(mage.LifePayCost(100)),
					mage.WithFlashback(core.ParseManaCost("{R}")),
				)
			})
		}
		if !mage.CardRegistered("Cast-Alt Combined Mana") {
			mage.Register("Cast-Alt Combined Mana", func() mage.Card {
				return mage.NewSorcery("Cast-Alt Combined Mana", "{3}",
					mage.NewSpellAbility(mage.GainLife(1)),
					mage.WithAdditionalCost(mage.ManaCostOf("{W}")),
					mage.WithFlashback(core.ParseManaCost("{1}")),
				)
			})
		}
	})
}

func TestCastFromGraveyardWithoutPaying(t *testing.T) {
	registerCastAltTestCards()
	tg := NewTestGame(t)
	pA := tg.GetPlayer(PlayerA)

	// Add the spell directly to the graveyard with no mana available.
	tg.AddCard(core.ZoneGraveyard, PlayerA, "Cast-Alt Heal")
	cardID := pA.Graveyard()[0].ID()

	pA.ManaPool().Clear() // ensure no mana
	if err := tg.CastCardFromZoneWithoutPaying(pA.PlayerID(), cardID, core.ZoneGraveyard, nil, 0); err != nil {
		t.Fatalf("CastCardFromZoneWithoutPaying: %v", err)
	}

	if tg.Game.GetStack().Size() != 1 {
		t.Fatalf("expected stack size 1, got %d", tg.Game.GetStack().Size())
	}

	// Resolve and check life gain + that the card returned to graveyard.
	tg.ResolveStack()
	if pA.Life() != 25 {
		t.Errorf("expected life 25 after Heal resolves, got %d", pA.Life())
	}
	found := false
	for _, c := range pA.Graveyard() {
		if c.Name() == "Cast-Alt Heal" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected Cast-Alt Heal back in graveyard after resolve")
	}
}

func TestCastFromGraveyardCreaturePermanent(t *testing.T) {
	registerCastAltTestCards()
	tg := NewTestGame(t)
	pA := tg.GetPlayer(PlayerA)

	tg.AddCard(core.ZoneGraveyard, PlayerA, "Cast-Alt Bear")
	cardID := pA.Graveyard()[0].ID()
	pA.ManaPool().Clear()

	if err := tg.CastCardFromZoneWithoutPaying(pA.PlayerID(), cardID, core.ZoneGraveyard, nil, 0); err != nil {
		t.Fatalf("CastCardFromZoneWithoutPaying: %v", err)
	}
	tg.ResolveStack()

	if got := tg.AllBattlefield(); len(got) != 1 || got[0].Name() != "Cast-Alt Bear" {
		t.Errorf("expected Cast-Alt Bear on battlefield, got %v", got)
	}
}

func TestCastFromExileWithoutPaying(t *testing.T) {
	registerCastAltTestCards()
	tg := NewTestGame(t)
	pA := tg.GetPlayer(PlayerA)

	// Build the card and exile it directly.
	card, err := mage.CreateCard("Cast-Alt Heal")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	card.SetOwner(pA.PlayerID())
	tg.ExileCard(card, pA.PlayerID())
	pA.ManaPool().Clear()

	if err := tg.CastCardFromZoneWithoutPaying(pA.PlayerID(), card.ID(), core.ZoneExile, nil, 0); err != nil {
		t.Fatalf("CastCardFromZoneWithoutPaying: %v", err)
	}
	tg.ResolveStack()

	if pA.Life() != 25 {
		t.Errorf("expected life 25, got %d", pA.Life())
	}
}

func TestCastFromExileWithAnyColorMana(t *testing.T) {
	registerCastAltTestCards()
	tg := NewTestGame(t)
	pA := tg.GetPlayer(PlayerA)

	card, err := mage.CreateCard("Cast-Alt Heal")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	card.SetOwner(pA.PlayerID())
	tg.ExileCard(card, pA.PlayerID())

	// Grant Gonti-style permission with any-color mana.
	tg.GrantCastFromExile(pA.PlayerID(), card.ID(), true)

	// Pool: 4 black mana — none white, but the card costs {2}{W}{W}. With
	// any-color mana permission, four black should be enough.
	pA.ManaPool().Clear()
	pA.ManaPool().Add(core.Black, 4)

	if err := tg.CastExiledCardWithPermission(pA.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatalf("CastExiledCardWithPermission: %v", err)
	}
	tg.ResolveStack()

	if pA.Life() != 25 {
		t.Errorf("expected life 25, got %d", pA.Life())
	}
	// Permission must be cleared once the card has left exile.
	if tg.CastFromExilePermissionFor(pA.PlayerID(), card.ID()) != nil {
		t.Errorf("expected permission cleared after cast")
	}
}

func TestCastFromExileWithAnyColorMana_NotEnoughMana(t *testing.T) {
	registerCastAltTestCards()
	tg := NewTestGame(t)
	pA := tg.GetPlayer(PlayerA)

	card, err := mage.CreateCard("Cast-Alt Heal")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	card.SetOwner(pA.PlayerID())
	tg.ExileCard(card, pA.PlayerID())
	tg.GrantCastFromExile(pA.PlayerID(), card.ID(), true)

	pA.ManaPool().Clear()
	pA.ManaPool().Add(core.Black, 3) // need 4 total

	if err := tg.CastExiledCardWithPermission(pA.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("expected error for insufficient mana")
	}
	// Card should remain in exile.
	if tg.FindExiledCard(card.ID()) == nil {
		t.Error("expected card still in exile after failed cast")
	}
}

func TestExileInsteadOfGraveyardOnResolve(t *testing.T) {
	registerCastAltTestCards()
	tg := NewTestGame(t)
	pA := tg.GetPlayer(PlayerA)

	tg.AddCard(core.ZoneGraveyard, PlayerA, "Cast-Alt Heal")
	cardID := pA.Graveyard()[0].ID()
	pA.ManaPool().Clear()

	// Tag the card as "if it would go to the graveyard, exile it instead".
	tg.AddExileIfWouldGoToGraveyardThisTurn(cardID, cardID)

	if err := tg.CastCardFromZoneWithoutPaying(pA.PlayerID(), cardID, core.ZoneGraveyard, nil, 0); err != nil {
		t.Fatalf("CastCardFromZoneWithoutPaying: %v", err)
	}
	tg.ResolveStack()

	// The instant should have been exiled, not put into the graveyard.
	for _, c := range pA.Graveyard() {
		if c.ID() == cardID {
			t.Errorf("Cast-Alt Heal went to graveyard; expected exile")
		}
	}
	if tg.FindExiledCard(cardID) == nil {
		t.Errorf("expected Cast-Alt Heal in exile after resolution")
	}

	// And the life gain still happened.
	if pA.Life() != 25 {
		t.Errorf("expected life 25, got %d", pA.Life())
	}
}

func TestCastFromZoneWithAlternateCost(t *testing.T) {
	registerCastAltTestCards()
	tg := NewTestGame(t)
	pA := tg.GetPlayer(PlayerA)

	// Place the spell in the graveyard.
	tg.AddCard(core.ZoneGraveyard, PlayerA, "Cast-Alt Heal")
	cardID := pA.Graveyard()[0].ID()

	// Pool: just enough to pay an alternate cost {1}{B}.
	pA.ManaPool().Clear()
	pA.ManaPool().Add(core.Black, 1)
	pA.ManaPool().Add(core.Colorless, 1)

	alt := core.ParseManaCost("{1}{B}")
	if err := tg.CastCardFromZoneWithAlternateCost(pA.PlayerID(), cardID, core.ZoneGraveyard, alt, nil, 0); err != nil {
		t.Fatalf("CastCardFromZoneWithAlternateCost: %v", err)
	}
	tg.ResolveStack()

	if pA.Life() != 25 {
		t.Errorf("expected life 25, got %d", pA.Life())
	}
	if pA.ManaPool().TotalMana() != 0 {
		t.Errorf("expected mana pool empty after paying alt cost, got %d", pA.ManaPool().TotalMana())
	}
}

func TestAlternateCastUnpayableTotalCostDoesNotTapManaSources(t *testing.T) {
	registerCastAltTestCards()
	tg := NewTestGame(t)
	pA := tg.GetPlayer(PlayerA)
	tg.AddCard(core.ZoneGraveyard, PlayerA, "Cast-Alt Atomic Failure")
	cardID := pA.Graveyard()[0].ID()
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Mountain")
	mountain := tg.FindPermanentByName("Mountain", pA.PlayerID())

	if err := tg.CastCardWithAlternateCost(pA.PlayerID(), cardID, 0, nil, 0); err == nil {
		t.Fatal("expected the unpayable total alternate cost to reject the cast")
	}
	if mountain.Tapped {
		t.Fatal("failed alternate cast tapped its Mountain")
	}
	if got := pA.ManaPool().TotalMana(); got != 0 {
		t.Fatalf("failed alternate cast left %d mana in the pool, want 0", got)
	}
	found := false
	for _, card := range pA.Graveyard() {
		found = found || card.ID() == cardID
	}
	if !found {
		t.Fatal("failed alternate cast removed the spell from the graveyard")
	}
}

func TestAlternateCastCombinesManaComponents(t *testing.T) {
	registerCastAltTestCards()
	tg := NewTestGame(t)
	pA := tg.GetPlayer(PlayerA)
	tg.AddCard(core.ZoneGraveyard, PlayerA, "Cast-Alt Combined Mana")
	cardID := pA.Graveyard()[0].ID()
	pA.ManaPool().Add(core.White, 1)
	pA.ManaPool().Add(core.Blue, 1)

	if err := tg.CastCardWithAlternateCost(pA.PlayerID(), cardID, 0, nil, 0); err != nil {
		t.Fatalf("casting alternate {1} plus additional {W} from {W}{U}: %v", err)
	}
	if got := pA.ManaPool().TotalMana(); got != 0 {
		t.Fatalf("alternate total payment left %d mana, want 0", got)
	}
}

// CR 702.34: a card cast via flashback is exiled instead of being put into
// its owner's graveyard when it resolves.
func TestFlashbackExilesAfterResolution(t *testing.T) {
	registerCastAltTestCards()
	tg := NewTestGame(t)
	pA := tg.GetPlayer(PlayerA)

	tg.AddCard(core.ZoneGraveyard, PlayerA, "Cast-Alt Flashback Heal")
	cardID := pA.Graveyard()[0].ID()

	pA.ManaPool().Clear()
	pA.ManaPool().Add(core.White, 1)
	pA.ManaPool().Add(core.Colorless, 4)

	if err := tg.CastCardWithAlternateCost(pA.PlayerID(), cardID, 0, nil, 0); err != nil {
		t.Fatalf("CastCardWithAlternateCost: %v", err)
	}
	tg.ResolveStack()

	if pA.Life() != 23 {
		t.Errorf("expected life 23 after flashback Heal resolves, got %d", pA.Life())
	}
	for _, c := range pA.Graveyard() {
		if c.ID() == cardID {
			t.Errorf("flashback card returned to graveyard; expected exile")
		}
	}
	if tg.FindExiledCard(cardID) == nil {
		t.Errorf("expected flashback card in exile after resolution")
	}
}

// CR 702.34: a fizzled flashback spell is also exiled, not put into the
// graveyard. Build a flashback sorcery with a target so we can fizzle it.
func TestFlashbackExilesOnFizzle(t *testing.T) {
	registerCastAltTestCards()
	if !mage.CardRegistered("Cast-Alt Flashback Bolt") {
		mage.Register("Cast-Alt Flashback Bolt", func() mage.Card {
			return mage.NewSorcery("Cast-Alt Flashback Bolt", "{R}",
				mage.NewTargetedSpell(mage.TargetCreature(), mage.DealDamage(mage.Fixed(2))),
				mage.WithFlashback(core.ParseManaCost("{2}{R}")),
			)
		})
	}
	tg := NewTestGame(t)
	pA := tg.GetPlayer(PlayerA)
	pB := tg.GetPlayer(PlayerB)

	tg.AddCard(core.ZoneGraveyard, PlayerA, "Cast-Alt Flashback Bolt")
	cardID := pA.Graveyard()[0].ID()
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Grizzly Bears")
	bear := tg.AllBattlefield()[0]

	pA.ManaPool().Clear()
	pA.ManaPool().Add(core.Red, 1)
	pA.ManaPool().Add(core.Colorless, 2)

	if err := tg.CastCardWithAlternateCost(pA.PlayerID(), cardID, 0, []uuid.UUID{bear.ID()}, 0); err != nil {
		t.Fatalf("CastCardWithAlternateCost: %v", err)
	}
	// Remove the target so the spell fizzles.
	tg.RemoveFromBattlefield(bear)
	pB.AddToGraveyard(bear.Card)

	tg.ResolveStack()

	for _, c := range pA.Graveyard() {
		if c.ID() == cardID {
			t.Errorf("fizzled flashback card returned to graveyard; expected exile")
		}
	}
	if tg.FindExiledCard(cardID) == nil {
		t.Errorf("expected fizzled flashback card in exile")
	}
}
