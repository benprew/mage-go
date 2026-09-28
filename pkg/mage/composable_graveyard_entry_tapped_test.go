package mage

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestReturnCardToBattlefieldTappedAtEntry(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Guest", "", 2, 2,
		WithAbility(WhenEvent(Enters(This), IfState(SourceTappedCondition(), GainLifeFor(You, Fixed(1))))),
		WithAbility(WhenEvent(BecomesTapped(This), GainLifeFor(You, Fixed(3)))))
	card.SetOwner(b.PlayerID())
	b.AddToGraveyard(card)
	target := DeclareGraveyardCardTarget("guest", InAnyGraveyard())
	_, obj := graveyardTestSpell(a.PlayerID(), target, ReturnCardToBattlefield(target, You, EnterTapped()))
	obj.Targets = []uuid.UUID{card.ID()}
	g.pushStack(obj)
	clone := g.Clone()
	clone.ResolveTopOfStack()
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if a.Life() != 20 || g.FindPermanent(card.ID()) != nil {
		t.Fatal("clone changed the original game")
	}
	g.ResolveTopOfStack()
	g.PutTriggersOnStack()
	g.ResolveStack()
	perm := g.FindPermanent(card.ID())
	if perm == nil || !perm.Tapped || perm.ControllerID() != a.PlayerID() || perm.Card.Owner() != b.PlayerID() {
		t.Fatal("card did not enter tapped under the chosen controller")
	}
	if a.Life() != 21 || clone.GetPlayer(a.PlayerID()).Life() != 21 {
		t.Fatal("entry trigger did not observe tapped state, or a tap transition fired")
	}
}

func TestReturnCardToBattlefieldTappedStaleAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewLand("Guest")
	card.SetOwner(b.PlayerID())
	b.AddToGraveyard(card)
	target := DeclareGraveyardCardTarget("guest", InAnyGraveyard())
	_, obj := graveyardTestSpell(a.PlayerID(), target, ReturnCardToBattlefield(target, You, EnterTapped()))
	obj.Targets = []uuid.UUID{card.ID()}
	g.pushStack(obj)
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("could not copy the return spell")
	}
	g.ResolveTopOfStack()
	first := g.FindPermanent(card.ID())
	if first == nil || !first.Tapped || g.StackSize() != 1 {
		t.Fatal("copy did not return the card tapped")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(card.ID()) != first || !first.Tapped {
		t.Fatal("original spell followed the card out of the graveyard")
	}

	card2 := NewLand("Stale")
	card2.SetOwner(b.PlayerID())
	b.AddToGraveyard(card2)
	_, stale := graveyardTestSpell(a.PlayerID(), target, ReturnCardToBattlefield(target, You, EnterTapped()))
	stale.Targets = []uuid.UUID{card2.ID()}
	g.pushStack(stale)
	moved, ok := g.MoveFromGraveyard(b.PlayerID(), card2.ID(), ZoneHand)
	if !ok {
		t.Fatal("could not move the target")
	}
	b.AddToHand(moved)
	returned, ok := b.RemoveFromHand(card2.ID())
	if !ok {
		t.Fatal("could not return the target")
	}
	b.AddToGraveyard(returned)
	g.ResolveTopOfStack()
	if g.FindPermanent(card2.ID()) != nil {
		t.Fatal("stale graveyard target entered the battlefield")
	}
}

func TestReturnCardToBattlefieldTappedValidation(t *testing.T) {
	target := DeclareGraveyardCardTarget("guest")
	for _, tc := range []struct {
		name   string
		effect ResolvingEffect
		want   string
	}{
		{"valid", ReturnCardToBattlefield(target, You, EnterTapped()), ""},
		{"duplicate", ReturnCardToBattlefield(target, You, EnterTapped(), EnterTapped()), "one entry option"},
		{"nil option", ReturnCardToBattlefield(target, You, nil), "non-nil entry option"},
		{"missing card", ReturnCardToBattlefield(nil, You, EnterTapped()), ".card"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewSorcery("Return", "", WithAbility(Spell(tc.effect, WithTargets(target))))
			err := ValidateCardDefinition("Return", card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestCapturedDiscardedLandEntersTapped(t *testing.T) {
	a := &cardSetChoicePlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Chosen land")
	land.SetOwner(b.PlayerID())
	b.AddToHand(land)
	artifact := NewArtifact("Other card", "")
	artifact.SetOwner(b.PlayerID())
	b.AddToHand(artifact)
	a.choice = land.ID()
	discarded, picked := BindCards("discarded"), BindCard("picked")
	player := DeclarePlayerTarget("player")
	spell := NewInstant("Synthetic return", "", WithAbility(Spell(Sequence(
		Discard(player, Fixed(2), DiscardCardsInto(discarded)),
		ChooseUpToOneCard(You, FromCards(discarded), IsLandCard, picked),
		ReturnCardToBattlefield(FromCard(picked), You, EnterTapped()),
	), WithTargets(player))))
	spell.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	perm := g.FindPermanent(land.ID())
	if perm == nil || !perm.Tapped || perm.ControllerID() != a.PlayerID() || perm.Card.Owner() != b.PlayerID() {
		t.Fatal("chosen discarded land did not enter tapped under the spell controller")
	}
	if !slices.Contains(a.seen, land.ID()) || slices.Contains(a.seen, artifact.ID()) {
		t.Fatal("choice did not offer the discarded land")
	}
}
