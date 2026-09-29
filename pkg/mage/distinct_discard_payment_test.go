package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type distinctDiscardPlayer struct {
	*BasePlayer
	choices [][]Card
	calls   int
}

func (p *distinctDiscardPlayer) ChooseMayAbility(string) bool { return true }

func (p *distinctDiscardPlayer) ChooseCardsFromHand(_ int, _ string, _ GameReader) []Card {
	i := p.calls
	p.calls++
	return p.choices[i%len(p.choices)]
}

func TestDistinctDiscardPaymentLocksChoiceAndVisit(t *testing.T) {
	for _, mode := range []string{"unchanged", "returned", "missing", "clone"} {
		t.Run(mode, func(t *testing.T) {
			a := &distinctDiscardPlayer{BasePlayer: NewBasePlayer("A")}
			g := NewGame(a, NewBasePlayer("B"))
			first, second, third := NewLand("First"), NewLand("Second"), NewLand("Third")
			for _, c := range []Card{first, second, third} {
				a.AddToHand(c)
			}
			a.choices = [][]Card{{first, second}, {second, third}}
			a.ManaPool().Add(Red, 1)
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{ManaPayment("{R}"), LifePayCost(2), DiscardCardsWithDifferentNamesCost(2)}})
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "returned", "missing":
				c, _ := a.RemoveFromHand(first.ID())
				if mode == "returned" {
					a.AddToHand(c)
				}
			case "clone":
				clone := g.Clone()
				cloned := *tx
				cloned.game = clone
				if err := cloned.Commit(); err != nil {
					t.Fatal(err)
				}
				if len(clone.GetPlayer(a.PlayerID()).Graveyard()) != 2 || len(a.Hand()) != 3 {
					t.Fatal("clone changed live payment")
				}
			}
			err = tx.Commit()
			if mode == "returned" || mode == "missing" {
				if err == nil {
					t.Fatal("stale visit paid cost")
				}
				if a.Life() != 20 || a.ManaPool().Count(Red) != 1 || len(a.Graveyard()) != 0 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 0 {
					t.Fatal("stale payment spent resources")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if len(a.Hand()) != 1 || a.Hand()[0].ID() != third.ID() || a.Life() != 18 {
					t.Fatal("commit used another choice")
				}
			}
			if a.calls != 1 {
				t.Fatalf("choice calls = %d; want 1", a.calls)
			}
		})
	}
}

func TestDistinctDiscardPaymentInvalidChoices(t *testing.T) {
	for _, mode := range []string{"same name", "duplicate", "outside", "nil", "short", "too many"} {
		t.Run(mode, func(t *testing.T) {
			a := &fixedDiscardChoicesPlayer{BasePlayer: NewBasePlayer("A")}
			g := NewGame(a, NewBasePlayer("B"))
			first, duplicateName, second := NewLand("First"), NewLand("First"), NewLand("Second")
			for _, c := range []Card{first, duplicateName, second} {
				a.AddToHand(c)
			}
			switch mode {
			case "same name":
				a.selected = []Card{first, duplicateName}
			case "duplicate":
				a.selected = []Card{first, first}
			case "outside":
				a.selected = []Card{first, NewLand("Outside")}
			case "nil":
				a.selected = []Card{first, nil}
			case "short":
				a.selected = []Card{first}
			case "too many":
				a.selected = []Card{first, duplicateName, second}
			}
			cost := DiscardCardsWithDifferentNamesCost(2)
			if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), cost}}); err == nil {
				t.Fatal("invalid choice accepted")
			}
			if err := cost.Pay(uuid.Nil, a.PlayerID(), g); err == nil {
				t.Fatal("invalid direct choice accepted")
			}
			if a.Life() != 20 || len(a.Hand()) != 3 || len(a.Graveyard()) != 0 {
				t.Fatal("invalid choice changed state")
			}
		})
	}
}

func TestDistinctDiscardPaymentBatchAndReplacements(t *testing.T) {
	for _, mode := range []string{"normal", "prevented", "hidden", "recheck"} {
		t.Run(mode, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			first, second := NewLand("First"), NewLand("Second")
			a.AddToHand(first)
			a.AddToHand(second)
			switch mode {
			case "prevented":
				g.effects.AddReplacement(&preventOneCostDiscardReplacement{card: second.ID()})
			case "hidden":
				g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
			case "recheck":
				g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
				g.effects.AddReplacement(&discardToHandReplacement{})
			}
			seen := 0
			watcher := NewTriggered(EvtDiscard, false, GainLifeFor(You, Fixed(1))).SetCondition(func(evt *GameEvent, game GameReader, _, _ uuid.UUID) bool {
				p := game.GetPlayer(evt.PlayerID)
				if p != a {
					return false
				}
				seen++
				if len(p.Hand()) != 0 || len(p.Graveyard()) != 2 {
					t.Error("event saw partial discard payment")
				}
				return false
			})
			g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(watcher)), a.PlayerID())
			cost := DiscardCardsWithDifferentNamesCost(2)
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), cost}})
			if mode == "normal" {
				if err != nil {
					t.Fatal(err)
				}
				if err = tx.Commit(); err != nil {
					t.Fatal(err)
				}
				if seen != 2 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 2 {
					t.Fatal("missing discard events")
				}
			} else {
				if err == nil {
					t.Fatal("illegal replacement paid characteristic cost")
				}
				if err = cost.Pay(uuid.Nil, a.PlayerID(), g); err == nil {
					t.Fatal("illegal direct discard paid cost")
				}
				if a.Life() != 20 || len(a.Hand()) != 2 || len(a.Graveyard()) != 0 || len(a.Library()) != 0 || seen != 0 {
					t.Fatal("failed payment changed live state")
				}
			}
		})
	}
}

func TestDistinctDiscardPaymentZeroAndRegistration(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	a.AddToHand(NewLand("First"))
	cost := DiscardCardsWithDifferentNamesCost(0)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 1 {
		t.Fatal("zero cost discarded a card")
	}
	for _, n := range []int{-1, 0, 3} {
		card := NewArtifact("Synthetic", "", WithAbility(Activate(Costs(DiscardCardsWithDifferentNamesCost(n)), GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); (err != nil) != (n < 0) {
			t.Fatalf("count %d: %v", n, err)
		}
	}
}

func TestDistinctDiscardPaymentReusedDeclarationsAndCopies(t *testing.T) {
	a := &distinctDiscardPlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewLand("First"), NewLand("Second")
	a.AddToHand(first)
	a.AddToHand(second)
	a.choices = [][]Card{{first}, {second}}
	cost := DiscardCardsWithDifferentNamesCost(1)
	firstTx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	secondTx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	if err = secondTx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = firstTx.Commit(); err != nil {
		t.Fatal(err)
	}
	if a.calls != 2 || len(a.Graveyard()) != 2 {
		t.Fatal("declaration shared choices")
	}

	for _, name := range []string{"Third", "Fourth", "Fifth"} {
		a.AddToHand(NewLand(name))
	}
	a.choices = [][]Card{append([]Card(nil), a.Hand()...)}
	a.calls = 0
	source := NewArtifact("Synthetic", "", WithAbility(Activate(Costs(DiscardCardsWithDifferentNamesCost(3)), GainLifeFor(You, Fixed(2)))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 24 || a.Life() != 20 {
		t.Fatal("copy or clone lost isolation")
	}
	g.ResolveStack()
	if a.Life() != 24 || a.calls != 1 || len(a.Graveyard()) != 5 {
		t.Fatal("copied activation paid again")
	}
}

func TestDistinctDiscardPaymentSpellAndWrappers(t *testing.T) {
	for _, mode := range []string{"direct", "optional", "alternative"} {
		t.Run(mode, func(t *testing.T) {
			a := &distinctDiscardPlayer{BasePlayer: NewBasePlayer("A")}
			g := NewGame(a, NewBasePlayer("B"))
			cost := DiscardCardsWithDifferentNamesCost(2)
			switch mode {
			case "optional":
				cost = OptionalCost(cost, "Pay?")
			case "alternative":
				cost = EitherCost(cost, ManaPayment("{99}"))
			}
			spell := NewInstant("Synthetic", "", WithAbility(Spell(GainLifeFor(You, Fixed(3)))), WithAdditionalCost(cost))
			if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
				t.Fatal(err)
			}
			a.AddToHand(spell)
			first, second := NewLand("First"), NewLand("Second")
			a.AddToHand(first)
			a.AddToHand(second)
			a.choices = [][]Card{{first, second}}
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
				t.Fatal(err)
			}
			if len(a.Graveyard()) != 2 || len(a.Hand()) != 0 {
				t.Fatal("spell did not pay full cost")
			}
			g.ResolveStack()
			if a.Life() != 23 {
				t.Fatal("paid spell did not resolve")
			}
		})
	}
}

func TestDistinctDiscardPaymentResolutionAndAvailability(t *testing.T) {
	a := &distinctDiscardPlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	cost := DiscardCardsWithDifferentNamesCost(2)
	first, same := NewLand("First"), NewLand("First")
	a.AddToHand(first)
	a.AddToHand(same)
	if cost.CanPay(uuid.Nil, a.PlayerID(), g) {
		t.Fatal("two equal names supplied distinct resources")
	}
	if cost.CanPay(uuid.Nil, uuid.New(), g) {
		t.Fatal("absent payer can pay")
	}
	if DiscardCardsWithDifferentNamesCost(-1).CanPay(uuid.Nil, a.PlayerID(), g) {
		t.Fatal("negative cost can be paid")
	}
	second := NewLand("Second")
	a.AddToHand(second)
	a.choices = [][]Card{{first, second}}
	effect := IfPaid(You, Costs(cost), GainLifeFor(You, Fixed(2)))
	spell := NewInstant("Synthetic", "", WithAbility(Spell(effect)))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 || len(a.Graveyard()) != 2 || len(a.Hand()) != 1 || a.Hand()[0].ID() != same.ID() || a.calls != 1 {
		t.Fatal("resolution payment did not retain the chosen group")
	}
}
