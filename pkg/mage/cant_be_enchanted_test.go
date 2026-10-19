package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCantBeEnchantedDefinition(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "self", Card: NewCreature("Good", "", 2, 3, WithAbility(Static(CantBeEnchanted(This))))},
		{Name: "nil", Card: NewCreature("Bad", "", 2, 3, WithAbility(Static(CantBeEnchanted(nil)))), WantErr: true},
		{Name: "unbound", Card: NewCreature("Bad", "", 2, 3, WithAbility(Static(CantBeEnchanted(DeclarePermanentTarget("unbound", AnyPermanent))))), WantErr: true},
	})
}

func TestCantBeEnchantedLifecycle(t *testing.T) {
	for _, change := range []string{"none", "remove", "phase", "phase-cycle", "control", "return"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			card := NewCreature("Host", "", 2, 3, WithAbility(Static(CantBeEnchanted(This))))
			host := g.PutOnBattlefield(card, a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 2, 3), b.PlayerID())
			aura := NewAura("Aura", "")
			if auraHostIsLegal(aura, host, g) || !auraHostIsLegal(aura, other, g) {
				t.Fatal("restriction did not select only the subject")
			}
			clone := g.Clone()
			switch change {
			case "remove":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), host.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "phase":
				g.PhaseOut(host.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: other.ID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "return":
				g.RemoveFromBattlefield(host)
				host = g.PutOnBattlefield(card, a.PlayerID())
			}
			if change != "phase" && auraHostIsLegal(aura, g.FindPermanent(host.ID()), g) != (change == "remove") {
				t.Fatal("wrong live restriction")
			}
			if auraHostIsLegal(aura, clone.FindPermanent(host.ID()), clone) {
				t.Fatal("clone lost restriction")
			}
			if change == "remove" {
				g.RunStepWithPriority(core.Cleanup)
				if auraHostIsLegal(aura, g.FindPermanent(host.ID()), g) {
					t.Fatal("cleanup did not restore restriction")
				}
			}
		})
	}
}

func TestCantBeEnchantedAttachments(t *testing.T) {
	g, a, b := newSpellCopyGame()
	host := g.PutOnBattlefield(NewCreature("Host", "", 2, 3, WithAbility(Static(CantBeEnchanted(This)))), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 2, 3), b.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Aura", ""), b.PlayerID())
	g.Attach(aura.ID(), other.ID())
	g.Attach(aura.ID(), host.ID())
	if g.FindPermanent(aura.ID()).AttachedTo != other.ID() {
		t.Fatal("illegal Aura moved")
	}
	equipment := g.PutOnBattlefield(NewArtifact("Equipment", "", WithSubTypes("Equipment")), a.PlayerID())
	g.Attach(equipment.ID(), host.ID())
	if g.FindPermanent(equipment.ID()).AttachedTo != host.ID() {
		t.Fatal("equipment was restricted")
	}
}

func TestCantBeEnchantedEntryAndStateActions(t *testing.T) {
	a := &handEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	host := g.PutOnBattlefield(NewCreature("Host", "", 2, 3, WithAbility(Static(CantBeEnchanted(This)))), a.PlayerID())
	card := NewAura("Entry", "")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	visit := a.handVisit(card.ID())
	if err := ApplyEffect(g, MayPutFromHandOntoBattlefield(You, HasSubTypeCardFilter("Aura")), host.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil || a.handVisit(card.ID()) != visit {
		t.Fatal("illegal entry changed the hand visit")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), host.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	first := g.PutOnBattlefield(NewAura("First", "", WithKeyword(core.Indestructible)), a.PlayerID())
	second := g.PutOnBattlefield(NewAura("Second", ""), a.PlayerID())
	g.Attach(first.ID(), host.ID())
	g.Attach(second.ID(), host.ID())
	g.AddRegenerationShield(second.ID())
	if g.FindPermanent(first.ID()).AttachedTo != host.ID() || g.FindPermanent(second.ID()).AttachedTo != host.ID() {
		t.Fatal("removed restriction blocked attachment")
	}
	g.RunStepWithPriority(core.Cleanup)
	g.CheckStateBasedActions()
	if g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil || len(a.Graveyard()) != 2 {
		t.Fatal("illegal Auras did not enter the graveyard")
	}
}
