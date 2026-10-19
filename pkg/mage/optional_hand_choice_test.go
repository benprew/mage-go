package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type optionalHandChoicePlayer struct {
	*BasePlayer
	accept          bool
	offers, choices int
	choose          func([]Card) Card
	offer           func()
}

func (p *optionalHandChoicePlayer) ChooseMayAbility(string) bool {
	p.offers++
	if p.offer != nil {
		p.offer()
	}
	return p.accept
}
func (p *optionalHandChoicePlayer) ChooseCardFromHand(cards []Card, _ string, _ GameReader) Card {
	p.choices++
	if p.choose != nil {
		return p.choose(cards)
	}
	return cards[len(cards)-1]
}

func TestOptionalHandChoice(t *testing.T) {
	for _, scenario := range []string{"chosen", "decline", "empty", "nonmatching", "nil", "foreign", "returned", "removed", "new", "changed", "snapshot", "canonical"} {
		t.Run(scenario, func(t *testing.T) {
			a := &optionalHandChoicePlayer{BasePlayer: NewBasePlayer("A"), accept: scenario != "decline"}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			first, second := NewCreature("First", "", 1, 1), NewCreature("Second", "", 2, 2)
			first.SetOwner(a.PlayerID())
			second.SetOwner(a.PlayerID())
			if scenario != "empty" && scenario != "nonmatching" {
				a.AddToHand(first)
				a.AddToHand(second)
			}
			if scenario == "nonmatching" {
				a.AddToHand(NewLand("Land"))
			}
			foreign := NewCreature("Foreign", "", 3, 3)
			foreign.SetOwner(b.PlayerID())
			b.AddToHand(foreign)
			switch scenario {
			case "nil":
				a.choose = func([]Card) Card { return nil }
			case "foreign":
				a.choose = func([]Card) Card { return foreign }
			case "returned":
				a.choose = func([]Card) Card { a.RemoveFromHand(second.ID()); a.AddToHand(second); return second }
			case "removed":
				a.choose = func([]Card) Card { a.RemoveFromHand(second.ID()); a.AddToGraveyard(second); return second }
			case "new":
				a.offer = func() { b.RemoveFromHand(foreign.ID()); foreign.SetOwner(a.PlayerID()); a.AddToHand(foreign) }
				a.choose = func([]Card) Card { return foreign }
			case "changed":
				a.choose = func([]Card) Card { second.SetOwner(b.PlayerID()); return second }
			case "snapshot":
				a.choose = func(cards []Card) Card { cards[1].SetOwner(b.PlayerID()); return cards[1] }
			case "canonical":
				a.choose = func([]Card) Card { fake := NewLand("Fake"); fake.SetID(second.ID()); return fake }
			}
			result := BindCard("chosen")
			seen := uuid.Nil
			paid := FuncEffect("observe chosen card", EffectProperties{}, func(g *Game, source, controller uuid.UUID, targets []uuid.UUID) error {
				card, ok := FromCard(result).resolveCard(&EffectContext{Game: g, Controller: controller})
				if !ok {
					t.Fatal("missing chosen hand visit")
				}
				seen = card.ID()
				if card.Owner() != a.PlayerID() {
					t.Fatal("choice changed the canonical owner")
				}
				return nil
			})
			unpaid := GainLifeFor(You, Fixed(3))
			filter := NewCardFilter("owned creature", func(c Card) bool { return c.HasType(core.TypeCreature) && c.Owner() == a.PlayerID() })
			effect := ChooseHandCardOrElse(You, filter, result, paid, unpaid)
			if err := ValidateCardDefinition("Choice", NewInstant("Choice", "", WithAbility(Spell(effect)))); err != nil {
				t.Fatal(err)
			}
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			chosen := scenario == "chosen" || scenario == "snapshot" || scenario == "canonical"
			if chosen && seen != second.ID() || !chosen && seen != uuid.Nil {
				t.Fatalf("wrong result: %s", seen)
			}
			if (a.Life() == 23) == chosen {
				t.Fatalf("wrong branch: life=%d", a.Life())
			}
			if _, ok := FromCard(result).resolveCard(&EffectContext{Game: g, Controller: a.PlayerID()}); ok {
				t.Fatal("local result escaped")
			}
			if len(g.PublicReveals()) != 0 {
				t.Fatal("selection revealed a card")
			}
			wantOffers, wantChoices := 1, 1
			if scenario == "empty" || scenario == "nonmatching" {
				wantOffers, wantChoices = 0, 0
			}
			if scenario == "decline" {
				wantChoices = 0
			}
			if a.offers != wantOffers || a.choices != wantChoices {
				t.Fatalf("offers/choices=%d/%d", a.offers, a.choices)
			}
		})
	}
}

func TestOptionalHandChoiceDefinition(t *testing.T) {
	result := BindCard("chosen")
	for _, effect := range []Effect{
		ChooseHandCardOrElse(You, CardFilter{}, result, ChooseHandCardOrElse(You, CardFilter{}, result, DiscardCard(FromCard(result)), nil), nil),
		ChooseHandCardOrElse(nil, CardFilter{}, result, GainLifeFor(You, Fixed(1)), nil),
		ChooseHandCardOrElse(You, CardFilter{}, nil, GainLifeFor(You, Fixed(1)), nil),
		ChooseHandCardOrElse(You, CardFilter{}, result, nil, nil),
		ChooseHandCardOrElse(You, CardFilter{}, result, nil, DiscardCard(FromCard(result))),
		Sequence(ChooseHandCardOrElse(You, CardFilter{}, result, GainLifeFor(You, Fixed(1)), nil), DiscardCard(FromCard(result))),
	} {
		if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", WithAbility(Spell(effect)))); err == nil {
			t.Fatal("invalid choice definition accepted")
		}
	}
	effect := ChooseHandCardOrElse(You, CardFilter{}, result, DiscardCard(FromCard(result)), nil)
	if err := ValidateCardDefinition("Valid", NewInstant("Valid", "", WithAbility(Spell(effect)))); err != nil {
		t.Fatal(err)
	}
}

func TestOptionalHandChoiceBindingAndClone(t *testing.T) {
	a := &optionalHandChoicePlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := NewLand("First"), NewLand("Second")
	first.SetOwner(a.PlayerID())
	second.SetOwner(a.PlayerID())
	a.AddToHand(first)
	a.AddToHand(second)
	result, inner := BindCard("outer"), BindCard("inner")
	var ids []uuid.UUID
	observe := func(binding *CardBinding) Effect {
		return FuncEffect("read local card", EffectProperties{}, func(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
			card, ok := FromCard(binding).resolveCard(&EffectContext{Game: g, Controller: controller})
			if !ok {
				t.Fatal("lost local result")
			}
			ids = append(ids, card.ID())
			return nil
		})
	}
	a.choose = func(cards []Card) Card {
		if a.choices == 1 {
			return cards[0]
		}
		return cards[1]
	}
	effect := ChooseHandCardOrElse(You, CardFilter{}, result, Sequence(observe(result), ChooseHandCardOrElse(You, CardFilter{}, inner, observe(inner), nil), observe(result)), nil)
	if err := ValidateCardDefinition("Nested", NewInstant("Nested", "", WithAbility(Spell(effect)))); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(ids) != 3 || ids[0] != first.ID() || ids[1] != second.ID() || ids[2] != first.ID() {
		t.Fatalf("nested identities=%v", ids)
	}
	chosen := ChooseHandCardOrElse(You, CardFilter{}, result, FuncEffect("fork local frame", EffectProperties{}, func(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
		clone := g.Clone()
		if err := DiscardCard(FromCard(result)).Apply(&EffectContext{Game: clone, Controller: controller}); err != nil {
			return err
		}
		if len(clone.GetPlayer(controller).Hand()) != 1 || len(g.GetPlayer(controller).Hand()) != 2 {
			t.Fatal("clone changed the original hand")
		}
		return nil
	}), nil)
	if err := ApplyEffect(g, chosen, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
}

func TestOptionalHandChoiceVisitAndRestore(t *testing.T) {
	for _, returned := range []bool{false, true} {
		a := &optionalHandChoicePlayer{BasePlayer: NewBasePlayer("A"), accept: true}
		g := NewGame(a, NewBasePlayer("B"))
		card := NewLand("Card")
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		result := BindCard("local")
		g.resolution.cardBindings = map[*CardBinding]capturedCard{result: {id: card.ID(), visit: g.handVisit(card.ID()), zone: core.ZoneHand}}
		saved := g.resolution.cardBindings[result]
		body := Sequence(FuncEffect("change the visit", EffectProperties{}, func(g *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
			a.RemoveFromHand(card.ID())
			if returned {
				a.AddToHand(card)
			} else {
				a.AddToGraveyard(card)
			}
			return nil
		}), DiscardCard(FromCard(result)))
		if err := ChooseHandCardOrElse(You, CardFilter{}, result, body, nil).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if returned && len(a.Hand()) != 1 || !returned && len(a.Graveyard()) != 1 {
			t.Fatal("local reference followed a departed visit")
		}
		if g.resolution.cardBindings[result] != saved {
			t.Fatal("choice did not restore the outer frame")
		}
	}
}
