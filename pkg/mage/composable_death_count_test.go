package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreatureDeathCountActions(t *testing.T) {
	for _, action := range []string{"destroy", "sacrifice", "graveyard", "token", "artifact", "exile", "return", "regenerate", "indestructible"} {
		t.Run(action, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 2, 2)
			if action == "indestructible" {
				card = NewCreature("Subject", "", 2, 2, WithKeyword(core.Indestructible))
			}
			if action == "artifact" {
				card = NewArtifact("Subject", "")
			}
			card.SetOwner(b.PlayerID())
			g.PutOnBattlefield(card, b.PlayerID())
			p := g.MutablePermanent(card.ID())
			want := 1
			switch action {
			case "destroy", "artifact":
				g.DestroyPermanent(p)
				if action == "artifact" {
					want = 0
				}
			case "sacrifice":
				g.DoSacrifice(p)
			case "graveyard":
				g.PutPermanentIntoGraveyard(p)
			case "token":
				p.IsToken = true
				g.DestroyPermanent(p)
			case "exile":
				g.ExilePermanent(p)
				want = 0
			case "return":
				if err := ApplyEffect(g, ReturnToHand(This), card.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want = 0
			case "regenerate":
				g.AddRegenerationShield(card.ID())
				g.DestroyPermanent(p)
				want = 0
			case "indestructible":
				g.DestroyPermanent(p)
				want = 0
			}
			for _, player := range []Player{a, b} {
				ctx := &EffectContext{Game: g, Controller: player.PlayerID()}
				for range 3 {
					if got := CreaturesDiedThisTurn().resolveValue(ctx); got != want {
						t.Fatalf("count = %d, want %d", got, want)
					}
				}
			}
		})
	}
}

func TestCreatureDeathCountResolutionCopyAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 2, 2, WithAbility(At(EachEndStep, GainLifeFor(You, CreaturesDiedThisTurn()))))
	g.PutOnBattlefield(source, a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || a.Life() != 20 {
		t.Fatal("end-step ability must use the stack")
	}
	original := g.StackObjects()[0]
	g.CopyStackObjectDirect(original, b.PlayerID(), false)
	g.ExilePermanent(g.FindPermanent(source.ID()))
	victims := make([]*Permanent, 0, 2)
	for _, player := range []Player{a, b} {
		c := NewCreature("Victim", "", 1, 1)
		c.SetOwner(player.PlayerID())
		g.PutOnBattlefield(c, player.PlayerID())
		victims = append(victims, g.FindPermanent(c.ID()))
	}
	g.DestroyPermanents(victims)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 22 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("copies must read resolution-time deaths in their own game")
	}
	clone.trackers.Turn.ResetForNewTurn(b.PlayerID())
	if CreaturesDiedThisTurn().resolveValue(&EffectContext{Game: clone}) != 0 || CreaturesDiedThisTurn().resolveValue(&EffectContext{Game: g}) != 2 {
		t.Fatal("turn reset must clear only the cloned count")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatal("source departure changed the turn count")
	}
}

func TestCreatureDeathCountSnapshot(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Victim", "", 1, 1)
	c.SetOwner(a.PlayerID())
	g.PutOnBattlefield(c, a.PlayerID())
	n := BindNumber("before")
	body := Sequence(SnapshotNumber(CreaturesDiedThisTurn(), n), Sacrifice(This),
		GainLifeFor(You, FromNumber(n)), GainLifeFor(You, CreaturesDiedThisTurn()))
	if err := ApplyEffect(g, body, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 21 {
		t.Fatalf("life = %d, want 21", a.Life())
	}
}
