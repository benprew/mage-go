package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDeclarationTargetQueryGameplay(t *testing.T) {
	for _, name := range []string{"Stone Rain", "Ice Storm", "Sinkhole", "Shatter"} {
		for _, state := range []string{"active", "removed", "phase", "phase-cycle", "return", "copy"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "LEG", "Wall of Shadows")
				victim := g.PutOnBattlefield(card, b.PlayerID())
				kind := core.TypeLand
				if name == "Shatter" {
					kind = core.TypeArtifact
				}
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.AddCardType(mage.This, kind)), victim.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				obj := castRONSpell(t, g, a, repositoryCard(t, "LEA", name), victim.ID())
				want := true
				switch state {
				case "removed":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), victim.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "phase", "phase-cycle":
					g.PhaseOut(victim.ID())
					want = false
					if state == "phase-cycle" {
						g.PhaseIn([]uuid.UUID{victim.ID()})
						want = true
					}
				case "return":
					g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
					g.PutOnBattlefield(card, b.PlayerID())
					want = false
				case "copy":
					obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
					if obj == nil {
						t.Fatal("copy failed")
					}
				}
				instructions, ok := mage.InspectDirectDestruction(obj.Effects[0])
				if !ok {
					t.Fatal("missing destruction")
				}
				plan, ok := mage.InspectPermanentTarget(instructions[0].Subject)
				if !ok {
					t.Fatal("missing target")
				}
				for _, game := range []*mage.Game{g, g.Clone()} {
					got, known := plan.Select(game, game.StackPeek())
					if !known || (len(got) == 1) != want {
						t.Fatalf("got %v known %v want %v", got, known, want)
					}
					game.ResolveStack()
					if (game.FindPermanentIncludingPhased(victim.ID()) == nil) != want {
						t.Fatal("native resolution differs")
					}
				}
			})
		}
	}
}

func TestRONDeclarationTargetQueryInvalid(t *testing.T) {
	for _, syntax := range []string{`Other(IsCreatureType(Wall))`, `CanTargetOnly()`, `CanTargetOnly(IsCreatureType(Wall()))`, `CanTargetOnly(IsCreatureType("Wall"))`, `CanTargetOnly(IsCardtype(Creature))`} {
		if _, err := compileTargetOnlySubtypeFilter(nameScopeNode(t, syntax)); err == nil {
			t.Fatalf("accepted %s", syntax)
		}
	}
}

func TestRONDeclarationTargetQueryAbilityGameplay(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phased", "copy", "control"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			victim := g.PutOnBattlefield(repositoryCard(t, "LEG", "Wall of Shadows"), b.PlayerID())
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(0)]),Targeted([TargetPermanent(AnyPermanent)],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`)
			card := factory()
			source := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			obj := g.StackPeek()
			switch state {
			case "departed", "returned":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if state == "returned" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phased":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: obj.ID, TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
			}
			if obj == nil {
				t.Fatal("missing stack object")
			}
			instructions, ok := mage.InspectDirectDestruction(obj.Effects[0])
			if !ok {
				t.Fatal("missing destruction")
			}
			plan, ok := mage.InspectPermanentTarget(instructions[0].Subject)
			if !ok {
				t.Fatal("missing target")
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				got, known := plan.Select(game, game.StackPeek())
				if !known || len(got) != 1 || got[0].ID != victim.ID() {
					t.Fatalf("got %v known %v", got, known)
				}
				game.ResolveStack()
				if game.FindPermanent(victim.ID()) != nil {
					t.Fatal("native ability resolution differs")
				}
			}
		})
	}
}

func TestRONDeclarationTargetQueryLowering(t *testing.T) {
	for _, tc := range []struct {
		syntax  string
		blocked bool
	}{
		{`CanTargetOnly(IsCreatureType(Wall))`, false},
		{`And([IsColor(Red),CanTargetOnly(IsCreatureType(Wall))])`, false},
		{`Or([IsColor(Red),CanTargetOnly(IsCreatureType(Wall))])`, true},
	} {
		t.Run(tc.syntax, func(t *testing.T) {
			g, p, obj, plan := ronTargetQueryFixture(t)
			filter, err := compileSpellCardFilter(nameScopeNode(t, tc.syntax), true)
			if err != nil {
				t.Fatal(err)
			}
			aura := g.PutOnBattlefield(mage.NewAura("Rule", "", mage.WithCastTarget(mage.TargetLand()), mage.WithAbility(mage.Static(mage.CantBeTargetedBySpells(mage.AttachedPermanent, filter)))), obj.Controller)
			g.Attach(aura.ID(), p.ID())
			got, known := plan.Select(g, obj)
			if !known || (len(got) == 0) != tc.blocked {
				t.Fatalf("got %v known %v blocked %v", got, known, tc.blocked)
			}
			if _, err := compileSpellCardFilter(nameScopeNode(t, tc.syntax), false); err == nil {
				t.Fatal("declaration predicate accepted in characteristic context")
			}
		})
	}
}
