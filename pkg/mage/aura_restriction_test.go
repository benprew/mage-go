package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCantBeEnchantedExceptByDefinition(t *testing.T) {
	target := DeclarePermanentTarget("unbound", AnyPermanent)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "host", Card: NewAura("Good", "", WithAbility(Static(CantBeEnchantedExceptBy(AttachedPermanent, This))))},
		{Name: "nil subject", Card: NewAura("Bad", "", WithAbility(Static(CantBeEnchantedExceptBy(nil, This)))), WantErr: true},
		{Name: "nil exception", Card: NewAura("Bad", "", WithAbility(Static(CantBeEnchantedExceptBy(AttachedPermanent, nil)))), WantErr: true},
		{Name: "unbound", Card: NewAura("Bad", "", WithAbility(Static(CantBeEnchantedExceptBy(target, This)))), WantErr: true},
	})
}

func TestCantBeEnchantedExceptByLifecycle(t *testing.T) {
	for _, change := range []string{"none", "remove", "phase", "phase-cycle", "control", "reattach", "source-return", "host-return"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			host := g.PutOnBattlefield(NewCreature("Host", "", 2, 5), b.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 2, 5), a.PlayerID())
			card := NewAura("Rule", "", WithAbility(Static(CantBeEnchantedExceptBy(AttachedPermanent, This))))
			rule := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(rule.ID(), host.ID())
			if !auraHostIsLegal(card, host, g) || rule.AttachedTo != host.ID() {
				t.Fatal("the granting Aura excluded itself")
			}
			aura := g.PutOnBattlefield(NewAura("Other Aura", ""), b.PlayerID())
			g.Attach(aura.ID(), other.ID())
			g.Attach(aura.ID(), host.ID())
			if g.FindPermanent(aura.ID()).AttachedTo != other.ID() {
				t.Fatal("illegal attachment moved")
			}
			clone := g.Clone()
			switch change {
			case "remove":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), rule.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "phase":
				g.PhaseOut(rule.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(rule.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: other.ID(), TargetID: rule.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "reattach":
				g.Attach(rule.ID(), other.ID())
			case "source-return":
				g.RemoveFromBattlefield(g.FindPermanent(rule.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
			case "host-return":
				g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
				host = g.PutOnBattlefield(host.Card, b.PlayerID())
			}
			allowed := change == "remove" || change == "phase" || change == "reattach" || change == "source-return" || change == "host-return"
			if auraHostIsLegal(aura.Card, host, g) != allowed {
				t.Fatal("wrong current attachment restriction")
			}
			cloneHost := clone.FindPermanent(host.ID())
			if auraHostIsLegal(aura.Card, cloneHost, clone) {
				t.Fatal("clone restriction changed")
			}
			if change == "remove" {
				g.RunStepWithPriority(core.Cleanup)
				if auraHostIsLegal(aura.Card, host, g) {
					t.Fatal("cleanup did not restore the rule")
				}
			}
		})
	}
}

func TestCantBeEnchantedExceptByStateAction(t *testing.T) {
	g, a, b := newSpellCopyGame()
	host := g.PutOnBattlefield(NewCreature("Host", "", 2, 5), b.PlayerID())
	first := g.PutOnBattlefield(NewAura("First", "", WithKeyword(core.Indestructible)), b.PlayerID())
	second := g.PutOnBattlefield(NewAura("Second", ""), b.PlayerID())
	g.Attach(first.ID(), host.ID())
	g.Attach(second.ID(), host.ID())
	g.AddRegenerationShield(second.ID())
	rule := g.PutOnBattlefield(NewAura("Rule", "", WithAbility(Static(CantBeEnchantedExceptBy(AttachedPermanent, This)))), a.PlayerID())
	g.Attach(rule.ID(), host.ID())
	seen := 0
	watcher := NewTriggered(core.EvtZoneChange, false).SetCondition(func(evt *core.GameEvent, before GameReader, _, _ uuid.UUID) bool {
		if evt.FromZone != core.ZoneBattlefield || evt.SourceID != first.ID() && evt.SourceID != second.ID() {
			return false
		}
		seen++
		if g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil || before.FindPermanent(first.ID()) == nil || before.FindPermanent(second.ID()) == nil {
			t.Error("event did not preserve the simultaneous move and former battlefield")
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), a.PlayerID())
	if g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) == nil {
		t.Fatal("state action ran during attachment")
	}
	g.CheckStateBasedActions()
	if g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil || len(b.Graveyard()) != 2 {
		t.Fatal("illegal Auras were destroyed instead of put into the graveyard")
	}
	if g.FindPermanent(rule.ID()) == nil {
		t.Fatal("exception Aura fell off")
	}
	if seen != 2 {
		t.Fatalf("zone events=%d, want 2", seen)
	}
}

func TestCantBeEnchantedExceptByEntryAndEquipment(t *testing.T) {
	a := &handEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	host := g.PutOnBattlefield(NewCreature("Host", "", 2, 5), b.PlayerID())
	rule := g.PutOnBattlefield(NewAura("Rule", "", WithAbility(Static(CantBeEnchantedExceptBy(AttachedPermanent, This)))), a.PlayerID())
	g.Attach(rule.ID(), host.ID())
	card := NewAura("Entry", "")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	visit := a.handVisit(card.ID())
	if err := ApplyEffect(g, MayPutFromHandOntoBattlefield(You, IsPermanentCard), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil || a.handVisit(card.ID()) != visit {
		t.Fatal("Aura with no legal host left its original hand visit")
	}
	equipment := g.PutOnBattlefield(NewArtifact("Equipment", "", WithSubTypes("Equipment")), a.PlayerID())
	g.Attach(equipment.ID(), host.ID())
	if g.FindPermanent(equipment.ID()).AttachedTo != host.ID() {
		t.Fatal("equipment was restricted")
	}
	g.MutablePermanent(host.ID()).GrantBaseAttr(core.Shroud)
	g.BouncePermanentToHand(g.FindPermanent(rule.ID()))
	if err := ApplyEffect(g, MayPutFromHandOntoBattlefield(You, HasSubTypeCardFilter("Aura")), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if p := g.FindPermanent(card.ID()); p == nil || p.AttachedTo != host.ID() {
		t.Fatal("nontargeted entry was blocked by shroud after the rule left")
	}
}
