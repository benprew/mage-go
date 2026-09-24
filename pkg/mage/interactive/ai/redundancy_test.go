package ai

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestAbilityActivationIsRedundant_DoesNotSuppressAdditionalEffects(t *testing.T) {
	controller := mage.NewBasePlayer("Alice")
	opponent := mage.NewBasePlayer("Bob")
	g := mage.NewGame(controller, opponent)
	card := mage.NewCreature("Rogue", "{1}{U}", 2, 2, mage.WithKeyword(core.Flying))
	card.SetOwner(controller.PlayerID())
	target := g.PutOnBattlefield(card, controller.PlayerID())
	effects := []mage.Effect{
		mage.GrantKeyword(core.Flying),
		mage.Boost(mage.Fixed(1), mage.Fixed(0)),
	}

	if AbilityActivationIsRedundant(g, controller.PlayerID(), target.ID(), effects, []uuid.UUID{target.ID()}) {
		t.Fatal("keyword already present must not suppress an activation with an additional boost")
	}
}

func TestTypedKeywordGrantRedundancy(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	flyer := mage.NewCreature("Flyer", "", 2, 2, mage.WithKeyword(core.Flying))
	bear := mage.NewCreature("Bear", "", 2, 2)
	g.PutOnBattlefield(flyer, a.PlayerID())
	g.PutOnBattlefield(bear, a.PlayerID())
	first := dsl.TargetPermanent("first", dsl.IsCreature)
	second := dsl.TargetPermanent("second", dsl.IsCreature)
	body := dsl.Until(core.EndOfTurn, dsl.GrantKeyword(second, core.Flying))
	ability := dsl.Activate(dsl.Costs(), body, dsl.WithTargets(first, second))
	if !AbilityActivationIsRedundant(g, a.PlayerID(), bear.ID(), ability.Effects(), []uuid.UUID{bear.ID(), flyer.ID()}, ability.Targets()...) {
		t.Fatal("typed grant to a flyer is redundant")
	}
	if AbilityActivationIsRedundant(g, a.PlayerID(), bear.ID(), ability.Effects(), []uuid.UUID{flyer.ID(), bear.ID()}, ability.Targets()...) {
		t.Fatal("used the first target instead of the keyword recipient")
	}
	if AbilityActivationIsRedundant(g, a.PlayerID(), bear.ID(), ability.Effects(), []uuid.UUID{flyer.ID()}) {
		t.Fatal("missing target declarations must not suppress an action")
	}
	for _, extra := range []dsl.ContinuousModification{
		dsl.ModifyPT(second, dsl.Fixed(1), dsl.Fixed(0)),
		dsl.SetColors(second, core.Blue),
		dsl.GrantKeyword(second, core.Banding),
	} {
		mixed := []mage.Effect{dsl.Until(core.EndOfTurn, dsl.GrantKeyword(second, core.Flying), extra)}
		if AbilityActivationIsRedundant(g, a.PlayerID(), bear.ID(), mixed, []uuid.UUID{bear.ID(), flyer.ID()}, ability.Targets()...) {
			t.Fatal("keyword grant suppressed an additional modification")
		}
	}
}

func TestTypedKeywordGrantCandidatesAndPending(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	flyer := mage.NewCreature("Flyer", "", 2, 2, mage.WithKeyword(core.Flying))
	bear := mage.NewCreature("Bear", "", 2, 2)
	g.PutOnBattlefield(flyer, a.PlayerID())
	g.PutOnBattlefield(bear, a.PlayerID())
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	ability := dsl.Activate(dsl.Costs(), dsl.Until(core.EndOfTurn, dsl.GrantKeyword(target, core.Flying)), dsl.WithTargets(target))
	source := mage.NewArtifact("Source", "", mage.WithAbility(ability))
	g.PutOnBattlefield(source, a.PlayerID())
	candidates := []uuid.UUID{flyer.ID(), bear.ID()}
	possible := NonRedundantKeywordGrantTargets(g, a.PlayerID(), source.ID(), ability.Effects(), candidates, ability.Targets()...)
	if len(possible) != 1 || possible[0] != bear.ID() {
		t.Fatalf("candidates = %v, want the creature without flying", possible)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{bear.ID()}); err != nil {
		t.Fatal(err)
	}
	if !AbilityActivationIsRedundant(g, a.PlayerID(), source.ID(), ability.Effects(), []uuid.UUID{bear.ID()}, ability.Targets()...) {
		t.Fatal("pending typed grant was not recognized")
	}
	if g.FindPermanent(bear.ID()).HasKeyword(core.Flying) || len(g.StackObjects()) != 1 {
		t.Fatal("inspection changed game state")
	}
}
