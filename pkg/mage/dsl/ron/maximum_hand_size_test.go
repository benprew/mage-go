package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONMaximumHandSizeCursedRack(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "ATQ", "Cursed Rack")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast accepted")
	}
	if g.StackSize() != 0 || len(a.Hand()) != 1 || g.MaximumHandSize(b.PlayerID()) != 7 {
		t.Fatal("failed casting changed state")
	}
	a.ManaPool().Add(core.Colorless, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.MaximumHandSize(b.PlayerID()) != 7 {
		t.Fatal("spell changed hand limit before entry")
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if p == nil || p.ChosenPlayer != b.PlayerID() || g.MaximumHandSize(a.PlayerID()) != 7 || g.MaximumHandSize(b.PlayerID()) != 4 {
		t.Fatal("entry choice or hand limit")
	}
	fillRONHand(a, 8)
	fillRONHand(b, 8)
	clone := g.Clone()
	g.RunStepWithPriority(core.Cleanup)
	if len(a.Hand()) != 7 || len(b.Hand()) != 8 {
		t.Fatal("inactive player discarded")
	}
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.Cleanup)
	if len(b.Hand()) != 4 || len(b.Graveyard()) != 4 || g.StackSize() != 0 {
		t.Fatal("chosen player cleanup")
	}
	if len(clone.GetPlayer(b.PlayerID()).Hand()) != 8 {
		t.Fatal("cleanup changed clone")
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	if g.MaximumHandSize(b.PlayerID()) != 4 || g.MaximumHandSize(a.PlayerID()) != 7 {
		t.Fatal("control repeated entry choice")
	}
	ids := g.PhaseOut(p.ID())
	if g.MaximumHandSize(b.PlayerID()) != 7 {
		t.Fatal("phased rule remained active")
	}
	g.PhaseIn(ids)
	if g.MaximumHandSize(b.PlayerID()) != 4 {
		t.Fatal("phase cycle lost choice")
	}
	g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.EndOfTurn))
	if g.MaximumHandSize(b.PlayerID()) != 7 {
		t.Fatal("removed ability still applied")
	}
	g.RunStepWithPriority(core.Cleanup)
	if g.MaximumHandSize(b.PlayerID()) != 4 {
		t.Fatal("restored ability lost choice")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, b.PlayerID())
	if g.MaximumHandSize(a.PlayerID()) != 4 || g.MaximumHandSize(b.PlayerID()) != 7 {
		t.Fatal("returned rack retained old choice")
	}
	if clone.MaximumHandSize(a.PlayerID()) != 7 || clone.MaximumHandSize(b.PlayerID()) != 4 {
		t.Fatal("clone rule changed")
	}
}

func TestRONMaximumHandSizeLibraryRule(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEA.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEA.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record Node
	for _, node := range nodes {
		if node.Fields["Name"].String == "Library of Leng" {
			record = node
		}
	}
	if _, err = Compile(record); err == nil || !strings.Contains(err.Error(), "ReplaceWouldDiscard") {
		t.Fatalf("remaining replacement blocker: %v", err)
	}
	rules := record.Fields["Rules"]
	rules.Args = rules.Args[:1]
	record.Fields["Rules"] = rules
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	fillRONHand(a, 10)
	g.RunStepWithPriority(core.Cleanup)
	if len(a.Hand()) != 10 || g.MaximumHandSize(a.PlayerID()) != -1 || g.MaximumHandSize(b.PlayerID()) != 7 {
		t.Fatal("no maximum rule")
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	if g.MaximumHandSize(a.PlayerID()) != 7 || g.MaximumHandSize(b.PlayerID()) != -1 {
		t.Fatal("rule did not follow current controller")
	}
	g.RunStepWithPriority(core.Cleanup)
	if len(a.Hand()) != 7 {
		t.Fatal("former controller kept no maximum")
	}
}

func TestRONMaximumHandSizeInvalid(t *testing.T) {
	for _, rule := range []string{
		`PlayerEffect(You,[HasNoMaximumHandSize()])`, `PlayerEffect(You,[SetMaximumHandSize])`,
		`PlayerEffect(You,[SetMaximumHandSize(Integer(-1))])`, `PlayerEffect(You,[SetMaximumHandSize(4)])`,
		`PlayerEffect(You,[SetMaximumHandSize(ValueX)])`, `PlayerEffect(You,[SetMaximumHandSize(Integer())])`,
		`PlayerEffect(You,[SetMaximumHandSize(Integer(4),Integer(5))])`, `PlayerEffect(You,[SetMaximumHandSize(Integer(4,5))])`,
		`PlayerEffect(You(),[HasNoMaximumHandSize])`, `PlayerEffect(TheChosenPlayer,[SetMaximumHandSize(Integer(4))])`,
		`PlayerEffect(Ref_TargetPlayer,[HasNoMaximumHandSize])`, `PlayerEffect(You,[HasNoMaximumHandSize,SetMaximumHandSize(Integer(4))])`,
		`Activated(PayMana([]),ActionList([PlayerEffect(You,[HasNoMaximumHandSize])]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONMaximumHandSizeStrictLoadAndInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/ATQ.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Cursed Rack"`) {
			record = line
		}
	}
	files := fstest.MapFS{"rack.ron": {Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Cursed Rack")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Cursed Rack")
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	if g.MaximumHandSize(a.PlayerID()) != 4 || g.MaximumHandSize(b.PlayerID()) != 4 {
		t.Fatal("instances shared choices")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory: %v", err)
	}
	for _, signature := range report.Signatures {
		if signature.Name == "Integer" && signature.Role != "value" {
			t.Fatal("hand limit value role", signature.Role)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[PlayerEffect(TheChosenPlayer,[HasNoMaximumHandSize])])`)}
	if err = LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection changed registry")
	}
}
