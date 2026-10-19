package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDestructionProposal(t *testing.T) {
	for _, state := range []string{"land", "pyramids", "pyramids source exit", "pyramids phase", "pyramids return", "clergy", "clergy regeneration", "clergy prohibited", "death ward"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronModalGame(0)
			g.SetActivePlayerIndex(0)
			var subjectCard mage.Card = mage.NewLand("Subject")
			spellName := "Stone Rain"
			if state == "clergy" || state == "clergy regeneration" || state == "clergy prohibited" || state == "death ward" {
				subjectCard = repositoryCard(t, "LEG", "Clergy of the Holy Nimbus")
				if state == "death ward" {
					subjectCard = mage.NewCreature("Subject", "", 3, 3)
				}
				spellName = "Wrath of God"
			}
			subjectCard.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(subjectCard, b.PlayerID())
			want := true
			switch state {
			case "pyramids", "pyramids source exit", "pyramids phase", "pyramids return":
				source := g.PutOnBattlefield(repositoryCard(t, "ARN", "Pyramids"), a.PlayerID())
				a.ManaPool().Add(core.Colorless, 2)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				want = false
				switch state {
				case "pyramids source exit":
					g.RemoveFromBattlefield(source)
				case "pyramids phase":
					g.PhaseOut(p.ID())
					g.PhaseIn([]uuid.UUID{p.ID()})
				case "pyramids return":
					g.RemoveFromBattlefield(p)
					p = g.PutOnBattlefield(subjectCard, b.PlayerID())
					want = true
				}
			case "clergy regeneration":
				want = false
			case "clergy prohibited":
				a.ManaPool().Add(core.Colorless, 1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
			case "death ward":
				castRONSpell(t, g, a.BasePlayer, repositoryCard(t, "LEA", "Death Ward"), p.ID())
				g.ResolveStack()
			}
			var obj *mage.StackObject
			if spellName == "Stone Rain" {
				obj = castRONSpell(t, g, a.BasePlayer, repositoryCard(t, "LEA", spellName), p.ID())
			} else if state == "clergy regeneration" {
				factory := permanentScopeFactory(t, "Sorcery", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`)
				obj = castRONSpell(t, g, a.BasePlayer, factory(), p.ID())
			} else {
				obj = castRONSpell(t, g, a.BasePlayer, repositoryCard(t, "LEA", spellName))
			}
			instructions, ok := mage.InspectDirectDestruction(obj.Effects[0])
			if !ok || len(instructions) != 1 {
				t.Fatal("missing typed destruction")
			}
			instruction := instructions[0]
			var visits []mage.PermanentVisit
			if instruction.Subject != nil {
				query, ok := mage.InspectPermanentTarget(instruction.Subject)
				if !ok {
					t.Fatal("missing target plan")
				}
				var known bool
				visits, known = query.Select(g, obj)
				if !known {
					t.Fatal("unknown target")
				}
			} else {
				query, ok := mage.InspectPermanentCollection(instruction.Subjects)
				if !ok {
					t.Fatal("missing collection plan")
				}
				visits = query.Select(g)
			}
			if len(visits) != 1 || visits[0].ID != p.ID() {
				t.Fatalf("unexpected visits %v", visits)
			}
			before := g.StackSize()
			for range 2 {
				proceeds, known := g.InspectDestructionProposal(visits[0], instruction.CannotRegenerate)
				if !known || proceeds != want {
					t.Fatalf("proceeds %v known %v want %v", proceeds, known, want)
				}
			}
			if g.StackSize() != before {
				t.Fatal("query changed stack")
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{clone, g} {
				game.ResolveStack()
				if (game.FindPermanent(p.ID()) == nil) != want {
					t.Fatal("query disagrees with RON gameplay")
				}
			}
		})
	}
}

func TestRONDestructionProposalInvalid(t *testing.T) {
	for _, action := range []string{`DestroyPermanent()`, `DestroyPermanentNoRegen(ThisPermanent,ThisPermanent)`, `DestroyEachPermanentNoRegen(You)`, `DestroyPermanent(WouldBeDestroyed_ThatPermanent)`} {
		node := nameScopeNode(t, `Card(Name:"Invalid proposal",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+action+`]))])`)
		if _, err := Compile(node); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}
