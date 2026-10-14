package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceZoneUpkeepVisit(t *testing.T) {
	for _, zone := range []core.Zone{core.ZoneGraveyard, core.ZoneExile} {
		for _, scenario := range []string{"present", "depart", "return", "copy", "pending-clone"} {
			t.Run(zone.String()+"/"+scenario, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				cardBinding, playerBinding := BindCard("source"), BindPlayer("upkeep")
				var move = ReturnCardToHand(FromCard(cardBinding))
				if zone == core.ZoneExile {
					move = PutExiledCardsInOwnersGraveyards(ExiledCardFrom(FromCard(cardBinding)))
				}
				trigger := WhenEvent(TurnStepBegins(YourUpkeep, playerBinding), Sequence(
					move, GainLifeFor(FromPlayer(playerBinding), Fixed(2)), GainLifeFor(You, Fixed(1)),
				), WithSourceCardInZone(zone, cardBinding))
				card := NewArtifact("Zone watcher", "", WithAbility(trigger))
				card.SetOwner(a.PlayerID())
				if err := ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				enter := func(game *Game) {
					if zone == core.ZoneGraveyard {
						game.GetPlayer(a.PlayerID()).AddToGraveyard(card)
					} else {
						game.ExileCard(card, card.ID())
					}
				}
				leave := func() {
					if zone == core.ZoneGraveyard {
						a.RemoveFromGraveyard(card.ID())
					} else {
						g.RemoveFromExile(card.ID())
					}
				}
				enter(g)
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("opponent upkeep triggered")
				}
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
				var clone *Game
				if scenario == "pending-clone" {
					clone = g.Clone()
				}
				if scenario == "depart" || scenario == "return" {
					leave()
					if scenario == "return" {
						enter(g)
					}
				}
				g.PutTriggersOnStack()
				if g.StackSize() != 1 || a.Life() != 20 {
					t.Fatal("detection or resolution timing")
				}
				if scenario == "copy" {
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy absent")
					}
				}
				g.ResolveStack()
				wantA, wantB := 23, 20
				if scenario == "copy" {
					wantA, wantB = 25, 21
				}
				if a.Life() != wantA || b.Life() != wantB {
					t.Fatalf("saved player/controller: %d/%d", a.Life(), b.Life())
				}
				moved := scenario != "depart" && scenario != "return"
				if zone == core.ZoneGraveyard && (len(a.Hand()) == 1) != moved {
					t.Fatal("graveyard read followed another visit")
				}
				if zone == core.ZoneExile && (len(a.Graveyard()) == 1) != moved {
					t.Fatal("exile read followed another visit")
				}
				if clone != nil {
					if clone.GetPlayer(a.PlayerID()).Life() != 20 {
						t.Fatal("clone changed before resolution")
					}
					clone.PutTriggersOnStack()
					clone.ResolveStack()
					if clone.GetPlayer(a.PlayerID()).Life() != 23 {
						t.Fatal("clone lost pending facts")
					}
				}
			})
		}
	}
}

func TestSourceZoneUpkeepDeclarations(t *testing.T) {
	for _, zone := range []core.Zone{core.ZoneBattlefield, core.ZoneHand, core.ZoneAny} {
		card := NewArtifact("Invalid", "", WithAbility(WhenEvent(
			TurnStepBegins(YourUpkeep, BindPlayer("player")), GainLifeFor(You, Fixed(1)),
			WithSourceCardInZone(zone, BindCard("source")),
		)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid source zone accepted")
		}
	}
	for _, binding := range []*CardBinding{nil, BindCard("")} {
		card := NewArtifact("Invalid", "", WithAbility(WhenEvent(
			TurnStepBegins(YourUpkeep, BindPlayer("player")), GainLifeFor(You, Fixed(1)),
			WithSourceCardInZone(core.ZoneGraveyard, binding),
		)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid source binding accepted")
		}
	}
	for _, scenario := range []string{"duplicate", "extra-zone", "state", "intervening", "battlefield-source", "target"} {
		t.Run(scenario, func(t *testing.T) {
			var effect = GainLifeFor(You, Fixed(1))
			if scenario == "battlefield-source" {
				effect = TapSubject(This)
			}
			trigger := WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("player")), effect,
				WithSourceCardInZone(core.ZoneGraveyard, BindCard("source"))).(*GenericTriggered)
			switch scenario {
			case "duplicate":
				WithSourceCardInZone(core.ZoneExile, BindCard("another")).applyTrigger(trigger)
			case "extra-zone":
				trigger.InZone(core.ZoneExile)
			case "state":
				trigger.AsStateTrigger()
			case "intervening":
				WithInterveningIf(PermanentMatchesFilter(This, IsCreature)).applyTrigger(trigger)
			case "target":
				trigger.WithTargets(DeclarePlayerTarget("target"))
			}
			card := NewArtifact("Invalid", "", WithAbility(trigger))
			if err := ValidateCardDefinition(card.Name(), card); err == nil {
				t.Fatal("invalid source context accepted")
			}
		})
	}
}

func TestSourceZoneUpkeepDetection(t *testing.T) {
	for _, zone := range []core.Zone{core.ZoneGraveyard, core.ZoneExile} {
		t.Run(zone.String(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			cardBinding, playerBinding := BindCard("source"), BindPlayer("player")
			trigger := WhenEvent(TurnStepBegins(EachUpkeep, playerBinding), GainLifeFor(You, Fixed(1)),
				WithSourceCardInZone(zone, cardBinding)).(*GenericTriggered)
			card := NewArtifact("Watcher", "", WithAbility(trigger))
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			fire := func() {
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
			}
			fire()
			a.RemoveFromHand(card.ID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			fire()
			g.PhaseOut(p.ID())
			fire()
			g.PhaseIn([]uuid.UUID{p.ID()})
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			a.RemoveFromHand(card.ID())
			g.ExileCardFaceDown(card, card.ID(), a.PlayerID())
			fire()
			if g.StackSize() != 0 {
				t.Fatal("wrong zone, phased permanent or hidden exile triggered")
			}
			g.RemoveFromExile(card.ID())
			if zone == core.ZoneGraveyard {
				a.AddToGraveyard(card)
			} else {
				g.ExileCard(card, card.ID())
			}
			second := card.Copy()
			second.SetOwner(b.PlayerID())
			if zone == core.ZoneGraveyard {
				b.AddToGraveyard(second)
			} else {
				g.ExileCard(second, card.ID())
			}
			previousSource, previousController := trigger.Source(), trigger.Controller()
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			if trigger.Source() != previousSource || trigger.Controller() != previousController {
				t.Fatal("detection changed an immutable declaration")
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 4 || g.StackPeek().Controller != b.PlayerID() {
				t.Fatal("distinct events or APNAP ordering")
			}
			g.ResolveStack()
			if a.Life() != 22 || b.Life() != 22 {
				t.Fatal("separate source controllers lost")
			}
		})
	}
}
