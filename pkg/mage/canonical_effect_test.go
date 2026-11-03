package mage

import (
	"testing"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCanonicalEffectsUseExplicitReferences(t *testing.T) {
	g := newPriorityTestGame()
	a, b := g.players[0], g.players[1]
	source := g.PutOnBattlefield(NewCreature("Source", "{0}", 2, 2), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "{0}", 2, 2), b.PlayerID())
	ctx := NewEffectContext(g, source.ID(), a.PlayerID(), nil)
	ctx.TriggerEvent = &GameEvent{PlayerID: b.PlayerID(), SourceID: victim.ID(), Amount: 3}
	ctx.Vars = map[string]any{"amount": 3}
	if err := GainLife(EventPlayer(), VarInt("amount")).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 23 || a.Life() != 20 {
		t.Fatal("life effect used the wrong player")
	}
	if err := DealDamage(SourceObject(), EventPlayer(), Fixed(2)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 21 {
		t.Fatal("damage effect ignored explicit recipient")
	}
	if err := Destroy(EventSource()).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(victim.ID()) != nil {
		t.Fatal("destruction ignored event reference")
	}
}

func TestCanonicalCounterAndZoneEffects(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	source := g.PutOnBattlefield(NewCreature("Source", "{0}", 2, 2), owner)
	ctx := NewEffectContext(g, source.ID(), owner, nil)
	ctx.Vars = map[string]any{"count": 2}
	if err := AddCounters(SourceObject(), P1P1, VarInt("count")).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if source.Counters[P1P1] != 2 {
		t.Fatal("counters were not added")
	}
	if err := RemoveCounters(SourceObject(), P1P1, Fixed(1)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if source.Counters[P1P1] != 1 {
		t.Fatal("counters were not removed")
	}
	if err := MoveObjects(SourceObject(), ZoneHand).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(source.ID()) != nil || len(g.players[0].Hand()) != 1 {
		t.Fatal("zone effect did not return the source")
	}
}

func TestCanonicalDescriptionsDoNotNeedExecutionVariables(t *testing.T) {
	for _, effect := range []Effect{
		GainLife(SelectController(), VarInt("n")),
		DealDamage(SourceObject(), FirstTarget(), VarInt("n")),
		DrawForPlayers(SelectController(), VarInt("n")),
		Discard(SelectController(), VarInt("n")),
		AddCounters(SourceObject(), P1P1, VarInt("n")),
		RemoveCounters(SourceObject(), P1P1, VarInt("n")),
	} {
		if effect.Text() == "" {
			t.Fatal("missing effect description")
		}
		_ = effect.Properties()
	}
}

func TestMoveObjectsExilesAnUncounterableSpell(t *testing.T) {
	g := newPriorityTestGame()
	player := g.players[0]
	spell := NewInstant("Uncounterable", "{0}", WithUncounterable())
	player.AddToHand(spell)
	if err := g.CastSpellByID(player.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	ctx := NewEffectContext(g, uuid.New(), player.PlayerID(), nil)
	ctx.TargetGroups = [][]uuid.UUID{{spell.ID()}}
	if err := MoveObjects(TargetGroup(0), ZoneExile).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if !g.stack.IsEmpty() || g.CardZone(spell.ID()) != ZoneExile {
		t.Fatal("zone movement failed to exile the spell")
	}
}

func TestMoveObjectsEmitsOneEntryEvent(t *testing.T) {
	for _, from := range []Zone{ZoneGraveyard, ZoneStack} {
		t.Run(from.String(), func(t *testing.T) {
			g := newPriorityTestGame()
			g.turns.SetStep(PrecombatMain)
			player := g.players[0]
			card := NewCreature("Entry", "{0}", 2, 2)
			count := 0
			observer := NewTriggered(EvtZoneChange, false, contextProbe{inspect: func(*EffectContext) { count++ }}).SetCondition(func(event *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
				return event.SourceID == card.ID() && event.ToZone == ZoneBattlefield
			})
			g.PutOnBattlefield(NewArtifact("Observer", "{0}", WithAbility(observer)), player.PlayerID())
			if from == ZoneStack {
				player.AddToHand(card)
				if err := g.CastSpellByID(player.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
			} else {
				player.AddToGraveyard(card)
			}
			ctx := NewEffectContext(g, uuid.New(), player.PlayerID(), nil)
			ctx.TargetGroups = [][]uuid.UUID{{card.ID()}}
			if err := MoveObjects(TargetGroup(0), ZoneBattlefield, EnterTapped()).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			g.PutTriggersOnStack()
			g.ResolveStack()
			if count != 1 {
				t.Fatalf("entry event count = %d, want 1", count)
			}
			if permanent := g.FindPermanent(card.ID()); permanent == nil || !permanent.Tapped {
				t.Fatal("permanent did not enter tapped")
			}
		})
	}
}

func TestMillMovesTheLibraryTopToItsOwnersGraveyard(t *testing.T) {
	g := newPriorityTestGame()
	player := g.players[0]
	first, second := NewLand("First"), NewLand("Second")
	player.SetLibrary([]*Card{first, second})
	ctx := NewEffectContext(g, uuid.New(), player.PlayerID(), []uuid.UUID{player.PlayerID()})
	if err := MillTargetPlayer(Fixed(1)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(player.Graveyard()) != 1 || player.Graveyard()[0].ID() != first.ID() {
		t.Fatal("mill did not move the library top to its owner's graveyard")
	}
	if len(player.Library()) != 1 || player.Library()[0].ID() != second.ID() {
		t.Fatal("mill changed the wrong library card")
	}
}
