package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDealDamageFromCapturesActualAmount(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	if err := ApplyEffect(g, PreventNextDamage(You, Fixed(2)), uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	dealt := BindNumber("damage dealt")
	effect := Sequence(DealDamageFrom(This, You, Fixed(3), DamageAmountInto(dealt)), GainLifeFor(You, FromNumber(dealt)))
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

func TestDealDamageFromResultDeclaration(t *testing.T) {
	dealt := BindNumber("dealt")
	valid := NewInstant("Valid", "", Spell(Sequence(DealDamageFrom(This, You, Fixed(2), DamageAmountInto(dealt)), GainLifeFor(You, FromNumber(dealt)))))
	if err := ValidateCardDefinition("Valid", valid); err != nil {
		t.Fatal(err)
	}
	invalid := NewInstant("Invalid", "", Spell(GainLifeFor(You, FromNumber(dealt))))
	if err := ValidateCardDefinition("Invalid", invalid); err == nil || !strings.Contains(err.Error(), "binding") {
		t.Fatalf("missing producer: %v", err)
	}
	for _, effect := range []ResolvingEffect{
		DealDamageFrom(This, You, Fixed(1), DamageAmountInto(nil)),
		DealDamageFrom(This, You, Fixed(1), nil),
		DealDamageFrom(This, You, Fixed(1), DamageAmountInto(dealt), DamageAmountInto(dealt)),
	} {
		card := NewInstant("Bad", "", Spell(effect))
		if err := ValidateCardDefinition("Bad", card); err == nil {
			t.Fatal("invalid result declaration passed registration")
		}
	}
}

func TestDealDamageFromResultZeroForZeroDamage(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	dealt := BindNumber("dealt")
	effect := Sequence(
		DealDamageFrom(This, ActivePlayer, Fixed(2), DamageAmountInto(dealt)),
		DealDamageFrom(This, ActivePlayer, Fixed(0), DamageAmountInto(dealt)),
		GainLifeFor(You, FromNumber(dealt)),
	)
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 18 || b.Life() != 20 {
		t.Fatal("zero damage kept a prior result")
	}
}

func TestDealDamageFromResultAbsentTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	victimCard := NewCreature("Victim", "", 2, 5)
	victim := g.PutOnBattlefield(victimCard, b.PlayerID())
	target := DeclareDamageTarget("recipient")
	g.resolution.targetSpecs = []Target{target.targetDeclaration}
	g.resolution.targetIncarnations = map[uuid.UUID]uuid.UUID{victim.ID(): victim.incarnationID}
	g.RemoveFromBattlefield(victim)
	returned := g.PutOnBattlefield(victimCard, b.PlayerID())
	dealt := BindNumber("dealt")
	effect := Sequence(DealDamageFrom(This, target, Fixed(2), DamageAmountInto(dealt)), GainLifeFor(You, FromNumber(dealt)))
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	if returned.Damage != 0 || a.Life() != 20 {
		t.Fatal("returned target received damage or stale damage result")
	}
}

func TestDealDamageFromResultAfterRedirect(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1, WithKeyword(core.Lifelink)), a.PlayerID())
	redirect := g.PutOnBattlefield(NewCreature("Redirect", "", 2, 5), b.PlayerID())
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: redirect.ID()})
	dealt := BindNumber("dealt")
	effect := Sequence(DealDamageFrom(This, You, Fixed(2), DamageAmountInto(dealt)), GainLifeFor(You, FromNumber(dealt)))
	if err := ApplyEffect(g, effect, source.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if redirect.Damage != 2 || b.Life() != 22 || a.Life() != 22 {
		t.Fatalf("redirect damage = %d; life = %d, %d", redirect.Damage, a.Life(), b.Life())
	}
}
