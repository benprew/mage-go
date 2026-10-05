package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTargetToughnessBelowSourcePower(t *testing.T) {
	for _, change := range []string{"live", "equal", "negative", "departure", "return", "phase", "phase-cycle", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			target := DeclarePermanentTarget("small", IsCreature, TargetToughnessLessThanSourcePower())
			card := NewCreature("Source", "", 3, 8, WithAbility(Activate(Costs(TapCost(This)), TapSubject(target), WithTargets(target))))
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(AttrSummonSick)
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 3), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err == nil {
				t.Fatal("equal toughness accepted")
			}
			if g.StackSize() != 0 || g.FindPermanent(source.ID()).Tapped {
				t.Fatal("illegal activation changed state")
			}
			g.FindPermanent(source.ID()).AddCounter(P1P1, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "equal":
				g.FindPermanent(source.ID()).RemoveCounter(P1P1, 1)
			case "negative":
				g.FindPermanent(source.ID()).AddCounter(M1M1, 6)
			case "departure", "return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "return" {
					g.PutOnBattlefield(card, a.PlayerID()).AddCounter(M1M1, 3)
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			want := change != "equal" && change != "negative"
			if g.FindPermanent(victim.ID()).Tapped != want {
				t.Fatal("wrong source power at resolution")
			}
			if clone.FindPermanent(victim.ID()).Tapped {
				t.Fatal("clone shared mutation")
			}
			clone.ResolveStack()
			if clone.FindPermanent(victim.ID()).Tapped != want {
				t.Fatal("clone lost source power")
			}
		})
	}
}

func TestTargetToughnessBelowSourcePowerLayerLKI(t *testing.T) {
	for _, change := range []string{"depart", "phase", "return", "live"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			target := DeclarePermanentTarget("small", IsCreature, TargetToughnessLessThanSourcePower())
			card := NewCreature("Source", "", 3, 8, WithAbility(Static(While(PermanentMatchesFilter(This, AnyPermanent), ModifyPT(This, Fixed(2), Fixed(0))))), WithAbility(Activate(Costs(TapCost(This)), TapSubject(target), WithTargets(target))))
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(AttrSummonSick)
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 4), b.PlayerID())
			if source.CurrentPower(g) != 5 {
				t.Fatal("source layer fixture failed")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "depart", "return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "return" {
					g.PutOnBattlefield(card, a.PlayerID()).AddCounter(M1M1, 3)
				}
			case "phase":
				g.PhaseOut(source.ID())
			}
			g.ApplyContinuousEffects()
			g.ResolveStack()
			if !g.FindPermanent(victim.ID()).Tapped {
				t.Fatal("source layer power lost after absence")
			}
		})
	}
}
