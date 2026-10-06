package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSingularEntryCounterJavelineers(t *testing.T) {
	for _, kind := range []string{"player", "creature", "planeswalker", "self"} {
		t.Run(kind, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "FEM", "Icatian Javelineers")
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed cast changed state")
			}
			a.ManaPool().Add(core.White, 1)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(c.ID()) != nil {
				t.Fatal("entry before spell resolution")
			}
			g.ResolveStack()
			p := g.FindPermanent(c.ID())
			if p == nil || p.Counters[core.Javelin] != 1 || p.CurrentPower(g) != 1 || p.CurrentToughness(g) != 1 {
				t.Fatal("entry characteristics")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{b.PlayerID()}); err == nil {
				t.Fatal("summoning sickness ignored")
			}
			if p.Tapped || p.Counters[core.Javelin] != 1 || g.StackSize() != 0 {
				t.Fatal("failed tap payment changed state")
			}
			g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
			target := b.PlayerID()
			switch kind {
			case "creature":
				target = g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 3), b.PlayerID()).ID()
			case "planeswalker":
				target = g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 3), b.PlayerID()).ID()
			case "self":
				target = p.ID()
			}
			if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 1, []uuid.UUID{target}); err == nil {
				t.Fatal("opponent activated source")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{uuid.New()}); err == nil {
				t.Fatal("invalid target accepted")
			}
			if g.FindPermanent(p.ID()).Tapped || g.FindPermanent(p.ID()).Counters[core.Javelin] != 1 || g.StackSize() != 0 {
				t.Fatal("illegal announcement spent costs")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{target}); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(p.ID()).Tapped || g.FindPermanent(p.ID()).Counters[core.Javelin] != 0 || b.Life() != 20 {
				t.Fatal("payment timing")
			}
			g.UntapPermanent(g.FindPermanent(p.ID()))
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{b.PlayerID()}); err == nil {
				t.Fatal("spent counter paid again")
			}
			if g.FindPermanent(p.ID()).Tapped || g.StackSize() != 1 {
				t.Fatal("counter failure did not roll back tap")
			}
			g.ResolveStack()
			switch kind {
			case "player":
				if b.Life() != 19 {
					t.Fatal("player damage")
				}
			case "creature":
				if g.FindPermanent(target).Damage != 1 {
					t.Fatal("creature damage")
				}
			case "planeswalker":
				if g.FindPermanent(target).Counters[core.Loyalty] != 2 {
					t.Fatal("planeswalker damage")
				}
			case "self":
				if g.FindPermanent(target) != nil {
					t.Fatal("self damage was not lethal")
				}
			}
		})
	}
}

func TestRONSingularEntryCounterLifecycle(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "FEM", "Icatian Javelineers")
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("ability copy failed")
	}
	clone := g.Clone()
	g.PhaseIn(g.PhaseOut(p.ID()))
	if g.FindPermanent(p.ID()).Counters[core.Javelin] != 0 {
		t.Fatal("phase cycle repeated entry")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	a.RemoveFromHand(c.ID())
	returned := g.PutOnBattlefield(c, a.PlayerID())
	if returned.Counters[core.Javelin] != 1 {
		t.Fatal("new incarnation lost entry counter")
	}
	q := g.PutOnBattlefield(c.Copy(), b.PlayerID())
	if q.Counters[core.Javelin] != 1 {
		t.Fatal("card copy lost entry counter")
	}
	g.MutablePermanent(q.ID()).RemoveCounter(core.Javelin, 1)
	if returned.Counters[core.Javelin] != 1 {
		t.Fatal("instances shared counters")
	}
	g.ResolveStack()
	if b.Life() != 18 || returned.Counters[core.Javelin] != 1 {
		t.Fatal("departed source or copied ability failed")
	}
	if clone.GetPlayer(b.PlayerID()).Life() != 20 || clone.FindPermanent(p.ID()).Counters[core.Javelin] != 0 {
		t.Fatal("live operations changed clone")
	}
	clone.ResolveStack()
	if clone.GetPlayer(b.PlayerID()).Life() != 18 {
		t.Fatal("cloned abilities failed")
	}
}

func TestRONSingularEntryCounterTargetRecheck(t *testing.T) {
	for _, change := range []string{"return", "phase", "shroud", "prevention"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			p := g.PutOnBattlefield(repositoryCard(t, "FEM", "Icatian Javelineers"), a.PlayerID())
			g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
			c := mage.NewCreature("Victim", "", 2, 3)
			c.SetOwner(b.PlayerID())
			q := g.PutOnBattlefield(c, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{q.ID()}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.BouncePermanentToHand(q)
				b.RemoveFromHand(c.ID())
				g.PutOnBattlefield(c, b.PlayerID())
			case "phase":
				g.PhaseOut(q.ID())
			case "shroud":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Shroud)), q.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "prevention":
				g.AddPreventionShield(q.ID(), 1)
			}
			g.ResolveStack()
			if g.FindPermanentIncludingPhased(q.ID()).Damage != 0 || g.FindPermanent(p.ID()).Counters[core.Javelin] != 0 {
				t.Fatal("target recheck or spent cost")
			}
		})
	}
}

func TestRONSingularEntryCounterReplacements(t *testing.T) {
	for _, choice := range []int{0, 1} {
		t.Run(fmt.Sprint(choice), func(t *testing.T) {
			a := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("A"), index: 1 - choice}
			b := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("B"), index: choice}
			g := mage.NewGame(a, b)
			donor := g.PutOnBattlefield(mage.NewArtifact("Donor", ""), a.PlayerID())
			g.AddETBAdditionalCounters(donor.ID(), core.Javelin, 1, mage.IsCreature, false)
			g.AddCounterDoubler(donor.ID(), core.Javelin, mage.IsCreature)
			p := g.PutOnBattlefield(repositoryCard(t, "FEM", "Icatian Javelineers"), b.PlayerID())
			want := 4
			if choice == 1 {
				want = 3
			}
			if int(p.Counters[core.Javelin]) != want || a.choices != 0 || b.choices != 1 {
				t.Fatalf("replacement count=%d choices=%d/%d, want %d", p.Counters[core.Javelin], a.choices, b.choices, want)
			}
		})
	}
}

func TestRONSingularEntryCounterDeclarations(t *testing.T) {
	for _, rule := range []string{
		`AsPermanentEnters(ThisPermanent,[EntersWithACounter])`,
		`AsPermanentEnters(ThisPermanent,[EntersWithACounter()])`,
		`AsPermanentEnters(ThisPermanent,[EntersWithACounter(TideCounter,JavelinCounter)])`,
		`AsPermanentEnters(ThisPermanent,[EntersWithACounter(PoisonCounter)])`,
		`AsPermanentEnters(ThisPermanent,[EntersWithACounter(TideCounter())])`,
		`AsPermanentEnters(ThisPermanent,[EntersWithACounter(PTCounter(2,2))])`,
		`AsPermanentEnters(HostPermanent,[EntersWithACounter(TideCounter)])`,
		`AsPermanentEnters(ThisPermanent(),[EntersWithACounter(TideCounter)])`,
		`AsPermanentEnters(ThisPermanent,[EntersWithACounter(TideCounter),EntersTapped])`,
		`Activated(PayMana([]),ActionList([EntersWithACounter(TideCounter)]))`,
		`SpellActions(ActionList([EntersWithACounter(TideCounter)]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", rule)
		}
	}
	for _, counter := range []string{"TideCounter", "JavelinCounter", "PTCounter(1,1)"} {
		factory := permanentScopeFactory(t, "Artifact", `AsPermanentEnters(ThisPermanent,[EntersWithACounter(`+counter+`)])`)
		first, second := factory(), factory()
		first.Abilities()[0].(*mage.EntersWithNCountersAbility).Count = 2
		if second.Abilities()[0].(*mage.EntersWithNCountersAbility).Count != 1 {
			t.Fatal("factories shared entry declaration")
		}
		if err := mage.ValidateCardDefinition(second.Name(), second); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRONSingularEntryCounterCompleteRecords(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/FEM.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "FEM.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, n := range nodes {
		name := n.Fields["Name"].String
		if name != "Homarid" && name != "Tidal Influence" && name != "Icatian Javelineers" {
			continue
		}
		seen++
		for _, rule := range n.Fields["Rules"].Args {
			if rule.Name != "AsPermanentEnters" {
				continue
			}
			option, err := compilePermanentEntry(rule)
			if err != nil {
				t.Fatal(err)
			}
			c := mage.NewArtifact("Entry contract", "", option())
			g, a, _ := ronGame()
			p := g.PutOnBattlefield(c, a.PlayerID())
			counter := core.Tide
			if name == "Icatian Javelineers" {
				counter = core.Javelin
			}
			if p.Counters[counter] != 1 || g.StackSize() != 0 {
				t.Fatalf("%s entry clause", name)
			}
		}
		_, err := Compile(n)
		if (name == "Icatian Javelineers" || name == "Homarid") && err != nil {
			t.Fatal(err)
		}
		if name == "Tidal Influence" && err == nil {
			t.Fatalf("incomplete %s accepted", name)
		}
	}
	if seen != 3 {
		t.Fatalf("record coverage %d", seen)
	}
}

func TestRONSingularEntryCounterInventory(t *testing.T) {
	src := `Card(Name:"Entry",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[AsPermanentEnters(ThisPermanent,[EntersWithACounter(PTCounter(1,1))])])`
	report, err := InspectFS(fstest.MapFS{"entry.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 || len(report.Issues) != 0 {
		t.Fatal("valid entry rejected")
	}
	found := false
	for _, s := range report.Signatures {
		if s.Name == "PTCounter" && s.Role == "permanent_counter_type" {
			found = true
		}
	}
	if !found {
		t.Fatal("counter type role missing")
	}
}

func TestRONSingularEntryCounterObserver(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `AsPermanentEnters(ThisPermanent,[EntersWithACounter(TideCounter)]),TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([GainLife(Integer(2))]))`)
	c := factory()
	trigger := c.Abilities()[1].(*mage.GenericTriggered)
	condition := trigger.Condition
	trigger.SetCondition(func(event *core.GameEvent, game mage.GameReader, source, controller uuid.UUID) bool {
		p := game.FindPermanent(source)
		return condition(event, game, source, controller) && p != nil && p.Counters[core.Tide] == 1
	})
	g, a, _ := ronGame()
	g.PutOnBattlefield(c, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || a.Life() != 20 {
		t.Fatal("entry observer did not see counter before resolution")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("entry observer failed")
	}
}
