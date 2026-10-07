package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func controllerAura(opponent, wrapped bool) Card {
	option := TargetYouControl()
	if opponent {
		option = TargetOpponentControls()
	}
	target := DeclarePermanentTarget("enchant", IsCreature, option)
	var castTarget Target = target
	if !wrapped {
		castTarget = TargetPermanentYouControl(IsCreature)
		if opponent {
			castTarget = TargetPermanentOpponentControls(IsCreature)
		}
	}
	return NewAura("Controller aura", "", WithCastTarget(castTarget))
}

func TestAuraControllerUntargetedEntry(t *testing.T) {
	for _, opponent := range []bool{false, true} {
		for _, legal := range []bool{false, true} {
			a := &handEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			controller := a.PlayerID()
			if opponent == legal {
				controller = b.PlayerID()
			}
			host := g.PutOnBattlefield(NewCreature("Host", "", 2, 2), controller)
			host.GrantBaseAttr(core.Shroud)
			card := controllerAura(opponent, false)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			visit := a.handVisit(card.ID())
			if err := ApplyEffect(g, MayPutFromHandOntoBattlefield(You, IsPermanentCard), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			p := g.FindPermanent(card.ID())
			if (p != nil) != legal {
				t.Fatal("untargeted entry ignored enchant controller")
			}
			if legal && p.AttachedTo != host.ID() || !legal && (len(a.Hand()) != 1 || a.handVisit(card.ID()) != visit) {
				t.Fatal("entry used wrong host or changed blocked hand visit")
			}
		}
	}
}

func TestAuraControllerSimultaneousStateAction(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := g.PutOnBattlefield(NewCreature("Host", "", 2, 2), a.PlayerID())
	firstCard, secondCard := controllerAura(false, false), controllerAura(false, true)
	firstCard.SetOwner(b.PlayerID())
	secondCard.SetOwner(a.PlayerID())
	first := g.PutOnBattlefield(firstCard, a.PlayerID())
	second := g.PutOnBattlefield(secondCard, a.PlayerID())
	g.Attach(first.ID(), host.ID())
	g.Attach(second.ID(), host.ID())
	g.MutablePermanent(first.ID()).GrantBaseAttr(core.Indestructible)
	g.AddRegenerationShield(second.ID())
	seen := 0
	watcher := NewTriggered(core.EvtZoneChange, false).SetCondition(func(evt *core.GameEvent, before GameReader, _, _ uuid.UUID) bool {
		if evt.FromZone != core.ZoneBattlefield || evt.SourceID != first.ID() && evt.SourceID != second.ID() {
			return false
		}
		seen++
		if g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil || before.FindPermanent(first.ID()) == nil || before.FindPermanent(second.ID()) == nil {
			t.Error("Aura moves did not retain simultaneous state and departure information")
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), a.PlayerID())
	g.AddControlEffect(ControlEffectSpec{SourceID: host.ID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.CheckStateBasedActions()
	if seen != 2 || len(a.Graveyard()) != 1 || len(b.Graveyard()) != 1 {
		t.Fatal("controller legality did not use the shared Aura move batch")
	}
}

func TestAuraControllerAttachment(t *testing.T) {
	for _, opponent := range []bool{false, true} {
		for _, wrapped := range []bool{false, true} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			legalPlayer, illegalPlayer := a.PlayerID(), b.PlayerID()
			if opponent {
				legalPlayer, illegalPlayer = illegalPlayer, legalPlayer
			}
			legal := g.PutOnBattlefield(NewCreature("Legal", "", 2, 2), legalPlayer)
			illegal := g.PutOnBattlefield(NewCreature("Illegal", "", 2, 2), illegalPlayer)
			card := controllerAura(opponent, wrapped)
			card.SetOwner(b.PlayerID())
			aura := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(aura.ID(), illegal.ID())
			if g.FindPermanent(aura.ID()).IsAttached() {
				t.Fatal("controller restriction ignored on direct attachment")
			}
			legal.GrantBaseAttr(core.Shroud)
			g.Attach(aura.ID(), legal.ID())
			if g.FindPermanent(aura.ID()).AttachedTo != legal.ID() {
				t.Fatal("shroud or ownership changed untargeted enchant legality")
			}
			g.Attach(aura.ID(), illegal.ID())
			if g.FindPermanent(aura.ID()).AttachedTo != legal.ID() {
				t.Fatal("illegal move changed old attachment")
			}
			clone := g.Clone()
			g.AddControlEffect(ControlEffectSpec{SourceID: illegal.ID(), TargetID: legal.ID(), ControllerID: illegalPlayer, Duration: core.Indefinite})
			g.CheckStateBasedActions()
			if g.FindPermanent(aura.ID()) != nil || len(b.Graveyard()) != 1 || len(a.Graveyard()) != 0 {
				t.Fatal("illegal Aura did not move to its owner's graveyard")
			}
			clone.CheckStateBasedActions()
			if clone.FindPermanent(aura.ID()) == nil {
				t.Fatal("control change leaked into clone")
			}
		}
	}
}

func TestAuraControllerCurrentAuraControl(t *testing.T) {
	for _, opponent := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		hostPlayer := a.PlayerID()
		if opponent {
			hostPlayer = b.PlayerID()
		}
		host := g.PutOnBattlefield(NewCreature("Host", "", 2, 2), hostPlayer)
		card := controllerAura(opponent, false)
		card.SetOwner(a.PlayerID())
		aura := g.PutOnBattlefield(card, a.PlayerID())
		g.Attach(aura.ID(), host.ID())
		g.PhaseIn(g.PhaseOut(host.ID()))
		g.CheckStateBasedActions()
		if g.FindPermanent(aura.ID()) == nil {
			t.Fatal("phase cycle changed enchant legality")
		}
		g.AddControlEffect(ControlEffectSpec{SourceID: host.ID(), TargetID: aura.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
		g.CheckStateBasedActions()
		if g.FindPermanent(aura.ID()) != nil || len(a.Graveyard()) != 1 {
			t.Fatal("enchant restriction did not follow Aura control")
		}
	}
}
