package ron

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestRONCreatedValueSnapshot(t *testing.T) {
	for _, state := range []string{"live", "phased", "returned", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			f := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([CreateValueX(`+counterInfoValue+`),RemoveCounters([AllCountersOfTypeFromPermanent(ChargeCounter,ThisPermanent)]),GainLife(ValueX)]))]))`)
			card := f()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.MutablePermanent(p.ID()).Counters[core.Charge] = 3
			saved := g.CaptureDamageSource(p.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			if len(g.WaitingDelayedPermanentSources()) != 0 {
				t.Fatal("snapshot ran at announcement")
			}
			g.ResolveStack()
			if state == "phased" {
				g.PhaseOut(p.ID())
			}
			if state == "returned" {
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				q := g.PutOnBattlefield(card, b.PlayerID())
				g.MutablePermanent(q.ID()).Counters[core.Charge] = 9
			}
			for range 3 {
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(saved) || a.Life() != 20 {
					t.Fatal("query lost exact snapshot source or ran body")
				}
			}
			game := g
			if state == "clone" {
				game = g.Clone()
			}
			game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			game.PutTriggersOnStack()
			game.ResolveStack()
			wantA := 23
			if state == "copy" {
				wantA = 20
			}
			if game.GetPlayer(a.PlayerID()).Life() != wantA || len(game.WaitingDelayedPermanentSources()) != 0 {
				t.Fatal("snapshot did not retain pre-removal value")
			}
			if state == "copy" && b.Life() != 23 {
				t.Fatal("copy shared snapshot value")
			}
			if state == "clone" && (a.Life() != 20 || len(g.WaitingDelayedPermanentSources()) != 1) {
				t.Fatal("clone shared frame")
			}
		})
	}
}

func TestRONCreatedValueOrdering(t *testing.T) {
	g, a, _ := ronGame()
	f := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateValueX(Integer(2)),GainLife(ValueX),CreateValueX(Plus(ValueX,Integer(1))),GainLife(ValueX),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([GainLife(ValueX)]))]))`)
	for range 2 {
		p := g.PutOnBattlefield(f(), a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	if a.Life() != 30 || len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("producer order or frame isolation failed")
	}
	clone := g.Clone()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 36 || clone.GetPlayer(a.PlayerID()).Life() != 30 {
		t.Fatal("saved values lost or shared")
	}
}

func TestRONCreatedValueInvalid(t *testing.T) {
	for _, body := range []string{
		`GainLife(ValueX),CreateValueX(Integer(1))`,
		`CreateValueX(ValueX),GainLife(ValueX)`,
		`CreateValueX(),GainLife(ValueX)`,
		`CreateValueX(Integer(1),Integer(2)),GainLife(ValueX)`,
		`CreateValueX(PowerOfPermanent(Ref_TargetPermanent)),GainLife(ValueX)`,
		`CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([CreateValueX(Integer(1))])),GainLife(ValueX)`,
		`If(PermanentPassesFilter(ThisPermanent,IsTapped),[CreateValueX(Integer(1))]),GainLife(ValueX)`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`), "invalid-value.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid producer accepted", body)
		}
	}
}

func TestRONCreatedValuePrimordialOoze(t *testing.T) {
	for _, paid := range []bool{false, true} {
		t.Run(map[bool]string{false: "unpaid", true: "paid"}[paid], func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: paid}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			card := repositoryCard(t, "LEG", "Primordial Ooze")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			if !p.HasAttr(core.AttrMustAttack) {
				t.Fatal("attack requirement missing")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if p.Counters[core.P1P1] != 0 || a.offers != 0 {
				t.Fatal("producer ran before resolution")
			}
			a.ManaPool().Add(core.Colorless, 1)
			g.ResolveStack()
			if g.FindPermanent(p.ID()).Counters[core.P1P1] != 1 || p.CurrentPower(g) != 2 || a.offers != 1 {
				t.Fatal("counter before saved payment failed")
			}
			if paid {
				if a.Life() != 20 || p.Tapped || a.ManaPool().TotalMana() != 0 {
					t.Fatal("paid branch failed")
				}
			} else {
				if a.Life() != 19 || !p.Tapped || a.ManaPool().TotalMana() != 1 {
					t.Fatal("unpaid branch failed")
				}
			}
		})
	}
}

func TestRONCreatedValueProtection(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		color     core.Color
	}{
		{"LEB", "Circle of Protection: Black", core.Black}, {"LEA", "Circle of Protection: Blue", core.Blue}, {"LEA", "Circle of Protection: Green", core.Green}, {"LEA", "Circle of Protection: Red", core.Red}, {"LEA", "Circle of Protection: White", core.White}, {"ATQ", "Circle of Protection: Artifacts", core.Colorless}, {"LEG", "Greater Realm of Preservation", core.Black},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := staticBaseRecord(t, tc.set, tc.name)
			rule := record.Fields["Rules"].Args[0]
			actions := rule.Args[1].Args[0].Args
			plan, err := compileChosenDamageSourceChoice(actions[0])
			if err != nil {
				t.Fatal(err)
			}
			shield, err := compileExactSourcePrevention(actions[1], scope{chosenDamageSource: plan})
			if err != nil {
				t.Fatal(err)
			}
			costs, _, err := compileCosts(rule.Args[0])
			if err != nil {
				t.Fatal(err)
			}
			a := &ronSourceChoicePlayer{BasePlayer: mage.NewBasePlayer("A")}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([CreateValueX(PowerOfPermanent(ThisPermanent))]))]))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.ChangeObjectColor(p.ID(), tc.color)
			saved := g.CaptureDamageSource(p.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			g.BouncePermanentToHand(p)
			binding := mage.BindDamageSource("waiting snapshot")
			bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
			payment := make([]mage.Cost, len(costs))
			for i, cost := range costs {
				payment[i] = cost()
			}
			consumer := mage.NewArtifact("Consumer", "", mage.WithAbility(mage.Activate(mage.Costs(payment...), mage.Sequence(plan.waitingDelayedPermanentChoice(bindings), shield(bindings)))))
			if err := mage.ValidateCardDefinition(consumer.Name(), consumer); err != nil {
				t.Fatal(err)
			}
			q := g.PutOnBattlefield(consumer, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err == nil || a.calls != 0 || g.StackSize() != 0 {
				t.Fatal("unpaid choice changed state")
			}
			a.ManaPool().Add(core.White, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if a.calls != 0 {
				t.Fatal("early source selection")
			}
			g.ResolveStack()
			if a.calls != 1 || !a.source.SameObject(saved) {
				t.Fatal("waiting snapshot body omitted departed source")
			}
			g.DealDamageToPlayerFromSource(a, 2, saved)
			g.DealDamageToPlayerFromSource(a, 1, saved)
			if a.Life() != 19 {
				t.Fatal("shield prevention or consumption failed")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if len(g.WaitingDelayedPermanentSources()) != 0 {
				t.Fatal("snapshot delay remained waiting")
			}
			if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "ChooseADamageSource") {
				t.Fatal("general aggregation was accepted", err)
			}
		})
	}
}

func TestRONCreatedValueInventory(t *testing.T) {
	source := `Card(Name:"Snapshot",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateValueX(Integer(2)),GainLife(ValueX)]))])`
	report, err := InspectFS(fstest.MapFS{"snapshot.ron": &fstest.MapFile{Data: []byte(source)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("snapshot inventory rejected", err)
	}
	if !slices.ContainsFunc(report.Signatures, func(s InventorySignature) bool { return s.Name == "Integer" && s.Role == "value" }) {
		t.Fatal("snapshot operand has no value role")
	}
}

func TestRONCreatedValueAnnouncedX(t *testing.T) {
	g, a, _ := ronGame()
	f := permanentScopeFactory(t, "Artifact", `Activated(PayManaX([ManaCostX],ValueX),ActionList([CreateValueX(Plus(ValueX,Integer(2))),GainLife(ValueX)]))`)
	p := g.PutOnBattlefield(f(), a.PlayerID())
	a.ManaPool().Add(core.Colorless, 3)
	g.SetXValue(3)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 25 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("created value did not shadow announced X")
	}
}

func TestRONCreatedValueOozeAttack(t *testing.T) {
	for _, state := range []string{"able", "sick", "tapped", "phased", "disabled"} {
		t.Run(state, func(t *testing.T) {
			a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			p := g.PutOnBattlefield(repositoryCard(t, "LEG", "Primordial Ooze"), a.PlayerID())
			if state != "sick" {
				g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
			}
			switch state {
			case "tapped":
				g.TapPermanent(p)
			case "phased":
				g.PhaseOut(p.ID())
			case "disabled":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
			}
			g.ApplyContinuousEffects()
			g.RunStepWithPriority(core.DeclareAttackers)
			if g.GetCombat().IsAttacking(p.ID()) != (state == "able") {
				t.Fatal("Ooze ignored attack legality")
			}
		})
	}
}

func TestRONCreatedValueOozePaymentTiming(t *testing.T) {
	for _, state := range []string{"short", "returned", "phased", "control", "copy"} {
		t.Run(state, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			card := repositoryCard(t, "LEG", "Primordial Ooze")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.MutablePermanent(p.ID()).Counters[core.P1P1] = 2
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep triggered")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			a.ManaPool().Add(core.Colorless, 1)
			switch state {
			case "returned":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				q := g.PutOnBattlefield(card, b.PlayerID())
				g.MutablePermanent(q.ID()).Counters[core.P1P1] = 9
			case "phased":
				g.PhaseOut(p.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.ResolveStack()
			damage := 3
			if state == "returned" || state == "phased" {
				damage = 2
			}
			if state == "copy" {
				damage = 7
			}
			if a.Life() != 20-damage || b.Life() != 20 || a.ManaPool().TotalMana() != 1 {
				t.Fatal("wrong saved amount, payer or failed payment", a.Life(), b.Life())
			}
			if state == "returned" && (g.FindPermanent(p.ID()).Counters[core.P1P1] != 9 || g.FindPermanent(p.ID()).Tapped) {
				t.Fatal("old trigger affected returned Ooze")
			}
		})
	}
}
