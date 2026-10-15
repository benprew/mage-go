package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func stickyAnimationGame(t *testing.T) (*Game, *Permanent, *Permanent) {
	t.Helper()
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	artifact := g.PutOnBattlefield(NewArtifact("Artifact", "{3}", WithKeyword(core.Flying), WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(core.Green, Fixed(1))))), a.PlayerID())
	source := g.PutOnBattlefield(resolvingCopyCard(t, func() Card {
		return NewEnchantment("Animation", "", WithAbility(Static(AnimateArtifactsUntilCleanupAfterExit())))
	}), b.PlayerID())
	return g, source, artifact
}

func stickyAnimationCheck(t *testing.T, g *Game, p *Permanent, want bool) {
	t.Helper()
	p = g.FindPermanent(p.ID())
	if p == nil || p.HasType(core.TypeCreature) != want || !p.HasType(core.TypeArtifact) {
		t.Fatal("wrong artifact membership")
	}
	if want && (p.CurrentPower(g) != 3 || p.CurrentToughness(g) != 3 || p.HasKeyword(core.Flying) || len(p.RuntimeAbilities) != 0) {
		t.Fatalf("wrong animation result: %d/%d, abilities %d", p.CurrentPower(g), p.CurrentToughness(g), len(p.RuntimeAbilities))
	}
	if !want && (!p.HasKeyword(core.Flying) || len(p.RuntimeAbilities) == 0) {
		t.Fatal("printed abilities did not return")
	}
}

func TestStickyAnimationLifetime(t *testing.T) {
	for _, change := range []string{"live", "cleanup-live", "exit", "return", "source-phase", "source-phase-cycle", "subject-phase-cycle", "remove-source-ability", "clone-exit", "clone-cleanup"} {
		t.Run(change, func(t *testing.T) {
			g, source, p := stickyAnimationGame(t)
			stickyAnimationCheck(t, g, p, true)
			want := true
			switch change {
			case "cleanup-live":
				g.effects.RemoveEndOfTurn()
			case "exit", "return", "clone-cleanup":
				g.RemoveFromBattlefield(source)
				if change == "return" {
					g.PutOnBattlefield(source.Card, source.ControllerID())
				}
				if change == "clone-cleanup" {
					clone := g.Clone()
					clone.effects.RemoveEndOfTurn()
					clone.ApplyContinuousEffects()
					stickyAnimationCheck(t, clone, p, false)
				}
			case "source-phase", "source-phase-cycle":
				ids := g.PhaseOut(source.ID())
				want = false
				if change == "source-phase-cycle" {
					g.PhaseIn(ids)
					want = true
				}
			case "subject-phase-cycle":
				ids := g.PhaseOut(p.ID())
				g.ApplyContinuousEffects()
				g.PhaseIn(ids)
			case "remove-source-ability":
				g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.EndOfTurn))
			case "clone-exit":
				clone := g.Clone()
				clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
				clone.effects.RemoveEndOfTurn()
				clone.ApplyContinuousEffects()
				stickyAnimationCheck(t, clone, p, false)
			}
			g.ApplyContinuousEffects()
			stickyAnimationCheck(t, g, p, want)
			if change == "exit" || change == "return" {
				late := g.PutOnBattlefield(NewArtifact("Late", "{3}", WithKeyword(core.Flying), WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(core.Green, Fixed(1))))), source.ControllerID())
				stickyAnimationCheck(t, g, late, true)
				g.effects.RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				stickyAnimationCheck(t, g, p, change == "return")
				stickyAnimationCheck(t, g, late, change == "return")
			}
		})
	}
}

func TestStickyAnimationLayers(t *testing.T) {
	g, source, p := stickyAnimationGame(t)
	creature := g.PutOnBattlefield(NewCreature("Printed creature", "{5}", 2, 4, WithCardType(core.TypeArtifact), WithKeyword(core.Flying)), source.ControllerID())
	if !creature.HasKeyword(core.Flying) || creature.CurrentToughness(g) != 4 {
		t.Fatal("printed creature changed")
	}
	g.AddContinuousEffect(TemporaryKeyword(p.ID(), core.Flying))
	if !g.FindPermanent(p.ID()).HasKeyword(core.Flying) {
		t.Fatal("later grant lost")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, SetBasePowerToughness(This, Fixed(4), Fixed(5)), ModifyPT(This, Fixed(1), Fixed(2))), p.ID(), p.ControllerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.MutablePermanent(p.ID()).AddCounter(core.P1P1, 1)
	g.ApplyContinuousEffects()
	current := g.FindPermanent(p.ID())
	if current.CurrentPower(g) != 6 || current.CurrentToughness(g) != 8 {
		t.Fatal("base, bonus or counter order failed")
	}
	g.RemoveFromBattlefield(source)
	current = g.FindPermanent(p.ID())
	if current.CurrentPower(g) != 6 || current.CurrentToughness(g) != 8 || !current.HasKeyword(core.Flying) {
		t.Fatal("source exit changed effect timestamp")
	}
}

func TestStickyAnimationDependencyAndIdentity(t *testing.T) {
	g, source, p := stickyAnimationGame(t)
	land := g.PutOnBattlefield(NewLand("Land"), source.ControllerID())
	if err := ApplyEffect(g, Until(core.EndOfTurn, AddCardType(This, core.TypeArtifact)), land.ID(), land.ControllerID(), nil); err != nil {
		t.Fatal(err)
	}
	current := g.FindPermanent(land.ID())
	if !current.HasType(core.TypeCreature) || current.CurrentToughness(g) != 0 {
		t.Fatal("animation ran before its type dependency")
	}
	g.RemoveFromBattlefield(source)
	g.RemoveFromBattlefield(p)
	if lki := g.LKI(p.ID()); lki == nil || lki.Power != 3 || !lki.Snapshot.HasType(core.TypeCreature) || len(lki.Snapshot.RuntimeAbilities) != 0 {
		t.Fatal("departure lost animation LKI")
	}
	returned := g.PutOnBattlefield(p.Card, p.ControllerID())
	stickyAnimationCheck(t, g, returned, true)
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	stickyAnimationCheck(t, g, returned, false)
	if g.FindPermanent(land.ID()).HasType(core.TypeCreature) || g.FindPermanent(land.ID()).HasType(core.TypeArtifact) {
		t.Fatal("type dependency or continued animation survived cleanup")
	}
}

func TestStickyAnimationSourceCopy(t *testing.T) {
	for _, departed := range []bool{false, true} {
		t.Run(map[bool]string{false: "live", true: "departed"}[departed], func(t *testing.T) {
			g, source, p := stickyAnimationGame(t)
			donor := g.PutOnBattlefield(resolvingCopyCard(t, func() Card { return NewEnchantment("Donor", "") }), source.ControllerID())
			if departed {
				g.RemoveFromBattlefield(source)
				source = g.PutOnBattlefield(source.Card, source.ControllerID())
			}
			if err := g.BecomeCopyOfPermanent(source, donor, false); err != nil {
				t.Fatal(err)
			}
			stickyAnimationCheck(t, g, p, departed)
			if departed {
				g.effects.RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				stickyAnimationCheck(t, g, p, false)
			}
		})
	}
}

func TestStickyAnimationCombatAndDefinition(t *testing.T) {
	g, source, p := stickyAnimationGame(t)
	if err := ValidateCardDefinition("Animation", source.Card); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(p.ID()).CanTapForEffect(g) {
		t.Fatal("animation bypassed summoning sickness")
	}
	g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
	if !g.FindPermanent(p.ID()).CanTapForEffect(g) || !g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) || !g.FindPermanent(p.ID()).CanDeclareAsBlocker(g) {
		t.Fatal("old artifact has wrong combat permissions")
	}
	g.AddContinuousEffect(PreventAttackingUntilEndOfTurn(p.ID()))
	if g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("animation bypassed attack prohibition")
	}
}

func TestStickyAnimationZeroManaValue(t *testing.T) {
	g, source, _ := stickyAnimationGame(t)
	zero := g.PutOnBattlefield(NewArtifact("Zero", ""), source.ControllerID())
	if p := g.FindPermanent(zero.ID()); !p.HasType(core.TypeCreature) || p.CurrentToughness(g) != 0 {
		t.Fatal("zero mana value was not used")
	}
	g.CheckStateBasedActions()
	if g.FindPermanent(zero.ID()) != nil {
		t.Fatal("zero toughness artifact survived")
	}
}

func TestStickyAnimationClonedDepartureAndSharedDeclaration(t *testing.T) {
	g, source, p := stickyAnimationGame(t)
	other := g.PutOnBattlefield(source.Card.Copy(), source.ControllerID())
	g.RemoveFromBattlefield(source)
	clone := g.Clone()
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	stickyAnimationCheck(t, g, p, true)
	g.RemoveFromBattlefield(other)
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	stickyAnimationCheck(t, g, p, false)
	clone.ApplyContinuousEffects()
	stickyAnimationCheck(t, clone, p, true)
}

func TestStickyAnimationCreatureTypeDependency(t *testing.T) {
	g, source, p := stickyAnimationGame(t)
	if err := ApplyEffect(g, Until(core.EndOfTurn, AddCardType(This, core.TypeCreature)), p.ID(), p.ControllerID(), nil); err != nil {
		t.Fatal(err)
	}
	current := g.FindPermanent(p.ID())
	if !current.HasKeyword(core.Flying) || current.BasePTOverride != nil || len(current.RuntimeAbilities) == 0 {
		t.Fatal("prior creature type effect did not exclude the artifact")
	}
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	stickyAnimationCheck(t, g, p, true)
	g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.EndOfTurn))
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	stickyAnimationCheck(t, g, p, true)
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	stickyAnimationCheck(t, g, p, false)
}
