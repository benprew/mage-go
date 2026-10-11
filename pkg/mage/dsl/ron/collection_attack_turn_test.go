package ron

import (
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCollectionAttackTurnFestival(t *testing.T) {
	for _, change := range []string{"normal", "copy", "illegal-cast", "own-upkeep", "countered", "phase", "return"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			g.SetActivePlayerIndex(1)
			g.SetStep(core.Upkeep)
			c := repositoryCard(t, "DRK", "Festival")
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			a.ManaPool().Add(core.White, 1)
			if change == "illegal-cast" {
				g.SetStep(core.PrecombatMain)
			}
			if change == "own-upkeep" {
				g.SetActivePlayerIndex(0)
			}
			err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0)
			if change == "illegal-cast" || change == "own-upkeep" {
				if err == nil || a.ManaPool().TotalMana() != 1 || len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("casting restriction or rollback")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			if change == "countered" {
				g.GetStack().RemoveByID(g.StackPeek().ID)
			}
			g.ResolveStack()
			late := mage.NewCreature("Late", "", 2, 3)
			late.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(late, b.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			if change == "phase" {
				g.PhaseIn(g.PhaseOut(p.ID()))
			}
			if change == "return" {
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				p = g.PutOnBattlefield(late, b.PlayerID())
				p.RevokeBaseAttr(core.AttrSummonSick)
			}
			if change == "countered" {
				if !p.CanDeclareAsAttacker(g) {
					t.Fatal("countered spell installed rule")
				}
				return
			}
			clone := g.Clone()
			g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{p.ID()})
			if g.GetCombat().IsAttacking(p.ID()) || g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("live collection restriction")
			}
			clone.RunStepWithPriority(core.Cleanup)
			if !clone.FindPermanent(p.ID()).CanDeclareAsAttacker(clone) || g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("cleanup or clone isolation")
			}
		})
	}
}

func TestRONCollectionAttackTurnSirenClause(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEB.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEB.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var card Node
	for _, n := range nodes {
		if n.Fields["Name"].String == "Siren's Call" {
			card = n
			break
		}
	}
	if _, err := Compile(card); err != nil {
		t.Fatalf("full Siren record: %v", err)
	}
	rules := card.Fields["Rules"].Args
	action := rules[1].Args[0].Args[0].Args[0]
	card.Fields["Name"] = Node{Kind: String, String: "Siren clause"}
	card.Fields["Rules"] = Node{Kind: List, Args: []Node{rules[1]}}
	card.Fields["Rules"].Args[0].Args[0].Args[0].Args = []Node{action}
	factory, err := Compile(card)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	g.SetActivePlayerIndex(1)
	c := factory()
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.PutOnBattlefield(mage.NewCreature("Late", "", 2, 3), b.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	g.RunStepWithPriority(core.DeclareAttackers)
	if !g.GetCombat().IsAttacking(p.ID()) {
		t.Fatal("active player requirement not applied")
	}
}

func TestRONCollectionAttackTurnInvalid(t *testing.T) {
	for _, action := range []string{
		`CreateEachPermanentRuleEffectUntil(IsCardtype(Creature),[],UntilEndOfTurn)`,
		`CreateEachPermanentRuleEffectUntil(IsCardtype(Creature),[CantAttack()],UntilEndOfTurn)`,
		`CreateEachPermanentRuleEffectUntil(IsCardtype(Creature),[CantAttack],UntilEndOfCombat)`,
		`CreateEachPermanentRuleEffectUntil(IsCardtype(Creature),[CantAttack],UntilEndOfTurn())`,
		`CreateEachPermanentRuleEffectUntil(IsCardtype(Creature),[Flying],UntilEndOfTurn)`,
		`CreateEachPermanentRuleEffectUntil(Ref_TargetPermanent,[CantAttack],UntilEndOfTurn)`,
		`CreateEachPermanentRuleEffectUntil(ControlledByPlayer(Ref_TargetPlayer),[MustAttack],UntilEndOfTurn)`,
		`CreateEachPermanentRuleEffectUntil(And([]),[CantAttack],UntilEndOfTurn)`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+action+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}

func TestRONCollectionAttackTurnInventory(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Inventory",Rules:[SpellActions(ActionList([CreateEachPermanentRuleEffectUntil(IsCardtype(Creature),[CantAttack],UntilEndOfTurn)]))])`), "inventory.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Inventory", "Card", "card", "collection", false, entries)
	for name, role := range map[string]string{"IsCardtype": "permanent_filter", "CantAttack": "permanent_rule_operation", "UntilEndOfTurn": "duration"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Errorf("%s missing %s role", name, role)
		}
	}
}
