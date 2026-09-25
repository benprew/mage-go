package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestAttachmentTargetResolution(t *testing.T) {
	for _, change := range []string{"none", "opponent aura", "host control", "host type", "move to own", "move to opponent", "detach", "phase", "return"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			host := g.PutOnBattlefield(NewCreature("Host", "", 2, 2), a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 2, 2), a.PlayerID())
			opponent := g.PutOnBattlefield(NewCreature("Opponent", "", 2, 2), b.PlayerID())
			aura := NewAura("Aura", "")
			aura.SetOwner(a.PlayerID())
			controller := a.PlayerID()
			if change == "opponent aura" {
				controller = b.PlayerID()
			}
			p := g.PutOnBattlefield(aura, controller)
			p.AttachedTo = host.ID()
			target := DeclarePermanentTarget("aura", And(IsEnchantment, HasSubType("Aura"), IsAttachedTo(IsCreature)), TargetAttachedToYouControl())
			ability := Spell(ReturnToHand(target), WithTargets(target))
			spell := NewInstant("Return", "", WithAbility(ability))
			g.pushStack(&StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{p.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
			clone := g.Clone()
			cp := clone.MutablePermanent(p.ID())
			switch change {
			case "host control":
				clone.AddControlEffect(ControlEffectSpec{SourceID: spell.ID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "host type":
				cp.AttachedTo = clone.PutOnBattlefield(NewArtifact("Noncreature", ""), a.PlayerID()).ID()
			case "move to own":
				cp.AttachedTo = other.ID()
			case "move to opponent":
				cp.AttachedTo = opponent.ID()
			case "detach":
				cp.AttachedTo = uuid.Nil
			case "phase":
				clone.PhaseOut(p.ID())
			case "return":
				clone.RemoveFromBattlefield(cp)
				clone.PutOnBattlefield(cp.Card, controller).AttachedTo = host.ID()
			}
			clone.ResolveStack()
			want := 0
			if change == "none" || change == "opponent aura" || change == "move to own" {
				want = 1
			}
			if got := len(clone.GetPlayer(a.PlayerID()).Hand()); got != want {
				t.Fatalf("hand = %d, want %d", got, want)
			}
			if g.FindPermanent(p.ID()).AttachedTo != host.ID() || len(a.Hand()) != 0 {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestAttachmentTargetCopyController(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := make([]uuid.UUID, 0, 2)
	for _, player := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
		host := g.PutOnBattlefield(NewCreature("Host", "", 2, 2), player)
		aura := NewAura("Aura", "")
		aura.SetOwner(player)
		g.PutOnBattlefield(aura, player).AttachedTo = host.ID()
		targets = append(targets, aura.ID())
	}
	target := DeclarePermanentTarget("aura", And(IsEnchantment, HasSubType("Aura"), IsAttachedTo(IsCreature)), TargetAttachedToYouControl())
	ability := Spell(ReturnToHand(target), WithTargets(target))
	spell := NewInstant("Return", "", WithAbility(ability))
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: targets[:1], TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	b.chooseQueue = [][]uuid.UUID{{targets[1]}}
	cp := g.CopyStackObjectDirect(obj, b.PlayerID(), true)
	if cp == nil || cp.Targets[0] != targets[1] || obj.Targets[0] != targets[0] {
		t.Fatal("copy did not use its controller for attachment targeting")
	}
	g.ResolveStack()
	if len(a.Hand()) != 1 || len(b.Hand()) != 1 {
		t.Fatal("copies did not return their legal targets")
	}
}
