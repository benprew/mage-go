package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestRONReplacementActionDomain(t *testing.T) {
	for _, spellName := range []string{"Stone Rain", "Wrath of God"} {
		for _, prevention := range []string{"Fog", "Healing Salve", "Reverse Damage"} {
			t.Run(spellName+"/"+prevention, func(t *testing.T) {
				g, a, b := ronModalGame(1)
				g.SetActivePlayerIndex(0)
				var card mage.Card = mage.NewLand("Subject")
				if spellName == "Wrath of God" {
					card = mage.NewCreature("Subject", "", 3, 3)
				}
				card.SetOwner(b.PlayerID())
				p := g.PutOnBattlefield(card, b.PlayerID())
				var spell *mage.StackObject
				if spellName == "Stone Rain" {
					spell = castRONSpell(t, g, a.BasePlayer, repositoryCard(t, "LEA", spellName), p.ID())
				} else {
					spell = castRONSpell(t, g, a.BasePlayer, repositoryCard(t, "LEA", spellName))
				}
				if prevention == "Healing Salve" {
					castRONSpell(t, g, a.BasePlayer, repositoryCard(t, "LEA", prevention), b.PlayerID())
				} else {
					castRONSpell(t, g, a.BasePlayer, repositoryCard(t, "LEA", prevention))
				}
				g.ResolveTopOfStack()
				instructions, ok := mage.InspectDirectDestruction(spell.Effects[0])
				if !ok || len(instructions) != 1 {
					t.Fatal("missing destruction declaration")
				}
				instruction := instructions[0]
				var visits []mage.PermanentVisit
				if instruction.Subject != nil {
					query, ok := mage.InspectPermanentTarget(instruction.Subject)
					if !ok {
						t.Fatal("missing target declaration")
					}
					var known bool
					visits, known = query.Select(g, spell)
					if !known {
						t.Fatal("unknown target")
					}
				} else {
					query, ok := mage.InspectPermanentCollection(instruction.Subjects)
					if !ok {
						t.Fatal("missing collection declaration")
					}
					visits = query.Select(g)
				}
				if len(visits) != 1 || visits[0].ID != p.ID() {
					t.Fatal("lost subject identity")
				}
				for range 2 {
					if proceeds, known := g.InspectDestructionProposal(visits[0], instruction.CannotRegenerate); !known || !proceeds {
						t.Fatalf("damage-only %s obscured %s: proceeds %v known %v", prevention, spellName, proceeds, known)
					}
				}
				for _, game := range []*mage.Game{g.Clone(), g} {
					game.ResolveStack()
					if game.FindPermanent(p.ID()) != nil || len(game.GetPlayer(b.PlayerID()).Graveyard()) != 1 {
						t.Fatal("inspection disagrees with native resolution")
					}
				}
				if a.Life() != 20 || b.Life() != 20 {
					t.Fatal("inspection or destruction consumed prevention")
				}
			})
		}
	}
}

func TestRONReplacementActionDomainPlan(t *testing.T) {
	for _, action := range []string{`DestroyPermanent(ThisPermanent)`, `DestroyPermanentNoRegen(ThisPermanent)`, `DestroyEachPermanent(IsCardtype(Creature))`, `DestroyEachPermanentNoRegen(IsCardtype(Creature))`} {
		nodes, err := Parse(strings.NewReader(action), "plan")
		if err != nil {
			t.Fatal(err)
		}
		plan, err := compileDirectDestructionPlan(nodes[0], scope{permanent: true})
		if err != nil {
			t.Fatal(err)
		}
		for range 2 {
			instruction := plan.build(instanceBindings{})
			effects, ok := mage.InspectDirectDestruction(instruction.Effect())
			if !ok || len(effects) != 1 || effects[0].CannotRegenerate != instruction.CannotRegenerate {
				t.Fatal("plan lost semantics")
			}
		}
	}
	for _, action := range []string{`DestroyPermanent()`, `DestroyPermanent(You)`, `DestroyPermanent(WouldBeDestroyed_ThatPermanent)`, `DestroyEachPermanent(You)`, `DestroyPermanentNoRegen(ThisPermanent,ThisPermanent)`, `Unknown(ThisPermanent)`} {
		nodes, err := Parse(strings.NewReader(action), "plan")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileDirectDestructionPlan(nodes[0], scope{permanent: true}); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}
