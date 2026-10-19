package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestEntryGraveyardPaymentZero(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	card := NewCreature("Entry payment", "{X}", 0, 1, WithAbility(ExileGraveyardAsEntersWithCounterChoice(AnnouncedXForEnteringPermanent(), NewCardFilter("creature", func(c Card) bool { return c.HasType(core.TypeCreature) }), core.P2P0, core.P1P1, core.P0P2)))
	g.SetXValue(12)
	if p := g.PutOnBattlefield(card, a.PlayerID()); p == nil || p.CurrentPower(g) != 0 || p.CurrentToughness(g) != 1 {
		t.Fatal("noncast entry inherited X")
	}
}

type entryExileChooser struct {
	*BasePlayer
	mode    int
	calls   int
	change  func()
	invalid bool
}

func (p *entryExileChooser) ChooseMode(_ []string, _ string) int { p.calls++; return p.mode }
func (p *entryExileChooser) ChooseCardFromGraveyard(cards []Card, _ string, _ GameReader) Card {
	if p.change != nil {
		f := p.change
		p.change = nil
		f()
	}
	if p.invalid {
		return nil
	}
	return cards[0]
}
func entryExileDefinition() Card {
	return NewCreature("Entry payment", "{X}", 0, 1, WithAbility(ExileGraveyardAsEntersWithCounterChoice(AnnouncedXForEnteringPermanent(), NewCardFilter("creature", func(c Card) bool { return c.HasType(core.TypeCreature) }), core.P2P0, core.P1P1, core.P0P2)))
}
func castEntryExile(t *testing.T, g *Game, p Player, card Card, x int) {
	t.Helper()
	g.SetActivePlayerIndex(0)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	card.SetOwner(p.PlayerID())
	p.AddToHand(card)
	p.ManaPool().Add(core.Colorless, x)
	if err := g.CastSpellByID(p.PlayerID(), card.ID(), nil, x); err != nil {
		t.Fatal(err)
	}
}
func TestEntryGraveyardPaymentRechecks(t *testing.T) {
	for _, scenario := range []string{"invalid", "return", "short"} {
		t.Run(scenario, func(t *testing.T) {
			a := &entryExileChooser{BasePlayer: NewBasePlayer("A"), invalid: scenario == "invalid"}
			g := NewGame(a, NewBasePlayer("B"))
			c := NewCreature("Offering", "", 1, 1)
			c.SetOwner(a.PlayerID())
			a.AddToGraveyard(c)
			if scenario == "return" {
				a.change = func() { a.RemoveFromGraveyard(c.ID()); a.AddToGraveyard(c) }
			}
			x := 1
			if scenario == "short" {
				x = 2
			}
			card := entryExileDefinition()
			castEntryExile(t, g, a, card, x)
			g.ResolveStack()
			if scenario == "invalid" {
				if g.FindPermanent(card.ID()) == nil || len(g.GetExile()) != 1 || a.calls != 1 || g.graveyardVisit(c.ID()) != uuid.Nil {
					t.Fatal("invalid chooser declined mandatory payment")
				}
				return
			}
			if g.FindPermanent(card.ID()) != nil || len(g.GetExile()) != 0 || a.calls != 0 || g.graveyardVisit(c.ID()) == uuid.Nil {
				t.Fatal("invalid payment moved cards or entered")
			}
		})
	}
}
func TestEntryGraveyardPaymentReplacementCount(t *testing.T) {
	for _, scenario := range []string{"omit", "hand", "double"} {
		t.Run(scenario, func(t *testing.T) {
			a := &entryExileChooser{BasePlayer: NewBasePlayer("A"), mode: 1}
			g := NewGame(a, NewBasePlayer("B"))
			offering := NewCreature("Offering", "", 1, 1)
			offering.SetOwner(a.PlayerID())
			a.AddToGraveyard(offering)
			if scenario != "double" {
				g.effects.AddReplacement(&testPrioritizedReplacement{matchFunc: func(action Action) bool {
					m, ok := action.(*CardZoneMoveAction)
					return ok && m.CardID() == offering.ID()
				}, replaceFunc: func(action Action, _ *Game) Action {
					if scenario == "omit" {
						return nil
					}
					m := *action.(*CardZoneMoveAction)
					m.destination = core.ZoneHand
					return &m
				}})
			}
			donor := g.PutOnBattlefield(NewArtifact("Doubler", ""), a.PlayerID())
			if scenario == "double" {
				g.AddCounterDoubler(donor.ID(), core.P1P1, IsCreature)
			}
			card := entryExileDefinition()
			castEntryExile(t, g, a, card, 1)
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p == nil {
				t.Fatal("replaced payment must still enter")
			}
			want := 0
			if scenario == "double" {
				want = 2
			}
			if p.Counters[core.P1P1] != want || a.calls != want/2 {
				t.Fatal("actual count or counter replacement failed")
			}
			if scenario == "hand" && len(a.Hand()) != 1 {
				t.Fatal("replacement destination ignored")
			}
		})
	}
}
func TestEntryGraveyardPaymentFailedDestination(t *testing.T) {
	g, a, b := newSpellCopyGame()
	card := entryExileDefinition()
	castEntryExile(t, g, a, card, 1)
	g.effects.AddReplacement(&testPrioritizedReplacement{matchFunc: func(action Action) bool {
		m, ok := action.(*CardZoneMoveAction)
		return ok && m.CardID() == card.ID() && m.Destination() == core.ZoneGraveyard
	}, replaceFunc: func(action Action, _ *Game) Action { return action.(*CardZoneMoveAction).WithExile(b.PlayerID()) }})
	g.ResolveStack()
	if g.FindPermanent(card.ID()) != nil || g.FindExiledCard(card.ID()) == nil || len(a.Graveyard()) != 0 {
		t.Fatal("failed entry bypassed zone replacement")
	}
}

func TestEntryGraveyardPaymentSimultaneous(t *testing.T) {
	g, a, b := newSpellCopyGame()
	offerings := make([]Card, 0, 3)
	for range 3 {
		c := NewCreature("Offering", "", 1, 1)
		c.SetOwner(a.PlayerID())
		a.AddToGraveyard(c)
		offerings = append(offerings, c)
	}
	zones, batches, entries := 0, 0, 0
	observer := NewTriggered(core.EvtZoneChange, false).SetCondition(func(event *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if event.FromZone == core.ZoneGraveyard && event.ToZone == core.ZoneExile {
			zones++
			for _, c := range offerings {
				if g.FindExiledCard(c.ID()) == nil {
					t.Fatal("zone event before all exile moves")
				}
			}
		}
		return false
	})
	batch := NewTriggered(core.EvtCardsLeftGraveyard, false).SetCondition(func(event *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		batches++
		if event.PlayerID != a.PlayerID() || event.Amount != 3 {
			t.Fatal("split graveyard batch")
		}
		return false
	})
	entry := NewTriggered(core.EvtZoneChange, false).SetCondition(func(event *core.GameEvent, r GameReader, _, _ uuid.UUID) bool {
		if p := r.FindPermanent(event.SourceID); event.ToZone == core.ZoneBattlefield && p != nil && p.Name() == "Entry payment" {
			entries++
			if p.CurrentPower(r) != 6 || p.CurrentToughness(r) != 1 {
				t.Fatal("entry observer did not see counters")
			}
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(observer), WithAbility(batch), WithAbility(entry)), b.PlayerID())
	card := entryExileDefinition()
	castEntryExile(t, g, a, card, 3)
	g.ResolveStack()
	if zones != 3 || batches != 1 || entries != 1 {
		t.Fatalf("events zones=%d batches=%d entries=%d", zones, batches, entries)
	}
}
func TestEntryGraveyardPaymentDefinition(t *testing.T) {
	filter := NewCardFilter("creature", func(c Card) bool { return c.HasType(core.TypeCreature) })
	valid := func() Ability {
		return ExileGraveyardAsEntersWithCounterChoice(AnnouncedXForEnteringPermanent(), filter, core.P1P1)
	}
	for _, card := range []Card{
		NewCreature("Bad", "", 1, 1, WithAbility(ExileGraveyardAsEntersWithCounterChoice(nil, filter, core.P1P1))),
		NewCreature("Bad", "", 1, 1, WithAbility(ExileGraveyardAsEntersWithCounterChoice(AnnouncedXForEnteringPermanent(), CardFilter{}, core.P1P1))),
		NewCreature("Bad", "", 1, 1, WithAbility(ExileGraveyardAsEntersWithCounterChoice(AnnouncedXForEnteringPermanent(), filter))),
		NewCreature("Bad", "", 1, 1, WithAbility(ExileGraveyardAsEntersWithCounterChoice(AnnouncedXForEnteringPermanent(), filter, core.CounterType(999)))),
		NewCreature("Bad", "", 1, 1, WithAbility(valid()), WithAbility(valid())), NewInstant("Bad", "", WithAbility(valid())),
	} {
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid entry definition accepted")
		}
	}
}

func TestEntryGraveyardPaymentAcquiredCopy(t *testing.T) {
	a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	donorPlan, err := NewCardDefinitionPlan(entryExileDefinition)
	if err != nil {
		t.Fatal(err)
	}
	donor, err := donorPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	donor.SetOwner(b.PlayerID())
	source := g.PutOnBattlefield(donor, b.PlayerID())
	source.AddCounter(core.P2P0, 7)
	plan, err := NewCardDefinitionPlan(func() Card { return NewCreature("Copy entrant", "{X}", 1, 1, WithAbility(MayCopyAsEnters(IsCreature))) })
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	offering := NewCreature("Offering", "", 1, 1)
	offering.SetOwner(a.PlayerID())
	a.AddToGraveyard(offering)
	castEntryExile(t, g, a, card, 1)
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if p == nil || p.Name() != "Entry payment" || p.Counters[core.P2P0] != 1 || len(g.GetExile()) != 1 || source.Counters[core.P2P0] != 7 || a.offers != 1 {
		t.Fatal("acquired entry payment repeated or inherited donor state")
	}
}

func TestEntryGraveyardPaymentStackExileRule(t *testing.T) {
	for _, marked := range []bool{false, true} {
		g, a, _ := newSpellCopyGame()
		card := entryExileDefinition()
		castEntryExile(t, g, a, card, 1)
		if marked {
			g.zones.AddExileInstead(card.ID(), uuid.New())
		} else {
			g.StackPeek().ExileOnLeaveStack = true
		}
		g.ResolveStack()
		if g.FindExiledCard(card.ID()) == nil || len(a.Graveyard()) != 0 {
			t.Fatal("failed entry lost stack exile rule")
		}
	}
}
