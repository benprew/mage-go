package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDealDamageFromSourceAndRecipient(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 1, 2, WithKeyword(Lifelink))
	old := g.PutOnBattlefield(card, a.PlayerID())
	shield := PreventNextDamage(You, Fixed(1))
	if err := ApplyEffect(g, shield, uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.resolution.sourceCaptured = true
	g.resolution.sourceIncarnation = old.incarnationID
	g.RemoveFromBattlefield(old)
	g.PutOnBattlefield(card, b.PlayerID())
	g.zones.ClearLKI()
	if got := g.CaptureDamageSource(card.ID()); !got.Valid() || got.Incarnation() != old.incarnationID {
		t.Fatalf("source snapshot = %+v; captured = %t; incarnation = %s", got, g.resolution.sourceCaptured, g.resolution.sourceIncarnation)
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		if err := ApplyEffect(game, DealDamageFrom(This, You, Fixed(3)), card.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if game.GetPlayer(a.PlayerID()).Life() != 22 || game.GetPlayer(b.PlayerID()).Life() != 18 {
			t.Fatalf("life totals = %d, %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
	if err := ApplyEffect(clone, DealDamageFrom(This, You, Fixed(1)), card.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 23 {
		t.Fatal("clone damage affected original")
	}
}

func TestDealDamageFromPermanentTargetAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 5), b.PlayerID())
	target := DeclareDamageTarget("recipient")
	effect := DealDamageFrom(This, target, Fixed(2))
	g.resolution.targetSpecs = []Target{target.targetDeclaration}
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	if victim.Damage != 2 {
		t.Fatalf("damage = %d", victim.Damage)
	}
	g.PhaseOut(victim.ID())
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	if victim.Damage != 2 {
		t.Fatal("phased recipient received damage")
	}
	g.PhaseIn([]uuid.UUID{victim.ID()})
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 18 {
		t.Fatal("player recipient did not receive damage")
	}
	if err := ApplyEffect(g, DealDamageFrom(This, You, Fixed(2)), uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 {
		t.Fatal("absent source dealt damage")
	}
}

func TestDealDamageFromDefinitionValidation(t *testing.T) {
	target := DeclareDamageTarget("recipient")
	valid := NewInstant("Valid", "", Spell(DealDamageFrom(This, target, Fixed(1)), WithTargets(target)))
	if err := ValidateCardDefinition("Valid", valid); err != nil {
		t.Fatal(err)
	}
	invalid := NewInstant("Invalid", "", Spell(DealDamageFrom(This, target, Fixed(1))))
	if err := ValidateCardDefinition("Invalid", invalid); err == nil || !strings.Contains(err.Error(), "target binding requires") {
		t.Fatalf("missing target declaration: %v", err)
	}
	if got := InspectDefinition(DealDamageFrom(This, You, Fixed(1))); got.Role != "resolving-effect" || len(got.Children) != 3 {
		t.Fatalf("inspection = %+v", got)
	}
	missing := NewInstant("Missing", "", Spell(DealDamageFrom(nil, You, Fixed(1))))
	if err := ValidateCardDefinition("Missing", missing); err == nil || !strings.Contains(err.Error(), "damage source") {
		t.Fatalf("nil source declaration: %v", err)
	}
}

func TestDealDamageFromSpellSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewInstant("Borrowed Bolt", "", nil)
	card.SetOwner(a.PlayerID())
	end := g.resolution.Begin(&StackObject{Card: card, Controller: b.PlayerID(), SourceID: card.ID()})
	defer end()
	if err := ApplyEffect(g, DealDamageFrom(This, You, Fixed(2)), card.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 18 || !g.HasDealtDamageToPlayer(card.ID(), b.PlayerID()) {
		t.Fatal("resolving spell did not deal damage as source")
	}
}

func TestDealDamageFromDoesNotFollowReturnedTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	card := NewCreature("Victim", "", 2, 5)
	old := g.PutOnBattlefield(card, b.PlayerID())
	target := DeclareDamageTarget("recipient")
	g.resolution.targetSpecs = []Target{target.targetDeclaration}
	g.resolution.targetIncarnations = map[uuid.UUID]uuid.UUID{old.ID(): old.incarnationID}
	g.RemoveFromBattlefield(old)
	returned := g.PutOnBattlefield(card, b.PlayerID())
	if err := ApplyEffect(g, DealDamageFrom(This, target, Fixed(2)), source.ID(), a.PlayerID(), []uuid.UUID{card.ID()}); err != nil {
		t.Fatal(err)
	}
	if returned.Damage != 0 {
		t.Fatal("damage followed a returned target")
	}
}
