package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestObjectFilterUsesCurrentCharacteristicsAcrossZones(t *testing.T) {
	g := newPriorityTestGame()
	card := NewCreature("Creature", "{G}", 2, 3, WithSubTypes("Elf"))
	permanent := g.PutOnBattlefield(card, g.players[0].PlayerID())
	filter := ObjectAnd(ObjectHasType(TypeCreature), ObjectHasSubtype("Elf"), ObjectHasColor(Green))
	if !filter.Match(card, g) || !filter.Match(permanent, g) {
		t.Fatal("shared filter disagrees across zones")
	}
	permanent.SubTypeOverride = []string{"Goblin"}
	if filter.Match(permanent, g) || !filter.Match(card, g) {
		t.Fatal("filter ignored current battlefield characteristics")
	}
	if !filter.Cards().Match(card) || filter.Permanents().Match(permanent, g) {
		t.Fatal("filter adapter changed meaning")
	}
}

func TestObjectFilterSupportsSuperTypesKeywordsAndCombinators(t *testing.T) {
	g := newPriorityTestGame()
	card := NewCreature("Legend", "{W}", 2, 2, WithSuperTypes(SuperLegendary), WithKeyword(Flying))
	permanent := g.PutOnBattlefield(card, g.players[0].PlayerID())
	filter := ObjectAnd(
		ObjectHasSuperType(SuperLegendary),
		ObjectHasKeyword(Flying),
		ObjectNot(ObjectHasColor(Black)),
		ObjectOr(ObjectHasColor(White), ObjectHasColor(Blue)),
	)
	if !filter.Match(card, g) || !filter.Match(permanent, g) {
		t.Fatal("expected filter to match legendary flying white creature in both card and permanent views")
	}
}

func TestProtectionFromColorUsesCardColorsNotJustManaCost(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	knight := g.PutOnBattlefield(NewCreature("White Knight", "{W}{W}", 2, 2, WithAbility(ProtectionFromColor(Black))), owner)
	blackToken := NewToken("Zombie Token", 2, 2, []CardType{TypeCreature}, []string{"Zombie"}, []Color{Black})
	if !knight.HasProtectionFrom(blackToken) {
		t.Fatal("expected protection from black to apply against a black token with no mana cost")
	}
}

func TestActivatedAbilitySupportsGraveyardActivationZoneDirectly(t *testing.T) {
	g := newPriorityTestGame()
	g.turns.SetStep(PrecombatMain)
	player := g.players[0]
	card := NewCreature("Reassembling", "{1}{B}", 1, 1,
		WithAbility(NewActivated(ExileSelfFromGraveyardCost(), GainLife(2), WithSorcerySpeed(), WithActivationZone(ZoneGraveyard))),
	)
	player.AddToGraveyard(card)
	cardID, idx, ok := g.FindGraveyardActivatableCard(player.PlayerID(), "Reassembling")
	if !ok || cardID != card.ID() || idx != 0 {
		t.Fatalf("expected graveyard-zone ActionDefinition to be discoverable, got ok=%v id=%v idx=%d", ok, cardID, idx)
	}
	if err := g.ActivateGraveyardAbility(player.PlayerID(), card.ID(), idx, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if player.Life() != 22 {
		t.Fatalf("expected life 22 after graveyard ability resolved, got %d", player.Life())
	}
}

type customObjectView struct{ *Card }

func TestObjectFilterRejectsTypedNilViews(t *testing.T) {
	for _, tc := range []struct {
		name   string
		object ObjectView
	}{
		{"nil", nil},
		{"permanent", (*Permanent)(nil)},
		{"card", (*Card)(nil)},
		{"LKI", (*PermanentLKI)(nil)},
		{"custom", (*customObjectView)(nil)},
	} {
		for _, filter := range []ObjectFilter{{}, ObjectHasType(TypeCreature), ObjectNot(ObjectHasType(TypeCreature))} {
			t.Run(tc.name+"/"+filter.Text(), func(t *testing.T) {
				defer func() {
					if err := recover(); err != nil {
						t.Errorf("nil object caused a panic: %v", err)
					}
				}()
				if filter.Match(tc.object, nil) {
					t.Fatal("nil object matched a filter")
				}
			})
		}
	}
	for _, filter := range []ObjectFilter{{}, ObjectHasType(TypeCreature)} {
		t.Run("adapters/"+filter.Text(), func(t *testing.T) {
			defer func() {
				if err := recover(); err != nil {
					t.Errorf("nil adapter input caused a panic: %v", err)
				}
			}()
			if filter.Cards().Match(nil) || filter.Permanents().Match(nil, nil) {
				t.Fatal("nil adapter input matched")
			}
		})
	}
}

func TestObjectPlayerFiltersRejectMissingIdentity(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	card := NewCreature("Unowned", "{G}", 1, 1)
	for _, filter := range []ObjectFilter{ObjectControlledBy(SelectController()), ObjectOwnedBy(SelectController())} {
		if filter.Match(card, g) || filter.Cards().Match(card) {
			t.Errorf("%s matched a missing identity", filter.Text())
		}
	}
	card.SetOwner(owner)
	permanent := g.PutOnBattlefield(card, owner)
	ctx := AbilityContext{Controller: owner}
	for _, filter := range []ObjectFilter{ObjectControlledBy(SelectController()), ObjectOwnedBy(SelectController())} {
		if filter.Match(permanent, g) {
			t.Errorf("%s matched without a controller binding", filter.Text())
		}
		if !filter.Match(permanent, g, ctx) {
			t.Errorf("%s rejected a bound controller", filter.Text())
		}
	}
	if ObjectControlledBy(SelectController()).Match(card, g, ctx) {
		t.Fatal("card outside the battlefield has a controller")
	}
	if !ObjectOwnedBy(SelectController()).Cards(ctx).Match(card) {
		t.Fatal("owned card did not match its owner")
	}
}
