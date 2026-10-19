package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronPartitionPlayer struct {
	mage.Player
	calls   int
	reasons []string
}

func (p *ronPartitionPlayer) ChooseMode(_ []string, reason string) int {
	p.calls++
	p.reasons = append(p.reasons, reason)
	return p.calls % 2
}

const ronPartitionAction = `EachPlayerAction(Trigger_ThoseDefendingPlayers,SeparatePermanentsIntoNamedPiles(And([IsCardtype(Creature),DoesntHaveAbility(Flying),ControlledByAPlayer(SinglePlayer(EachablePlayer))]),["Left","Right"]))`

func TestRONNamedPermanentPartition(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := hostUntapRecord(t, set, "Raging River")
			if _, err := Compile(record); err != nil {
				t.Fatal("complete record must compile", err)
			}
			name := record.Fields["Name"]
			name.String = "Partition projection"
			record.Fields["Name"] = name
			actions := record.Fields["Rules"].Args[0].Args[1].Args[0]
			actions.Args = actions.Args[:1]
			record.Fields["Rules"].Args[0].Args[1].Args[0] = actions
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			chooser := &ronPartitionPlayer{Player: b}
			g.SetPlayerAt(1, chooser)
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 8), a.PlayerID())
			attacker.RevokeBaseAttr(core.AttrSummonSick)
			g.PutOnBattlefield(mage.NewCreature("Ground one", "", 1, 4), b.PlayerID())
			g.PutOnBattlefield(mage.NewCreature("Ground two", "", 1, 4), b.PlayerID())
			g.PutOnBattlefield(mage.NewCreature("Flying", "", 1, 4, mage.WithKeyword(core.Flying)), b.PlayerID())
			phased := g.PutOnBattlefield(mage.NewCreature("Phased", "", 1, 4), b.PlayerID())
			g.PhaseOut(phased.ID())
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
			g.PutTriggersOnStack()
			if chooser.calls != 0 {
				t.Fatal("choice made before resolution")
			}
			g.BouncePermanentToHand(source)
			clone := g.Clone()
			cloneChooser := &ronPartitionPlayer{Player: clone.GetPlayer(b.PlayerID())}
			clone.SetPlayerAt(1, cloneChooser)
			clone.ResolveStack()
			if chooser.calls != 0 || cloneChooser.calls != 2 {
				t.Fatal("clone choice isolation", chooser.calls, cloneChooser.calls)
			}
			g.ResolveStack()
			if chooser.calls != 2 {
				t.Fatal("wrong chooser or membership", chooser.calls)
			}
			if !strings.Contains(chooser.reasons[1], "Ground one") {
				t.Fatal("prior choices missing", chooser.reasons)
			}
		})
	}
}

func TestRONNamedPermanentPartitionInvalid(t *testing.T) {
	for _, action := range []string{
		`SeparatePermanentsIntoNamedPiles`, `SeparatePermanentsIntoNamedPiles()`,
		`SeparatePermanentsIntoNamedPiles(AnyPermanent)`,
		`SeparatePermanentsIntoNamedPiles(AnyPermanent,[])`,
		`SeparatePermanentsIntoNamedPiles(AnyPermanent,["Left"])`,
		`SeparatePermanentsIntoNamedPiles(AnyPermanent,["Left",""])`,
		`SeparatePermanentsIntoNamedPiles(AnyPermanent,["Left","Left"])`,
		`SeparatePermanentsIntoNamedPiles(AnyPermanent,["Left",Right])`,
		`SeparatePermanentsIntoNamedPiles(AnyPermanent,["Left","Right"],AnyPermanent)`,
		`SeparatePermanentsIntoNamedPiles(EachablePlayer,["Left","Right"])`,
	} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerA(WhenAnyNumberOfCreaturesAttack(IsCardtype(Creature)),ActionList([EachPlayerAction(Trigger_ThoseDefendingPlayers,` + action + `)]))])`
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", action)
		}
	}
	for _, rule := range []string{
		`SpellActions(ActionList([` + ronPartitionAction + `]))`,
		`TriggerA(WhenAnyNumberOfCreaturesAttack(IsCardtype(Creature)),ActionList([` + ronPartitionAction + `,PlayerAction(EachablePlayer,GainLife(Integer(1)))]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[`+rule+`])`), "scope.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("scope leak", rule)
		}
	}
}

func TestRONNamedPermanentPartitionInventory(t *testing.T) {
	source := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerA(WhenAnyNumberOfCreaturesAttack(IsCardtype(Creature)),ActionList([` + ronPartitionAction + `]))])`
	report, err := InspectFS(fstest.MapFS{"inventory.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range report.Signatures {
		if entry.Name == "DoesntHaveAbility" && entry.Role == "permanent_filter" {
			found = true
		}
	}
	if !found {
		t.Fatal("partition lost its permanent filter role")
	}
}
