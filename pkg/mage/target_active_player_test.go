package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTargetActivePlayerContract(t *testing.T) {
	for _, change := range []string{"normal", "active", "control", "return", "phase", "cycle", "source-return", "copy", "clone-active"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			g.SetActivePlayerIndex(1)
			target := DeclarePermanentTarget("active creature", IsCreature, TargetControlledBy(ActivePlayer))
			card := NewArtifact("Contract", "", WithAbility(Activate(Costs(), TapSubject(target), WithTargets(target))))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			own := g.PutOnBattlefield(NewCreature("Own", "", 2, 3), a.PlayerID())
			victimCard := NewCreature("Victim", "", 2, 3)
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			possible := target.targetSpec().Possible(a.PlayerID(), card, g)
			if !slices.Equal(possible, []uuid.UUID{victim.ID()}) {
				t.Fatalf("active-player targets: %v", possible)
			}
			g.SetRandomResults([]int{7})
			query := target.targetSpec().Possible(b.PlayerID(), card, g)
			random := g.RandIntn(10)
			if !slices.Equal(query, possible) || random != 7 || victim.Tapped || g.StackSize() != 0 {
				t.Fatal("target query changed state or used the query controller")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			legal := true
			switch change {
			case "active":
				g.SetActivePlayerIndex(0)
				legal = false
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: own.ID(), TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
				legal = false
			case "return":
				g.RemoveFromBattlefield(victim)
				g.PutOnBattlefield(victimCard, b.PlayerID())
				legal = false
			case "phase":
				g.PhaseOut(victim.ID())
				legal = false
			case "cycle":
				g.PhaseIn(g.PhaseOut(victim.ID()))
			case "source-return":
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(card, b.PlayerID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			if change == "clone-active" {
				clone.SetActivePlayerIndex(0)
			}
			g.ResolveStack()
			if clone.FindPermanentIncludingPhased(victim.ID()).Tapped {
				t.Fatal("resolution changed clone")
			}
			clone.ResolveStack()
			for _, game := range []*Game{g, clone} {
				want := legal && (change != "clone-active" || game != clone)
				if game.FindPermanent(own.ID()).Tapped {
					t.Fatal("used action controller instead of active player")
				}
				p := game.FindPermanentIncludingPhased(victim.ID())
				if p == nil || p.Tapped != want {
					t.Fatal("target legality changed")
				}
			}
		})
	}
}
