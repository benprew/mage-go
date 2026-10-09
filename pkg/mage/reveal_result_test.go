package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRevealResultSnapshots(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	card := NewArtifact("Cecily, Haunted Mage", "")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	binding := BindCard("reveal")
	name := BindCardName("name")
	g.resolution.cardNameBindings = map[*CardNameBinding]string{name: "Eleven, the Mage"}
	yes := GainLifeFor(You, Fixed(2))
	no := GainLifeFor(You, Fixed(1))
	tests := []struct {
		matcher RevealedCardMatcher
		want    int
	}{
		{RevealedCardHasType(core.TypeArtifact), 2},
		{RevealedCardHasType(core.TypeLand), 1},
		{RevealedCardLacksType(core.TypeLand), 2},
		{RevealedCardNamed(MatchesChosenCardName(FromCardName(name))), 2},
	}
	for _, tc := range tests {
		before := a.Life()
		if err := IfRevealedCardMatches(binding, tc.matcher, yes, no).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if a.Life() != before+1 {
			t.Fatal("missing result matched")
		}
	}
	if err := RevealRandomHandCardInto(You, binding).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	card.AddType(core.TypeLand)
	a.RemoveFromHand(card.ID())
	a.AddToHand(card)
	for _, tc := range tests {
		before := a.Life()
		if err := IfRevealedCardMatches(binding, tc.matcher, yes, no).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if a.Life() != before+tc.want {
			t.Fatal("reveal facts changed after return or mutation")
		}
	}
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("snapshot permitted an action on a returned visit")
	}
	a.RemoveFromHand(card.ID())
	if err := RevealRandomHandCardInto(You, binding).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	before := a.Life()
	if err := IfRevealedCardMatches(binding, RevealedCardLacksType(core.TypeLand), yes, no).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != before+1 {
		t.Fatal("empty reveal retained facts")
	}
	cc := &EffectContext{Game: clone, Controller: a.PlayerID()}
	before = clone.GetPlayer(a.PlayerID()).Life()
	if err := IfRevealedCardMatches(binding, tests[0].matcher, yes, no).Apply(cc); err != nil {
		t.Fatal(err)
	}
	if clone.GetPlayer(a.PlayerID()).Life() != before+2 {
		t.Fatal("clone lost its facts")
	}
	if err := ApplyEffect(clone, IfRevealedCardMatches(binding, tests[0].matcher, yes, no), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if clone.GetPlayer(a.PlayerID()).Life() != before+3 {
		t.Fatal("nested frame read outer facts")
	}
}

func TestRevealResultDeclarations(t *testing.T) {
	binding := BindCard("reveal")
	body := GainLifeFor(You, Fixed(1))
	name := BindCardName("name")
	valid := IfRevealedCardMatches(binding, RevealedCardHasType(core.TypeLand), body, nil)
	for _, effect := range []Effect{
		valid,
		Sequence(valid, RevealRandomHandCardInto(You, binding)),
		Sequence(RevealRandomHandCardInto(You, binding), IfRevealedCardMatches(nil, RevealedCardHasType(core.TypeLand), body, body)),
		Sequence(RevealRandomHandCardInto(You, binding), IfRevealedCardMatches(binding, nil, body, body)),
		Sequence(RevealRandomHandCardInto(You, binding), IfRevealedCardMatches(binding, RevealedCardHasType(core.CardType(-1)), body, body)),
		Sequence(RevealRandomHandCardInto(You, binding), IfRevealedCardMatches(binding, RevealedCardLacksType(core.CardType(99)), body, body)),
		Sequence(RevealRandomHandCardInto(You, binding), IfRevealedCardMatches(binding, RevealedCardNamed(nil), body, body)),
		Sequence(RevealRandomHandCardInto(You, binding), IfRevealedCardMatches(binding, RevealedCardNamed(MatchesChosenCardName(FromCardName(name))), body, body)),
		Sequence(RevealRandomHandCardInto(You, binding), IfRevealedCardMatches(binding, RevealedCardHasType(core.TypeLand), nil, nil)),
	} {
		c := NewInstant("Invalid", "", Spell(effect))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	c := NewInstant("Valid", "", Spell(Sequence(RevealRandomHandCardInto(You, binding), valid)))
	if err := ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
}

func TestRevealResultWaitingAndFrames(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewArtifact("Source", "")
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	held := NewLand("Held")
	held.SetOwner(a.PlayerID())
	a.AddToHand(held)
	binding := BindCard("result")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID()}
	if err := RevealRandomHandCardInto(You, binding).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	for _, body := range []ResolvingEffect{
		IfRevealedCardMatches(binding, RevealedCardHasType(core.TypeLand), Destroy(This), GainLifeFor(You, Fixed(0))),
		IfRevealedCardMatches(binding, RevealedCardHasType(core.TypeLand), GainLifeFor(You, Fixed(0)), Destroy(This)),
	} {
		saved := g.CaptureDamageSource(p.ID())
		if err := AtNext(EachEndStep, body).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		refs := g.WaitingDelayedPermanentSources()
		if len(refs) != 1 || !refs[0].SameObject(saved) || len(a.Hand()) != 1 || len(g.PublicReveals()) != 1 {
			t.Fatal("waiting query lost a branch source or executed a branch")
		}
	}
	g.PhaseOut(p.ID())
	if len(g.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("phasing lost the saved source")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	second := NewArtifact("Second", "")
	second.SetOwner(a.PlayerID())
	a.AddToHand(second)
	g.SetRandomResults([]int{1})
	if err := ForEach(MatchingPermanents(AnyPermanent), BindPermanent("member"), RevealRandomHandCardInto(You, binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	before := a.Life()
	if err := IfRevealedCardMatches(binding, RevealedCardHasType(core.TypeLand), GainLifeFor(You, Fixed(2)), GainLifeFor(You, Fixed(1))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != before+2 {
		t.Fatal("loop overwrote outer reveal facts")
	}
}
