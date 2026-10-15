package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func castOrdinalSpell(t *testing.T, g *Game, player Player, instant bool) *StackObject {
	t.Helper()
	var spell Card = NewSorcery("Other spell", "")
	if instant {
		spell = NewInstant("Instant", "")
	}
	spell.SetOwner(player.PlayerID())
	player.AddToHand(spell)
	if err := g.CastSpellByID(player.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	return g.StackPeek()
}

func instantOrdinalWatcher() Card {
	caster := BindPlayer("caster")
	return NewArtifact("Watcher", "", WithAbility(WhenEvent(SpellCast().AfterFirstInstant().ByOpponent().WithCaster(caster), GainLifeFor(FromPlayer(caster), Fixed(1)))))
}

func TestSpellCastOrdinalHistory(t *testing.T) {
	g, a, b := newSpellCopyGame()
	g.SetStep(core.PrecombatMain)
	g.SetActivePlayerIndex(1)
	castOrdinalSpell(t, g, b, false)
	g.ResolveStack()
	first := castOrdinalSpell(t, g, b, true)
	if g.CopyStackObjectDirect(first, b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if g.GetInstantsCastThisTurn(b.PlayerID()) != 1 {
		t.Fatal("copy counted as cast")
	}
	source := g.PutOnBattlefield(instantOrdinalWatcher(), a.PlayerID())
	castOrdinalSpell(t, g, b, true)
	g.ResolveStack()
	if b.Life() != 21 {
		t.Fatal("earlier instant before entry was not counted")
	}
	castOrdinalSpell(t, g, b, true)
	g.ResolveStack()
	if b.Life() != 22 {
		t.Fatal("third instant did not trigger")
	}
	castOrdinalSpell(t, g, b, false)
	g.ResolveStack()
	if b.Life() != 22 {
		t.Fatal("noninstant used the earlier instant count")
	}
	g.SetActivePlayerIndex(0)
	for range 2 {
		castOrdinalSpell(t, g, a, true)
		g.ResolveStack()
	}
	if a.Life() != 20 || b.Life() != 22 {
		t.Fatal("controller cast triggered")
	}
	g.PhaseOut(source.ID())
	castOrdinalSpell(t, g, b, true)
	g.ResolveStack()
	if b.Life() != 22 || g.GetInstantsCastThisTurn(b.PlayerID()) != 4 {
		t.Fatal("phasing changed history or triggered")
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	castOrdinalSpell(t, g, b, true)
	g.ResolveStack()
	if b.Life() != 23 {
		t.Fatal("phase cycle lost history")
	}
	g.trackers.Turn.ResetForNewTurn(a.PlayerID())
	castOrdinalSpell(t, g, b, true)
	g.ResolveStack()
	if b.Life() != 23 {
		t.Fatal("first instant of new turn triggered")
	}
}

func TestSpellCastOrdinalFailedCast(t *testing.T) {
	g, a, b := newSpellCopyGame()
	g.SetStep(core.PrecombatMain)
	g.PutOnBattlefield(instantOrdinalWatcher(), a.PlayerID())
	card := NewInstant("Unpaid", "{U}")
	card.SetOwner(b.PlayerID())
	b.AddToHand(card)
	if err := g.CastSpellByID(b.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast succeeded")
	}
	if g.GetInstantsCastThisTurn(b.PlayerID()) != 0 || g.StackSize() != 0 || len(b.Hand()) != 1 {
		t.Fatal("failed cast changed history, stack or hand")
	}
	castOrdinalSpell(t, g, b, true)
	g.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("failed cast counted as first instant")
	}
}

func TestSpellCastOrdinalPending(t *testing.T) {
	for _, change := range []string{"leave", "return", "phase", "control", "counter spell", "copy trigger", "two sources"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			g.SetStep(core.PrecombatMain)
			card := instantOrdinalWatcher()
			source := g.PutOnBattlefield(card, a.PlayerID())
			castOrdinalSpell(t, g, b, true)
			g.ResolveStack()
			if change == "two sources" {
				g.PutOnBattlefield(instantOrdinalWatcher(), a.PlayerID())
			}
			spell := castOrdinalSpell(t, g, b, true)
			g.PutTriggersOnStack()
			want := 21
			switch change {
			case "leave", "return":
				g.RemoveFromBattlefield(source)
				if change == "return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "counter spell":
				g.GetStack().RemoveByID(spell.ID)
			case "copy trigger":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("trigger copy failed")
				}
				want++
			case "two sources":
				want++
			}
			g.trackers.Turn.ResetForNewTurn(a.PlayerID())
			clone := g.Clone()
			g.ResolveStack()
			if a.Life() != 20 || b.Life() != want {
				t.Fatalf("saved caster lost: %d/%d want %d", a.Life(), b.Life(), want)
			}
			if clone.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("clone changed")
			}
			clone.ResolveStack()
			if clone.GetPlayer(b.PlayerID()).Life() != want {
				t.Fatal("clone lost pending trigger")
			}
		})
	}
}

func TestSpellCastOrdinalSnapshot(t *testing.T) {
	g, a, b := newSpellCopyGame()
	pattern := SpellCast().AfterFirstInstant()
	trigger := pattern.makeTrigger(GainLifeFor(You, Fixed(1)))
	for _, ordinal := range []int{0, 1, 2, 3} {
		event := core.GameEvent{Type: core.EvtSpellCast, PlayerID: b.PlayerID(), CastInstantOrdinal: ordinal}
		g.trackers.Turn.RecordInstantCast(b.PlayerID())
		if got := trigger.Condition(&event, g, uuid.Nil, a.PlayerID()); got != (ordinal > 1) {
			t.Fatalf("ordinal %d matched %v", ordinal, got)
		}
	}
}

func TestSpellCastOrdinalInvalid(t *testing.T) {
	for _, pattern := range []EventPattern{
		SpellCast().AfterFirstInstant().AfterFirstInstant(),
		SpellCast().AfterFirstInstant().WithCaster(nil),
	} {
		card := NewArtifact("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid event accepted")
		}
	}
}

func TestSpellCastOrdinalCloneHistoryAndControl(t *testing.T) {
	g, a, b := newSpellCopyGame()
	g.SetStep(core.PrecombatMain)
	source := g.PutOnBattlefield(instantOrdinalWatcher(), a.PlayerID())
	castOrdinalSpell(t, g, b, true)
	g.ResolveStack()
	clone := g.Clone()
	for range 2 {
		castOrdinalSpell(t, g, b, true)
		g.ResolveStack()
	}
	castOrdinalSpell(t, clone, clone.GetPlayer(b.PlayerID()), true)
	clone.ResolveStack()
	if b.Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 21 || g.GetInstantsCastThisTurn(b.PlayerID()) != 3 || clone.GetInstantsCastThisTurn(b.PlayerID()) != 2 {
		t.Fatal("clones share turn history")
	}
	for range 2 {
		castOrdinalSpell(t, g, a, true)
		g.ResolveStack()
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	castOrdinalSpell(t, g, a, true)
	g.ResolveStack()
	castOrdinalSpell(t, g, b, true)
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 22 {
		t.Fatal("controller change did not use each player's cast history")
	}
	g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.Indefinite))
	g.ApplyContinuousEffects()
	castOrdinalSpell(t, g, a, true)
	g.ResolveStack()
	if a.Life() != 21 || g.GetInstantsCastThisTurn(a.PlayerID()) != 4 {
		t.Fatal("ability removal changed history or triggered")
	}
}
