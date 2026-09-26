package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestDealDamageFromCapturesActualPlayer(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	player := BindPlayer("damaged player")
	amount := BindNumber("damage dealt")
	effect := Sequence(DealDamageFrom(This, You, Fixed(3), DamageAmountInto(amount), DamagePlayerInto(player)), GainLifeFor(FromPlayer(player), FromNumber(amount)))
	for _, game := range []*Game{g, g.Clone()} {
		if err := ApplyEffect(game, effect, source.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if got := game.GetPlayer(b.PlayerID()).Life(); got != 20 {
			t.Fatalf("life = %d, want 20", got)
		}
	}
	if b.Life() != 20 {
		t.Fatal("clone changed original")
	}
}

func TestDealDamageFromPlayerResultAbsentAfterRedirectOrPrevention(t *testing.T) {
	for _, redirected := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
		if redirected {
			victim := g.PutOnBattlefield(NewCreature("Redirect", "", 2, 5), b.PlayerID())
			g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
		} else if err := ApplyEffect(g, PreventNextDamage(You, Fixed(3)), uuid.Nil, b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		player := BindPlayer("damaged player")
		effect := Sequence(SnapshotPlayer(You, player), DealDamageFrom(This, You, Fixed(3), DamagePlayerInto(player)), GainLifeFor(FromPlayer(player), Fixed(4)))
		if err := ApplyEffect(g, effect, source.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if b.Life() != 20 {
			t.Fatalf("redirected=%t: stale player result changed life to %d", redirected, b.Life())
		}
	}
}

func TestDealDamageFromPlayerResultRedirectedFromPermanent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 5), b.PlayerID())
	g.effects.AddReplacement(&creatureDamageRedirectReplacement{creatureID: victim.ID(), targetPlayerID: b.PlayerID()})
	player := BindPlayer("damaged player")
	amount := BindNumber("damage dealt")
	target := DeclareDamageTarget("victim")
	g.resolution.targetSpecs = []Target{target.targetDeclaration}
	g.resolution.targetIncarnations = map[uuid.UUID]uuid.UUID{victim.ID(): victim.incarnationID}
	effect := Sequence(DealDamageFrom(This, target, Fixed(3), DamageAmountInto(amount), DamagePlayerInto(player)), GainLifeFor(FromPlayer(player), FromNumber(amount)))
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	if victim.Damage != 0 || b.Life() != 20 {
		t.Fatalf("victim damage = %d, player life = %d", victim.Damage, b.Life())
	}
}

func TestDealDamageFromPlayerResultDeclaration(t *testing.T) {
	player := BindPlayer("damaged player")
	valid := NewInstant("Valid", "", Spell(Sequence(DealDamageFrom(This, You, Fixed(1), DamagePlayerInto(player)), GainLifeFor(FromPlayer(player), Fixed(1)))))
	if err := ValidateCardDefinition("Valid", valid); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{
		DealDamageFrom(This, You, Fixed(1), DamagePlayerInto(nil)),
		DealDamageFrom(This, You, Fixed(1), DamagePlayerInto(BindPlayer(""))),
		DealDamageFrom(This, You, Fixed(1), DamagePlayerInto(player), DamagePlayerInto(player)),
	} {
		if err := ValidateCardDefinition("Bad", NewInstant("Bad", "", Spell(effect))); err == nil {
			t.Fatal("invalid damage player result passed registration")
		}
	}
	missing := NewInstant("Missing", "", Spell(GainLifeFor(FromPlayer(player), Fixed(1))))
	if err := ValidateCardDefinition("Missing", missing); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing producer: %v", err)
	}
}
