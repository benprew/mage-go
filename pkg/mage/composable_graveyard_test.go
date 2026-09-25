package mage

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func graveyardTestSpell(owner uuid.UUID, target *CardTargetBinding, effects ResolvingEffect) (Card, *StackObject) {
	ability := Spell(effects, WithTargets(target))
	card := NewSorcery("Recover", "", WithAbility(ability))
	card.SetOwner(owner)
	return card, &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: owner, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
}

func TestTypedGraveyardReturnUsesOwnerAndTargetBinding(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewCreature("First", "", 1, 1)
	first.SetOwner(a.PlayerID())
	second := NewCreature("Second", "", 1, 1)
	second.SetOwner(b.PlayerID())
	a.AddToGraveyard(first)
	b.AddToGraveyard(second)
	player := DeclarePlayerTarget("player")
	cardTarget := DeclareGraveyardCardTarget("card", InAnyGraveyard(), GraveyardCardFilter(IsCreatureCard))
	ability := Spell(Sequence(GainLifeFor(player, Fixed(1)), ReturnCardToHand(cardTarget)), WithTargets(player, cardTarget))
	spell := NewSorcery("Recover", "", WithAbility(ability))
	spell.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition("Recover", spell); err != nil {
		t.Fatal(err)
	}
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{a.PlayerID(), second.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	g.ResolveTopOfStack()
	if a.Life() != 21 || len(a.Hand()) != 0 || len(b.Hand()) != 1 || b.Hand()[0].ID() != second.ID() {
		t.Fatal("return used the wrong target or destination")
	}
	if !slices.ContainsFunc(a.Graveyard(), func(card Card) bool { return card.ID() == first.ID() }) {
		t.Fatal("return moved an unrelated card")
	}
}

func TestTypedGraveyardTargetVisitAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(a.PlayerID())
	a.AddToGraveyard(victim)
	target := DeclareGraveyardCardTarget("card")
	spell, obj := graveyardTestSpell(a.PlayerID(), target, ReturnCardToHand(target))
	obj.Targets = []uuid.UUID{victim.ID()}
	g.pushStack(obj)
	clone := g.Clone()
	clone.ResolveTopOfStack()
	if len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 || len(a.Hand()) != 0 {
		t.Fatal("clone changed the original game")
	}
	moved, ok := g.MoveFromGraveyard(a.PlayerID(), victim.ID(), ZoneHand)
	if !ok {
		t.Fatal("could not move target")
	}
	a.AddToHand(moved)
	removed, ok := a.RemoveFromHand(victim.ID())
	if !ok {
		t.Fatal("could not return target")
	}
	a.AddToGraveyard(removed)
	if !slices.Contains(target.Possible(a.PlayerID(), spell, g), victim.ID()) {
		t.Fatal("card is no longer targetable")
	}
	g.ResolveTopOfStack()
	if slices.ContainsFunc(a.Hand(), func(card Card) bool { return card.ID() == victim.ID() }) || !slices.ContainsFunc(a.Graveyard(), func(card Card) bool { return card.ID() == victim.ID() }) {
		t.Fatal("stale target followed a new graveyard visit")
	}
}

func TestTypedGraveyardTargetScopeAndFilter(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	artifact := NewArtifact("Artifact", "")
	artifact.SetOwner(b.PlayerID())
	b.AddToGraveyard(artifact)
	yours := DeclareGraveyardCardTarget("yours", GraveyardCardFilter(IsCreatureCard))
	fromAny := DeclareGraveyardCardTarget("any", InAnyGraveyard())
	if slices.Contains(yours.Possible(a.PlayerID(), nil, g), artifact.ID()) || !slices.Contains(fromAny.Possible(a.PlayerID(), nil, g), artifact.ID()) {
		t.Fatal("graveyard scope or card filter is wrong")
	}
	for _, tc := range []struct {
		name     string
		body     ResolvingEffect
		declared []TargetBinding
		want     string
	}{
		{"nil", ReturnCardToHand(nil), nil, ".card"},
		{"undeclared", ReturnCardToHand(fromAny), nil, ".card"},
		{"valid", ReturnCardToHand(fromAny), []TargetBinding{fromAny}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewSorcery("Test", "", WithAbility(Spell(tc.body, WithTargets(tc.declared...))))
			err := ValidateCardDefinition("Test", card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation error = %v, want %q", err, tc.want)
			}
		})
	}
}
