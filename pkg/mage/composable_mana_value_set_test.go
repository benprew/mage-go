package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestManaValueAtMostSetPaidXAndSimultaneousDestroy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindNumber("paid")
	spell := NewInstant("Sweep", "", WithAbility(Spell(DestroyEach(ManaValueAtMost(MatchingPermanents(Or(IsArtifact, IsCreature)), FromNumber(paid))))), WithAdditionalCost(LifePayXCostInto(paid)))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	makeCard := func(name, cost string, types ...core.CardType) *Permanent {
		card := NewArtifact(name, cost)
		card.types = types
		card.power, card.toughness = 2, 2
		card.SetOwner(b.PlayerID())
		return g.PutOnBattlefield(card, b.PlayerID())
	}
	zero := makeCard("Zero", "", core.TypeArtifact)
	boundary := makeCard("Boundary", "{2}", core.TypeArtifact, core.TypeCreature)
	above := makeCard("Above", "{3}", core.TypeCreature)
	land := makeCard("Land", "", core.TypeLand)
	phased := makeCard("Phased", "{1}", core.TypeArtifact)
	phased.PhasedOut = true
	g.AddRegenerationShield(boundary.ID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil, 2); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if !clone.HasRegenerationShield(boundary.ID()) {
		t.Fatal("clone lost regeneration shield")
	}
	clone.ResolveStack()
	if clone.FindPermanent(zero.ID()) != nil || clone.FindPermanent(boundary.ID()) == nil || !clone.FindPermanent(boundary.ID()).Tapped {
		t.Fatalf("paid threshold or regeneration result is wrong: zero=%v boundary=%v", clone.FindPermanent(zero.ID()), clone.FindPermanent(boundary.ID()))
	}
	for _, p := range []*Permanent{above, land, phased} {
		if clone.FindPermanentIncludingPhased(p.ID()) == nil {
			t.Fatalf("excluded permanent %s was destroyed", p.Name())
		}
	}
	if g.FindPermanent(zero.ID()) == nil || g.FindPermanent(boundary.ID()).Tapped {
		t.Fatal("clone changed original game")
	}
	g.ResolveStack()
	if g.FindPermanent(zero.ID()) != nil || g.FindPermanent(boundary.ID()) == nil || !g.FindPermanent(boundary.ID()).Tapped {
		t.Fatal("original resolution differed from clone")
	}
}

func TestManaValueAtMostSetValidation(t *testing.T) {
	paid := BindNumber("paid")
	for _, tc := range []struct {
		name string
		set  PermanentSet
		want string
	}{
		{"nil set", ManaValueAtMost(nil, Fixed(2)), "non-nil"},
		{"nil limit", ManaValueAtMost(MatchingPermanents(IsCreature), nil), "non-nil"},
		{"unbound", ManaValueAtMost(MatchingPermanents(IsCreature), FromNumber(paid)), "producer"},
	} {
		card := NewSorcery(tc.name, "", WithAbility(Spell(DestroyEach(tc.set))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
}
