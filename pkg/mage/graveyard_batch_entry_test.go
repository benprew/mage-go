package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestGraveyardBatchEntry(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewCreature("First", "", 2, 2, WithAbility(EnterTappedAsEnters()))
	second := NewCreature("Second", "", 3, 3, WithAbility(EntersWithNCounters(core.P1P1, 2)))
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	a.AddToGraveyard(first)
	b.AddToGraveyard(second)
	spell := NewSorcery("Stay", "")
	spell.SetOwner(a.PlayerID())
	a.AddToGraveyard(spell)
	observed := 0
	observer := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		if e.FromZone == core.ZoneGraveyard && e.ToZone == core.ZoneBattlefield {
			observed++
			if reader.FindPermanent(first.ID()) == nil || reader.FindPermanent(second.ID()) == nil || len(a.Graveyard()) != 1 || len(b.Graveyard()) != 0 {
				t.Error("entry observation preceded the complete batch")
			}
			if e.ActorID != b.PlayerID() {
				t.Error("wrong actor")
			}
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(observer)), a.PlayerID())
	clone := g.Clone()
	body := PutGraveyardCreaturesOntoBattlefieldUnderOwnersControl()
	if err := ApplyEffect(g, body, uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	p, q := g.FindPermanent(first.ID()), g.FindPermanent(second.ID())
	if p == nil || q == nil || !p.Tapped || q.Counters[core.P1P1] != 2 || p.ControllerID() != a.PlayerID() || q.ControllerID() != b.PlayerID() || observed != 2 {
		t.Fatal("fresh owner entry or replacement rules failed")
	}
	if clone.FindPermanent(first.ID()) != nil || len(clone.GetPlayer(a.PlayerID()).Graveyard()) != 2 {
		t.Fatal("clone shared zone mutations")
	}
	if err := ApplyEffect(g, body, uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if observed != 2 {
		t.Fatal("repeat entered absent visits")
	}
	card := NewSorcery("Definition", "", WithAbility(Spell(body)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestGraveyardBatchEntryChoicesAndVisits(t *testing.T) {
	for _, scenario := range []string{"normal", "returned", "new-card", "blocked-copy", "failed"} {
		t.Run(scenario, func(t *testing.T) {
			a := NewBasePlayer("A")
			b := &graveyardEntryPlayer{NewBasePlayer("B")}
			g := NewGame(a, b)
			first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
			first.SetOwner(a.PlayerID())
			second.SetOwner(b.PlayerID())
			order := []string{}
			first.AddAbility(&opponentEntryProbe{check: func(_ *Game, _ *Permanent) {
				order = append(order, "A")
				for _, live := range g.AllBattlefield() {
					if live.ID() == first.ID() || live.ID() == second.ID() {
						t.Error("entry choice ran after movement")
					}
				}
				if scenario == "returned" {
					c, _ := b.RemoveFromGraveyard(second.ID())
					b.AddToGraveyard(c)
				}
				if scenario == "new-card" {
					c := NewCreature("Late", "", 1, 1)
					c.SetOwner(b.PlayerID())
					b.AddToGraveyard(c)
				}
			}})
			second.AddAbility(&opponentEntryProbe{check: func(_ *Game, _ *Permanent) { order = append(order, "B") }})
			if scenario == "failed" {
				second.AddAbility(&graveyardRejectEntry{})
			}
			if scenario == "blocked-copy" {
				g.PutOnBattlefield(NewLand("Land donor"), a.PlayerID())
				second.AddAbility(MayCopyAsEnters(IsLand))
				g.effects.Rules.SetLandsCantEnter(true)
			}
			a.AddToGraveyard(first)
			b.AddToGraveyard(second)
			visit := g.graveyardVisit(second.ID())
			g.resolution.SetX(9)
			if err := ApplyEffect(g, PutGraveyardCreaturesOntoBattlefieldUnderOwnersControl(), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(first.ID()) == nil || g.resolution.X() != 9 || len(order) == 0 || order[0] != "A" {
				t.Fatal("APNAP order or frame changed")
			}
			if scenario == "returned" || scenario == "blocked-copy" || scenario == "failed" {
				if g.FindPermanent(second.ID()) != nil || len(b.Graveyard()) != 1 {
					t.Fatal("invalid visit or failed entry moved")
				}
				if scenario != "returned" && g.graveyardVisit(second.ID()) != visit {
					t.Fatal("blocked entry changed graveyard visit")
				}
			} else if g.FindPermanent(second.ID()) == nil || len(order) != 2 || order[1] != "B" {
				t.Fatal("nonactive owner entry lost")
			}
			if scenario == "new-card" && len(b.Graveyard()) != 1 {
				t.Fatal("batch selected a late addition")
			}
		})
	}
}

type graveyardRejectEntry struct{ BaseAbility }

func (*graveyardRejectEntry) OnEnter(*Game, *Permanent) bool { return false }

type graveyardEntryPlayer struct{ *BasePlayer }

func (*graveyardEntryPlayer) ChooseMayAbility(string) bool { return true }

func TestGraveyardBatchEntryAPNAPAndFinalRecheck(t *testing.T) {
	for _, active := range []int{0, 1} {
		for _, returned := range []bool{false, true} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetActivePlayerIndex(active)
			players := []Player{a, b}
			firstOwner, lastOwner := players[active], players[1-active]
			first := NewCreature("First choice", "", 2, 2, WithAbility(EntersWithCounters(core.P1P1, AnnouncedXForEnteringPermanent())))
			last := NewCreature("Last choice", "", 2, 2)
			first.SetOwner(firstOwner.PlayerID())
			last.SetOwner(lastOwner.PlayerID())
			order := []uuid.UUID{}
			first.AddAbility(&opponentEntryProbe{check: func(game *Game, _ *Permanent) {
				order = append(order, firstOwner.PlayerID())
				if game.XValue() != 0 {
					t.Error("entry inherited enclosing X")
				}
			}})
			last.AddAbility(&opponentEntryProbe{check: func(_ *Game, _ *Permanent) {
				order = append(order, lastOwner.PlayerID())
				if returned {
					card, _ := firstOwner.RemoveFromGraveyard(first.ID())
					firstOwner.AddToGraveyard(card)
				}
			}})
			firstOwner.AddToGraveyard(first)
			lastOwner.AddToGraveyard(last)
			g.resolution.SetX(7)
			if err := ApplyEffect(g, PutGraveyardCreaturesOntoBattlefieldUnderOwnersControl(), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if len(order) != 2 || order[0] != firstOwner.PlayerID() || order[1] != lastOwner.PlayerID() || g.resolution.X() != 7 {
				t.Fatal("APNAP choice order or enclosing X failed")
			}
			if returned {
				if g.FindPermanent(first.ID()) != nil || len(firstOwner.Graveyard()) != 1 {
					t.Fatal("final recheck followed a returned visit")
				}
			} else if p := g.FindPermanent(first.ID()); p == nil || p.Counters[core.P1P1] != 0 {
				t.Fatal("uncast entry inherited X")
			}
		}
	}
}

func TestGraveyardBatchEntryEntrantObservations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	entries, left := 0, 0
	observer := NewCreature("Entering observer", "", 2, 2, WithAbility(NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		if e.FromZone == core.ZoneGraveyard && e.ToZone == core.ZoneBattlefield {
			entries++
			if len(reader.AllBattlefield()) != 3 {
				t.Error("entrant did not observe all other entrants")
			}
		}
		return false
	})), WithAbility(NewTriggered(core.EvtCardsLeftGraveyard, false).SetCondition(func(e *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		left++
		for _, permanent := range reader.AllBattlefield() {
			if permanent.CurrentPower(reader) < 3 {
				t.Error("graveyard departure observation missed entrant layers")
			}
		}
		if e.Amount != 3 || len(reader.GetPlayer(a.PlayerID()).Graveyard()) != 0 {
			t.Error("graveyard departure notification was not a batch")
		}
		return false
	})))
	observer.AddAbility(StaticAbility(BoostAllCreaturesIncludingSelf(2, 2, AnyPermanent)))
	for _, c := range []Card{observer, NewCreature("One", "", 1, 1), NewCreature("Two", "", 1, 1)} {
		c.SetOwner(a.PlayerID())
		a.AddToGraveyard(c)
	}
	if err := ApplyEffect(g, PutGraveyardCreaturesOntoBattlefieldUnderOwnersControl(), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if entries != 3 || left != 1 {
		t.Fatalf("entries/left = %d/%d", entries, left)
	}
}
