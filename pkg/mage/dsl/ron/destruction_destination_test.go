package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDestructionDestinationPlan(t *testing.T) {
	action := `CreateReplaceWouldPutIntoGraveyardUntil(APermanentWouldDie(SinglePermanent(ThisPermanent)),[ExileItInstead],UntilEndOfTurn)`
	nodes, err := Parse(strings.NewReader(action), "plan.ron")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := compileTemporaryDeathExilePlan(nodes[0], scope{permanent: true})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(mage.NewCreature("Subject", "", 3, 3), a.PlayerID())
		if err := mage.ApplyEffect(g, plan.build(instanceBindings{}), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.DestroyPermanent(p)
		if g.FindExiledCard(p.ID()) == nil {
			t.Fatal("plan lost subject")
		}
	}
	for _, bad := range []string{
		strings.Replace(action, "SinglePermanent(ThisPermanent)", "SinglePermanent(Ref_TargetPermanent)", 1),
		strings.Replace(action, "APermanentWouldDie(", "Unknown(", 1),
		strings.Replace(action, "ExileItInstead]", "ExileItInstead()]", 1),
		strings.Replace(action, "UntilEndOfTurn", "UntilEndOfCombat", 1),
	} {
		nodes, err := Parse(strings.NewReader(bad), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileTemporaryDeathExilePlan(nodes[0], scope{permanent: true}); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}

func TestRONDestructionDestination(t *testing.T) {
	for _, spellName := range []string{"Stone Rain", "Wrath of God"} {
		for _, replacement := range []string{"Disintegrate", "Firestorm Phoenix"} {
			t.Run(spellName+"/"+replacement, func(t *testing.T) {
				g, a, b := ronGame()
				g.SetActivePlayerIndex(0)
				card := mage.Card(mage.NewCreature("Subject", "", 3, 4))
				if replacement == "Firestorm Phoenix" {
					card = repositoryCard(t, "LEG", replacement)
				}
				card.SetOwner(b.PlayerID())
				p := g.PutOnBattlefield(card, b.PlayerID())
				if replacement == "Disintegrate" {
					spell := repositoryCard(t, "LEA", replacement)
					a.AddToHand(spell)
					a.ManaPool().Add(core.Red, 1)
					if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
				}
				if spellName == "Stone Rain" {
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.AddCardType(mage.This, core.TypeLand)), p.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				var spell *mage.StackObject
				if spellName == "Stone Rain" {
					spell = castRONSpell(t, g, a, repositoryCard(t, "LEA", spellName), p.ID())
				} else {
					spell = castRONSpell(t, g, a, repositoryCard(t, "LEA", spellName))
				}
				instructions, ok := mage.InspectDirectDestruction(spell.Effects[0])
				if !ok || len(instructions) != 1 {
					t.Fatal("missing destruction plan")
				}
				var visits []mage.PermanentVisit
				if instructions[0].Subject != nil {
					query, ok := mage.InspectPermanentTarget(instructions[0].Subject)
					if !ok {
						t.Fatal("target plan")
					}
					var known bool
					visits, known = query.Select(g, spell)
					if !known {
						t.Fatal("unknown target")
					}
				} else {
					query, ok := mage.InspectPermanentCollection(instructions[0].Subjects)
					if !ok {
						t.Fatal("collection plan")
					}
					visits = query.Select(g)
				}
				if len(visits) != 1 {
					t.Fatal("lost visit")
				}
				want := mage.DestructionToExile
				if replacement == "Firestorm Phoenix" {
					want = mage.DestructionToHand
				}
				for _, game := range []*mage.Game{g.Clone(), g} {
					for range 2 {
						if got, known := game.InspectDestructionDestinations(visits[0], instructions[0].CannotRegenerate); !known || got != want {
							t.Fatalf("destination %v known %v", got, known)
						}
					}
					game.ResolveStack()
					if game.FindPermanent(p.ID()) != nil || len(game.GetPlayer(b.PlayerID()).Graveyard()) != 0 || game.CreatureDeaths() != 0 {
						t.Fatal("native resolution differs")
					}
					if replacement == "Disintegrate" && game.FindExiledCard(card.ID()) == nil {
						t.Fatal("missing exile")
					}
					if replacement == "Firestorm Phoenix" && !game.IsHandCardRevealed(b.PlayerID(), card.ID()) {
						t.Fatal("missing hand lock")
					}
				}
			})
		}
	}
}
