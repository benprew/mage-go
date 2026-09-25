package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSingularNoRegenerationPreservesCharacteristics(t *testing.T) {
	for _, kind := range []string{"typed", "positional", "step", "gathered"} {
		t.Run(kind, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Victim", "", 2, 2)
			c.SetOwner(a.PlayerID())
			g.PutOnBattlefield(c, a.PlayerID())
			g.AddRegenerationShield(c.ID())
			ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID(), Targets: []uuid.UUID{c.ID()}}
			var effect Effect
			switch kind {
			case "typed":
				effect = DestroyWithoutRegeneration(This)
			case "positional":
				effect = DestroyTargetNoRegen()
			case "step":
				effect = DestroyTargetNoRegenStep()
			case "gathered":
				ctx.Vars = make(map[string]any)
				ctx.SetUUID("victim", c.ID())
				effect = DestroyGatheredNoRegen("victim")
			}
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(c.ID()) != nil {
				t.Fatal("regeneration prevented destruction")
			}
			lki := g.LookupObject(c.ID())
			if lki == nil || lki.ViewHasKeyword(core.CantRegenerate) {
				t.Fatal("destruction changed the victim's last-known characteristics")
			}
		})
	}
}

func TestSingularNoRegenerationSavedLifetime(t *testing.T) {
	for _, change := range []string{"same", "return", "absent", "phase", "phase cycle", "control", "indestructible"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Subject", "", 2, 2)
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.AddRegenerationShield(c.ID())
			saved := BindPermanent("saved")
			ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(c, a.PlayerID())
			case "absent":
				g.RemoveFromBattlefield(p)
			case "phase":
				g.PhaseOut(c.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(c.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: c.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "indestructible":
				p.GrantBaseAttr(core.Indestructible)
			}
			clone := g.Clone()
			ctx.Game = clone
			if err := DestroyWithoutRegeneration(From(saved)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			survives := change == "return" || change == "phase" || change == "indestructible"
			got := clone.FindPermanentIncludingPhased(c.ID())
			if (got != nil) != survives {
				t.Fatal("incorrect saved subject lifetime or destruction")
			}
			if got != nil && got.HasKeyword(core.CantRegenerate) {
				t.Fatal("destruction changed survivor characteristics")
			}
			if change != "absent" && g.FindPermanentIncludingPhased(c.ID()) == nil {
				t.Fatal("clone destruction changed original")
			}
			if change == "indestructible" {
				if !clone.HasRegenerationShield(c.ID()) {
					t.Fatal("indestructible destruction consumed shield")
				}
			}
		})
	}
}
