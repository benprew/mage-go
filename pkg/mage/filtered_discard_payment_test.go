package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type filteredDiscardPlayer struct {
	*BasePlayer
	choice     Card
	candidates []Card
	calls      int
	decline    bool
}

func (p *filteredDiscardPlayer) ChooseMayAbility(string) bool { return !p.decline }
func (p *filteredDiscardPlayer) ChooseCardFromHand(cards []Card, _ string, _ GameReader) Card {
	p.calls++
	p.candidates = append([]Card(nil), cards...)
	return p.choice
}

func TestFilteredDiscardPaymentLockedVisit(t *testing.T) {
	for _, mode := range []string{"normal", "returned", "missing", "changed", "clone"} {
		t.Run(mode, func(t *testing.T) {
			a := &filteredDiscardPlayer{BasePlayer: NewBasePlayer("A")}
			g := NewGame(a, NewBasePlayer("B"))
			first, second, land := NewArtifact("First", ""), NewArtifact("Second", ""), NewLand("Land")
			for _, c := range []Card{land, first, second} {
				a.AddToHand(c)
			}
			a.choice = second
			allowed := true
			filter := NewCardFilter("artifact card", func(c Card) bool { return allowed && c.HasType(TypeArtifact) })
			a.ManaPool().Add(Red, 1)
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{ManaPayment("{R}"), LifePayCost(2), DiscardMatchingCardCost(filter)}})
			if err != nil {
				t.Fatal(err)
			}
			if len(a.candidates) != 2 || a.candidates[0].ID() != first.ID() || a.candidates[1].ID() != second.ID() {
				t.Fatal("choice included a nonmatching card")
			}
			switch mode {
			case "returned", "missing":
				c, _ := a.RemoveFromHand(second.ID())
				if mode == "returned" {
					a.AddToHand(c)
				}
			case "changed":
				allowed = false
			case "clone":
				clone := g.Clone()
				copied := *tx
				copied.game = clone
				if err := copied.Commit(); err != nil {
					t.Fatal(err)
				}
				if len(a.Hand()) != 3 || len(clone.GetPlayer(a.PlayerID()).Graveyard()) != 1 {
					t.Fatal("clone payment changed original")
				}
			}
			err = tx.Commit()
			if mode == "returned" || mode == "missing" || mode == "changed" {
				if err == nil {
					t.Fatal("stale or ineligible choice paid")
				}
				if a.Life() != 20 || a.ManaPool().Count(Red) != 1 || len(a.Graveyard()) != 0 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 0 {
					t.Fatal("failed payment spent resources")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != second.ID() || a.Life() != 18 {
					t.Fatal("payment used another card")
				}
			}
			if a.calls != 1 {
				t.Fatalf("choice calls = %d", a.calls)
			}
		})
	}
}

func TestFilteredDiscardPaymentInvalidAndReplacement(t *testing.T) {
	for _, mode := range []string{"nonmatching", "outside", "nil", "empty", "prevented", "hidden", "recheck"} {
		t.Run(mode, func(t *testing.T) {
			a := &filteredDiscardPlayer{BasePlayer: NewBasePlayer("A")}
			g := NewGame(a, NewBasePlayer("B"))
			artifact, land := NewArtifact("Artifact", ""), NewLand("Land")
			a.AddToHand(land)
			if mode != "empty" {
				a.AddToHand(artifact)
			}
			a.choice = artifact
			switch mode {
			case "nonmatching":
				a.choice = land
			case "outside":
				a.choice = NewArtifact("Outside", "")
			case "nil":
				a.choice = nil
			case "prevented":
				g.effects.AddReplacement(&preventOneCostDiscardReplacement{card: artifact.ID()})
			case "hidden":
				g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
			case "recheck":
				g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
				g.effects.AddReplacement(&discardToHandReplacement{})
			}
			cost := DiscardMatchingCardCost(IsArtifactCard)
			before := len(a.Hand())
			if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), cost}}); err == nil {
				t.Fatal("invalid payment prepared")
			}
			if err := cost.Pay(uuid.Nil, a.PlayerID(), g); err == nil {
				t.Fatal("invalid direct payment succeeded")
			}
			if a.Life() != 20 || len(a.Hand()) != before || len(a.Graveyard()) != 0 || len(a.Library()) != 0 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 0 {
				t.Fatal("invalid payment changed state")
			}
		})
	}
}

func TestFilteredDiscardPaymentContextsAndIsolation(t *testing.T) {
	a := &filteredDiscardPlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	a.AddToHand(first)
	a.AddToHand(second)
	cost := DiscardMatchingCardCost(IsArtifactCard)
	a.choice = first
	one, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	a.choice = second
	two, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	if err = two.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = one.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(a.Graveyard()) != 2 || a.calls != 2 {
		t.Fatal("declaration shared choices")
	}
	third := NewArtifact("Third", "")
	a.AddToHand(third)
	a.choice = third
	if err = ApplyEffect(g, UnlessPaid(You, Costs(cost), LoseLifeFor(You, Fixed(2))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || len(a.Graveyard()) != 3 {
		t.Fatal("resolution payment failed")
	}
	for _, card := range []Card{
		NewArtifact("Activation", "", WithAbility(Activate(Costs(cost), GainLifeFor(You, Fixed(1))))),
		NewInstant("Spell", "", WithAdditionalCost(cost), Spell(GainLifeFor(You, Fixed(1)))),
		NewInstant("Resolution", "", Spell(UnlessPaid(You, Costs(cost), LoseLifeFor(You, Fixed(2))))),
	} {
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFilteredDiscardPaymentSpellAndStackCopy(t *testing.T) {
	for _, mode := range []string{"activation", "spell", "optional", "alternative"} {
		t.Run(mode, func(t *testing.T) {
			a := &filteredDiscardPlayer{BasePlayer: NewBasePlayer("A")}
			g := NewGame(a, NewBasePlayer("B"))
			artifact := NewArtifact("Payment", "")
			a.AddToHand(artifact)
			a.choice = artifact
			cost := DiscardMatchingCardCost(IsArtifactCard)
			switch mode {
			case "optional":
				cost = OptionalCost(cost, "Pay?")
			case "alternative":
				cost = EitherCost(cost, ManaPayment("{99}"))
			}
			if mode == "activation" {
				source := NewArtifact("Source", "", WithAbility(Activate(Costs(cost), GainLifeFor(You, Fixed(3)))))
				g.PutOnBattlefield(source, a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			} else {
				spell := NewInstant("Synthetic", "", WithAbility(Spell(GainLifeFor(You, Fixed(3)))), WithAdditionalCost(cost))
				a.AddToHand(spell)
				if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != artifact.ID() {
				t.Fatal("wrong payment card")
			}
			g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
			clone := g.Clone()
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 26 || a.Life() != 20 {
				t.Fatal("copy or clone changed original")
			}
			g.ResolveStack()
			if a.Life() != 26 || a.calls != 1 {
				t.Fatal("copy paid again")
			}
		})
	}
}

func TestFilteredDiscardPaymentAbsentAndUnfiltered(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	cost := DiscardMatchingCardCost(IsArtifactCard)
	if cost.CanPay(uuid.Nil, uuid.New(), g) {
		t.Fatal("missing player can pay")
	}
	if err := cost.Pay(uuid.Nil, uuid.New(), g); err == nil {
		t.Fatal("missing player paid")
	}
	if err := ApplyEffect(g, UnlessPaid(You, Costs(cost), LoseLifeFor(You, Fixed(2))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 18 {
		t.Fatal("unavailable payment suppressed fallback")
	}
	a.AddToHand(NewLand("Land"))
	if err := DiscardMatchingCardCost(CardFilter{}).Pay(uuid.Nil, a.PlayerID(), g); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 0 || len(a.Graveyard()) != 1 {
		t.Fatal("zero filter did not match all cards")
	}
}

func TestFilteredDiscardPaymentAfterDraw(t *testing.T) {
	for _, mode := range []string{"pay", "decline", "unavailable"} {
		t.Run(mode, func(t *testing.T) {
			a := &filteredDiscardPlayer{BasePlayer: NewBasePlayer("A"), decline: mode == "decline"}
			g := NewGame(a, NewBasePlayer("B"))
			var first Card = NewArtifact("Drawn artifact", "")
			if mode == "unavailable" {
				first = NewLand("Drawn land")
			}
			a.SetLibrary([]Card{first, NewLand("Second"), NewLand("Third")})
			a.choice = first
			body := Sequence(DrawFor(You, Fixed(3)), UnlessPaid(You, Costs(DiscardMatchingCardCost(IsArtifactCard)), Discard(You, Fixed(2))))
			if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if mode == "pay" {
				if len(a.Hand()) != 2 || len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != first.ID() || a.calls != 1 {
					t.Fatal("payment did not choose after draw")
				}
			} else {
				if len(a.Hand()) != 1 || len(a.Graveyard()) != 2 || a.calls != 0 {
					t.Fatal("fallback or optional choice is wrong")
				}
			}
		})
	}
}
