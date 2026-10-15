package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNumericConsumerLivingArmor(t *testing.T) {
	for _, change := range []string{"live", "zero", "x", "control", "phase", "phase-cycle", "leave", "return", "shroud", "copy", "counter"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "DRK", "Living Armor")
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			s := g.PutOnBattlefield(c, a.PlayerID())
			cost, want := "{2}{U}", int(3)
			if change == "zero" {
				cost, want = "", 0
			}
			if change == "x" {
				cost, want = "{X}{G}", 1
			}
			card := mage.NewCreature("Target", cost, 1, 8)
			card.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			bad := g.PutOnBattlefield(mage.NewLand("Illegal"), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), s.ID(), 0, []uuid.UUID{bad.ID()}); err == nil || g.FindPermanent(s.ID()) == nil || g.FindPermanent(s.ID()).Tapped {
				t.Fatal("illegal target paid tap or sacrifice")
			}
			g.MutablePermanent(s.ID()).Tapped = true
			if err := g.ActivateAbilityByIndex(a.PlayerID(), s.ID(), 0, []uuid.UUID{p.ID()}); err == nil || g.FindPermanent(s.ID()) == nil {
				t.Fatal("tapped source paid sacrifice without the tap cost")
			}
			g.MutablePermanent(s.ID()).Tapped = false
			if err := g.ActivateAbilityByIndex(a.PlayerID(), s.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(s.ID()) != nil || len(a.Graveyard()) != 1 || g.FindPermanent(p.ID()).Counters[core.P0P1] != 0 || g.StackSize() != 1 {
				t.Fatal("payment or resolution timing")
			}
			switch change {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "phase", "phase-cycle":
				g.PhaseOut(p.ID())
				if change == "phase-cycle" {
					g.PhaseIn([]uuid.UUID{p.ID()})
				} else {
					want = 0
				}
			case "leave", "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if change == "return" {
					b.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
				want = 0
			case "shroud":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
				want = 0
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				want *= 2
			case "counter":
				g.CounterStackObject(g.StackPeek().ID)
				want = 0
			}
			pending := g.StackSize()
			clone := g.Clone()
			g.ResolveStack()
			if clone.StackSize() != pending {
				t.Fatal("clone stack changed")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				target := game.FindPermanentIncludingPhased(p.ID())
				if target != nil && (target.Counters[core.P0P1] != want || target.CurrentPower(game) != 1 || target.CurrentToughness(game) != 8+want) {
					t.Fatalf("counters or P/T: %+v, want %d", target.Counters, want)
				}
				if len(game.GetPlayer(a.PlayerID()).Graveyard()) != 1 {
					t.Fatal("sacrifice refunded")
				}
			}
			if target := g.FindPermanent(p.ID()); target != nil {
				g.MutablePermanent(p.ID()).AddCounter(core.P0P1, 1)
				if clone.FindPermanent(p.ID()).Counters[core.P0P1] != want {
					t.Fatal("clone counters shared")
				}
			}
		})
	}
}

func TestRONNumericConsumerMoneychanger(t *testing.T) {
	for _, change := range []string{"live", "return", "copy", "zero", "control", "phase-before", "counter"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "FEM", "Icatian Moneychanger")
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("unpaid casting changed state")
			}
			a.ManaPool().Add(core.White, 1)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(c.ID())
			if p == nil || p.Counters[core.Credit] != 3 || p.CurrentPower(g) != 0 || p.CurrentToughness(g) != 2 || a.Life() != 17 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("entry counters, damage, metadata or casting cost")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep triggered")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if p.Counters[core.Credit] != 3 || g.StackSize() != 1 {
				t.Fatal("upkeep timing")
			}
			g.ResolveStack()
			if g.FindPermanent(p.ID()).Counters[core.Credit] != 4 {
				t.Fatal("upkeep counter missing")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), boundedActivationIndex(t, p), nil); err == nil || g.FindPermanent(p.ID()) == nil {
				t.Fatal("non-upkeep sacrifice accepted")
			}
			g.SetStep(core.Upkeep)
			g.SetActivePlayerIndex(1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), boundedActivationIndex(t, p), nil); err == nil || g.FindPermanent(p.ID()) == nil {
				t.Fatal("opponent turn sacrifice accepted")
			}
			g.SetActivePlayerIndex(0)
			if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), boundedActivationIndex(t, p), nil); err == nil {
				t.Fatal("opponent permission")
			}
			if change == "phase-before" {
				g.PhaseOut(p.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), boundedActivationIndex(t, p), nil); err == nil || g.FindPermanentIncludingPhased(p.ID()) == nil {
					t.Fatal("phased source paid")
				}
				return
			}
			who, lifeA, lifeB := a.PlayerID(), 21, 20
			if change == "zero" {
				g.MutablePermanent(p.ID()).Counters[core.Credit] = 0
				lifeA = 17
			}
			if change == "control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.SetActivePlayerIndex(1)
				who, lifeA, lifeB = b.PlayerID(), 17, 24
			}
			if err := g.ActivateAbilityByIndex(who, p.ID(), boundedActivationIndex(t, p), nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(p.ID()) != nil || a.Life() != 17 || b.Life() != 20 || len(a.Graveyard()) != 1 {
				t.Fatal("sacrifice or gain timing")
			}
			g.SetStep(core.Draw)
			switch change {
			case "return":
				a.RemoveFromGraveyard(c.ID())
				returned := g.PutOnBattlefield(c, b.PlayerID())
				returned.AddCounter(core.Credit, 6)
				lifeB = 17
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				lifeB = 24
			case "counter":
				g.CounterStackObject(g.StackPeek().ID)
				lifeA = 17
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if game.GetPlayer(a.PlayerID()).Life() != lifeA || game.GetPlayer(b.PlayerID()).Life() != lifeB {
					t.Fatalf("LKI/controller life %d/%d want %d/%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life(), lifeA, lifeB)
				}
				if change == "return" && game.FindPermanent(c.ID()).Counters[core.Credit] != 9 {
					t.Fatal("old resolution changed returned source")
				}
			}
		})
	}
}

func TestRONNumericConsumerInvalid(t *testing.T) {
	for _, action := range []string{
		`GainLifeForEach`, `GainLifeForEach(Integer(1))`, `GainLifeForEach(Integer(1),Integer(2),Integer(3))`,
		`GainLifeForEach(You,Integer(1))`, `GainLifeForEach(Integer(1),ManaValueOfPermanent(Ref_TargetPermanent))`,
		`GainLifeForEach(Integer(1),TheTotalNumberOfCountersOfTypeAmongPermanents(CreditCounter,SinglePermanent(HostPermanent)))`,
		`GainLifeForEach(Integer(1),TheNumberOfCountersRemovedThisWay)`,
		`PutCounters([NumberCountersOfTypeOnPermanent(ManaValueOfPermanent(Ref_TargetPermanent),ChargeCounter,ThisPermanent)])`,
		`PutCounters([NumberCountersOfTypeOnPermanent(ManaValueOfPermanent(You),ChargeCounter,ThisPermanent)])`,
		`PutCounters([NumberCountersOfTypeOnPermanent(ManaValueOfPermanent(ThisPermanent()),ChargeCounter,ThisPermanent)])`,
		`PutCounters([NumberCountersOfTypeOnPermanent(Plus(Integer(1),ValueX),ChargeCounter,ThisPermanent)])`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([%s]))])`, action)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}

func TestRONNumericConsumerInventory(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Inventory",Rules:[SpellActions(ActionList([GainLifeForEach(Integer(2),ManaValueOfPermanent(ThisPermanent))]))])`), "inventory.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Inventory", "Card", "card", "collection", false, entries)
	for name, role := range map[string]string{"Integer": "value", "ManaValueOfPermanent": "value", "ThisPermanent": "permanent_reference"} {
		found := false
		for _, sig := range entries {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s missing %s role", name, role)
		}
	}
	report, err := InspectFS(os.DirFS("../../../.."), "cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 2305 || len(report.Issues) != 140 {
		t.Fatalf("compiled/rejected %d/%d", report.Compiled, len(report.Issues))
	}
	for _, issue := range report.Issues {
		if issue.Card == "Living Armor" || issue.Card == "Icatian Moneychanger" {
			t.Fatal(issue)
		}
	}
}

func TestRONNumericConsumerResolutionAndInstances(t *testing.T) {
	value := `TheTotalNumberOfCountersOfTypeAmongPermanents(ChargeCounter,SinglePermanent(ThisPermanent))`
	rule := `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([PutCounters([NumberCountersOfTypeOnPermanent(Plus(Integer(1),` + value + `),PTCounter(0,1),Ref_TargetPermanent)]),GainLifeForEach(Integer(2),` + value + `)])))`
	factory := permanentScopeFactory(t, "Artifact", rule)
	first, second := factory(), factory()
	if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("shared target declaration")
	}
	g, a, b := ronGame()
	s := g.PutOnBattlefield(first, a.PlayerID())
	q := g.PutOnBattlefield(second, b.PlayerID())
	one := g.PutOnBattlefield(mage.NewCreature("One", "", 1, 3), a.PlayerID())
	two := g.PutOnBattlefield(mage.NewCreature("Two", "", 1, 3), b.PlayerID())
	s.AddCounter(core.Charge, 1)
	q.AddCounter(core.Charge, 4)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), s.ID(), 0, []uuid.UUID{one.ID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), q.ID(), 0, []uuid.UUID{two.ID()}); err != nil {
		t.Fatal(err)
	}
	g.MutablePermanent(s.ID()).AddCounter(core.Charge, 2)
	clone := g.Clone()
	g.MutablePermanent(q.ID()).AddCounter(core.Charge, 1)
	g.ResolveStack()
	clone.ResolveStack()
	if g.FindPermanent(one.ID()).Counters[core.P0P1] != 4 || g.FindPermanent(two.ID()).Counters[core.P0P1] != 6 || a.Life() != 26 || b.Life() != 30 {
		t.Fatal("values read at announcement or shared between instances")
	}
	if clone.FindPermanent(one.ID()).Counters[core.P0P1] != 4 || clone.FindPermanent(two.ID()).Counters[core.P0P1] != 5 || clone.GetPlayer(b.PlayerID()).Life() != 28 {
		t.Fatal("clone lost independent current values")
	}
}

func TestRONNumericConsumerReplacementOrder(t *testing.T) {
	for _, index := range []int{0, 1} {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			a := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("A"), index: index}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			c := repositoryCard(t, "FEM", "Icatian Moneychanger")
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.SetStep(core.Upkeep)
			first := g.PutOnBattlefield(mage.NewArtifact("Double", ""), a.PlayerID())
			second := g.PutOnBattlefield(mage.NewArtifact("Add", ""), a.PlayerID())
			g.AddReplacementEffect(&informationLifeReplacement{source: first.ID(), double: true})
			g.AddReplacementEffect(&informationLifeReplacement{source: second.ID()})
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), boundedActivationIndex(t, p), nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if a.Life() != 24+index || a.choices != 1 {
				t.Fatalf("life=%d choices=%d", a.Life(), a.choices)
			}
		})
	}
	for _, index := range []int{0, 1} {
		a := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("A"), index: index}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		value := `TheTotalNumberOfCountersOfTypeAmongPermanents(PTCounter(1,1),SinglePermanent(ThisPermanent))`
		c := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([PutCounters([NumberCountersOfTypeOnPermanent(Plus(Integer(1),`+value+`),PTCounter(1,1),ThisPermanent)]),GainLifeForEach(Integer(2),`+value+`)]))`)()
		p := g.PutOnBattlefield(c, a.PlayerID())
		g.AddReplacementEffect(&counterPlacementReplacement{source: p.ID()})
		g.AddCounterDoubler(p.ID(), core.P1P1, mage.AnyPermanent)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		want := 4 - index
		if g.FindPermanent(p.ID()).Counters[core.P1P1] != want || a.Life() != 20+2*want || a.choices != 1 {
			t.Fatal("counter amount or later life read bypassed replacement order")
		}
	}
}

func TestRONNumericConsumerContexts(t *testing.T) {
	actions := `PutCounters([NumberCountersOfTypeOnPermanent(ManaValueOfPermanent(Ref_TargetPermanent),ChargeCounter,Ref_TargetPermanent)]),GainLifeForEach(Integer(2),ManaValueOfPermanent(Ref_TargetPermanent))`
	targeted := `Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + actions + `]))`
	for _, context := range []string{"spell", "mode", "activation", "trigger", "host", "player", "players"} {
		t.Run(context, func(t *testing.T) {
			g, a, b := ronGame()
			target := g.PutOnBattlefield(mage.NewCreature("Target", "{2}{U}", 1, 8), b.PlayerID())
			var c mage.Card
			switch context {
			case "spell", "mode":
				body := targeted
				if context == "mode" {
					body = `Modal_ChooseOne([` + body + `,` + body + `])`
				}
				nodes, err := Parse(strings.NewReader(`Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[],Rules:[SpellActions(`+body+`)])`), "probe.ron")
				if err != nil {
					t.Fatal(err)
				}
				f, err := Compile(nodes[0])
				if err != nil {
					t.Fatal(err)
				}
				c = f()
				c.SetOwner(a.PlayerID())
				a.AddToHand(c)
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
					t.Fatal(err)
				}
			case "activation", "player", "players":
				body := targeted
				if context == "player" {
					body = strings.Replace(targeted, `GainLifeForEach(Integer(2),ManaValueOfPermanent(Ref_TargetPermanent))`, `PlayerAction(ControllerOfPermanent(Ref_TargetPermanent),GainLifeForEach(Integer(2),ManaValueOfPermanent(Ref_TargetPermanent)))`, 1)
				}
				if context == "players" {
					body = strings.Replace(targeted, `GainLifeForEach(Integer(2),ManaValueOfPermanent(Ref_TargetPermanent))`, `PlayerActions(ControllerOfPermanent(Ref_TargetPermanent),[GainLifeForEach(Integer(2),ManaValueOfPermanent(Ref_TargetPermanent))])`, 1)
				}

				c = permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),`+body+`)`)()
				s := g.PutOnBattlefield(c, a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), s.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
					t.Fatal(err)
				}
			case "trigger":
				c = permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),`+targeted+`)`)()
				g.PutOnBattlefield(c, a.PlayerID())
				g.PutTriggersOnStack()
			case "host":
				c = staticPermanentFixture(t, "Enchantment", "Aura", `EnchantPermanent(IsCardtype(Creature)),Activated(PayMana([]),ActionList([`+strings.ReplaceAll(actions, "Ref_TargetPermanent", "HostPermanent")+`]))`)()
				s := g.PutOnBattlefield(c, a.PlayerID())
				g.Attach(s.ID(), target.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), s.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			}
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(target.ID()).Counters[core.Charge] != 3 {
				t.Fatal("counter context")
			}
			wantA, wantB := 26, 20
			if context == "player" || context == "players" {
				wantA, wantB = 20, 26
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life context: %d/%d", a.Life(), b.Life())
			}
		})
	}
}

func TestRONNumericConsumerNonpositive(t *testing.T) {
	for _, n := range []int{-2, 0, 3} {
		g, a, _ := ronGame()
		c := permanentScopeFactory(t, "Artifact", fmt.Sprintf(`Activated(PayMana([]),ActionList([PutCounters([NumberCountersOfTypeOnPermanent(Plus(Integer(%d),Integer(0)),ChargeCounter,ThisPermanent)]),GainLifeForEach(Integer(2),Integer(%d))]))`, n, n))()
		p := g.PutOnBattlefield(c, a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		want := max(n, 0)
		if g.FindPermanent(p.ID()).Counters[core.Charge] != want || a.Life() != 20+2*want {
			t.Fatal("nonpositive amount or multiplication")
		}
	}
}

func TestRONNumericConsumerLivingArmorRetarget(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "DRK", "Living Armor")
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil || len(a.Hand()) != 1 {
		t.Fatal("unpaid casting")
	}
	a.ManaPool().Add(core.Colorless, 4)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.ManaPool().TotalMana() != 0 {
		t.Fatal("casting cost")
	}
	first := g.PutOnBattlefield(mage.NewCreature("First", "{U}", 1, 8), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "{4}{B}", 1, 8), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), c.ID(), 0, []uuid.UUID{first.ID()}); err != nil {
		t.Fatal(err)
	}
	phased := g.PhaseOut(first.ID())
	cp := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
	if cp == nil || len(cp.Targets) != 1 || cp.Targets[0] != second.ID() {
		t.Fatal("retarget failed")
	}
	g.PhaseIn(phased)
	clone := g.Clone()
	g.ResolveTopOfStack()
	if g.FindPermanent(second.ID()).Counters[core.P0P1] != 5 || g.FindPermanent(first.ID()).Counters[core.P0P1] != 0 {
		t.Fatal("copy reused original amount or target")
	}
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		if game.FindPermanent(first.ID()).Counters[core.P0P1] != 1 || game.FindPermanent(second.ID()).Counters[core.P0P1] != 5 {
			t.Fatal("copy/original/clone target identity")
		}
	}
}

func TestRONNumericConsumerMoneychangerPendingIncarnations(t *testing.T) {
	g, a, _ := ronGame()
	g.SetStep(core.Upkeep)
	c := repositoryCard(t, "FEM", "Icatian Moneychanger")
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	c.SetOwner(a.PlayerID())
	first := g.PutOnBattlefield(c, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), boundedActivationIndex(t, first), nil); err != nil {
		t.Fatal(err)
	}
	a.RemoveFromGraveyard(c.ID())
	second := g.PutOnBattlefield(c, a.PlayerID())
	second.AddCounter(core.Credit, 6)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), second.ID(), boundedActivationIndex(t, second), nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		if game.GetPlayer(a.PlayerID()).Life() != 26 || game.FindPermanent(c.ID()) != nil || len(game.GetPlayer(a.PlayerID()).Graveyard()) != 1 {
			t.Fatal("second departure replaced first incarnation counter information")
		}
	}
}

func TestRONNumericConsumerBoundedPlacement(t *testing.T) {
	a := &ronBoundedPlayer{BasePlayer: mage.NewBasePlayer("A"), choice: 3}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	c := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([PutCountersWithRestriction([UptoNumberCountersOfTypeOnPermanent(ManaValueOfPermanent(Ref_TargetPermanent),ChargeCounter,Ref_TargetPermanent)],CantCauseTheNumberOfCountersOfTypeOnPermanentToBeGreaterThan(ChargeCounter,Ref_TargetPermanent,Integer(4)))])))`)()
	s := g.PutOnBattlefield(c, a.PlayerID())
	p := g.PutOnBattlefield(mage.NewCreature("Target", "{4}{G}", 1, 8), a.PlayerID())
	p.AddCounter(core.Charge, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), s.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(p.ID()).Counters[core.Charge] != 4 {
		t.Fatal("dynamic bound or existing cap")
	}
}
