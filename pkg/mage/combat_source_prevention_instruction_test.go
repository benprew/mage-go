package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCombatSourcePreventionInstructionDeclaration(t *testing.T) {
	target := DeclarePermanentTarget("source", AnyPermanent)
	effect := (CombatSourcePreventionInstruction{Subject: target}).Effect()
	got, ok := InspectCombatSourcePreventionInstruction(effect)
	if !ok || got.Subject != target {
		t.Fatal("lost source declaration")
	}
	for _, subject := range []PermanentRef{nil, (*PermanentTargetBinding)(nil)} {
		if (CombatSourcePreventionInstruction{Subject: subject}).Effect() != nil {
			t.Fatal("missing source accepted")
		}
	}
	for _, effect := range []Effect{nil, (*combatSourcePreventionInstallation)(nil), PreventCombatDamageByThisTurn(nil), PreventCombatDamageByThisTurn((*PermanentTargetBinding)(nil)), PreventCombatDamageBy(target), PreventCombatDamageByEachThisTurn(PermanentsControlledBy(You, AnyPermanent)), PreventCombatDamageToAndBy(target), &combatSourcePreventionInstallation{subject: target, subjects: PermanentsControlledBy(You, AnyPermanent), duration: core.EndOfTurn}, Sequence(effect), destructionInspectionTrap{}} {
		if _, ok := InspectCombatSourcePreventionInstruction(effect); ok {
			t.Fatal("unsupported prevention accepted")
		}
	}
	if _, ok := InspectCombatSourcePreventionInstruction(PreventCombatDamageByThisTurn(target)); !ok {
		t.Fatal("compatible caller lost native inspection")
	}
}

func TestCombatSourcePreventionInstructionInspection(t *testing.T) {
	for _, state := range []string{"alone", "prefix", "suffix", "foreign", "opaque subject", "opaque suffix", "illegal", "unknown illegal", "duplicate", "copy", "clone", "returned", "phase", "phase cycle", "empty prefix", "illegal prefix"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			subject := PermanentRef(target)
			certain, known := false, true
			switch state {
			case "foreign":
				subject = DeclarePermanentTarget("foreign", AnyPermanent)
				known = false
			case "opaque subject":
				subject = This
				known = false
			}
			prevention := (CombatSourcePreventionInstruction{Subject: subject}).Effect()
			obj.Effects = []Effect{prevention}
			switch state {
			case "prefix":
				obj.Effects = append(obj.Effects, Destroy(target))
				known = false
			case "suffix", "copy", "clone":
				obj.Effects = append([]Effect{Destroy(target)}, obj.Effects...)
				certain = true
			case "opaque suffix":
				obj.Effects = []Effect{Destroy(target), prevention, destructionInspectionTrap{}}
				known = false
			case "illegal":
				p.GrantBaseAttr(core.Shroud)
			case "unknown illegal":
				p.GrantBaseAttr(core.Shroud)
				obj.Effects = append(obj.Effects, destructionInspectionTrap{})
				known = false
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
				obj.Targets = append(obj.Targets, p.ID())
				known = false
			case "returned":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, obj.Controller)
			case "phase":
				g.PhaseOut(p.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "illegal prefix":
				p.GrantBaseAttr(core.Shroud)
				other := g.PutOnBattlefield(NewLand("Other"), obj.Controller)
				otherTarget := DeclarePermanentTarget("other", HasCardTypeFilter(core.TypeLand))
				obj.TargetSpecs = append(obj.TargetSpecs, otherTarget.targetSpec())
				obj.Targets = append(obj.Targets, other.ID())
				obj.TargetZones[other.ID()] = core.ZoneBattlefield
				obj.targetIncarnations[other.ID()] = other.incarnationID
				obj.Effects = []Effect{prevention, Destroy(otherTarget)}
				p = other
				certain = true
			case "empty prefix":
				p.GrantBaseAttr(core.Shroud)
				obj.Effects = append(obj.Effects, Destroy(target))
			}
			if state == "copy" {
				obj = g.CopyStackObjectDirect(obj, g.GetOpponent(obj.Controller).PlayerID(), false)
			}
			if state == "clone" {
				g = g.Clone()
				obj = g.StackPeek()
			}
			for range 2 {
				c, k := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}})
				if c != certain || k != known {
					t.Fatalf("got %v/%v want %v/%v", c, k, certain, known)
				}
			}
			if len(g.effects.replacements) != 0 {
				t.Fatal("inspection installed prevention")
			}
		})
	}
}

func TestCombatSourcePreventionInstructionExecution(t *testing.T) {
	for _, change := range []string{"normal", "phase cycle", "returned", "control", "lost creature type"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "", 3, 20)
			p := g.PutOnBattlefield(card, a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 1, 20), b.PlayerID())
			if err := ApplyEffect(g, (CombatSourcePreventionInstruction{Subject: This}).Effect(), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			switch change {
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "returned":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, a.PlayerID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: other.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "lost creature type":
				g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrIsCreature)
			}
			g.effects.RemoveEndOfCombat()
			g.damage.SetResolvingCombatDamage(true)
			for range 2 {
				g.DealDamageToPlayer(b, 2, p.ID())
				g.DealDamageToPermanent(g.FindPermanent(other.ID()), 2, p.ID())
			}
			wantLife, wantDamage := 20, 0
			if change == "returned" {
				wantLife, wantDamage = 16, 4
			}
			if b.Life() != wantLife || g.FindPermanent(other.ID()).Damage != wantDamage {
				t.Fatal("wrong source visit or duration")
			}
			g.DealDamageToPlayer(b, 1, other.ID())
			g.damage.SetResolvingCombatDamage(false)
			g.DealDamageToPlayer(b, 1, p.ID())
			if b.Life() != wantLife-2 {
				t.Fatal("prevented unrelated or noncombat damage")
			}
			g.RunStepWithPriority(core.Cleanup)
			g.damage.SetResolvingCombatDamage(true)
			g.DealDamageToPlayer(b, 1, p.ID())
			if b.Life() != wantLife-3 {
				t.Fatal("prevention survived cleanup")
			}
			clone.damage.SetResolvingCombatDamage(true)
			clone.DealDamageToPlayer(clone.GetPlayer(b.PlayerID()), 2, p.ID())
			if clone.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("clone lost prevention")
			}
		})
	}
}
