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

func TestRONEntryCountersTriskelion(t *testing.T) {
	for _, kind := range []string{"player", "creature", "planeswalker", "self"} {
		t.Run(kind, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "ATQ", "Triskelion")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 || g.FindPermanent(card.ID()) != nil {
				t.Fatal("failed cast changed state")
			}
			a.ManaPool().Add(core.Colorless, 6)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil {
				t.Fatal("entry happened before resolution")
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p == nil || p.Counters[core.P1P1] != 3 || p.CurrentPower(g) != 4 || p.CurrentToughness(g) != 4 || g.StackSize() != 0 {
				t.Fatal("entry counters or characteristics")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{uuid.New()}); err == nil {
				t.Fatal("invalid target accepted")
			}
			if g.FindPermanent(p.ID()).Counters[core.P1P1] != 3 || g.StackSize() != 0 {
				t.Fatal("invalid target consumed a counter")
			}
			target := b.PlayerID()
			var victim *mage.Permanent
			switch kind {
			case "creature":
				victim = g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 5), b.PlayerID())
				target = victim.ID()
			case "planeswalker":
				victim = g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 5), b.PlayerID())
				target = victim.ID()
			case "self":
				target = p.ID()
			}
			if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 1, []uuid.UUID{target}); err == nil {
				t.Fatal("opponent activated source")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{target}); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(p.ID()).Counters[core.P1P1] != 2 || g.FindPermanent(p.ID()).Tapped || b.Life() != 20 {
				t.Fatal("counter payment, tap status or timing")
			}
			g.ResolveStack()
			switch kind {
			case "player":
				if b.Life() != 19 {
					t.Fatal("player damage")
				}
			case "planeswalker":
				if g.FindPermanent(victim.ID()).Counters[core.Loyalty] != 4 {
					t.Fatal("loyalty damage")
				}
			case "creature", "self":
				if g.FindPermanent(target).Damage != 1 {
					t.Fatal("creature damage")
				}
			}
			for range 2 {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{b.PlayerID()}); err != nil {
					t.Fatal(err)
				}
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{b.PlayerID()}); err == nil {
				t.Fatal("spent counters paid again")
			}
			if g.StackSize() != 2 {
				t.Fatal("failed payment changed stack")
			}
			g.ResolveStack()
		})
	}
}

func TestRONEntryCountersLifecycle(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "ATQ", "Triskelion")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 1, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("stack copy failed")
	}
	clone := g.Clone()
	g.PhaseIn(g.PhaseOut(p.ID()))
	if g.FindPermanent(p.ID()).Counters[core.P1P1] != 2 {
		t.Fatal("phase cycle repeated entry")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if returned.Counters[core.P1P1] != 3 {
		t.Fatal("return did not reset entry counters")
	}
	g.ResolveStack()
	if a.Life() != 18 || returned.Counters[core.P1P1] != 3 {
		t.Fatal("departed damage source or copy lost identity")
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.FindPermanent(p.ID()).Counters[core.P1P1] != 2 {
		t.Fatal("live operations changed clone")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 18 {
		t.Fatal("cloned abilities failed")
	}
	copied := card.Copy()
	q := g.PutOnBattlefield(copied, b.PlayerID())
	if q.Counters[core.P1P1] != 3 {
		t.Fatal("card copy lost entry ability")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), q.ID(), 1, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(returned.ID()).Counters[core.P1P1] != 3 {
		t.Fatal("instances shared counter state")
	}
	spell := repositoryCard(t, "LEB", "Lightning Bolt")
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{q.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(q.ID()) != nil || b.Life() != 19 {
		t.Fatal("lethal damage or independent ability after source death")
	}
}

func TestRONEntryCountersReplacements(t *testing.T) {
	for _, choice := range []int{0, 1} {
		t.Run(fmt.Sprint(choice), func(t *testing.T) {
			a := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("A"), index: 1 - choice}
			b := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("B"), index: choice}
			g := mage.NewGame(a, b)
			donor := g.PutOnBattlefield(mage.NewArtifact("Donor", ""), a.PlayerID())
			g.AddETBAdditionalCounters(donor.ID(), core.P1P1, 1, mage.IsCreature, false)
			g.AddCounterDoubler(donor.ID(), core.P1P1, mage.IsCreature)
			p := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Triskelion"), b.PlayerID())
			want := 8
			if choice == 1 {
				want = 7
			}
			if p.Counters[core.P1P1] != want || a.choices != 0 || b.choices != 1 {
				t.Fatalf("entry replacement counters=%d, choices=%d/%d, want %d", p.Counters[core.P1P1], a.choices, b.choices, want)
			}
		})
	}
}

func TestRONEntryCountersDeclarations(t *testing.T) {
	for _, operation := range []string{
		`EntersWithNumberCounters`, `EntersWithNumberCounters(Integer(3))`,
		`EntersWithNumberCounters(Integer(3), PTCounter(1,1), ChargeCounter)`,
		`EntersWithNumberCounters(3, PTCounter(1,1))`, `EntersWithNumberCounters(Integer(-1), PTCounter(1,1))`,
		`EntersWithNumberCounters(Integer("3"), PTCounter(1,1))`, `EntersWithNumberCounters(Integer(3,4), PTCounter(1,1))`,
		`EntersWithNumberCounters(ValueX, PTCounter(1,1))`, `EntersWithNumberCounters(Integer(3), PoisonCounter)`,
		`EntersWithNumberCounters(Integer(3), CreditCounter())`, `EntersWithNumberCounters(Integer(3), PTCounter(Integer(1),1))`,
		`EntersWithNumberCounters(Integer(3), PTCounter(2,2))`,
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[AsPermanentEnters(ThisPermanent,[%s])])`, operation)
		nodes, err := Parse(strings.NewReader(src), "invalid-entry.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", operation)
		}
	}
	for _, rule := range []string{
		`AsPermanentEnters(HostPermanent,[EntersWithNumberCounters(Integer(3),ChargeCounter)])`,
		`AsPermanentEnters(ThisPermanent(),[EntersWithNumberCounters(Integer(3),ChargeCounter)])`,
		`AsPermanentEnters(Ref_TargetPermanent,[EntersWithNumberCounters(Integer(3),ChargeCounter)])`,
		`AsPermanentEnters(ThisPermanent,[EntersWithNumberCounters(Integer(3),ChargeCounter),EntersTapped])`,
		`Activated(PayMana([]),ActionList([EntersWithNumberCounters(Integer(3),ChargeCounter)]))`,
		`SpellActions(ActionList([EntersWithNumberCounters(Integer(3),ChargeCounter)]))`,
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[%s])`, rule)
		nodes, err := Parse(strings.NewReader(src), "invalid-entry.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", rule)
		}
	}
	for _, counter := range []string{`CreditCounter`, `PTCounter(+1,+0)`, `PTCounter(+1,+1)`} {
		factory := permanentScopeFactory(t, "Artifact", fmt.Sprintf(`AsPermanentEnters(ThisPermanent,[EntersWithNumberCounters(Integer(0),%s)])`, counter))
		if err := mage.ValidateCardDefinition("zero", factory()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRONEntryCountersInventory(t *testing.T) {
	src := `Card(Name:"Entry",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[AsPermanentEnters(ThisPermanent,[EntersWithNumberCounters(Integer(3),PTCounter(1,1))])])`
	report, err := InspectFS(fstest.MapFS{"entry.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"Integer": "entry_value", "PTCounter": "permanent_counter_type"} {
		found := false
		for _, s := range report.Signatures {
			if s.Name == name && s.Role == role {
				found = true
			}
		}
		if !found {
			t.Errorf("missing %s role %s", name, role)
		}
	}
}

func TestRONEntryCountersObserverAndFactoryIsolation(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `AsPermanentEnters(ThisPermanent,[EntersWithNumberCounters(Integer(3),CreditCounter)]),TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([GainLife(Integer(2))]))`)
	first, second := factory(), factory()
	first.Abilities()[0].(*mage.EntersWithNCountersAbility).Count = 5
	if second.Abilities()[0].(*mage.EntersWithNCountersAbility).Count != 3 {
		t.Fatal("factories shared entry declarations")
	}
	trigger := second.Abilities()[1].(*mage.GenericTriggered)
	condition := trigger.Condition
	trigger.SetCondition(func(event *core.GameEvent, game mage.GameReader, source, controller uuid.UUID) bool {
		p := game.FindPermanent(source)
		return condition(event, game, source, controller) && p != nil && p.Counters[core.Credit] == 3
	})
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(second, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || a.Life() != 20 || p.Counters[core.Credit] != 3 {
		t.Fatal("entry observer did not see counters before trigger resolution")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("entry trigger failed")
	}
	zero := permanentScopeFactory(t, "Artifact", `AsPermanentEnters(ThisPermanent,[EntersWithNumberCounters(Integer(0),CreditCounter)])`)()
	if g.PutOnBattlefield(zero, a.PlayerID()).Counters[core.Credit] != 0 {
		t.Fatal("zero amount added counters")
	}
}

func TestRONEntryCountersTargetRecheckAndPrevention(t *testing.T) {
	for _, change := range []string{"shroud", "return", "phase", "prevention"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			p := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Triskelion"), a.PlayerID())
			card := mage.NewCreature("Victim", "", 2, 2)
			card.SetOwner(b.PlayerID())
			q := g.PutOnBattlefield(card, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{q.ID()}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "shroud":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Shroud)), q.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "return":
				g.BouncePermanentToHand(q)
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "phase":
				g.PhaseOut(q.ID())
			case "prevention":
				g.AddPreventionShield(q.ID(), 1)
			}
			g.ResolveStack()
			if g.FindPermanent(p.ID()).Counters[core.P1P1] != 2 {
				t.Fatal("resolution refunded cost")
			}
			if target := g.FindPermanentIncludingPhased(q.ID()); target == nil || target.Damage != 0 {
				t.Fatal("illegal or protected target took damage")
			}
		})
	}
}

func TestRONEntryCountersCompleteRecordInventory(t *testing.T) {
	want := map[string]int{"Clockwork Avian": 4, "Clockwork Beast": 7, "Divine Intervention": 2, "Icatian Moneychanger": 3, "Merseine": 3, "Rasputin Dreamweaver": 7, "Tetravus": 3, "Triskelion": 3}
	seen := map[string]int{}
	files, err := os.ReadDir("../../../../cards/mtgish")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".ron") {
			continue
		}
		data, err := os.ReadFile("../../../../cards/mtgish/" + file.Name())
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), file.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range nodes {
			name := n.Fields["Name"].String
			amount, ok := want[name]
			if !ok {
				continue
			}
			for _, rule := range n.Fields["Rules"].Args {
				if rule.Name != "AsPermanentEnters" {
					continue
				}
				option, err := compilePermanentEntry(rule)
				if err != nil {
					t.Fatal(err)
				}
				c := mage.NewArtifact("Entry contract", "", option())
				if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
					t.Fatal(err)
				}
				g, a, _ := ronGame()
				p := g.PutOnBattlefield(c, a.PlayerID())
				ability := c.Abilities()[0].(*mage.EntersWithNCountersAbility)
				if p.Counters[ability.CounterType] != amount {
					t.Fatalf("%s entry count", name)
				}
				seen[name]++
			}
		}
	}
	if len(seen) != len(want) || seen["Clockwork Beast"] != 6 {
		t.Fatalf("entry record coverage %v", seen)
	}
}
