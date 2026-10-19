package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const ronPileChoice = `ChooseNamedPileForPermanent(["Left","Right"],EachablePermanent)`
const ronPileRead = `InThePileChosenForPermanent(EachablePermanent)`

func pileSelectionSource(actions string) string {
	return `Card(Name:"Pile selection projection",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerA(WhenAnyNumberOfCreaturesAttack(IsCardtype(Creature)),ActionList([` + actions + `]))])`
}

func TestRONNamedPileSelection(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := hostUntapRecord(t, set, "Raging River")
			if _, err := Compile(record); err != nil {
				t.Fatal("complete record must compile", err)
			}
			name := record.Fields["Name"]
			name.String = "Pile selection projection"
			record.Fields["Name"] = name
			loop := record.Fields["Rules"].Args[0].Args[1].Args[0].Args[1]
			predicate := loop.Args[1].Args[1].Args[1].Args[0].Args[0]
			loop.Args[1].Args[1] = Node{Kind: Tuple, Name: "TapEachPermanent", Args: []Node{predicate}}
			record.Fields["Rules"].Args[0].Args[1].Args[0].Args[1] = loop
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			attackerChooser := &ronPartitionPlayer{Player: a}
			defenderChooser := &ronPartitionPlayer{Player: b}
			g.SetPlayerAt(0, attackerChooser)
			g.SetPlayerAt(1, defenderChooser)
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 8), a.PlayerID())
			attacker.RevokeBaseAttr(core.AttrSummonSick)
			right := g.PutOnBattlefield(mage.NewCreature("Right defender", "", 1, 4), b.PlayerID())
			left := g.PutOnBattlefield(mage.NewCreature("Left defender", "", 1, 4), b.PlayerID())
			flying := g.PutOnBattlefield(mage.NewCreature("Flying", "", 1, 4, mage.WithKeyword(core.Flying)), b.PlayerID())
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
			g.PutTriggersOnStack()
			if attackerChooser.calls+defenderChooser.calls != 0 {
				t.Fatal("choice before resolution")
			}
			g.BouncePermanentToHand(source)
			clone := g.Clone()
			ca := &ronPartitionPlayer{Player: clone.GetPlayer(a.PlayerID())}
			cb := &ronPartitionPlayer{Player: clone.GetPlayer(b.PlayerID())}
			clone.SetPlayerAt(0, ca)
			clone.SetPlayerAt(1, cb)
			clone.ResolveStack()
			if ca.calls != 1 || cb.calls != 2 || attackerChooser.calls+defenderChooser.calls != 0 {
				t.Fatal("choice controller or clone isolation")
			}
			g.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if !game.FindPermanent(right.ID()).Tapped || game.FindPermanent(left.ID()).Tapped || !game.FindPermanent(flying.ID()).Tapped {
					t.Fatal("scoped reader selected wrong pile")
				}
			}
		})
	}
}

func TestRONNamedPileSelectionInvalid(t *testing.T) {
	loop := func(body string) string { return `ActionForEachPermanent(IsCardtype(Creature),[` + body + `])` }
	for _, actions := range []string{
		loop(ronPileChoice),
		ronPartitionAction + `,` + ronPileChoice,
		ronPartitionAction + `,` + loop(`TapEachPermanent(`+ronPileRead+`),`+ronPileChoice),
		ronPartitionAction + `,` + loop(ronPileChoice) + `,TapEachPermanent(` + ronPileRead + `)`,
		`If(PermanentPassesFilter(ThisPermanent,IsAttacking),[` + ronPartitionAction + `]),` + loop(ronPileChoice),
		ronPartitionAction + `,` + loop(ronPileChoice+`,`+loop(`TapEachPermanent(`+ronPileRead+`)`)),
	} {
		assertPileSelectionRejected(t, actions)
	}
	for _, choice := range []string{
		`ChooseNamedPileForPermanent`, `ChooseNamedPileForPermanent()`,
		`ChooseNamedPileForPermanent(["Left","Right"])`,
		`ChooseNamedPileForPermanent(["Left","Right"],EachablePermanent,You)`,
		`ChooseNamedPileForPermanent([],EachablePermanent)`,
		`ChooseNamedPileForPermanent(["Left"],EachablePermanent)`,
		`ChooseNamedPileForPermanent(["Left","Left"],EachablePermanent)`,
		`ChooseNamedPileForPermanent(["Left","Absent"],EachablePermanent)`,
		`ChooseNamedPileForPermanent(["Left",Right],EachablePermanent)`,
		`ChooseNamedPileForPermanent(["Left","Right"],EachablePermanent())`,
		`ChooseNamedPileForPermanent(["Left","Right"],ThisPermanent)`,
	} {
		assertPileSelectionRejected(t, ronPartitionAction+`,`+loop(choice))
	}
	for _, read := range []string{`InThePileChosenForPermanent`, `InThePileChosenForPermanent()`, `InThePileChosenForPermanent(ThisPermanent)`, `InThePileChosenForPermanent(EachablePermanent())`, `InThePileChosenForPermanent(EachablePermanent,You)`} {
		assertPileSelectionRejected(t, ronPartitionAction+`,`+loop(ronPileChoice+`,TapEachPermanent(`+read+`)`))
	}
}

func assertPileSelectionRejected(t *testing.T, actions string) {
	t.Helper()
	nodes, err := Parse(strings.NewReader(pileSelectionSource(actions)), "invalid-pile.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Compile(nodes[0]); err == nil {
		t.Fatal("accepted", actions)
	}
}

func TestRONNamedPileSelectionInventory(t *testing.T) {
	source := pileSelectionSource(ronPartitionAction + `,ActionForEachPermanent(IsCardtype(Creature),[` + ronPileChoice + `,TapEachPermanent(` + ronPileRead + `)])`)
	report, err := InspectFS(fstest.MapFS{"pile.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	foundChoice, foundReader := false, false
	for _, s := range report.Signatures {
		if s.Name == "EachablePermanent" && s.Role == "permanent_reference" && strings.Contains(s.Context, "ChooseNamedPileForPermanent") {
			foundChoice = true
		}
		if s.Name == "EachablePermanent" && s.Role == "permanent_reference" && strings.Contains(s.Context, "InThePileChosenForPermanent") {
			foundReader = true
		}
	}
	if !foundChoice || !foundReader {
		t.Fatal("pile choice lost permanent reference role")
	}
}

func TestRONNamedPileSelectionLoopFrames(t *testing.T) {
	record := hostUntapRecord(t, "LEA", "Raging River")
	name := record.Fields["Name"]
	name.String = "Delayed pile projection"
	record.Fields["Name"] = name
	nodes, err := Parse(strings.NewReader(`CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([TapEachPermanent(`+ronPileRead+`)]))`), "delayed-pile.ron")
	if err != nil {
		t.Fatal(err)
	}
	loop := record.Fields["Rules"].Args[0].Args[1].Args[0].Args[1]
	loop.Args[1].Args[1] = nodes[0]
	record.Fields["Rules"].Args[0].Args[1].Args[0].Args[1] = loop
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	ca, cb := &ronPartitionPlayer{Player: a}, &ronPartitionPlayer{Player: b}
	g.SetPlayerAt(0, ca)
	g.SetPlayerAt(1, cb)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	attackers := make([]uuid.UUID, 0, 2)
	for _, label := range []string{"One", "Two"} {
		p := g.PutOnBattlefield(mage.NewCreature(label, "", 1, 8), a.PlayerID())
		p.RevokeBaseAttr(core.AttrSummonSick)
		attackers = append(attackers, p.ID())
	}
	right := g.PutOnBattlefield(mage.NewCreature("Right", "", 1, 4), b.PlayerID())
	left := g.PutOnBattlefield(mage.NewCreature("Left", "", 1, 4), b.PlayerID())
	g.ExecuteAttackers(a.PlayerID(), attackers)
	g.PutTriggersOnStack()
	g.ResolveStack()
	if ca.calls != 2 || cb.calls != 2 || g.FindPermanent(left.ID()).Tapped || g.FindPermanent(right.ID()).Tapped {
		t.Fatal("wrong loop choices or timing")
	}
	g.BouncePermanentToHand(source)
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if !game.FindPermanent(left.ID()).Tapped || !game.FindPermanent(right.ID()).Tapped {
			t.Fatal("loop result not retained in delayed frame")
		}
	}
}

func TestRONNamedPileSelectionNames(t *testing.T) {
	source := pileSelectionSource(strings.Replace(ronPartitionAction, `["Left","Right"]`, `["Left","Middle","Right"]`, 1) + `,ActionForEachPermanent(IsAttacking,[ChooseNamedPileForPermanent(["Right","Left"],EachablePermanent),TapEachPermanent(` + ronPileRead + `)])`)
	nodes, err := Parse(strings.NewReader(source), "three-piles.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	g.PutOnBattlefield(factory(), a.PlayerID())
	attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 8), a.PlayerID())
	attacker.RevokeBaseAttr(core.AttrSummonSick)
	defender := g.PutOnBattlefield(mage.NewCreature("Defender", "", 1, 4), b.PlayerID())
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if defender.Tapped {
		t.Fatal("reordered subset selected a different declared pile")
	}
}

func TestRONNamedPileSelectionBranchScope(t *testing.T) {
	branch := `If(PermanentPassesFilter(ThisPermanent,IsAttacking),[` + ronPartitionAction + `])`
	nodes, err := Parse(strings.NewReader(pileSelectionSource(branch)), "valid-branch.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Compile(nodes[0]); err != nil {
		t.Fatal("branch prerequisite", err)
	}
	nodes, err = Parse(strings.NewReader(pileSelectionSource(branch+`,ActionForEachPermanent(IsCardtype(Creature),[`+ronPileChoice+`])`)), "branch-leak.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Compile(nodes[0]); err == nil || !strings.Contains(err.Error(), "preceding partition") {
		t.Fatal("branch exported partition", err)
	}
	nodes, err = Parse(strings.NewReader(pileSelectionSource(ronPartitionAction+`,ActionForEachPermanent(IsCardtype(Creature),[If(PermanentPassesFilter(EachablePermanent,IsAttacking),[`+ronPileChoice+`]),TapEachPermanent(`+ronPileRead+`)])`)), "choice-branch-leak.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Compile(nodes[0]); err == nil || !strings.Contains(err.Error(), "preceding choice") {
		t.Fatal("branch exported selected pile", err)
	}
}
