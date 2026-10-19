package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONAttackGroup(t *testing.T) {
	for _, filter := range []string{`IsCardtype(Creature)`, `And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))])`} {
		t.Run(filter, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Enchantment", `TriggerA(WhenAnyNumberOfCreaturesAttack(`+filter+`),ActionList([EachPlayerAction(Trigger_ThoseDefendingPlayers,GainLife(Integer(2))),GainLife(Integer(1))]))`)
			g, a, b := ronGame()
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			g.PutOnBattlefieldAttacking(mage.NewCreature("Entered", "", 1, 8), a.PlayerID(), b.PlayerID())
			g.ExecuteAttackers(a.PlayerID(), nil)
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("entering attacking or empty declaration triggered")
			}
			ids := make([]uuid.UUID, 0, 2)
			for range 2 {
				p := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 8), a.PlayerID())
				p.RevokeBaseAttr(core.AttrSummonSick)
				ids = append(ids, p.ID())
			}
			g.ExecuteAttackers(a.PlayerID(), ids)
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("expected one declaration trigger", g.StackSize())
			}
			g.FireEvent(core.GameEvent{Type: core.EvtAttackersDeclared, PlayerID: b.PlayerID(), Amount: 2})
			g.PutTriggersOnStack()
			g.BouncePermanentToHand(source)
			clone := g.Clone()
			clone.ResolveStack()
			g.ResolveStack()
			wantA, wantB := 24, 22
			if strings.HasPrefix(filter, "And") {
				wantA = 21
			}
			for _, game := range []*mage.Game{g, clone} {
				if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB {
					t.Fatal("wrong saved defender or controller", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
				}
			}
		})
	}
}

func TestRONAttackGroupInvalid(t *testing.T) {
	for _, rule := range []string{
		`TriggerA(WhenAnyNumberOfCreaturesAttack(),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAnyNumberOfCreaturesAttack(IsCardtype(Artifact)),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAnyNumberOfCreaturesAttack(And([])),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAnyNumberOfCreaturesAttack(And([IsCardtype(Creature),HasAbility(Flying)])),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAnyNumberOfCreaturesAttack(IsCardtype(Creature)),ActionList([PlayerAction(Trigger_DefendingPlayer,GainLife(Integer(1)))]))`,
		`SpellActions(ActionList([EachPlayerAction(Trigger_ThoseDefendingPlayers,GainLife(Integer(1)))]))`,
		`TriggerA(WhenACreatureAttacks(SinglePermanent(ThisPermanent)),ActionList([EachPlayerAction(Trigger_ThoseDefendingPlayers,GainLife(Integer(1)))]))`,
		`TriggerA(WhenAnyNumberOfCreaturesAttack(IsCardtype(Creature)),ActionList([EachPlayerAction(Trigger_ThoseDefendingPlayers(),GainLife(Integer(1)))]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONAttackGroupRagingRiverBoundary(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		n := hostUntapRecord(t, set, "Raging River")
		if _, err := Compile(n); err != nil {
			t.Fatal("complete record must compile", err)
		}
		event := n.Fields["Rules"].Args[0].Args[0]
		body, err := Parse(strings.NewReader(`Card(Name:"Attack projection",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerA(WhenAnyNumberOfCreaturesAttack(IsCardtype(Creature)),ActionList([EachPlayerAction(Trigger_ThoseDefendingPlayers,GainLife(Integer(2)))]))])`), "projection.ron")
		if err != nil {
			t.Fatal(err)
		}
		body[0].Fields["Rules"].Args[0].Args[0] = event
		factory, err := Compile(body[0])
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		g.PutOnBattlefield(factory(), a.PlayerID())
		p := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 8), a.PlayerID())
		p.RevokeBaseAttr(core.AttrSummonSick)
		g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
		g.PutTriggersOnStack()
		g.ResolveStack()
		if b.Life() != 22 || a.Life() != 20 {
			t.Fatal("record event projection")
		}
	}
}

func TestRONAttackGroupInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerA(WhenAnyNumberOfCreaturesAttack(IsCardtype(Creature)),ActionList([EachPlayerAction(Trigger_ThoseDefendingPlayers,GainLife(Integer(1)))]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	foundFilter, foundSet := false, false
	for _, entry := range report.Signatures {
		if entry.Name == "IsCardtype" && entry.Role == "permanent_filter" {
			foundFilter = true
		}
		if entry.Name == "Trigger_ThoseDefendingPlayers" && entry.Role == "player_collection_reference" {
			foundSet = true
		}
	}
	if !foundFilter || !foundSet {
		t.Fatal("lost event or collection roles")
	}
}
