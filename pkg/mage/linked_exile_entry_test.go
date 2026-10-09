package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLinkedExileEntryBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	link := NewExileLink()
	source := NewArtifact("Source", "")
	source.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(source, a.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	cards := []*BaseCard{NewCreature("First", "", 2, 2, WithAbility(EnterTappedAsEnters())), NewCreature("Second", "", 3, 3, WithAbility(EntersWithNCounters(core.P1P1, 2)))}
	for i, card := range cards {
		owner := a.PlayerID()
		if i == 1 {
			owner = b.PlayerID()
		}
		card.SetOwner(owner)
		target := g.PutOnBattlefield(card, a.PlayerID())
		target.AddCounter(core.P1P1, 4)
		bound := BindPermanent("subject")
		g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{bound: {target.ID(), target.incarnationID}}
		if err := ExilePermanentLinked(From(bound), link).Apply(ctx); err != nil {
			t.Fatal(err)
		}
	}
	unlinked := NewCreature("Unlinked", "", 1, 1)
	unlinked.SetOwner(a.PlayerID())
	g.ExileCard(unlinked, p.ID())
	events := 0
	watcher := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if e.FromZone == core.ZoneExile && e.ToZone == core.ZoneBattlefield {
			events++
			for _, c := range cards {
				if g.FindPermanent(c.ID()) == nil || g.FindExiledCard(c.ID()) != nil {
					t.Error("entry event preceded the batch")
				}
			}
			if e.ActorID != a.PlayerID() {
				t.Error("wrong actor")
			}
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), a.PlayerID())
	body := PutExiledCardsOntoBattlefieldUnderOwnersControl(CardsExiledWith(link, This))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(p)
	for range 2 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || len(g.GetExile()) != 3 {
			t.Fatal("waiting query lost source or moved cards")
		}
	}
	clone := g.Clone()
	if err := body.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if events != 2 || g.FindExiledCard(unlinked.ID()) == nil {
		t.Fatal("wrong batch membership")
	}
	first, second := g.FindPermanent(cards[0].ID()), g.FindPermanent(cards[1].ID())
	if first.ControllerID() != a.PlayerID() || second.ControllerID() != b.PlayerID() || !first.Tapped || first.Counters[core.P1P1] != 0 || second.Counters[core.P1P1] != 2 {
		t.Fatal("owner or fresh entry rules lost")
	}
	if len(clone.GetExile()) != 3 || clone.FindPermanent(cards[0].ID()) != nil {
		t.Fatal("clone changed")
	}
}

func TestLinkedExileEntryVisitsAndProhibition(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g, err := NewGameWithAnte(a, b, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	cards := []*BaseCard{NewCreature("Old visit", "", 1, 1), NewCreature("New owner", "", 1, 1), NewLand("Blocked"), NewInstant("Spell", ""), NewToken("Token", 1, 1, []core.CardType{core.TypeCreature}, nil)}
	saved := make(savedTransportVisits, 0, len(cards)+1)
	for _, c := range cards {
		c.SetOwner(a.PlayerID())
		g.ExileCardFaceDown(c, uuid.New())
		saved = append(saved, capturedCard{id: c.ID(), visit: g.FindExiledCard(c.ID()).Visit, zone: core.ZoneExile})
	}
	saved = append(saved, saved[1])
	c, _ := g.RemoveFromExile(cards[0].ID())
	g.ExileCard(c, uuid.New())
	if err := g.ChangeOwner(cards[1].ID(), b.PlayerID()); err != nil {
		t.Fatal(err)
	}
	g.effects.Rules.SetLandsCantEnter(true)
	if err := PutExiledCardsOntoBattlefieldUnderOwnersControl(saved).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if p := g.FindPermanent(cards[1].ID()); p == nil || p.ControllerID() != b.PlayerID() {
		t.Fatal("current owner ignored")
	}
	for _, i := range []int{0, 2, 3, 4} {
		if g.FindPermanent(cards[i].ID()) != nil || g.FindExiledCard(cards[i].ID()) == nil {
			t.Fatal("ineligible visit moved")
		}
	}
	if len(g.AllBattlefield()) != 1 || len(a.Graveyard()) != 0 {
		t.Fatal("duplicate entry or prohibited card lost")
	}
}

func TestLinkedExileEntryAura(t *testing.T) {
	for _, scenario := range []string{"legal", "no-host", "phased", "protected", "returned-host"} {
		t.Run(scenario, func(t *testing.T) {
			a := &exileEntryPlayer{BasePlayer: NewBasePlayer("A")}
			g := NewGame(a, NewBasePlayer("B"))
			hostCard := NewCreature("Host", "", 1, 1, WithKeyword(core.Shroud))
			hostCard.SetOwner(a.PlayerID())
			host := g.PutOnBattlefield(hostCard, a.PlayerID())
			switch scenario {
			case "no-host":
				g.BouncePermanentToHand(host)
			case "phased":
				g.PhaseOut(host.ID())
			case "protected":
				host.GrantBaseAttr(core.AttrCantBeEnchanted)
			case "returned-host":
				a.choose = func(options []*Permanent) *Permanent {
					g.BouncePermanentToHand(host)
					a.RemoveFromHand(host.ID())
					g.PutOnBattlefield(hostCard, a.PlayerID())
					return options[0]
				}
			}
			aura := NewAura("Aura", "", WithCastTarget(TargetCreature()))
			aura.SetOwner(a.PlayerID())
			g.ExileCard(aura, uuid.New())
			visit := g.FindExiledCard(aura.ID()).Visit
			if err := PutExiledCardsOntoBattlefieldUnderOwnersControl(savedTransportVisits{{id: aura.ID(), visit: visit, zone: core.ZoneExile}}).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if scenario == "legal" {
				if p := g.FindPermanent(aura.ID()); p == nil || p.AttachedTo != host.ID() {
					t.Fatal("Aura entry incorrectly targeted or unattached")
				}
			} else if g.FindPermanent(aura.ID()) != nil || g.FindExiledCard(aura.ID()) == nil || g.FindExiledCard(aura.ID()).Visit != visit {
				t.Fatal("illegal Aura lost its exile visit")
			}
		})
	}
}

type exileEntryPlayer struct {
	*BasePlayer
	choose func([]*Permanent) *Permanent
}

func (p *exileEntryPlayer) ChoosePermanent(options []*Permanent, _ string, _ GameReader) *Permanent {
	if p.choose != nil {
		return p.choose(options)
	}
	return options[0]
}

func TestLinkedExileEntryDefinition(t *testing.T) {
	for _, set := range []ExiledCardSet{nil, CardsExiledWith(nil, This), CardsExiledWith(NewExileLink(), nil)} {
		c := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), PutExiledCardsOntoBattlefieldUnderOwnersControl(set))))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid entry declaration accepted")
		}
	}
}

func TestLinkedExileEntryChoiceRechecks(t *testing.T) {
	for _, scenario := range []string{"reexile", "owner", "blocked", "failure"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g, err := NewGameWithAnte(a, b, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			first := NewLand("First")
			first.SetOwner(a.PlayerID())
			var choiceOrder []uuid.UUID
			probeA := &optionalEntryProbe{run: func(g *Game, _ *Permanent) bool {
				choiceOrder = append(choiceOrder, a.PlayerID())
				if len(g.GetExile()) != 2 {
					t.Error("movement preceded entry choices")
				}
				return true
			}}
			first.AddAbility(probeA)
			probeB := &optionalEntryProbe{run: func(g *Game, _ *Permanent) bool {
				choiceOrder = append(choiceOrder, b.PlayerID())
				if len(g.GetExile()) != 2 {
					t.Error("movement preceded all owners' entry choices")
				}
				switch scenario {
				case "reexile":
					card, _ := g.RemoveFromExile(first.ID())
					g.ExileCard(card, uuid.New())
				case "owner":
					if err := g.ChangeOwner(first.ID(), b.PlayerID()); err != nil {
						t.Fatal(err)
					}
				case "blocked":
					g.effects.Rules.SetLandsCantEnter(true)
				case "failure":
					return false
				}
				return true
			}}
			second := NewCreature("Second", "", 1, 1, WithAbility(probeB))
			second.SetOwner(b.PlayerID())
			g.ExileCardFaceDown(second, uuid.New())
			g.ExileCardFaceDown(first, uuid.New())
			visits := savedTransportVisits{{id: second.ID(), visit: g.FindExiledCard(second.ID()).Visit, zone: core.ZoneExile}, {id: first.ID(), visit: g.FindExiledCard(first.ID()).Visit, zone: core.ZoneExile}}
			if err := PutExiledCardsOntoBattlefieldUnderOwnersControl(visits).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if len(choiceOrder) != 2 || choiceOrder[0] != a.PlayerID() || choiceOrder[1] != b.PlayerID() {
				t.Fatal("entry choices were not APNAP")
			}
			if scenario == "failure" {
				if g.FindExiledCard(second.ID()) != nil || g.FindPermanent(second.ID()) != nil || len(b.Graveyard()) != 1 || g.FindPermanent(first.ID()) == nil {
					t.Fatal("entry operation failure did not retain its graveyard destination")
				}
			} else if g.FindExiledCard(first.ID()) == nil || g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) == nil {
				t.Fatal("entry choices substituted a visit, owner or prohibited land")
			}
		})
	}
}

func TestLinkedExileEntryDoesNotCast(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("X counters", "{X}", 1, 1, WithAbility(EntersWithXCounters(core.Javelin)))
	card.SetOwner(a.PlayerID())
	g.ExileCardFaceDown(card, uuid.New())
	saved := savedTransportVisits{{id: card.ID(), visit: g.FindExiledCard(card.ID()).Visit, zone: core.ZoneExile}}
	g.SetXValue(7)
	if err := PutExiledCardsOntoBattlefieldUnderOwnersControl(saved).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if p := g.FindPermanent(card.ID()); p == nil || p.Counters[core.Javelin] != 0 || g.XValue() != 7 || g.StackSize() != 0 {
		t.Fatal("entry inherited casting state or erased the caller's X")
	}
}

func TestLinkedExileEntryChoiceOrder(t *testing.T) {
	a := &millOrderPlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	var order []string
	cards := make([]Card, 0, 2)
	visits := make(savedTransportVisits, 0, 2)
	for _, name := range []string{"First", "Second"} {
		probe := &optionalEntryProbe{run: func(_ *Game, p *Permanent) bool { order = append(order, p.Name()); return true }}
		card := NewCreature(name, "", 1, 1, WithAbility(probe))
		card.SetOwner(a.PlayerID())
		g.ExileCardFaceDown(card, uuid.New())
		cards = append(cards, card)
		visits = append(visits, capturedCard{id: card.ID(), visit: g.FindExiledCard(card.ID()).Visit, zone: core.ZoneExile})
	}
	if err := PutExiledCardsOntoBattlefieldUnderOwnersControl(visits).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if a.calls != 1 || len(order) != 2 || order[0] != "Second" || order[1] != "First" {
		t.Fatal("owner could not order simultaneous entry choices")
	}
	for _, card := range cards {
		if g.FindPermanent(card.ID()) == nil {
			t.Fatal("ordered entry lost a card")
		}
	}
}
