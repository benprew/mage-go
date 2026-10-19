package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestGlobalCombatPreventionInstructionDeclaration(t *testing.T) {
	effect := (GlobalCombatPreventionInstruction{}).Effect()
	for _, e := range []Effect{effect, PreventAllCombatDamage()} {
		if _, ok := InspectGlobalCombatPreventionInstruction(e); !ok {
			t.Fatal("lost native global prevention")
		}
		card := NewInstant("Prevention", "", Spell(e))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	for _, e := range []Effect{
		nil, (*preventAllCombatDamageEffect)(nil), (*sequenceEffect)(nil),
		PreventCombatDamageByThisTurn(This), PreventCombatDamageToAndBy(This),
		PreventCombatDamageByEachThisTurn(MatchingPermanents(IsCreature)),
		PreventNextDamage(You, Fixed(3)), PreventDamageToYouByCreaturesWithFlying(),
		Sequence(effect), CompositeEffects("wrapped", effect), destructionInspectionTrap{},
	} {
		if _, ok := InspectGlobalCombatPreventionInstruction(e); ok {
			t.Fatal("accepted a different prevention declaration")
		}
	}
}

func TestGlobalCombatPreventionInstructionQuery(t *testing.T) {
	for _, state := range []string{
		"alone", "nested", "prefix", "suffix", "empty watch", "copy", "clone",
		"all illegal", "illegal prefix", "phase", "cycle", "return", "foreign",
		"opaque prefix", "opaque suffix", "opaque all illegal", "nil suffix",
		"typed nil suffix", "duplicate", "legacy",
	} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			prevention := (GlobalCombatPreventionInstruction{}).Effect()
			obj.Effects = []Effect{prevention}
			watched := []PermanentVisit{{p.ID(), p.incarnationID}}
			certain, known := false, true
			switch state {
			case "nested":
				obj.Effects = []Effect{Sequence(CompositeEffects("nested", prevention))}
			case "prefix":
				obj.Effects = []Effect{Sequence(prevention, Destroy(target))}
				known = false
			case "suffix", "copy", "clone", "cycle":
				obj.Effects = []Effect{Sequence(Destroy(target), prevention)}
				certain = true
			case "empty watch":
				obj.Effects = []Effect{Sequence(Destroy(target), prevention)}
				watched = nil
			case "all illegal", "illegal prefix":
				p.GrantBaseAttr(core.Shroud)
				obj.Effects = []Effect{Sequence(prevention, Destroy(target))}
			case "phase":
				g.PhaseOut(p.ID())
				obj.Effects = []Effect{Destroy(target), prevention}
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, obj.Controller)
				obj.Effects = []Effect{Destroy(target), prevention}
			case "foreign":
				obj.Effects = []Effect{Destroy(DeclarePermanentTarget("foreign", AnyPermanent)), prevention}
				known = false
			case "opaque prefix":
				obj.Effects = []Effect{destructionInspectionTrap{}, prevention, Destroy(target)}
				known = false
			case "opaque suffix", "opaque all illegal":
				obj.Effects = []Effect{Destroy(target), prevention, destructionInspectionTrap{}}
				known = false
				if state == "opaque all illegal" {
					p.GrantBaseAttr(core.Shroud)
				}
			case "nil suffix":
				obj.Effects = []Effect{Destroy(target), prevention, nil}
				known = false
			case "typed nil suffix":
				obj.Effects = []Effect{Destroy(target), (*preventAllCombatDamageEffect)(nil)}
				known = false
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
				obj.Targets = append(obj.Targets, p.ID())
				known = false
			case "legacy":
				obj.Effects = []Effect{PreventAllCombatDamage()}
			}
			if state == "cycle" {
				g.PhaseIn(g.PhaseOut(p.ID()))
			}
			if state == "copy" {
				obj = g.CopyStackObjectDirect(obj, g.GetOpponent(obj.Controller).PlayerID(), false)
			}
			if state == "clone" {
				g = g.Clone()
				obj = g.StackPeek()
			}
			before := g.Clone()
			for range 2 {
				c, k := g.InspectSpellDestruction(obj, watched)
				if c != certain || k != known {
					t.Fatalf("got %v/%v; want %v/%v", c, k, certain, known)
				}
			}
			if !reflect.DeepEqual(g.random, before.random) || len(g.effects.replacements) != len(before.effects.replacements) || g.StackSize() != before.StackSize() || g.FindPermanentIncludingPhased(p.ID()) == nil && state != "return" {
				t.Fatal("inspection changed game state")
			}
		})
	}
}

func TestGlobalCombatPreventionInstructionExecution(t *testing.T) {
	for _, state := range []string{"normal", "phase", "return", "control"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewCreature("Source", "", 1, 20), a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 20), b.PlayerID())
			if err := ApplyEffect(g, (GlobalCombatPreventionInstruction{}).Effect(), source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "phase":
				g.PhaseOut(source.ID())
			case "return":
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(source.Card, a.PlayerID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			clone := g.Clone()
			g.effects.RemoveEndOfCombat()
			g.damage.SetResolvingCombatDamage(true)
			for range 2 {
				g.DealDamageToPlayer(a, 2, uuid.Nil)
				g.DealDamageToPlayer(b, 2, uuid.Nil)
				g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 2, uuid.Nil)
			}
			if a.Life() != 20 || b.Life() != 20 || g.FindPermanent(victim.ID()).Damage != 0 {
				t.Fatal("global prevention lost a recipient or a later combat")
			}
			g.damage.SetResolvingCombatDamage(false)
			g.DealDamageToPlayer(b, 1, uuid.Nil)
			g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, uuid.Nil)
			if b.Life() != 19 || g.FindPermanent(victim.ID()).Damage != 1 {
				t.Fatal("noncombat damage was prevented")
			}
			g.RunStepWithPriority(core.Cleanup)
			if result := g.effects.ApplyReplacements(NewDamageToPlayerAction(uuid.Nil, b.PlayerID(), 2, true), g); result == nil {
				t.Fatal("prevention survived cleanup")
			}
			if result := clone.effects.ApplyReplacements(NewDamageToPlayerAction(uuid.Nil, b.PlayerID(), 2, true), clone); result != nil {
				t.Fatal("cleanup changed the clone")
			}
		})
	}
}
