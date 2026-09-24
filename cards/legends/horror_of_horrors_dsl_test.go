package legends

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestHorrorOfHorrorsTypedDefinition(t *testing.T) {
	card, err := mage.CreateCard("Horror of Horrors")
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	ability := mage.UnwrapAbility(card.Abilities()[0]).(mage.ActivatedAbility)
	if len(ability.Targets()) != 1 {
		t.Fatal("missing target")
	}
	if _, ok := ability.Targets()[0].(mage.TargetBinding); !ok {
		t.Fatal("missing typed target")
	}
	if !mage.IsRegenerationEffect(ability.Effects()[0]) || ability.Effects()[0].Properties().Outcome != mage.OutcomeBenefit {
		t.Fatal("missing regeneration metadata")
	}
}

func TestHorrorOfHorrorsPayment(t *testing.T) {
	for _, scenario := range []string{"opponent", "own", "tapped", "protected swamp", "nonbasic swamp", "source swamp", "sacrifice target", "no swamp", "opponent swamp", "phased swamp", "not swamp", "nonblack", "noncreature", "player", "shroud"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			card, err := mage.CreateCard("Horror of Horrors")
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "source swamp" {
				card.(*mage.BaseCard).AddType(core.TypeLand)
				card.(*mage.BaseCard).AddSubType("Swamp")
			}
			source := g.PutOnBattlefield(card, a.PlayerID()).ID()
			swamp := uuid.Nil
			if scenario == "source swamp" {
				swamp = source
			} else if scenario != "no swamp" {
				owner, name := gametest.PlayerA, "Swamp"
				if scenario == "opponent swamp" {
					owner = gametest.PlayerB
				}
				if scenario == "not swamp" {
					name = "Forest"
				}
				if scenario == "nonbasic swamp" {
					name = "Badlands"
				}
				if scenario == "sacrifice target" {
					swamp = g.PutOnBattlefield(mage.NewCreature("Creature Swamp", "{B}", 2, 2, mage.WithCardType(core.TypeLand), mage.WithSubTypes("Swamp")), a.PlayerID()).ID()
				} else {
					swamp = g.AddCard(core.ZoneBattlefield, owner, name)
				}
			}
			owner, name := gametest.PlayerB, "Scathe Zombies"
			if scenario == "own" {
				owner = gametest.PlayerA
			}
			if scenario == "nonblack" {
				name = "Grizzly Bears"
			}
			if scenario == "noncreature" {
				name = "Horror of Horrors"
			}
			target := g.AddCard(core.ZoneBattlefield, owner, name)
			if scenario == "sacrifice target" {
				target = swamp
			}
			if scenario == "player" {
				target = b.PlayerID()
			}
			if scenario == "shroud" {
				g.FindPermanent(target).GrantBaseAttr(core.Shroud)
			}
			if scenario == "phased swamp" {
				g.PhaseOut(swamp)
			}
			if scenario == "protected swamp" {
				g.FindPermanent(swamp).GrantBaseAttr(core.Indestructible)
				g.AddRegenerationShield(swamp)
			}
			if scenario == "tapped" {
				g.FindPermanent(source).Tapped = true
				g.FindPermanent(swamp).Tapped = true
			}
			g.SetStep(core.Upkeep)
			g.SetActivePlayerIndex(1)
			err = g.ActivateAbilityByIndex(a.PlayerID(), source, 0, []uuid.UUID{target})
			rejected := scenario == "no swamp" || scenario == "opponent swamp" || scenario == "phased swamp" || scenario == "not swamp" || scenario == "nonblack" || scenario == "noncreature" || scenario == "player" || scenario == "shroud"
			if rejected {
				if err == nil || g.StackSize() != 0 || len(a.Graveyard()) != 0 || len(b.Graveyard()) != 0 {
					t.Fatal("invalid activation changed payment state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(swamp) != nil || len(a.Graveyard()) != 1 || g.StackSize() != 1 || g.HasRegenerationShield(target) {
				t.Fatal("sacrifice was not paid before resolution")
			}
			g.ResolveStack()
			if g.HasRegenerationShield(target) != (scenario != "sacrifice target") {
				t.Fatal("incorrect shield or target selection after payment")
			}
			if scenario != "sacrifice target" {
				g.DestroyPermanent(g.FindPermanent(target))
				if g.FindPermanent(target) == nil || !g.FindPermanent(target).Tapped || g.HasRegenerationShield(target) {
					t.Fatal("shield did not replace destruction")
				}
			}
		})
	}
}

func TestHorrorOfHorrorsLifetimeAndCopies(t *testing.T) {
	for _, scenario := range []string{"source exit", "returned", "phased", "phase cycle", "shroud", "color loss", "control", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			source := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Horror of Horrors")
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
			target := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Scathe Zombies")
			card := g.FindPermanent(target).Card
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source, 0, []uuid.UUID{target}); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "source exit":
				g.RemoveFromBattlefield(g.FindPermanent(source))
			case "returned":
				g.RemoveFromBattlefield(g.FindPermanent(target))
				g.PutOnBattlefield(card, b.PlayerID())
			case "phased":
				g.PhaseOut(target)
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(target))
			case "shroud":
				g.FindPermanent(target).GrantBaseAttr(core.Shroud)
			case "color loss":
				card.(*mage.BaseCard).SetColorOverride([]core.Color{core.Green})
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source, TargetID: target, ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "copy":
				second := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Drudge Skeletons")
				g.ChooseTarget(gametest.PlayerB, "Drudge Skeletons")
				cp := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
				if cp == nil || len(cp.Targets) != 1 || cp.Targets[0] != second {
					t.Fatal("copy lost separate target")
				}
				clone := g.Clone()
				clone.ResolveStack()
				if !clone.HasRegenerationShield(target) || !clone.HasRegenerationShield(second) || g.HasRegenerationShield(target) || g.HasRegenerationShield(second) || g.StackSize() != 2 {
					t.Fatal("clone changed original or lost targets")
				}
				g.ResolveTopOfStack()
				if !g.HasRegenerationShield(second) || g.HasRegenerationShield(target) || len(b.Graveyard()) != 0 {
					t.Fatal("copy paid costs or resolved wrong target")
				}
			}
			g.ResolveStack()
			want := scenario == "source exit" || scenario == "phase cycle" || scenario == "control" || scenario == "copy"
			if g.HasRegenerationShield(target) != want {
				t.Fatal("incorrect target legality or lifetime")
			}
			g.AssertGraveyardCount(gametest.PlayerA, "Swamp", 1)
		})
	}
}
