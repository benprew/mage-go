package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSourceCounterCapRasputin(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Rasputin Dreamweaver")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0) == nil {
		t.Fatal("unpaid cast")
	}
	a.ManaPool().Add(core.White, 1)
	a.ManaPool().Add(core.Blue, 1)
	a.ManaPool().Add(core.Colorless, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if p == nil || p.Counters[core.Dream] != 7 {
		t.Fatal("entry")
	}
	donor := g.PutOnBattlefield(mage.NewArtifact("Doubler", ""), a.PlayerID())
	g.AddCounterDoubler(donor.ID(), core.Dream, mage.IsCreature)
	for _, count := range []int{7, 6} {
		if count == 6 {
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
		}
		g.RunStepWithPriority(core.Untap)
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
		g.PutTriggersOnStack()
		if g.StackSize() != 1 {
			t.Fatal("upkeep")
		}
		pending := g.Clone()
		if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
			t.Fatal("copy")
		}
		g.ResolveStack()
		pending.ResolveStack()
		if g.FindPermanent(p.ID()).Counters[core.Dream] != 7 || pending.FindPermanent(p.ID()).Counters[core.Dream] != 7 {
			t.Fatal("cap after doubling/copies")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("opponent upkeep")
	}
}

func TestRONSourceCounterCapInvalid(t *testing.T) {
	rule := `PermanentRuleEffect(ThisPermanent,[CantHaveMoreThanNumberCountersOfType(Integer(7),DreamCounter)])`
	for _, bad := range []string{
		strings.Replace(rule, "ThisPermanent", "HostPermanent", 1),
		strings.Replace(rule, "Integer(7)", "Integer(-1)", 1),
		strings.Replace(rule, "Integer(7)", "ValueX", 1),
		strings.Replace(rule, "DreamCounter", "UnknownCounter", 1),
		strings.Replace(rule, "DreamCounter)", "DreamCounter,Integer(1))", 1),
		strings.Replace(rule, "ThisPermanent", "ThisPermanent()", 1),
	} {
		n, err := Parse(strings.NewReader(`Card(Name:"Cap",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),ManaCost:[],Rules:[`+bad+`])`), "invalid")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(n[0]); err == nil {
			t.Fatal("accepted", bad)
		}
	}
}

func TestRONSourceCounterCapReplacementOrder(t *testing.T) {
	for _, choice := range []int{0, 1} {
		a := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("A"), index: 1 - choice}
		b := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("B"), index: choice}
		g := mage.NewGame(a, b)
		donor := g.PutOnBattlefield(mage.NewArtifact("Donor", ""), a.PlayerID())
		g.AddETBAdditionalCounters(donor.ID(), core.Dream, 3, mage.AnyPermanent, false)
		g.AddCounterDoubler(donor.ID(), core.Dream, mage.AnyPermanent)
		factory := permanentScopeFactory(t, "Artifact", `AsPermanentEnters(ThisPermanent,[EntersWithNumberCounters(Integer(1),DreamCounter)]),PermanentRuleEffect(ThisPermanent,[CantHaveMoreThanNumberCountersOfType(Integer(7),DreamCounter)])`)
		p := g.PutOnBattlefield(factory(), b.PlayerID())
		want := 7
		if choice == 1 {
			want = 5
		}
		if p.Counters[core.Dream] != want || a.choices != 0 || b.choices != 1 {
			t.Fatal("cap changed affected-player replacement order", p.Counters[core.Dream], a.choices, b.choices)
		}
	}
}

func TestRONSourceCounterCapInventoryAndAtomicLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	n := hostUntapRecord(t, "LEG", "Rasputin Dreamweaver")
	src := strings.Split(string(data), "\n")[n.Position.Line-1]
	files := fstest.MapFS{"cap.ron": {Data: []byte(src)}}
	report, err := InspectFS(fstest.MapFS{"only-cap.ron": {Data: []byte(`Card(Name:"Cap",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[PermanentRuleEffect(ThisPermanent,[CantHaveMoreThanNumberCountersOfType(Integer(7),DreamCounter)])])`)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"CantHaveMoreThanNumberCountersOfType": "permanent_rule_operation", "DreamCounter": "permanent_counter_type"} {
		found := false
		for _, entry := range report.Signatures {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatal("missing semantic role", name, role)
		}
	}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	one, err := registry.CreateCard("Rasputin Dreamweaver")
	if err != nil {
		t.Fatal(err)
	}
	two, err := registry.CreateCard("Rasputin Dreamweaver")
	if err != nil || one.ID() == two.ID() {
		t.Fatal("factory identity", err)
	}
	if err := mage.ValidateCardDefinition(one.Name(), one); err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(strings.Replace(src, "Rasputin Dreamweaver", "Bad Cap", 1), "Integer(7), DreamCounter)])]", "Integer(-1), DreamCounter)])]", 1))}
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("partial collection loaded")
	}
	if _, err := registry.CreateCard("Bad Cap"); err == nil {
		t.Fatal("bad definition published")
	}
	if _, err := registry.CreateCard("Rasputin Dreamweaver"); err != nil {
		t.Fatal("atomic failure removed accepted record", err)
	}
}
