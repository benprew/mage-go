package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestEnchantedTargetResolution(t *testing.T) {
	for _, change := range []string{"none", "opponent aura", "remove aura", "phase aura", "return host", "second aura"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := randomTestGame()
			host := addRandomTargetCreature(g, b, "Host")
			auraCard := NewAura("Aura", "{W}")
			auraCard.SetOwner(a.PlayerID())
			controller := a.PlayerID()
			if change == "opponent aura" {
				controller = b.PlayerID()
			}
			aura := g.PutOnBattlefield(auraCard, controller)
			aura.AttachedTo = host.ID()
			host.Attachments = append(host.Attachments, aura.ID())
			target := DeclarePermanentTarget("creature", And(IsCreature, IsEnchanted))
			ability := Activate(Costs(TapCost(This)), Destroy(target), WithTargets(target))
			source := g.PutOnBattlefield(NewCreature("Assassin", "", 4, 3, WithAbility(ability)), a.PlayerID())
			g.pushStack(&StackObject{ID: uuid.New(), SourceID: source.ID(), Controller: a.PlayerID(), Targets: []uuid.UUID{host.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
			clone := g.Clone()
			switch change {
			case "remove aura":
				clone.RemoveFromBattlefield(clone.FindPermanent(aura.ID()))
			case "phase aura":
				clone.PhaseOut(aura.ID())
			case "return host":
				p := clone.FindPermanent(host.ID())
				clone.RemoveFromBattlefield(p)
				clone.PutOnBattlefield(p.Card, b.PlayerID())
			case "second aura":
				card := NewAura("Second Aura", "{W}")
				card.SetOwner(b.PlayerID())
				p := clone.PutOnBattlefield(card, b.PlayerID())
				p.AttachedTo = host.ID()
				h := clone.MutablePermanent(host.ID())
				h.Attachments = append(h.Attachments, p.ID())
				clone.PhaseOut(aura.ID())
			}
			clone.ResolveStack()
			wantDestroyed := change == "none" || change == "opponent aura" || change == "second aura"
			if gotDestroyed := clone.FindPermanent(host.ID()) == nil; gotDestroyed != wantDestroyed {
				t.Fatalf("destroyed = %v, want %v", gotDestroyed, wantDestroyed)
			}
			if g.FindPermanent(host.ID()) == nil || !IsEnchanted.Match(g.FindPermanent(host.ID()), g) {
				t.Fatal("clone changed original target or attachment")
			}
		})
	}
}
