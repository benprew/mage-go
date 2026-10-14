package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestMatchingPermanentExileCostTransaction(t *testing.T) {
	for _, scenario := range []string{"success", "phased", "opponent", "wrong-type", "double", "life"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			card := NewCreature("Victim", "", 2, 2)
			card.SetOwner(b.PlayerID())
			controller := a.PlayerID()
			if scenario == "opponent" {
				controller = b.PlayerID()
			}
			victim := g.PutOnBattlefield(card, controller)
			if scenario == "phased" {
				g.PhaseOut(victim.ID())
			}
			filter := IsCreature
			if scenario == "wrong-type" {
				filter = IsLand
			}
			costs := []Cost{Tap(), ExileMatchingPermanentCost(filter, "Exile a creature")}
			if scenario == "double" {
				costs = append(costs, ExileMatchingPermanentCost(filter, "Exile another creature"))
			}
			if scenario == "life" {
				costs = append(costs, LifePayCost(21))
			}
			before := g.Clone()
			for range 2 {
				costs[1].CanPay(source.ID(), a.PlayerID(), g)
			}
			tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: costs}, nil)
			if err == nil {
				err = tx.Commit()
			}
			if scenario == "success" {
				if err != nil {
					t.Fatal(err)
				}
				exiled := g.FindExiledCard(card.ID())
				if g.FindPermanent(victim.ID()) != nil || exiled == nil || exiled.Card.Owner() != b.PlayerID() || exiled.ExiledBy != source.ID() || !g.FindPermanent(source.ID()).Tapped || len(b.Graveyard()) != 0 {
					t.Fatal("wrong payment destination or resources")
				}
			} else if err == nil || g.FindPermanentIncludingPhased(victim.ID()) == nil || g.FindPermanent(source.ID()).Tapped || len(g.GetExile()) != 0 || a.Life() != 20 {
				t.Fatal("failed payment changed state")
			}
			if before.FindPermanentIncludingPhased(victim.ID()) == nil || before.FindPermanent(source.ID()).Tapped || len(before.GetExile()) != 0 {
				t.Fatal("payment changed clone")
			}
		})
	}
}

func TestMatchingPermanentExileCostStaleVisit(t *testing.T) {
	for _, change := range []string{"return", "phase", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			card := NewCreature("Victim", "", 2, 2)
			card.SetOwner(a.PlayerID())
			victim := g.PutOnBattlefield(card, a.PlayerID())
			_, err := g.prepareCostPaymentTransaction(costPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Mana: core.ParseManaCost("{1}"), Costs: []Cost{Tap(), ExileMatchingPermanentCost(IsCreature, "Exile")}}, nil)
			if err == nil {
				t.Fatal("missing mana accepted")
			}
			a.ManaPool().Add(core.Colorless, 1)
			tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Mana: core.ParseManaCost("{1}"), Costs: []Cost{Tap(), ExileMatchingPermanentCost(IsCreature, "Exile")}}, nil)
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(victim)
				g.PutOnBattlefield(card, a.PlayerID())
			case "phase":
				g.PhaseOut(victim.ID())
			case "control":
				victim.computedController = b.PlayerID()
				victim.baseController = b.PlayerID()
			}
			if tx.Commit() == nil || g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 1 || len(g.GetExile()) != 0 {
				t.Fatal("stale choice spent resources")
			}
		})
	}
}

func TestMatchingPermanentExileCostSourceAndDefinition(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 2, 2)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	cost := ExileMatchingPermanentCost(IsCreature, "Exile")
	if !cost.CanPay(p.ID(), a.PlayerID(), g) || cost.Pay(p.ID(), a.PlayerID(), g) != nil || g.FindExiledCard(card.ID()) == nil {
		t.Fatal("source cannot pay")
	}
	if cost.CanPay(uuid.Nil, uuid.New(), g) {
		t.Fatal("missing player can pay")
	}
	for _, bad := range []Cost{ExileMatchingPermanentCost(PermanentFilter{}, "Exile"), ExileMatchingPermanentCost(IsCreature, "")} {
		c := NewArtifact("Bad", "", WithAbility(Activate(Costs(bad), GainLifeFor(You, Fixed(1)))))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid cost accepted")
		}
	}
}

type matchingExileChooser struct {
	*BasePlayer
	choices int
	invalid *Permanent
}

func (p *matchingExileChooser) ChoosePermanent(candidates []*Permanent, _ string, _ GameReader) *Permanent {
	p.choices++
	if p.invalid != nil {
		return p.invalid
	}
	return candidates[0]
}

func TestMatchingPermanentExileCostQueriesAndEvents(t *testing.T) {
	for _, token := range []bool{false, true} {
		t.Run(fmt.Sprint(token), func(t *testing.T) {
			a := &matchingExileChooser{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			card := NewCreature("Victim", "", 2, 2)
			card.SetOwner(b.PlayerID())
			victim := g.PutOnBattlefield(card, a.PlayerID())
			victim.IsToken = token
			for _, attr := range []core.Attr{core.Shroud, core.Hexproof, core.Indestructible} {
				victim.GrantBaseAttr(attr)
			}
			g.AddRegenerationShield(victim.ID())
			events := 0
			observer := NewTriggered(core.EvtZoneChange, false).SetCondition(func(event *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
				if reader != g {
					return false
				}
				if event.SourceID == victim.ID() && event.FromZone == core.ZoneBattlefield {
					events++
					if event.ToZone != core.ZoneExile || event.PlayerID != a.PlayerID() || g.FindPermanent(victim.ID()) != nil {
						t.Fatal("wrong exile event")
					}
				}
				return false
			})
			g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(observer)), a.PlayerID())
			cost := ExileMatchingPermanentCost(IsCreature, "Exile")
			for range 3 {
				if !cost.CanPay(source.ID(), a.PlayerID(), g) {
					t.Fatal("protected creature cannot pay")
				}
			}
			if a.choices != 0 || len(g.GetExile()) != 0 || events != 0 {
				t.Fatal("query chose or moved")
			}
			tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{cost}}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if a.choices != 1 || events != 0 || g.FindPermanent(victim.ID()) == nil {
				t.Fatal("preparation changed live state")
			}
			if err = tx.Commit(); err != nil {
				t.Fatal(err)
			}
			if a.choices != 1 || events != 1 || (g.FindExiledCard(card.ID()) != nil) == token {
				t.Fatal("wrong token or event contract")
			}
			second := NewCreature("Second", "", 1, 1)
			second.SetOwner(a.PlayerID())
			g.PutOnBattlefield(second, a.PlayerID())
			if err = cost.Pay(source.ID(), a.PlayerID(), g); err != nil {
				t.Fatal("cost definition retained the previous choice", err)
			}
		})
	}
}
