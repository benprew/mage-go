package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageSourceResultKeepsDepartedIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Old Source", "", 1, 2, WithKeyword(core.Lifelink))
	old := g.PutOnBattlefield(card, a.PlayerID())
	source := BindDamageSource("damage source")
	effect := Sequence(
		DealDamageFrom(This, You, Fixed(1), DamageSourceInto(source)),
		DealDamageFrom(FromDamageSource(source), You, Fixed(2)),
	)
	g.resolution.sourceCaptured = true
	g.resolution.sourceIncarnation = old.incarnationID
	g.RemoveFromBattlefield(old)
	returned := g.PutOnBattlefield(card, b.PlayerID())
	for _, game := range []*Game{g, g.Clone()} {
		if err := ApplyEffect(game, effect, card.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if game.GetPlayer(a.PlayerID()).Life() != 23 || game.GetPlayer(b.PlayerID()).Life() != 17 {
			t.Fatalf("old source controller or lifelink lost: A=%d B=%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
	if got := g.FindPermanent(card.ID()); got == nil || got.incarnationID != returned.incarnationID {
		t.Fatal("saved source changed returned incarnation")
	}
}

func TestDamageSourceResultClearsOnPreventedDamage(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	binding := BindDamageSource("source")
	effect := Sequence(
		DealDamageFrom(This, You, Fixed(1), DamageSourceInto(binding)),
		PreventNextDamage(You, Fixed(1)),
		DealDamageFrom(This, You, Fixed(1), DamageSourceInto(binding)),
		DealDamageFrom(FromDamageSource(binding), You, Fixed(2)),
	)
	if err := ApplyEffect(g, effect, source.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 19 {
		t.Fatal("prevented damage retained a source")
	}
}

func TestDamageSourceResultCloneIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 1, WithKeyword(core.Lifelink)), a.PlayerID())
	binding := BindDamageSource("source")
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: damageSourceFromPermanent(p)}
	clone := g.Clone()
	saved := clone.resolution.damageSourceBindings[binding]
	saved.colors = append(saved.colors, core.Red)
	saved.keywords[core.Lifelink] = false
	clone.resolution.damageSourceBindings[binding] = saved
	if !g.resolution.damageSourceBindings[binding].HasKeyword(core.Lifelink) || g.resolution.damageSourceBindings[binding].HasColor(core.Red) {
		t.Fatal("clone changed original saved source")
	}
}

func TestDamageSourceResultSpellAfterRedirect(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewInstant("Two Bolts", "", nil)
	card.SetOwner(a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 5), b.PlayerID())
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
	end := g.resolution.Begin(&StackObject{Card: card, Controller: a.PlayerID(), SourceID: card.ID()})
	defer end()
	binding := BindDamageSource("spell")
	effect := Sequence(
		DealDamageFrom(This, You, Fixed(1), DamageSourceInto(binding)),
		DealDamageFrom(FromDamageSource(binding), You, Fixed(1)),
	)
	if err := ApplyEffect(g, effect, card.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if victim.Damage != 2 || b.Life() != 20 || !g.HasDealtDamageToPermanent(card.ID(), victim.ID()) {
		t.Fatal("redirected spell damage lost source identity")
	}
}

func TestDamageSourceResultDeclaration(t *testing.T) {
	binding := BindDamageSource("source")
	valid := NewInstant("Valid", "", Spell(Sequence(DealDamageFrom(This, You, Fixed(1), DamageSourceInto(binding)), DealDamageFrom(FromDamageSource(binding), You, Fixed(1)))))
	if err := ValidateCardDefinition("Valid", valid); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{
		DealDamageFrom(This, You, Fixed(1), DamageSourceInto(nil)),
		DealDamageFrom(This, You, Fixed(1), DamageSourceInto(BindDamageSource(""))),
		DealDamageFrom(This, You, Fixed(1), DamageSourceInto(binding), DamageSourceInto(binding)),
		DealDamageFrom(FromDamageSource(binding), You, Fixed(1)),
	} {
		if err := ValidateCardDefinition("Bad", NewInstant("Bad", "", Spell(effect))); err == nil {
			t.Fatal("invalid source result passed registration")
		}
	}
	missing := NewInstant("Missing", "", Spell(DealDamageFrom(FromDamageSource(binding), You, Fixed(1))))
	if err := ValidateCardDefinition("Missing", missing); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing producer: %v", err)
	}
}
