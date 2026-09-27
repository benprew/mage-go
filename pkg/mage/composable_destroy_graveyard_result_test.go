package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDestroyEachGraveyardResult(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	count := BindNumber("mountains put into graveyard")
	put := func(name string) *Permanent {
		card := NewLand(name)
		card.SetOwner(b.PlayerID())
		return g.PutOnBattlefield(card, b.PlayerID())
	}
	first := put("First")
	second := put("Second")
	regenerated := put("Regenerated")
	g.AddRegenerationShield(regenerated.ID())
	indestructible := put("Indestructible")
	indestructible.GrantBaseAttr(core.Indestructible)
	phased := put("Phased")
	phased.PhasedOut = true
	set := MatchingPermanents(IsLand)
	body := Sequence(DestroyEach(set, PermanentsPutIntoGraveyardThisWayInto(count)), GainLifeFor(You, FromNumber(count)))
	clone := g.Clone()
	if err := body.Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.resolution.numberBindings[count] != 2 || clone.GetPlayer(a.PlayerID()).Life() != 22 {
		t.Fatal("clone did not count exactly two graveyard moves")
	}
	if g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) == nil || a.Life() != 20 {
		t.Fatal("clone changed the original game")
	}
	if err := body.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.resolution.numberBindings[count] != 2 || a.Life() != 22 {
		t.Fatal("result did not count exactly two graveyard moves")
	}
	for _, p := range []*Permanent{regenerated, indestructible, phased} {
		if g.FindPermanentIncludingPhased(p.ID()) == nil {
			t.Fatalf("%s did not survive", p.Name())
		}
	}
	if err := DestroyEach(MatchingPermanents(IsCreature), PermanentsPutIntoGraveyardThisWayInto(count)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.resolution.numberBindings[count] != 0 {
		t.Fatal("empty instruction retained a stale result")
	}
}

func TestDestroyEachGraveyardResultDefinition(t *testing.T) {
	count := BindNumber("graveyard count")
	set := MatchingPermanents(IsLand)
	valid := NewSorcery("Valid", "", WithAbility(Spell(Sequence(
		DestroyEach(set, PermanentsPutIntoGraveyardThisWayInto(count)),
		DealDamageFrom(This, You, FromNumber(count)),
	))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		body ResolvingEffect
		want string
	}{
		{DestroyEach(set, PermanentsPutIntoGraveyardThisWayInto(nil)), "binding"},
		{DestroyEach(set, PermanentsPutIntoGraveyardThisWayInto(BindNumber(""))), "name"},
		{DestroyEach(set, PermanentsPutIntoGraveyardThisWayInto(count), PermanentsPutIntoGraveyardThisWayInto(count)), "once"},
		{Sequence(DealDamageFrom(This, You, FromNumber(count)), DestroyEach(set, PermanentsPutIntoGraveyardThisWayInto(count))), "producer"},
	} {
		card := NewSorcery("Invalid", "", WithAbility(Spell(tc.body)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.want) {
			t.Fatalf("error %v does not contain %q", err, tc.want)
		}
	}
}

func TestDestroyEachGraveyardResultXTargetsAndReplacement(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareXPermanentTargets("mountains", And(IsLand, HasSubType("Mountain")))
	count := BindNumber("mountains put into graveyard")
	spell := NewSorcery("Synthetic eruption", "{X}{U}{U}{U}", WithAbility(Spell(Sequence(
		DestroyEach(targets, PermanentsPutIntoGraveyardThisWayInto(count)),
		DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), FromNumber(count)),
	), WithTargets(targets))))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	first := g.PutOnBattlefield(NewLand("First", WithSubTypes("Mountain")), b.PlayerID())
	second := g.PutOnBattlefield(NewLand("Second", WithSubTypes("Mountain")), b.PlayerID())
	plain := g.PutOnBattlefield(NewLand("Plain"), b.PlayerID())
	g.effects.AddReplacement(&exileInsteadOfGraveyardReplacement{cardID: second.ID()})
	g.SetStep(core.PrecombatMain)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{first.ID(), plain.ID()}, 2); err == nil {
		t.Fatal("accepted a non-Mountain target")
	}
	if g.StackSize() != 0 || a.Life() != 20 {
		t.Fatal("invalid targets changed game state")
	}
	a.ManaPool().Add(core.Blue, 3)
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{first.ID(), second.ID()}, 2); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 19 {
		t.Fatalf("life = %d/%d; want 19/19", a.Life(), b.Life())
	}
}

func TestDestroyEachGraveyardResultWithoutRegenerationAndToken(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := g.PutOnBattlefield(NewLand("Shielded"), b.PlayerID())
	g.AddRegenerationShield(land.ID())
	token := g.PutOnBattlefield(NewToken("Token", 1, 1, []core.CardType{core.TypeCreature}, nil), b.PlayerID())
	count := BindNumber("graveyard moves")
	if err := DestroyEachWithoutRegeneration(MatchingPermanents(Or(IsLand, IsCreature)), PermanentsPutIntoGraveyardThisWayInto(count)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.resolution.numberBindings[count] != 2 || g.FindPermanent(land.ID()) != nil || g.FindPermanent(token.ID()) != nil {
		t.Fatal("result did not count the shielded land and token")
	}
}
