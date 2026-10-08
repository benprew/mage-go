package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedManaOperands(t *testing.T) {
	for name, player := range map[string]PlayerRef{"controller": ControllerOf(This), "controller LKI": ControllerOfWithLKI(This), "owner": OwnerOf(This)} {
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewArtifact("Source", "")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			a.ManaPool().Add(core.Blue, 2)
			b.ManaPool().Add(core.Red, 3)
			if err := ApplyEffect(g, AtNext(EachEndStep, LoseUnspentManaFor(player)), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			check := func() {
				t.Helper()
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(saved) || a.ManaPool().TotalMana() != 2 || b.ManaPool().TotalMana() != 3 {
					t.Fatal("lost reference or cleared mana", got)
				}
			}
			check()
			phased := g.PhaseOut(p.ID())
			check()
			g.PhaseIn(phased)
			check()
			g.BouncePermanentToHand(p)
			g.zones.ClearLKI()
			check()
			a.RemoveFromHand(c.ID())
			g.PutOnBattlefield(c, b.PlayerID())
			check()
			clone := g.Clone()
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("shared wait lifetime")
			}
		})
	}
}

func TestWaitingDelayedManaBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	for _, body := range []ResolvingEffect{LoseUnspentManaFor(You), LoseUnspentManaFor(ControllerOf(AttachedPermanent)), AtNext(EachEndStep, LoseUnspentManaFor(OwnerOf(This)))} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("inferred implicit or future reference")
	}
	for _, player := range []PlayerRef{nil, FromPlayer(BindPlayer("missing")), DeclarePlayerTarget("missing")} {
		c := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, LoseUnspentManaFor(player)))))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("accepted undeclared player")
		}
	}
}
