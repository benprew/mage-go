package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronCounterPayer struct {
	*mage.BasePlayer
	choice, calls int
}

func (p *ronCounterPayer) ChooseNumber(_, _ int, _ string) int { p.calls++; return p.choice }

const anyChargeCost = `RemoveCounters([AnyNumberOfCountersOfTypeFromPermanent(ChargeCounter, ThisPermanent)])`
const paidCounterValue = `TheNumberOfCountersRemovedThisWay`

func counterPaymentCard(t *testing.T, cost, actions string) mage.Card {
	t.Helper()
	nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Payment",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(%s,%s)])`, cost, actions)), "payment.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	c := factory()
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRONCounterPaymentBatteries(t *testing.T) {
	for _, tc := range []struct {
		name  string
		color core.Color
	}{
		{"Black Mana Battery", core.Black}, {"Blue Mana Battery", core.Blue}, {"Green Mana Battery", core.Green}, {"Red Mana Battery", core.Red}, {"White Mana Battery", core.White},
	} {
		for _, count := range []int{0, 2, 3} {
			t.Run(fmt.Sprintf("%s/%d", tc.name, count), func(t *testing.T) {
				a := &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A"), choice: count}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				c := repositoryCard(t, "LEG", tc.name)
				if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
					t.Fatal(err)
				}
				p := g.PutOnBattlefield(c, a.PlayerID())
				a.ManaPool().Add(core.Colorless, 2)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().TotalMana() != 0 || !g.FindPermanent(p.ID()).Tapped || g.FindPermanent(p.ID()).Counters[core.Charge] != 0 || g.StackSize() != 1 {
					t.Fatal("charge payment or timing")
				}
				g.ResolveStack()
				g.MutablePermanent(p.ID()).Tapped = false
				g.MutablePermanent(p.ID()).AddCounter(core.Charge, 2)
				ability := c.Abilities()[1]
				if !mage.IsActivatedManaAbility(ability) {
					t.Fatal("not a mana ability")
				}
				if mage.ManaProductionsForAbilityInGame(ability, g, p.ID()) != nil || a.calls != 0 {
					t.Fatal("solver must not invent a counter choice")
				}
				clone := g.Clone()
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
					t.Fatal(err)
				}
				if g.StackSize() != 0 || a.ManaPool().Count(tc.color) != 1+count || b.ManaPool().TotalMana() != 0 || a.calls != 1 || int(g.FindPermanent(p.ID()).Counters[core.Charge]) != 3-count || !g.FindPermanent(p.ID()).Tapped {
					t.Fatal("counter payment or immediate mana")
				}
				if clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 || clone.FindPermanent(p.ID()).Counters[core.Charge] != 3 || clone.FindPermanent(p.ID()).Tapped {
					t.Fatal("activation changed clone")
				}
			})
		}
	}
}

func TestRONCounterPaymentInvalidChoicesAndRollback(t *testing.T) {
	for _, scenario := range []string{"negative", "excess", "mana", "opponent", "phase", "tapped"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A"), choice: 2}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			cost := `And([TapPermanent(ThisPermanent), ` + anyChargeCost + `])`
			if scenario == "mana" {
				cost = `And([TapPermanent(ThisPermanent), ` + anyChargeCost + `, PayMana([ManaCostGeneric(1)])])`
			}
			c := counterPaymentCard(t, cost, `ActionList([AddManaRepeated(ManaProduceU, `+paidCounterValue+`)])`)
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(p.ID()).AddCounter(core.Charge, 3)
			who := a.PlayerID()
			switch scenario {
			case "negative":
				a.choice = -1
			case "excess":
				a.choice = 4
			case "opponent":
				who = b.PlayerID()
			case "phase":
				g.PhaseOut(p.ID())
			case "tapped":
				g.MutablePermanent(p.ID()).Tapped = true
			}
			if err := g.ActivateAbilityByIndex(who, p.ID(), 0, nil); err == nil {
				t.Fatal("invalid activation accepted")
			}
			if scenario == "phase" {
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			if a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 || g.FindPermanent(p.ID()).Counters[core.Charge] != 3 || (scenario != "tapped" && g.FindPermanent(p.ID()).Tapped) {
				t.Fatal("failed payment changed state")
			}
		})
	}
}

func TestRONCounterPaymentSavedInstances(t *testing.T) {
	for _, change := range []string{"none", "leave", "return", "phase", "control"} {
		t.Run(change, func(t *testing.T) {
			a := &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A"), choice: 2}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			c := counterPaymentCard(t, anyChargeCost, `ActionList([GainLife(`+paidCounterValue+`)])`)
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(p.ID()).AddCounter(core.Charge, 5)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			a.choice = 3
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			switch change {
			case "leave", "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if change == "return" {
					a.RemoveFromHand(c.ID())
					g.PutOnBattlefield(c, b.PlayerID())
					g.MutablePermanent(c.ID()).AddCounter(core.Charge, 9)
				}
			case "phase":
				g.PhaseOut(p.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			clone := g.Clone()
			g.ResolveStack()
			if a.Life() != 25 || b.Life() != 23 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("saved count, copy controller or clone isolation")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 25 || clone.GetPlayer(b.PlayerID()).Life() != 23 {
				t.Fatal("clone lost counts")
			}
		})
	}
}

func TestRONCounterPaymentTargets(t *testing.T) {
	for _, change := range []string{"none", "return", "phase"} {
		t.Run(change, func(t *testing.T) {
			a := &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A"), choice: 2}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			c := counterPaymentCard(t, anyChargeCost, `Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([AddManaRepeated(ManaProduceB, `+paidCounterValue+`)]))`)
			if mage.IsActivatedManaAbility(c.Abilities()[0]) {
				t.Fatal("targeted ability classified as mana")
			}
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(p.ID()).AddCounter(core.Charge, 3)
			targetCard := mage.NewCreature("Target", "", 1, 1)
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{uuid.New()}); err == nil || a.calls != 0 {
				t.Fatal("invalid target chose a payment")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || g.StackSize() != 1 || g.FindPermanent(p.ID()).Counters[core.Charge] != 1 {
				t.Fatal("targeted timing")
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(target)
				g.PutOnBattlefield(targetCard, b.PlayerID())
			case "phase":
				g.PhaseOut(target.ID())
			}
			g.ResolveStack()
			want := 0
			if change == "none" {
				want = 2
			}
			if a.ManaPool().Count(core.Black) != want {
				t.Fatal("target legality")
			}
		})
	}
}

func TestRONCounterPaymentInvalidDeclarations(t *testing.T) {
	for _, rule := range []string{
		`Activated(TapPermanent(ThisPermanent),ActionList([AddManaRepeated(ManaProduceB,TheNumberOfCountersRemovedThisWay)]))`,
		`Activated(` + anyChargeCost + `,ActionList([GainLife(TheNumberOfCountersRemovedThisWay())]))`,
		`Activated(And([` + anyChargeCost + `,` + anyChargeCost + `]),ActionList([GainLife(TheNumberOfCountersRemovedThisWay)]))`,
		`Activated(RemoveCounters([AnyNumberOfCountersOfTypeFromPermanent(ChargeCounter,Ref_TargetPermanent)]),ActionList([GainLife(TheNumberOfCountersRemovedThisWay)]))`,
		`Activated(RemoveCounters([AnyNumberOfCountersOfTypeFromPermanent(UnknownCounter,ThisPermanent)]),ActionList([GainLife(TheNumberOfCountersRemovedThisWay)]))`,
		`Activated(RemoveCounters([AnyNumberOfCountersOfTypeFromPermanent(ChargeCounter,ThisPermanent())]),ActionList([GainLife(TheNumberOfCountersRemovedThisWay)]))`,
		`Activated(And([` + anyChargeCost + `,RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter,ThisPermanent)])]),ActionList([GainLife(TheNumberOfCountersRemovedThisWay)]))`,
		`Activated(` + anyChargeCost + `,ActionList([GainLife(Integer(1))])),Activated(TapPermanent(ThisPermanent),ActionList([GainLife(TheNumberOfCountersRemovedThisWay)]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONCounterPaymentFactoriesAndConsumers(t *testing.T) {
	for _, action := range []string{`AddManaRepeated(AnyManaColor, TheNumberOfCountersRemovedThisWay)`, `AddManaRepeated(And([ManaProduceU, ManaProduceR]), TheNumberOfCountersRemovedThisWay)`, `LoseLife(TheNumberOfCountersRemovedThisWay)`, `DrawNumberCards(TheNumberOfCountersRemovedThisWay)`, `CreateTokens([NumberTokens(TheNumberOfCountersRemovedThisWay, ` + saprolingRON + `)], [])`} {
		t.Run(action, func(t *testing.T) {
			a := &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A"), choice: 1}
			g := mage.NewGame(a, mage.NewBasePlayer("B"))
			fillRONLibrary(t, a.BasePlayer, 5)
			nodes, err := Parse(strings.NewReader(`Card(Name:"Payment",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(`+anyChargeCost+`,ActionList([`+action+`]))])`), "payment.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			for _, count := range []int{1, 3} {
				c := factory()
				if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
					t.Fatal(err)
				}
				p := g.PutOnBattlefield(c, a.PlayerID())
				g.MutablePermanent(p.ID()).AddCounter(core.Charge, count)
				a.choice = count
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			switch {
			case strings.Contains(action, "AnyManaColor"):
				if a.ManaPool().TotalMana() != 4 {
					t.Fatal("chosen-color mana count")
				}
			case strings.Contains(action, "And"):
				if a.ManaPool().Count(core.Blue) != 4 || a.ManaPool().Count(core.Red) != 4 {
					t.Fatal("multiple mana production")
				}
			case strings.HasPrefix(action, "LoseLife"):
				if a.Life() != 16 {
					t.Fatal("life amount")
				}
			case strings.HasPrefix(action, "Draw"):
				if len(a.Hand()) != 4 {
					t.Fatal("draw count")
				}
			case strings.HasPrefix(action, "CreateTokens"):
				if len(g.FilterBattlefield(mage.IsToken)) != 4 {
					t.Fatal("token count")
				}
			}
		})
	}
}

func TestRONCounterPaymentInventory(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Inventory",Rules:[Activated(`+anyChargeCost+`,ActionList([AddManaRepeated(ManaProduceB,TheNumberOfCountersRemovedThisWay)]))])`), "inventory.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Inventory", "Card", "card", "", false, entries)
	for name, want := range map[string]string{"AnyNumberOfCountersOfTypeFromPermanent": "counter_removal_instruction", "ChargeCounter": "permanent_counter_type", "ThisPermanent": "permanent_reference", "ManaProduceB": "mana_production", "TheNumberOfCountersRemovedThisWay": "value"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name {
				found = true
				if entry.Role != want {
					t.Errorf("%s role=%s want=%s", name, entry.Role, want)
				}
			}
		}
		if !found {
			t.Fatalf("missing %s", name)
		}
	}
}

func TestRONCounterPaymentZeroAndUnused(t *testing.T) {
	for _, tc := range []struct {
		cost, action string
		mana         int
	}{
		{anyChargeCost, `AddMana(ManaProduceB),AddManaRepeated(ManaProduceB,TheNumberOfCountersRemovedThisWay)`, 1},
		{`RemoveCounters([AnyNumberOfCountersOfTypeFromPermanent(PTCounter(1,1),ThisPermanent)])`, `GainLife(Integer(1))`, 0},
	} {
		a := &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A")}
		g := mage.NewGame(a, mage.NewBasePlayer("B"))
		c := counterPaymentCard(t, tc.cost, `ActionList([`+tc.action+`])`)
		p := g.PutOnBattlefield(c, a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if a.calls != 1 || a.ManaPool().Count(core.Black) != tc.mana {
			t.Fatal("zero payment or unused result")
		}
	}
}
