package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func counterStaticFactory(t *testing.T, filter, subject string) mage.CardFactory {
	t.Helper()
	metadata := `Cardtypes:[Creature],Subtypes:[]`
	rules := ""
	if subject == "HostPermanent" {
		metadata = `Cardtypes:[Enchantment],Subtypes:[Aura]`
		rules = `EnchantPermanent(IsCardtype(Creature)),`
	}
	source := fmt.Sprintf(`Card(Name:"Counter condition",Typeline:(Supertypes:[],%s),Rules:[%sIf(PermanentPassesFilter(%s,%s),[PermanentLayerEffect(%s,[AddAbility([Flying])])])],CardPT:(Power:2,Toughness:3))`, metadata, rules, subject, filter, subject)
	nodes, err := Parse(strings.NewReader(source), "counter-static.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Counter condition", factory()); err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONStaticCounterLiveSubjects(t *testing.T) {
	for _, subject := range []string{"ThisPermanent", "HostPermanent"} {
		for _, filter := range []string{`HasACounterOfType(NetCounter)`, `HasNumberCountersOfType(EqualTo(Integer(1)),NetCounter)`} {
			t.Run(subject+"/"+filter, func(t *testing.T) {
				g, a, b := ronGame()
				factory := counterStaticFactory(t, filter, subject)
				card := factory()
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				recipient := source
				if subject == "HostPermanent" {
					recipient = g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 3), b.PlayerID())
					g.Attach(source.ID(), recipient.ID())
				}
				check := func(game *mage.Game, want bool) {
					t.Helper()
					game.ApplyContinuousEffects()
					if got := game.FindPermanent(recipient.ID()).HasKeyword(core.Flying); got != want {
						t.Fatalf("flying %v, want %v", got, want)
					}
				}
				check(g, false)
				g.MutablePermanent(recipient.ID()).Counters[core.Sleep] = 1
				check(g, false)
				g.MutablePermanent(recipient.ID()).Counters[core.Net] = 1
				check(g, true)
				clone := g.Clone()
				g.MutablePermanent(recipient.ID()).Counters[core.Net] = 2
				check(g, strings.HasPrefix(filter, "HasACounter"))
				check(clone, true)
				g.MutablePermanent(recipient.ID()).Counters[core.Net] = 1
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				check(g, true)
				phase := g.PhaseOut(source.ID())
				if subject == "HostPermanent" {
					check(g, false)
				}
				g.PhaseIn(phase)
				check(g, true)
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), source.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				check(g, false)
				g.RunStepWithPriority(core.Cleanup)
				check(g, true)
				second := g.PutOnBattlefield(factory(), a.PlayerID())
				if subject == "HostPermanent" {
					other := g.PutOnBattlefield(mage.NewCreature("Other host", "", 2, 3), a.PlayerID())
					g.Attach(second.ID(), other.ID())
					g.Attach(source.ID(), other.ID())
					check(g, false)
					g.Attach(source.ID(), recipient.ID())
					check(g, true)
				} else if second.HasKeyword(core.Flying) {
					t.Fatal("factory shared counter state")
				}
				g.BouncePermanentToHand(g.FindPermanent(recipient.ID()))
				owner := g.GetPlayer(recipient.Card.Owner())
				owner.RemoveFromHand(recipient.ID())
				g.PutOnBattlefield(recipient.Card, a.PlayerID())
				check(g, false)
				check(clone, true)
			})
		}
	}
}

func TestRONStaticCounterRealClauses(t *testing.T) {
	for _, name := range []string{"Homarid", "Tidal Influence", "Merseine", "Venarian Gold"} {
		t.Run(name, func(t *testing.T) {
			set := "FEM"
			if name == "Venarian Gold" {
				set = "LEG"
			}
			data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
			if err != nil {
				t.Fatal(err)
			}
			nodes, err := Parse(strings.NewReader(string(data)), set+".lines.ron")
			if err != nil {
				t.Fatal(err)
			}
			var record Node
			for _, n := range nodes {
				if n.Fields["Name"].String == name {
					record = n
					break
				}
			}
			if record.Fields == nil {
				t.Fatal("missing complete record")
			}
			if _, err := Compile(record); err == nil {
				t.Fatal("incomplete card became accepted")
			}
			g, a, b := ronGame()
			var options []mage.CardOption
			for _, rule := range record.Fields["Rules"].Args {
				if rule.Name != "If" {
					continue
				}
				ctx := scope{permanent: true, aura: name == "Merseine" || name == "Venarian Gold"}
				if name == "Homarid" || name == "Tidal Influence" {
					build, err := compileConditionalStatic(rule, ctx)
					if err != nil {
						t.Fatal(err)
					}
					options = append(options, build())
				} else {
					condition, err := compileStaticCondition(rule.Args[0], ctx)
					if err != nil {
						t.Fatal(err)
					}
					options = append(options, mage.WithAbility(mage.Static(mage.While(condition, mage.GrantKeywordModification(mage.This, core.Flying)))))
				}
			}
			if len(options) == 0 {
				t.Fatal("no clauses tested")
			}
			var card mage.Card = mage.NewCreature("Clause fixture", "", 2, 3, options...)
			if name == "Merseine" || name == "Venarian Gold" {
				card = mage.NewEnchantment("Clause fixture", "", append(options, mage.WithSubTypes("Aura"))...)
			}
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			blue := g.PutOnBattlefield(mage.NewCreature("Blue", "{U}", 3, 3), b.PlayerID())
			subject := source
			if name == "Venarian Gold" {
				g.Attach(source.ID(), blue.ID())
				subject = blue
			}
			counter := core.Tide
			if name == "Merseine" {
				counter = core.Net
			}
			if name == "Venarian Gold" {
				counter = core.Sleep
			}
			for _, count := range []int{0, 1, 2, 3, 4, 0} {
				g.MutablePermanent(subject.ID()).Counters[counter] = uint8(count)
				g.ApplyContinuousEffects()
				switch name {
				case "Homarid":
					power, toughness := 2, 3
					if count == 1 {
						power--
						toughness--
					}
					if count == 3 {
						power++
						toughness++
					}
					checkConditionalPT(t, g, source.ID(), power, toughness)
				case "Tidal Influence":
					power := 3
					if count == 1 {
						power -= 2
					}
					if count == 3 {
						power += 2
					}
					checkConditionalPT(t, g, blue.ID(), power, 3)
					checkConditionalPT(t, g, source.ID(), 2, 3)
				default:
					if g.FindPermanent(source.ID()).HasKeyword(core.Flying) != (count > 0) {
						t.Fatal("real predicate read wrong counters")
					}
				}
			}
		})
	}
}

func TestRONStaticCounterInvalid(t *testing.T) {
	for _, filter := range []string{
		`HasACounterOfType`, `HasACounterOfType()`, `HasACounterOfType(NetCounter,NetCounter)`, `HasACounterOfType(NetCounter())`, `HasACounterOfType(UnknownCounter)`, `HasACounterOfType(PoisonCounter)`, `HasACounterOfType(PTCounter(3,3))`,
		`HasNumberCountersOfType(EqualTo(Integer(-1)),NetCounter)`, `HasNumberCountersOfType(EqualTo(ValueX),NetCounter)`, `HasNumberCountersOfType(EqualTo(Integer(1)),NetCounter,NetCounter)`, `HasNumberCountersOfType(GreaterThanOrEqualTo(Integer(1)),NetCounter)`, `HasNumberCountersOfType(EqualTo(Integer("1")),NetCounter)`, `HasNumberCountersOfType(EqualTo(Integer(1),Integer(2)),NetCounter)`,
	} {
		assertConditionalRejected(t, `If(PermanentPassesFilter(ThisPermanent,`+filter+`),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,1)])])`)
	}
	for _, ref := range []string{"ThisPermanent()", "HostPermanent", "Ref_TargetPermanent", "Trigger_ThatPermanent", "EachablePermanent"} {
		assertConditionalRejected(t, `If(PermanentPassesFilter(`+ref+`,HasACounterOfType(NetCounter)),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,1)])])`)
	}
}

func TestRONStaticCounterCompositionAndBounds(t *testing.T) {
	for _, tc := range []struct {
		filter string
		count  int
		want   bool
	}{
		{`HasNumberCountersOfType(EqualTo(Integer(0)),NetCounter)`, 0, true},
		{`HasNumberCountersOfType(EqualTo(Integer(0)),NetCounter)`, 1, false},
		{`HasNumberCountersOfType(EqualTo(Integer(255)),NetCounter)`, 255, true},
		{`HasNumberCountersOfType(EqualTo(Integer(256)),NetCounter)`, 0, false},
		{`And([HasACounterOfType(NetCounter),IsCardtype(Creature)])`, 1, true},
		{`And([HasACounterOfType(NetCounter),IsColor(Blue)])`, 1, false},
		{`Or([HasNumberCountersOfType(EqualTo(Integer(3)),NetCounter),And([HasACounterOfType(NetCounter),IsCardtype(Creature)])])`, 2, true},
	} {
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(counterStaticFactory(t, tc.filter, "ThisPermanent")(), a.PlayerID())
		g.MutablePermanent(p.ID()).Counters[core.Net] = uint8(tc.count)
		g.ApplyContinuousEffects()
		if got := g.FindPermanent(p.ID()).HasKeyword(core.Flying); got != tc.want {
			t.Fatalf("%s at %d: %v", tc.filter, tc.count, got)
		}
	}
}

func TestRONStaticCounterResolutionTiming(t *testing.T) {
	rules := `AsPermanentEnters(ThisPermanent,[EntersWithNumberCounters(Integer(0),NetCounter)]),If(PermanentPassesFilter(ThisPermanent,HasACounterOfType(NetCounter)),[PermanentLayerEffect(ThisPermanent,[AddAbility([Flying])])]),Activated(PayMana([ManaCostU]),ActionList([PutCounters([ACounterOfTypeOnPermanent(NetCounter,ThisPermanent)])]))`
	factory := permanentScopeFactory(t, "Artifact", rules)
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, nil); err == nil || g.StackSize() != 0 {
		t.Fatal("unpaid activation changed state")
	}
	a.ManaPool().Add(core.Blue, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(p.ID()).HasKeyword(core.Flying) || a.ManaPool().TotalMana() != 0 {
		t.Fatal("predicate changed before resolution or payment failed")
	}
	clone := g.Clone()
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	g.ApplyContinuousEffects()
	if g.FindPermanent(p.ID()).Counters[core.Net] != 2 || !g.FindPermanent(p.ID()).HasKeyword(core.Flying) {
		t.Fatalf("counter resolutions: count %d, flying %v", g.FindPermanent(p.ID()).Counters[core.Net], g.FindPermanent(p.ID()).HasKeyword(core.Flying))
	}
	if clone.FindPermanent(p.ID()).Counters[core.Net] != 0 || clone.FindPermanent(p.ID()).HasKeyword(core.Flying) {
		t.Fatal("live resolution changed clone")
	}
	clone.ResolveStack()
	clone.ApplyContinuousEffects()
	if clone.FindPermanent(p.ID()).Counters[core.Net] != 1 || !clone.FindPermanent(p.ID()).HasKeyword(core.Flying) {
		t.Fatal("clone resolution did not update static condition")
	}
}

func TestRONStaticCounterInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[If(PermanentPassesFilter(ThisPermanent,And([HasACounterOfType(NetCounter),HasNumberCountersOfType(EqualTo(Integer(3)),TideCounter)])),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,1)])])],CardPT:(Power:2,Toughness:3))`
	report, err := InspectFS(fstest.MapFS{"counter.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Issues) != 0 {
		t.Fatal(report.Issues)
	}
	for name, role := range map[string]string{"NetCounter": "permanent_counter_type", "TideCounter": "permanent_counter_type", "EqualTo": "static_counter_comparison", "Integer": "static_counter_value"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Errorf("missing %s role %s", name, role)
		}
	}
}

func TestRONStaticCounterPlayerConditions(t *testing.T) {
	for _, condition := range []string{
		`PlayerPassesFilter(You,ControlsA(HasACounterOfType(NetCounter)))`,
		`PlayerPassesFilter(You,ControlsNo(HasACounterOfType(NetCounter)))`,
		`APlayerPassesFilter(Opponent,ControlsA(HasNumberCountersOfType(EqualTo(Integer(1)),NetCounter)))`,
	} {
		factory := permanentScopeFactory(t, "Artifact", `If(`+condition+`,[PermanentLayerEffect(ThisPermanent,[AddAbility([Flying])])])`)
		g, a, b := ronGame()
		source := g.PutOnBattlefield(factory(), a.PlayerID())
		member := g.PutOnBattlefield(mage.NewArtifact("Counter member", ""), a.PlayerID())
		if strings.HasPrefix(condition, "APlayer") {
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: member.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
		}
		for _, count := range []int{0, 1, 2, 0} {
			g.MutablePermanent(member.ID()).Counters[core.Net] = uint8(count)
			g.ApplyContinuousEffects()
			want := count > 0
			if strings.Contains(condition, "ControlsNo") {
				want = count == 0
			}
			if strings.HasPrefix(condition, "APlayer") {
				want = count == 1
			}
			if got := g.FindPermanent(source.ID()).HasKeyword(core.Flying); got != want {
				t.Fatalf("%s at %d: %v", condition, count, got)
			}
		}
	}
}
