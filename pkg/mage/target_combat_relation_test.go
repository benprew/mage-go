package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTargetCombatRelation(t *testing.T) {
	for _, direction := range []string{"blocks", "blocked", "either"} {
		for _, change := range []string{"live", "tap", "source-exit", "target-exit", "source-return", "target-return", "source-phase", "target-phase", "phase-cycle", "control", "shroud", "source-abilities", "copy", "clone"} {
			t.Run(direction+"/"+change, func(t *testing.T) {
				g, a, b := newSpellCopyGame()
				option := TargetBlocksSource()
				if direction == "blocked" {
					option = TargetBlockedBySource()
				}
				if direction == "either" {
					option = TargetInCombatWithSource()
				}
				target := DeclarePermanentTarget("opponent", IsCreature, option)
				card := NewCreature("Source", "", 2, 8, WithAbility(Activate(Costs(ManaPayment("{B}")), AddCountersTo(target, P1P1, Fixed(1)), WithTargets(target))))
				source := g.PutOnBattlefield(card, a.PlayerID())
				victimCard := NewCreature("Victim", "", 2, 8)
				victim := g.PutOnBattlefield(victimCard, b.PlayerID())
				a.ManaPool().Add(Black, 1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err == nil {
					t.Fatal("unrelated target accepted")
				}
				if a.ManaPool().Count(Black) != 1 || g.StackSize() != 0 {
					t.Fatal("illegal target spent mana")
				}
				attacker, blocker := source, victim
				if direction == "blocked" {
					attacker, blocker = victim, source
				}
				g.combat.AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
				g.combat.AddBlockerIncarnation(blocker.ID(), blocker.incarnationID, attacker.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
					t.Fatal(err)
				}
				clone := g.Clone()
				switch change {
				case "tap":
					g.TapPermanent(victim)
				case "source-exit":
					g.combat.RemoveFromCombat(source.ID())
				case "target-exit":
					g.combat.RemoveFromCombat(victim.ID())
				case "source-return":
					g.RemoveFromBattlefield(source)
					source = g.PutOnBattlefield(card, a.PlayerID())
					if direction == "blocked" {
						g.combat.AddBlockerIncarnation(source.ID(), source.incarnationID, victim.ID())
					} else {
						g.combat.AddAttackerIncarnation(source.ID(), source.incarnationID, b.PlayerID())
						g.combat.AddBlockerIncarnation(victim.ID(), victim.incarnationID, source.ID())
					}
				case "target-return":
					g.RemoveFromBattlefield(victim)
					victim = g.PutOnBattlefield(victimCard, b.PlayerID())
					if direction == "blocked" {
						g.combat.AddAttackerIncarnation(victim.ID(), victim.incarnationID, a.PlayerID())
						g.combat.AddBlockerIncarnation(source.ID(), source.incarnationID, victim.ID())
					} else {
						g.combat.AddBlockerIncarnation(victim.ID(), victim.incarnationID, source.ID())
					}
				case "source-phase":
					g.PhaseOut(source.ID())
				case "target-phase":
					g.PhaseOut(victim.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(victim.ID()))
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
				case "shroud":
					g.MutablePermanent(victim.ID()).GrantBaseAttr(Shroud)
				case "source-abilities":
					g.AddContinuousEffect(RemoveAllAbilities(source.ID(), Indefinite))
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				case "clone":
					clone.combat.RemoveFromCombat(victim.ID())
				}
				g.ResolveStack()
				want := int(0)
				if change == "live" || change == "tap" || change == "clone" || change == "source-abilities" {
					want = 1
				}
				if change == "copy" {
					want = 2
				}
				if g.FindPermanentIncludingPhased(victim.ID()).Counters[P1P1] != want {
					t.Fatal("wrong live relationship or incarnation")
				}
				clone.ResolveStack()
				cloneWant := int(1)
				if change == "clone" {
					cloneWant = 0
				}
				if clone.FindPermanent(victim.ID()).Counters[P1P1] != cloneWant {
					t.Fatal("clone lost independent combat or targets")
				}
			})
		}
	}
}

func TestTargetCombatRelationDefinition(t *testing.T) {
	for _, option := range []PermanentTargetOption{TargetBlocksSource(), TargetBlockedBySource(), TargetInCombatWithSource()} {
		target := DeclarePermanentTarget("opponent", IsCreature, option)
		spell := NewInstant("Invalid", "", WithAbility(Spell(TapSubject(target), WithTargets(target))))
		if ValidateCardDefinition(spell.Name(), spell) == nil {
			t.Fatal("battlefield source restriction on spell accepted")
		}
		source := NewCreature("Valid", "", 2, 8, WithAbility(Activate(Costs(), TapSubject(target), WithTargets(target))))
		if err := ValidateCardDefinition(source.Name(), source); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTargetCombatRelationBandsAndMultiplePairs(t *testing.T) {
	g, a, b := newSpellCopyGame()
	target := DeclarePermanentTarget("blocker", IsCreature, TargetBlocksSource())
	source := g.PutOnBattlefield(NewCreature("Source", "", 2, 8, WithKeyword(Banding), WithAbility(Activate(Costs(), AddCountersTo(target, P1P1, Fixed(1)), WithTargets(target)))), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Band member", "", 2, 8), a.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 8), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 8), b.PlayerID())
	source.RevokeBaseAttr(AttrSummonSick)
	other.RevokeBaseAttr(AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{source.ID(), other.ID()})
	g.combat.AddBand([]uuid.UUID{source.ID(), other.ID()})
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: first.ID(), AttackerID: other.ID()}, {BlockerID: second.ID(), AttackerID: source.ID()}})
	for _, id := range []uuid.UUID{first.ID(), second.ID()} {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{id}); err != nil {
			t.Fatal(err)
		}
	}
	g.combat.RemoveFromCombat(first.ID())
	g.ResolveStack()
	if g.FindPermanent(first.ID()).Counters[P1P1] != 0 || g.FindPermanent(second.ID()).Counters[P1P1] != 1 {
		t.Fatal("band or multiple blockers lost pair identity")
	}
	if !g.combat.GroupFor(source.ID()).Blocked {
		t.Fatal("pair check changed blocked status")
	}
}
