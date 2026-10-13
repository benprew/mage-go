package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSingleGraveyardExileCostSelection(t *testing.T) {
	a := &graveyardExileChoicePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	var own, foreign []Card
	for _, player := range []Player{a, b} {
		for range 3 {
			card := NewCreature("Payment", "{G}", 1, 1)
			card.SetOwner(player.PlayerID())
			player.AddToGraveyard(card)
			if player == a {
				own = append(own, card)
			} else {
				foreign = append(foreign, card)
			}
		}
		land := NewLand("Excluded")
		land.SetOwner(player.PlayerID())
		player.AddToGraveyard(land)
	}
	cost := ExileMatchingCardsFromSingleGraveyardCost(2, IsCreatureCard, EachPlayer)
	for range 2 {
		if !cost.CanPay(uuid.Nil, a.PlayerID(), g) {
			t.Fatal("payable cost rejected")
		}
	}
	if len(a.seen) != 0 || len(g.GetExile()) != 0 {
		t.Fatal("query made choices or moved cards")
	}
	a.choices = []uuid.UUID{foreign[2].ID(), foreign[0].ID()}
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{cost, LifePayCost(2)}})
	if err != nil {
		t.Fatal(err)
	}
	if len(a.seen) != 2 || len(a.seen[0]) != 6 || len(a.seen[1]) != 2 {
		t.Fatalf("choice sets = %v", a.seen)
	}
	if len(g.GetExile()) != 0 || a.Life() != 20 {
		t.Fatal("preparation spent resources")
	}
	clone := g.Clone()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(g.GetExile()) != 2 || len(a.Graveyard()) != 4 || len(b.Graveyard()) != 2 || a.Life() != 18 {
		t.Fatal("wrong graveyard or cost payment")
	}
	if g.FindExiledCard(foreign[1].ID()) != nil || g.FindExiledCard(own[0].ID()) != nil || g.FindExiledCard(foreign[2].ID()).Card.Owner() != b.PlayerID() {
		t.Fatal("choice or ownership changed")
	}
	if len(clone.GetExile()) != 0 || len(clone.GetPlayer(b.PlayerID()).Graveyard()) != 4 {
		t.Fatal("payment changed clone")
	}
}

func TestSingleGraveyardExileCostRollback(t *testing.T) {
	for _, reason := range []string{"split", "invalid", "duplicate", "cross-graveyard", "returned", "absent", "type-change", "overlap"} {
		t.Run(reason, func(t *testing.T) {
			a := &graveyardExileChoicePlayer{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			first, second, foreign := NewCreature("First", "", 1, 1), NewCreature("Second", "", 1, 1), NewCreature("Foreign", "", 1, 1)
			for _, card := range []Card{first, second} {
				card.SetOwner(a.PlayerID())
				a.AddToGraveyard(card)
			}
			foreign.SetOwner(b.PlayerID())
			b.AddToGraveyard(foreign)
			if reason == "split" {
				a.RemoveFromGraveyard(second.ID())
			}
			cost := ExileMatchingCardsFromSingleGraveyardCost(2, IsCreatureCard, EachPlayer)
			a.choices = []uuid.UUID{first.ID(), second.ID()}
			switch reason {
			case "invalid":
				a.choices[0] = uuid.New()
			case "duplicate":
				a.choices[1] = first.ID()
			case "cross-graveyard":
				a.choices[1] = foreign.ID()
			}
			costs := []Cost{LifePayCost(2), cost}
			if reason == "overlap" {
				costs = append(costs, ExileMatchingCardsFromSingleGraveyardCost(1, IsCreatureCard, EachPlayer))
				a.choices = append(a.choices, first.ID())
			}
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: costs})
			if reason == "returned" || reason == "absent" || reason == "type-change" {
				if err != nil {
					t.Fatal(err)
				}
				if reason == "type-change" {
					first.types = []core.CardType{core.TypeLand}
				} else {
					removed, ok := g.MoveFromGraveyard(a.PlayerID(), first.ID(), core.ZoneHand)
					if !ok {
						t.Fatal("move failed")
					}
					if reason == "returned" {
						a.AddToGraveyard(removed)
					} else {
						a.AddToHand(removed)
					}
				}
				err = tx.Commit()
			}
			if err == nil {
				t.Fatal("invalid payment accepted")
			}
			if a.Life() != 20 || len(g.GetExile()) != 0 {
				t.Fatal("failed payment spent resources")
			}
		})
	}
}

func TestSingleGraveyardExileCostBatch(t *testing.T) {
	a := &graveyardExileChoicePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	for range 2 {
		card := NewCreature("Payment", "", 1, 1)
		card.SetOwner(b.PlayerID())
		b.AddToGraveyard(card)
		a.choices = append(a.choices, card.ID())
	}
	zones, batches := 0, 0
	observer := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, r GameReader, _, _ uuid.UUID) bool {
		if e.FromZone == core.ZoneGraveyard && e.ToZone == core.ZoneExile {
			zones++
			if len(r.GetPlayer(b.PlayerID()).Graveyard()) != 0 || len(g.GetExile()) != 2 {
				t.Error("event saw a partial batch")
			}
			if e.PlayerID != b.PlayerID() {
				t.Error("wrong zone player")
			}
		}
		return false
	})
	batch := NewTriggered(core.EvtCardsLeftGraveyard, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		batches++
		if e.Amount != 2 || e.PlayerID != b.PlayerID() {
			t.Error("wrong batch facts")
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(observer), WithAbility(batch)), a.PlayerID())
	cost := ExileMatchingCardsFromSingleGraveyardCost(2, IsCreatureCard, EachPlayer)
	if err := cost.Pay(uuid.Nil, a.PlayerID(), g); err != nil {
		t.Fatal(err)
	}
	if zones != 2 || batches != 1 {
		t.Fatalf("events = %d / %d", zones, batches)
	}
}

func TestSingleGraveyardExileCostDefinitionAndZero(t *testing.T) {
	for _, cost := range []Cost{
		ExileMatchingCardsFromSingleGraveyardCost(-1, IsCreatureCard, EachPlayer),
		ExileMatchingCardsFromSingleGraveyardCost(2, CardFilter{}, EachPlayer),
		ExileMatchingCardsFromSingleGraveyardCost(2, IsCreatureCard, nil),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(cost), GainLifeFor(You, Fixed(1)))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	a := &graveyardExileChoicePlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	cost := ExileMatchingCardsFromSingleGraveyardCost(0, IsCreatureCard, EachPlayer)
	if !cost.CanPay(uuid.Nil, a.PlayerID(), g) {
		t.Fatal("zero cost cannot be paid")
	}
	if err := cost.Pay(uuid.Nil, a.PlayerID(), g); err != nil {
		t.Fatal(err)
	}
	if len(a.seen) != 0 || len(g.GetExile()) != 0 {
		t.Fatal("zero cost chose or moved cards")
	}
	if cost.CanPay(uuid.Nil, uuid.New(), g) {
		t.Fatal("absent payer can pay")
	}
}

type singleGraveyardMutationPlayer struct {
	*graveyardExileChoicePlayer
	beforeChoice func()
}

func (p *singleGraveyardMutationPlayer) ChooseCardFromGraveyard(cards []Card, text string, reader GameReader) Card {
	chosen := p.graveyardExileChoicePlayer.ChooseCardFromGraveyard(cards, text, reader)
	if p.beforeChoice != nil {
		p.beforeChoice()
	}
	return chosen
}

func TestSingleGraveyardExileCostChoiceVisit(t *testing.T) {
	a := &singleGraveyardMutationPlayer{graveyardExileChoicePlayer: &graveyardExileChoicePlayer{BasePlayer: NewBasePlayer("A")}}
	g := NewGame(a, NewBasePlayer("B"))
	first := NewCreature("Payment", "", 1, 1)
	first.SetOwner(a.PlayerID())
	a.AddToGraveyard(first)
	a.choices = []uuid.UUID{first.ID()}
	a.beforeChoice = func() {
		card, ok := g.MoveFromGraveyard(a.PlayerID(), first.ID(), core.ZoneHand)
		if !ok {
			t.Fatal("move failed")
		}
		a.AddToGraveyard(card)
	}
	cost := ExileMatchingCardsFromSingleGraveyardCost(1, IsCreatureCard, EachPlayer)
	if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), cost}}); err == nil {
		t.Fatal("returned offered card paid the cost")
	}
	if a.Life() != 20 || len(g.GetExile()) != 0 || len(a.Graveyard()) != 1 {
		t.Fatal("stale choice spent resources")
	}
}

func TestSingleGraveyardExileCostPlayerSetAndAlternative(t *testing.T) {
	a := &graveyardExileChoicePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	for range 2 {
		card := NewCreature("Own", "", 1, 1)
		card.SetOwner(a.PlayerID())
		a.AddToGraveyard(card)
	}
	cost := ExileMatchingCardsFromSingleGraveyardCost(2, IsCreatureCard, OpponentsOf(You))
	if cost.CanPay(uuid.Nil, a.PlayerID(), g) {
		t.Fatal("own graveyard supplied opponent payment")
	}
	for range 2 {
		card := NewCreature("Foreign", "", 1, 1)
		card.SetOwner(b.PlayerID())
		b.AddToGraveyard(card)
		a.choices = append(a.choices, card.ID())
	}
	a.mode = 0
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{EitherCost(cost, LifePayCost(2))}})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if a.modes != 1 || len(b.Graveyard()) != 0 || len(a.Graveyard()) != 2 || a.Life() != 20 {
		t.Fatal("alternative changed the payment or selected player")
	}
}
