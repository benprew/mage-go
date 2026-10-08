package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func resolvingCounterClause(t *testing.T, name string) mage.Card {
	t.Helper()
	set, count := "DRK", 1
	if name == "Tourach's Gate" {
		set, count = "FEM", 3
	}
	n := hostUntapRecord(t, set, name)
	rules := n.Fields["Rules"]
	rules.Args = rules.Args[:count]
	n.Fields["Rules"] = rules
	factory, err := Compile(n)
	if err != nil {
		t.Fatal(err)
	}
	c := factory()
	if err := mage.ValidateCardDefinition(name, c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRONResolvingCounterRealClauses(t *testing.T) {
	for _, name := range []string{"Fasting", "Tourach's Gate"} {
		for _, count := range []int{0, 1, 2, 3, 4, 5, 6} {
			t.Run(fmt.Sprintf("%s/%d", name, count), func(t *testing.T) {
				g, a, b := ronGame()
				c := resolvingCounterClause(t, name)
				c.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(c, a.PlayerID())
				counter := core.Hunger
				if name == "Tourach's Gate" {
					counter = core.Time
					land := g.PutOnBattlefield(mage.NewLand("Host"), a.PlayerID())
					g.Attach(p.ID(), land.ID())
				}
				g.MutablePermanent(p.ID()).Counters[counter] = uint8(count)
				g.MutablePermanent(p.ID()).Counters[core.Charge] = 9
				queueRONUpkeep(g, b)
				if g.StackSize() != 0 {
					t.Fatal("opponent upkeep triggered")
				}
				queueRONUpkeep(g, a)
				if g.StackSize() != 1 || int(g.FindPermanent(p.ID()).Counters[counter]) != count {
					t.Fatal("early counter change")
				}
				g.ResolveStack()
				gone := count >= 4
				want := count + 1
				if name == "Tourach's Gate" {
					gone = count <= 1
					want = max(0, count-1)
				}
				live := g.FindPermanent(p.ID())
				if (live == nil) != gone {
					t.Fatal("wrong sequential counter condition")
				}
				if live != nil && (int(live.Counters[counter]) != want || live.Counters[core.Charge] != 9) {
					t.Fatal("wrong counter kind or amount")
				}
			})
		}
	}
}

func TestRONResolvingCounterTimingAndIdentity(t *testing.T) {
	for _, change := range []string{"none", "lower", "raise", "leave", "return", "phase", "phase-cycle", "control", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := resolvingCounterClause(t, "Fasting")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(p.ID()).Counters[core.Hunger] = 4
			queueRONUpkeep(g, a)
			clone := g.Clone()
			switch change {
			case "lower":
				g.MutablePermanent(p.ID()).Counters[core.Hunger] = 3
			case "raise":
				g.MutablePermanent(p.ID()).Counters[core.Hunger] = 5
			case "leave", "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if change == "return" {
					a.RemoveFromHand(c.ID())
					g.PutOnBattlefield(c, a.PlayerID())
					g.MutablePermanent(p.ID()).Counters[core.Hunger] = 8
				}
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{p.ID()})
			live := g.FindPermanent(p.ID())
			wantLive := change == "lower" || change == "return" || change == "phase"
			if (live != nil) != wantLive {
				t.Fatal("wrong source incarnation or current count")
			}
			if change == "return" && live.Counters[core.Hunger] != 8 {
				t.Fatal("old trigger changed returned source")
			}
			if change == "phase" && live.Counters[core.Hunger] != 4 {
				t.Fatal("phased source changed")
			}
			clone.ResolveStack()
			if clone.FindPermanent(p.ID()) != nil {
				t.Fatal("clone lost original pending condition")
			}
		})
	}
}

func TestRONResolvingCounterFilters(t *testing.T) {
	for _, filter := range []string{`HasACounterOfType(TimeCounter)`, `HasNoCountersOfType(TimeCounter)`, `HasNumberCountersOfType(EqualTo(Integer(1)),TimeCounter)`, `HasNumberCountersOfType(GreaterThanOrEqualTo(Integer(2)),TimeCounter)`, `HasNumberCountersOfType(GreaterThanOrEqualTo(Integer(0)),TimeCounter)`, `HasNumberCountersOfType(GreaterThanOrEqualTo(Integer(256)),TimeCounter)`, `And([HasACounterOfType(TimeCounter),HasNoCountersOfType(HungerCounter)])`, `Or([HasNoCountersOfType(TimeCounter),HasACounterOfType(HungerCounter)])`} {
		for _, count := range []int{0, 1, 2, 255} {
			g, a, _ := ronGame()
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([If(PermanentPassesFilter(ThisPermanent,`+filter+`),[GainLife(Integer(1))])]))`)
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			g.MutablePermanent(p.ID()).Counters[core.Time] = uint8(count)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			want := count > 0
			switch filter {
			case `HasNoCountersOfType(TimeCounter)`, `Or([HasNoCountersOfType(TimeCounter),HasACounterOfType(HungerCounter)])`:
				want = count == 0
			case `HasNumberCountersOfType(EqualTo(Integer(1)),TimeCounter)`:
				want = count == 1
			case `HasNumberCountersOfType(GreaterThanOrEqualTo(Integer(2)),TimeCounter)`:
				want = count >= 2
			case `HasNumberCountersOfType(GreaterThanOrEqualTo(Integer(0)),TimeCounter)`:
				want = true
			case `HasNumberCountersOfType(GreaterThanOrEqualTo(Integer(256)),TimeCounter)`:
				want = false
			}
			if (a.Life() == 21) != want {
				t.Fatal("wrong filter", filter, count, a.Life())
			}
		}
	}
}

func TestRONResolvingCounterContextBoundaries(t *testing.T) {
	for _, rule := range []string{
		`Activated(PayMana([]),ActionList([If(PermanentPassesFilter(Ref_TargetPermanent,HasNoCountersOfType(TimeCounter)),[GainLife(Integer(1))])]))`,
		`Activated(PayMana([]),ActionList([If(PermanentPassesFilter(HostPermanent,HasNoCountersOfType(TimeCounter)),[GainLife(Integer(1))])]))`,
		`If(PermanentPassesFilter(ThisPermanent,HasNumberCountersOfType(GreaterThanOrEqualTo(Integer(2)),TimeCounter)),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,1)])])`,
		`Activated(PayMana([]),ActionList([If(PermanentPassesFilter(ThisPermanent,HasNoCountersOfType(TimeCounter)),[AddMana(ManaProduceB)])]))`,
	} {
		source := `Card(Name:"Bad context",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[` + rule + `])`
		nodes, err := Parse(strings.NewReader(source), "counter-context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted unsupported counter context", rule)
		}
	}
	factory := permanentScopeFactory(t, "Artifact", `ActivatedWithModifiers(PayMana([]),ActionList([GainLife(Integer(1))]),ActivateOnlyIf(PermanentPassesFilter(ThisPermanent,HasNoCountersOfType(TimeCounter))))`)
	if err := mage.ValidateCardDefinition("Supported counter restriction", factory()); err != nil {
		t.Fatal(err)
	}

}

func TestRONResolvingCounterInvalid(t *testing.T) {
	for _, filter := range []string{`HasACounterOfType`, `HasNoCountersOfType()`, `HasNoCountersOfType(TimeCounter())`, `HasNoCountersOfType(UnknownCounter)`, `HasNoCountersOfType(TimeCounter,HungerCounter)`, `HasNumberCountersOfType(EqualTo(Integer(-1)),TimeCounter)`, `HasNumberCountersOfType(GreaterThanOrEqualTo(ValueX),TimeCounter)`, `HasNumberCountersOfType(GreaterThanOrEqualTo(Integer(2),Integer(3)),TimeCounter)`, `HasNumberCountersOfType(LessThanOrEqualTo(Integer(2)),TimeCounter)`, `HasNumberCountersOfType(GreaterThanOrEqualTo(Integer()),TimeCounter)`, `HasNumberCountersOfType(GreaterThanOrEqualTo(2),TimeCounter)`, `HasNumberCountersOfType(GreaterThanOrEqualTo(Integer(2)))`, `And([])`} {
		source := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([If(PermanentPassesFilter(ThisPermanent,` + filter + `),[GainLife(Integer(1))])]))])`
		nodes, err := Parse(strings.NewReader(source), "counter-invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", filter)
		}
	}
}

func TestRONResolvingCounterInformation(t *testing.T) {
	for _, change := range []string{"live", "lower", "leave", "return", "phase", "phase-cycle"} {
		t.Run(change, func(t *testing.T) {
			g, a, _ := ronGame()
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([If(PermanentPassesFilter(ThisPermanent,HasNumberCountersOfType(GreaterThanOrEqualTo(Integer(5)),HungerCounter)),[GainLife(Integer(1)),TapPermanent(ThisPermanent)])]))`)
			c := factory()
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(p.ID()).Counters[core.Hunger] = 5
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "lower":
				g.MutablePermanent(p.ID()).Counters[core.Hunger] = 4
			case "leave", "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if change == "return" {
					a.RemoveFromHand(c.ID())
					g.PutOnBattlefield(c, a.PlayerID())
				}
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			}
			g.ResolveStack()
			want := 21
			if change == "lower" {
				want = 20
			}
			if a.Life() != want {
				t.Fatal("counter information lost its original incarnation", a.Life())
			}
			g.PhaseIn([]uuid.UUID{p.ID()})
			if live := g.FindPermanent(p.ID()); live != nil {
				wantTapped := change == "live" || change == "phase-cycle"
				if live.Tapped != wantTapped {
					t.Fatal("information read changed an unavailable object")
				}
			}
		})
	}
}

func TestRONResolvingCounterTargetInstances(t *testing.T) {
	a := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B")}
	g := mage.NewGame(a, b)
	g.SetTurn(2)
	g.SetStep(core.PrecombatMain)
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([If(PermanentPassesFilter(Ref_TargetPermanent,HasNoCountersOfType(TimeCounter)),[TapPermanent(Ref_TargetPermanent),GainLife(Integer(1))])])) )`)
	first, second := factory(), factory()
	if first.Abilities()[0].(*mage.SpellAbility).Targets()[0] == second.Abilities()[0].(*mage.SpellAbility).Targets()[0] {
		t.Fatal("shared factory targets")
	}
	p := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 8), a.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 8), b.PlayerID())
	g.MutablePermanent(q.ID()).Counters[core.Time] = 1
	first.SetOwner(a.PlayerID())
	a.AddToHand(first)
	if err := g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	b.targets = []uuid.UUID{q.ID()}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	clone.MutablePermanent(q.ID()).Counters[core.Time] = 0
	g.ResolveStack()
	clone.ResolveStack()
	if !g.FindPermanent(p.ID()).Tapped || g.FindPermanent(q.ID()).Tapped || a.Life() != 21 || b.Life() != 20 {
		t.Fatal("copy lost its own target or count")
	}
	if !clone.FindPermanent(q.ID()).Tapped || clone.GetPlayer(b.PlayerID()).Life() != 21 {
		t.Fatal("clone lost its own counter condition")
	}
}

func TestRONResolvingCounterClauseCasting(t *testing.T) {
	g, a, _ := ronGame()
	c := resolvingCounterClause(t, "Fasting")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("unpaid cast did not roll back")
	}
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(c.ID()) != nil || a.ManaPool().TotalMana() != 0 {
		t.Fatal("casting timing or payment")
	}
	g.ResolveStack()
	for turn := 1; turn <= 5; turn++ {
		g.RunStepWithPriority(core.Upkeep)
		p := g.FindPermanent(c.ID())
		if turn == 5 {
			if p != nil {
				t.Fatal("fifth upkeep did not destroy source")
			}
		} else if p == nil || int(p.Counters[core.Hunger]) != turn {
			t.Fatal("upkeep progression", turn)
		}
	}
}

func TestRONResolvingCounterStrictInventory(t *testing.T) {
	source := `Card(Name:"Counter",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([If(PermanentPassesFilter(ThisPermanent,And([HasNoCountersOfType(TimeCounter),HasNumberCountersOfType(GreaterThanOrEqualTo(Integer(5)),HungerCounter)])),[GainLife(Integer(1))])]))])`
	fs := fstest.MapFS{"counter.ron": {Data: []byte(source)}}
	report, err := InspectFS(fs, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal(err, report.Issues)
	}
	foundCounterValue := false
	for _, signature := range report.Signatures {
		if signature.Name == "TimeCounter" && signature.Role != "permanent_counter_type" {
			t.Fatal("unknown counter role")
		}
		if signature.Name == "GreaterThanOrEqualTo" && signature.Role != "static_counter_comparison" {
			t.Fatal("wrong comparison role")
		}
		if signature.Name == "Integer" && signature.Role == "static_counter_value" {
			foundCounterValue = true
		}
	}
	if !foundCounterValue {
		t.Fatal("missing counter value role")
	}
	fs["gap.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Gap",Rules:[Unsupported])`)}
	if registry, err := LoadFS(fs, "*.ron"); err == nil || registry != nil {
		t.Fatal("published partial registry")
	}
	if _, err := Compile(hostUntapRecord(t, "DRK", "Fasting")); err != nil {
		t.Fatal("Fasting counter rules failed", err)
	}
	for _, tc := range []struct{ set, name, next string }{{"FEM", "Tourach's Gate", "unsupported cost reference (HostPermanent)"}} {
		_, err := Compile(hostUntapRecord(t, tc.set, tc.name))
		if err == nil || !strings.Contains(err.Error(), tc.next) {
			t.Fatal("wrong full-record blocker", tc.name, err)
		}
	}
}
