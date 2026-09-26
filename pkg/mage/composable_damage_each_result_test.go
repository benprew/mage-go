package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestDealDamageEachTotalRecordsActualDamage(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 9), b.PlayerID())
	g.AddPreventionShield(b.PlayerID(), 1)
	total := BindNumber("total damage")
	effect := Sequence(
		DealDamageEach(This, DamageRecipients(EachPlayer, WithoutPermanent(MatchingPermanents(IsCreature), This)), Fixed(2), DamageTotalInto(total)),
		GainLifeFor(You, FromNumber(total)),
	)
	clone := g.Clone()
	if err := ApplyEffect(clone, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || b.Life() != 20 || victim.Damage != 0 {
		t.Fatal("clone changed original game")
	}
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 23 || b.Life() != 19 || g.FindPermanent(victim.ID()).Damage != 2 {
		t.Fatalf("life = %d, %d; damage = %d; want 23, 19, 2", a.Life(), b.Life(), g.FindPermanent(victim.ID()).Damage)
	}
}

func TestDealDamageEachTotalClearsOnAbsentSourceAndZeroDamage(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	total := BindNumber("total damage")
	effect := Sequence(
		DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(2), DamageTotalInto(total)),
		DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(0), DamageTotalInto(total)),
		GainLifeFor(You, FromNumber(total)),
	)
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 18 || b.Life() != 18 {
		t.Fatal("zero damage retained an earlier result")
	}
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 16 || b.Life() != 16 {
		t.Fatal("a new resolution retained an earlier result")
	}
	if err := ApplyEffect(g, effect, uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 16 || b.Life() != 16 {
		t.Fatal("absent source retained an earlier result")
	}
}

func TestDealDamageEachTotalFollowsReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 9), b.PlayerID())
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
	total := BindNumber("total")
	effect := Sequence(
		DealDamageEach(This, DamageRecipients(EachPlayer, WithoutPermanent(MatchingPermanents(IsCreature), This)), Fixed(2), DamageTotalInto(total)),
		GainLifeFor(You, FromNumber(total)),
	)
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 24 || b.Life() != 20 || g.FindPermanent(victim.ID()).Damage != 4 {
		t.Fatalf("life = %d, %d; victim damage = %d; want 24, 20, 4", a.Life(), b.Life(), g.FindPermanent(victim.ID()).Damage)
	}
}

func TestDealDamageEachTotalExcludesReturnedTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	card := NewCreature("Target", "", 2, 9)
	old := g.PutOnBattlefield(card, b.PlayerID())
	targets := DeclarePermanentTargets("target", 1, 1, IsCreature)
	g.resolution.targetSpecs = []Target{targets.targetDeclaration}
	g.resolution.targetIncarnations = map[uuid.UUID]uuid.UUID{old.ID(): old.incarnationID}
	g.RemoveFromBattlefield(old)
	returned := g.PutOnBattlefield(card, b.PlayerID())
	total := BindNumber("total")
	effect := Sequence(
		DealDamageEach(This, DamageRecipients(EachPlayer, targets), Fixed(2), DamageTotalInto(total)),
		GainLifeFor(You, FromNumber(total)),
	)
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{old.ID()}); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 || b.Life() != 18 || returned.Damage != 0 {
		t.Fatalf("life = %d, %d; returned damage = %d; want 22, 18, 0", a.Life(), b.Life(), returned.Damage)
	}
}

func TestDealDamageEachTotalDeclaration(t *testing.T) {
	total := BindNumber("total")
	valid := NewInstant("Valid", "", Spell(Sequence(
		DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(2), DamageTotalInto(total)),
		GainLifeFor(You, FromNumber(total)),
	)))
	if err := ValidateCardDefinition("Valid", valid); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{
		DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(2), DamageTotalInto(nil)),
		DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(2), nil),
		DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(2), DamageTotalInto(total), DamageTotalInto(total)),
	} {
		card := NewInstant("Invalid", "", Spell(effect))
		if err := ValidateCardDefinition("Invalid", card); err == nil {
			t.Fatal("invalid total declaration passed registration")
		}
	}
	missing := NewInstant("Missing", "", Spell(GainLifeFor(You, FromNumber(total))))
	if err := ValidateCardDefinition("Missing", missing); err == nil || !strings.Contains(err.Error(), "binding") {
		t.Fatalf("missing producer: %v", err)
	}
}
