package mage

import (
	"testing"

	"github.com/google/uuid"
)

type countedMayPlayer struct {
	*BasePlayer
	accept bool
	offers int
}

func (p *countedMayPlayer) ChooseMayAbility(string) bool {
	p.offers++
	return p.accept
}

func TestUnlessPaidOffersOnlyPossibleSacrifice(t *testing.T) {
	for _, tc := range []struct {
		name       string
		cards      int
		permanents int
		other      bool
		phased     bool
		accept     bool
		wantOffers int
		wantHand   int
		wantBoard  int
	}{
		{"no permanent", 2, 0, false, false, true, 0, 0, 0},
		{"opponent permanent", 2, 1, true, false, true, 0, 0, 1},
		{"phased permanent", 2, 1, false, true, true, 0, 0, 0},
		{"pay once then discard", 2, 1, false, false, true, 1, 1, 0},
		{"decline each time", 2, 1, false, false, false, 2, 0, 1},
		{"no drawn cards", 0, 1, false, false, true, 0, 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: tc.accept}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			for range tc.cards {
				a.AddToLibrary(NewArtifact("Drawn", ""))
			}
			for range tc.permanents {
				controller := a.PlayerID()
				if tc.other {
					controller = b.PlayerID()
				}
				permanent := g.PutOnBattlefield(NewLand("Land"), controller)
				permanent.PhasedOut = tc.phased
			}
			cards, member := BindCards("drawn"), BindCard("member")
			effect := Sequence(
				DrawFor(You, Fixed(tc.cards), DrawCardsInto(cards)),
				ForEachSavedCard(cards, member, UnlessPaid(You,
					Costs(SacrificeMatchingIncludingSourceCost(AnyPermanent, "Sacrifice a permanent")),
					Discard(You, Fixed(1)))),
			)
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			board := len(g.FilterBattlefield(AnyPermanent))
			if a.offers != tc.wantOffers || len(a.Hand()) != tc.wantHand || board != tc.wantBoard {
				t.Fatalf("offers = %d, hand = %d, battlefield = %d", a.offers, len(a.Hand()), board)
			}
		})
	}
}

func TestUnlessPaidPerDrawnCardDeclaration(t *testing.T) {
	cards, member := BindCards("drawn"), BindCard("member")
	body := Sequence(DrawFor(You, Fixed(2), DrawCardsInto(cards)),
		ForEachSavedCard(cards, member, UnlessPaid(You,
			Costs(SacrificeMatchingIncludingSourceCost(AnyPermanent, "Sacrifice a permanent")),
			Discard(You, Fixed(1)))))
	for _, card := range []*BaseCard{
		NewInstant("Draw spell", "", Spell(body)),
		NewArtifact("Draw artifact", "", WithAbility(Activate(Costs(), body))),
		NewArtifact("Draw trigger", "", WithAbility(WhenEvent(Enters(This), body))),
	} {
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}
