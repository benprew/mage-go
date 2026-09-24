package legends

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestGwendlynTypedDefinition(t *testing.T) {
	card, err := mage.CreateCard("Gwendlyn Di Corci")
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	var target, discard, timing bool
	var visit func(mage.DefinitionMetadata)
	visit = func(n mage.DefinitionMetadata) {
		target = target || n.Role == "target-binding"
		discard = discard || n.Text == "discard at random for player" && len(n.Children) == 2 && n.Properties.Outcome == mage.OutcomeDetriment
		timing = timing || n.Role == "activation-timing"
		for _, child := range n.Children {
			visit(child)
		}
	}
	for _, ability := range card.Abilities() {
		visit(mage.InspectDefinition(ability))
	}
	if !target || !discard || !timing {
		t.Fatalf("typed target=%v, discard=%v, timing=%v", target, discard, timing)
	}
}

func TestGwendlynTypedActivation(t *testing.T) {
	for _, scenario := range []string{"opponent", "self", "empty", "upkeep", "source control", "opponent turn", "tapped", "summoning sick", "permanent", "phased"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			source := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gwendlyn Di Corci")
			g.FindPermanent(source).RevokeBaseAttr(core.AttrSummonSick)
			recipient, target := gametest.PlayerB, b.PlayerID()
			if scenario == "self" {
				recipient, target = gametest.PlayerA, a.PlayerID()
			}
			if scenario != "empty" {
				g.AddCard(core.ZoneHand, recipient, "Forest")
				g.AddCard(core.ZoneHand, recipient, "Mountain")
			}
			g.SetStep(core.PrecombatMain)
			switch scenario {
			case "upkeep":
				g.SetStep(core.Upkeep)
			case "opponent turn":
				g.SetActivePlayerIndex(1)
			case "tapped":
				g.FindPermanent(source).Tapped = true
			case "summoning sick":
				g.FindPermanent(source).GrantBaseAttr(core.AttrSummonSick)
			case "permanent":
				target = source
			case "phased":
				g.PhaseOut(source)
			}
			g.SetRandomResults([]int{1, 7})
			err := g.ActivateAbilityByIndex(a.PlayerID(), source, 0, []uuid.UUID{target})
			success := scenario == "opponent" || scenario == "self" || scenario == "empty" || scenario == "upkeep" || scenario == "source control"
			if !success {
				if scenario == "phased" {
					g.PhaseIn([]uuid.UUID{source})
				}
				if err == nil || g.StackSize() != 0 || g.FindPermanent(source).Tapped != (scenario == "tapped") {
					t.Fatal("failed activation changed payment or stack")
				}
				g.AssertHandCount(recipient, "Forest", 1)
				g.AssertHandCount(recipient, "Mountain", 1)
				if g.RandIntn(10) != 1 {
					t.Fatal("failed activation consumed randomness")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(source).Tapped || g.StackSize() != 1 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("incorrect payment or stack")
			}
			if scenario != "empty" {
				g.AssertHandCount(recipient, "Mountain", 1)
			}
			if scenario == "source control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source, TargetID: source, ControllerID: b.PlayerID(), Duration: core.Indefinite})
			} else {
				g.RemoveFromBattlefield(g.FindPermanent(source))
			}
			g.ResolveStack()
			next := 1
			if scenario != "empty" {
				g.AssertHandCount(recipient, "Forest", 1)
				g.AssertGraveyardCount(recipient, "Mountain", 1)
				next = 7
			}
			if g.RandIntn(10) != next {
				t.Fatal("incorrect random queue consumption")
			}
		})
	}
}

func TestGwendlynTypedCopyAndClone(t *testing.T) {
	g := gametest.NewTestGame(t)
	a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
	source := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gwendlyn Di Corci")
	g.FindPermanent(source).RevokeBaseAttr(core.AttrSummonSick)
	for _, player := range []gametest.PlayerRef{gametest.PlayerA, gametest.PlayerB} {
		g.AddCard(core.ZoneHand, player, "Forest")
		g.AddCard(core.ZoneHand, player, "Mountain")
	}
	g.SetStep(core.Upkeep)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source, 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	g.ChooseTarget(gametest.PlayerB, "PlayerA")
	cp := g.CopyStackObjectDirect(original, b.PlayerID(), true)
	if cp == nil || len(cp.Targets) != 1 || cp.Targets[0] != a.PlayerID() || original.Targets[0] != b.PlayerID() {
		t.Fatal("copy lost independent targets")
	}
	g.SetRandomResults([]int{1, 0, 7, 9})
	clone := g.Clone()
	clone.ResolveTopOfStack()
	if len(a.Hand()) != 2 || len(b.Hand()) != 2 || len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 {
		t.Fatal("clone changed original hands")
	}
	g.ResolveTopOfStack()
	g.AssertGraveyardCount(gametest.PlayerA, "Mountain", 1)
	g.AssertHandCount(gametest.PlayerB, "Mountain", 1)
	g.FindPermanent(source).Tapped = false
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source, 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatalf("activation with nonempty stack: %v", err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source))
	g.ResolveTopOfStack()
	g.AssertGraveyardCount(gametest.PlayerB, "Forest", 1)
	g.ResolveStack()
	g.AssertGraveyardCount(gametest.PlayerB, "Mountain", 1)
	g.AssertHandCount(gametest.PlayerA, "Forest", 1)
	if clone.RandIntn(10) != 0 || g.RandIntn(10) != 9 {
		t.Fatal("incorrect clone random queue")
	}
}

func TestGwendlynReplacementAndDiscardEvents(t *testing.T) {
	g := gametest.NewTestGame(t)
	a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Library of Leng")
	source := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gwendlyn Di Corci")
	g.FindPermanent(source).RevokeBaseAttr(core.AttrSummonSick)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Mountain")
	chosen := b.Hand()[0].ID()
	events := 0
	watcher := mage.NewTriggered(core.EvtDiscard, false, mage.DrawFor(mage.You, mage.Fixed(1))).SetCondition(func(evt *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
		events++
		if evt.PlayerID != b.PlayerID() || evt.SourceID != chosen {
			t.Error("incorrect discard event recipient or card")
		}
		return false
	})
	g.PutOnBattlefield(mage.NewArtifact("Discard observer", "", mage.WithAbility(watcher)), a.PlayerID())
	g.SetStep(core.PrecombatMain)
	g.SetRandomResults([]int{1, 7})
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source, 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.AssertHandCount(gametest.PlayerB, "Mountain", 0)
	g.AssertGraveyardCount(gametest.PlayerB, "Mountain", 0)
	if len(b.Library()) != 1 || b.Library()[0].ID() != chosen || events != 1 || g.RandIntn(10) != 7 {
		t.Fatal("random discard lost replacement, event, or queue order")
	}
}
