package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestAttachedCreatureDiesCapturesEventState(t *testing.T) {
	for _, simultaneous := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		controller := BindPlayer("former controller")
		toughness := BindNumber("former toughness")
		aura := NewAura("Watcher", "", WithAbility(WhenEvent(
			AttachedCreatureDies().WithController(controller).WithToughness(toughness),
			GainLifeFor(FromPlayer(controller), FromNumber(toughness)))))
		aura.SetOwner(a.PlayerID())
		host := NewCreature("Host", "", 2, 3)
		host.SetOwner(a.PlayerID())
		other := NewCreature("Other", "", 1, 4)
		other.SetOwner(a.PlayerID())
		g.PutOnBattlefield(host, b.PlayerID())
		g.PutOnBattlefield(other, a.PlayerID())
		g.PutOnBattlefield(aura, a.PlayerID())
		g.Attach(aura.ID(), host.ID())
		if err := ValidateCardDefinition(aura.Name(), aura); err != nil {
			t.Fatal(err)
		}
		if err := ApplyEffect(g, Until(EndOfTurn, ModifyPT(AttachedPermanent, Fixed(0), Fixed(2))), aura.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if simultaneous {
			g.DestroyPermanents([]*Permanent{g.FindPermanent(host.ID()), g.FindPermanent(other.ID()), g.FindPermanent(aura.ID())})
		} else {
			g.DestroyPermanent(g.FindPermanent(other.ID()))
			g.DestroyPermanent(g.FindPermanent(host.ID()))
		}
		g.PutTriggersOnStack()
		if g.StackSize() != 1 {
			t.Fatalf("simultaneous=%v stack=%d, want 1", simultaneous, g.StackSize())
		}
		clone := g.Clone()
		clone.ResolveStack()
		if b.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 25 {
			t.Fatal("clone did not keep event state separate")
		}
		if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
			t.Fatal("copy failed")
		}
		g.zones.ClearLKI()
		g.PutOnBattlefield(host, a.PlayerID())
		g.ResolveStack()
		if a.Life() != 20 || b.Life() != 30 {
			t.Fatalf("life=(%d,%d), want (20,30)", a.Life(), b.Life())
		}
	}
}

func TestAttachedCreatureDiesRejectsOtherEvents(t *testing.T) {
	for _, change := range []string{"unattached", "other", "phased", "exile", "land", "returned host"} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		aura := NewAura("Watcher", "", WithAbility(WhenEvent(AttachedCreatureDies(), GainLifeFor(You, Fixed(1)))))
		host := NewCreature("Host", "", 2, 2)
		host.SetOwner(b.PlayerID())
		g.PutOnBattlefield(host, b.PlayerID())
		g.PutOnBattlefield(aura, a.PlayerID())
		if change != "unattached" {
			g.Attach(aura.ID(), host.ID())
		}
		switch change {
		case "other":
			other := NewCreature("Other", "", 1, 1)
			other.SetOwner(b.PlayerID())
			g.PutOnBattlefield(other, b.PlayerID())
			g.DestroyPermanent(g.FindPermanent(other.ID()))
		case "phased":
			g.PhaseOut(aura.ID())
			g.DestroyPermanent(g.FindPermanent(host.ID()))
		case "exile":
			g.ExilePermanent(g.FindPermanent(host.ID()))
		case "land":
			land := NewLand("Land")
			land.SetOwner(b.PlayerID())
			g.PutOnBattlefield(land, b.PlayerID())
			g.Attach(aura.ID(), land.ID())
			g.DestroyPermanent(g.FindPermanent(land.ID()))
		case "returned host":
			g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
			g.PutOnBattlefield(host, b.PlayerID())
			g.DestroyPermanent(g.FindPermanent(host.ID()))
		default:
			g.DestroyPermanent(g.FindPermanent(host.ID()))
		}
		g.PutTriggersOnStack()
		if g.StackSize() != 0 {
			t.Fatalf("%s: stack=%d, want 0", change, g.StackSize())
		}
	}
}

func TestAttachedCreatureDiesValidation(t *testing.T) {
	b := BindNumber("toughness")
	good := NewAura("Good", "", WithAbility(WhenEvent(AttachedCreatureDies().WithToughness(b), GainLifeFor(You, FromNumber(b)))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	bad := NewAura("Bad", "", WithAbility(WhenEvent(AttachedCreatureDies().WithController(nil), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(err.Error(), "binding") {
		t.Fatalf("validation=%v, want binding error", err)
	}
}
