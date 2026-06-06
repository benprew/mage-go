package heuristic

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"

	_ "github.com/benprew/mage-go/cards" // register all card sets
)

// TestHeuristicPirateShipPingsFace verifies the AI activates Pirate Ship's
// "{T}: deals 1 damage to any target" ability to ping the opponent's face
// when there are no opposing creatures and the opponent is low enough that
// the ping advances the race.
func TestHeuristicPirateShipPingsFace(t *testing.T) {
	pa := mage.NewBasePlayer("AI")
	pb := mage.NewBasePlayer("Opp")
	g := mage.NewGame(pa, pb)

	pa.SetLife(20)
	pb.SetLife(2)

	put := func(name string, owner *mage.BasePlayer) *mage.Permanent {
		t.Helper()
		card, err := mage.CreateCard(name)
		if err != nil {
			t.Fatalf("CreateCard %q: %v", name, err)
		}
		card.SetOwner(owner.PlayerID())
		perm := g.PutOnBattlefield(card, owner.PlayerID())
		perm.RevokeBaseAttr(core.AttrSummonSick)
		return perm
	}

	ship := put("Pirate Ship", pa)
	// Pirate Ship sacrifices itself when its controller has no Island.
	put("Island", pa)

	personalities := []struct {
		name string
		w    ai.WeightedPersonality
	}{
		{"Midrange", ai.MidrangeWeighted},
		{"Aggro", ai.AggroWeighted},
		{"Control", ai.ControlWeighted},
		{"Tempo", ai.TempoWeighted},
		{"Burn", ai.BurnWeighted},
	}

	for _, p := range personalities {
		t.Run(p.name, func(t *testing.T) {
			s := New(p.w)
			act := s.PriorityAction(pa, g, 0, true)

			if act.Type != interactive.ActionActivateAbility {
				t.Fatalf("expected ActionActivateAbility, got %v", act.Type)
			}
			if act.PermanentID != ship.ID() {
				t.Fatalf("expected activation of Pirate Ship %v, got %v", ship.ID(), act.PermanentID)
			}
			if len(act.Targets) != 1 {
				t.Fatalf("expected exactly one target, got %v", act.Targets)
			}
			if act.Targets[0] != pb.PlayerID() {
				t.Fatalf("expected ping aimed at opponent face %v, got %v", pb.PlayerID(), act.Targets[0])
			}
		})
	}
}

// TestHeuristicPirateShipPingsFaceOverUnkillableCreature verifies that a free
// repeatable ping (no mana cost) still chips the opponent's face when the only
// opposing creature is too large for the ping to meaningfully damage. A free
// recurring ping always advances the game, so the AI should never decline it.
func TestHeuristicPirateShipPingsFaceOverUnkillableCreature(t *testing.T) {
	pa := mage.NewBasePlayer("AI")
	pb := mage.NewBasePlayer("Opp")
	g := mage.NewGame(pa, pb)

	pa.SetLife(20)
	pb.SetLife(20)

	put := func(name string, owner *mage.BasePlayer) *mage.Permanent {
		t.Helper()
		card, err := mage.CreateCard(name)
		if err != nil {
			t.Fatalf("CreateCard %q: %v", name, err)
		}
		card.SetOwner(owner.PlayerID())
		perm := g.PutOnBattlefield(card, owner.PlayerID())
		perm.RevokeBaseAttr(core.AttrSummonSick)
		return perm
	}

	ship := put("Pirate Ship", pa)
	put("Island", pa)

	// Craw Wurm is a 6/4 the 1-damage ping cannot profitably damage.
	put("Craw Wurm", pb)

	s := New(ai.MidrangeWeighted)
	act := s.PriorityAction(pa, g, 0, true)

	if act.Type != interactive.ActionActivateAbility {
		t.Fatalf("expected ActionActivateAbility (chip face), got %v", act.Type)
	}
	if act.PermanentID != ship.ID() {
		t.Fatalf("expected activation of Pirate Ship, got %v", act.PermanentID)
	}
	if len(act.Targets) != 1 || act.Targets[0] != pb.PlayerID() {
		t.Fatalf("expected ping aimed at opponent face, got %v", act.Targets)
	}
}

// TestHeuristicPirateShipPingsCreature verifies the AI activates Pirate Ship
// to kill an opposing X/1 creature when face damage is not the better line.
func TestHeuristicPirateShipPingsCreature(t *testing.T) {
	pa := mage.NewBasePlayer("AI")
	pb := mage.NewBasePlayer("Opp")
	g := mage.NewGame(pa, pb)

	pa.SetLife(20)
	pb.SetLife(20)

	put := func(name string, owner *mage.BasePlayer) *mage.Permanent {
		t.Helper()
		card, err := mage.CreateCard(name)
		if err != nil {
			t.Fatalf("CreateCard %q: %v", name, err)
		}
		card.SetOwner(owner.PlayerID())
		perm := g.PutOnBattlefield(card, owner.PlayerID())
		perm.RevokeBaseAttr(core.AttrSummonSick)
		return perm
	}

	ship := put("Pirate Ship", pa)
	put("Island", pa)

	// Mons's Goblin Raiders is a 1/1 the ping can kill.
	target := put("Mons's Goblin Raiders", pb)

	s := New(ai.MidrangeWeighted)
	act := s.PriorityAction(pa, g, 0, true)

	if act.Type != interactive.ActionActivateAbility {
		t.Fatalf("expected ActionActivateAbility, got %v", act.Type)
	}
	if act.PermanentID != ship.ID() {
		t.Fatalf("expected activation of Pirate Ship, got %v", act.PermanentID)
	}
	if len(act.Targets) != 1 || act.Targets[0] != target.ID() {
		t.Fatalf("expected ping aimed at opposing 1/1, got %v", act.Targets)
	}
}
