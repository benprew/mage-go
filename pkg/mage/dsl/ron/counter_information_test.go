package ron

import (
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const counterInfoValue = `TheTotalNumberOfCountersOfTypeAmongPermanents(ChargeCounter,SinglePermanent(ThisPermanent))`

func TestRONCounterInformationSource(t *testing.T) {
	for _, change := range []string{"live", "change", "leave", "return", "phase", "phase-cycle", "copy", "sacrifice"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			cost := `PayMana([])`
			if change == "sacrifice" {
				cost = `SacrificePermanent(ThisPermanent)`
			}
			f := permanentScopeFactory(t, "Artifact", `Activated(`+cost+`,ActionList([GainLife(`+counterInfoValue+`)]))`)
			c := f()
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(p.ID()).Counters[core.Charge] = 3
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			want := 23
			switch change {
			case "change":
				g.MutablePermanent(p.ID()).Counters[core.Charge] = 5
				want = 25
			case "leave", "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if change == "return" {
					a.RemoveFromHand(c.ID())
					q := g.PutOnBattlefield(c, b.PlayerID())
					g.MutablePermanent(q.ID()).Counters[core.Charge] = 11
				}
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
				g.MutablePermanent(p.ID()).Counters[core.Charge] = 5
				want = 25
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone changed")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if game.GetPlayer(a.PlayerID()).Life() != want {
					t.Fatal("wrong information count")
				}
				wantB := 20
				if change == "copy" {
					wantB = 23
				}
				if game.GetPlayer(b.PlayerID()).Life() != wantB {
					t.Fatal("wrong copy controller")
				}
			}
		})
	}
}

func TestRONCounterInformationClock(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "ATQ", "Armageddon Clock")
	c.SetOwner(a.PlayerID())
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(c, a.PlayerID())
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if g.FindPermanent(p.ID()).Counters[core.Doom] != 1 {
		t.Fatal("upkeep did not add doom")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 19 {
		t.Fatal("wrong clock damage")
	}
}

func TestRONCounterInformationInvalid(t *testing.T) {
	for _, value := range []string{`TheTotalNumberOfCountersOfTypeAmongPermanents`, `TheTotalNumberOfCountersOfTypeAmongPermanents()`, `TheTotalNumberOfCountersOfTypeAmongPermanents(ChargeCounter)`, `TheTotalNumberOfCountersOfTypeAmongPermanents(ChargeCounter(),SinglePermanent(ThisPermanent))`, `TheTotalNumberOfCountersOfTypeAmongPermanents(UnknownCounter,SinglePermanent(ThisPermanent))`, `TheTotalNumberOfCountersOfTypeAmongPermanents(ChargeCounter,SinglePermanent())`, `TheTotalNumberOfCountersOfTypeAmongPermanents(ChargeCounter,SinglePermanent(ThisPermanent()))`, `TheTotalNumberOfCountersOfTypeAmongPermanents(ChargeCounter,SinglePermanent(Ref_TargetPermanent))`, `TheTotalNumberOfCountersOfTypeAmongPermanents(ChargeCounter,SinglePermanent(HostPermanent))`, `TheTotalNumberOfCountersOfTypeAmongPermanents(ChargeCounter,EachPermanent(AnyPermanent))`} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([GainLife(` + value + `)]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-counter.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid value accepted", value)
		}
	}
}

func TestRONCounterInformationClockPayment(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "ATQ", "Armageddon Clock")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("unpaid cast changed state")
	}
	a.ManaPool().Add(core.Colorless, 6)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(c.ID())
	g.MutablePermanent(p.ID()).Counters[core.Doom] = 2
	g.MutablePermanent(p.ID()).Counters[core.Charge] = 9
	b.ManaPool().Add(core.Colorless, 4)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 2, nil); err == nil || b.ManaPool().TotalMana() != 4 || g.StackSize() != 0 {
		t.Fatal("non-upkeep activation changed state")
	}
	g.SetStep(core.Upkeep)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, nil); err == nil || g.FindPermanent(p.ID()).Counters[core.Doom] != 2 {
		t.Fatal("unpaid activation changed counters")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 2, nil); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().TotalMana() != 0 || g.FindPermanent(p.ID()).Counters[core.Doom] != 2 || g.StackPeek().Controller != b.PlayerID() {
		t.Fatal("payment or early removal")
	}
	g.ResolveStack()
	if g.FindPermanent(p.ID()).Counters[core.Doom] != 1 || g.FindPermanent(p.ID()).Counters[core.Charge] != 9 {
		t.Fatal("wrong counter removed")
	}
	a.ManaPool().Add(core.Colorless, 4)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, nil); err != nil {
		t.Fatal(err)
	}
	g.CounterStackObject(g.StackPeek().ID)
	g.ResolveStack()
	if a.ManaPool().TotalMana() != 0 || g.FindPermanent(p.ID()).Counters[core.Doom] != 1 {
		t.Fatal("countered activation refunded or removed counter")
	}
}

func TestRONCounterInformationClockLifecycle(t *testing.T) {
	for _, change := range []string{"live", "change", "leave", "return", "second-departure", "phase", "phase-cycle", "control", "copy", "prevent", "zero", "remove-ability", "phase-before", "opponent", "two"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "ATQ", "Armageddon Clock")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(p.ID()).Counters[core.Doom] = 3
			if change == "two" {
				other := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Armageddon Clock"), a.PlayerID())
				g.MutablePermanent(other.ID()).Counters[core.Doom] = 2
			}
			if change == "phase-before" {
				g.PhaseOut(p.ID())
			}
			who := a.PlayerID()
			if change == "opponent" {
				who = b.PlayerID()
			}
			g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: who})
			g.PutTriggersOnStack()
			if change == "phase-before" || change == "opponent" {
				if g.StackSize() != 0 {
					t.Fatal("ineligible event detected")
				}
				return
			}
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("early damage")
			}
			wantA, wantB := 17, 17
			switch change {
			case "change":
				g.MutablePermanent(p.ID()).Counters[core.Doom] = 5
				wantA, wantB = 15, 15
			case "zero":
				g.MutablePermanent(p.ID()).Counters[core.Doom] = 0
				wantA, wantB = 20, 20
			case "leave", "return", "second-departure":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if change != "leave" {
					a.RemoveFromHand(c.ID())
					q := g.PutOnBattlefield(c, b.PlayerID())
					g.MutablePermanent(q.ID()).Counters[core.Doom] = 11
					if change == "second-departure" {
						g.BouncePermanentToHand(g.FindPermanent(q.ID()))
					}
				}
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
				g.MutablePermanent(p.ID()).Counters[core.Doom] = 5
				wantA, wantB = 15, 15
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				wantA, wantB = 14, 14
			case "prevent":
				g.AddPreventionShield(b.PlayerID(), 2)
				wantB = 19
			case "remove-ability":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			case "two":
				wantA, wantB = 15, 15
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone damaged original")
			}
			g.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB {
					t.Fatalf("life=%d/%d want=%d/%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life(), wantA, wantB)
				}
			}
		})
	}
}

func TestRONCounterInformationTargets(t *testing.T) {
	for _, change := range []string{"live", "exile-in-body", "return", "phase", "shroud"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			victim := mage.NewCreature("Victim", "", 2, 8)
			victim.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(victim, b.PlayerID())
			g.MutablePermanent(p.ID()).Counters[core.Charge] = 4
			value := strings.ReplaceAll(counterInfoValue, "ThisPermanent", "Ref_TargetPermanent")
			body := `GainLife(` + value + `)`
			if change == "exile-in-body" {
				body = `Exile([Permanent(Ref_TargetPermanent)],[]),` + body
			}
			c := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([`+body+`])))`)()
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			want := 24
			switch change {
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				b.RemoveFromHand(victim.ID())
				q := g.PutOnBattlefield(victim, b.PlayerID())
				g.MutablePermanent(q.ID()).Counters[core.Charge] = 11
				want = 20
			case "phase":
				g.PhaseOut(p.ID())
				want = 20
			case "shroud":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
				want = 20
			}
			g.ResolveStack()
			if a.Life() != want {
				t.Fatal("target legality or LKI count")
			}
		})
	}
}

func TestRONCounterInformationInventoryAndBoundaries(t *testing.T) {
	n := hostUntapRecord(t, "ATQ", "Armageddon Clock")
	data, err := os.ReadFile("../../../../cards/mtgish/ATQ.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	record := strings.Split(string(data), "\n")[n.Position.Line-1]
	files := fstest.MapFS{"clock.ron": &fstest.MapFile{Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("registry count")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory compilation", err)
	}
	for name, role := range map[string]string{"TheTotalNumberOfCountersOfTypeAmongPermanents": "value", "DoomCounter": "permanent_counter_type", "SinglePermanent": "permanent_filter", "ThisPermanent": "permanent_reference"} {
		if !slices.ContainsFunc(report.Signatures, func(s InventorySignature) bool { return s.Name == name && s.Role == role }) {
			t.Fatal("missing semantic role", name, role)
		}
	}
	for _, tc := range []struct{ set, name, gap string }{{"ARN", "Cyclone", "PayManaForEach"}} {
		if _, err := Compile(hostUntapRecord(t, tc.set, tc.name)); err == nil || !strings.Contains(err.Error(), tc.gap) {
			t.Fatalf("%s gap: %v", tc.name, err)
		}
	}
	for _, tc := range []struct{ set, name string }{{"FEM", "Icatian Moneychanger"}, {"LEG", "Voodoo Doll"}, {"DRK", "City of Shadows"}} {
		if _, err := Compile(hostUntapRecord(t, tc.set, tc.name)); err != nil {
			t.Fatalf("supported counter information consumer %s: %v", tc.name, err)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(record, "SinglePermanent(ThisPermanent)", "SinglePermanent(Ref_TargetPermanent)"))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection published")
	}
}

func TestRONCounterInformationClockNativeTiming(t *testing.T) {
	g, a, b := ronGame()
	g.SetTurn(2)
	a.AddToLibrary(mage.NewLand("Draw A"))
	b.AddToLibrary(mage.NewLand("Draw B"))
	p := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Armageddon Clock"), a.PlayerID())
	g.MutablePermanent(p.ID()).Counters[core.Doom] = 3
	first := true
	g.SetOnPriority(func(game *mage.Game, _ int, _ bool) mage.PriorityAction {
		if first {
			first = false
			if len(a.Hand()) != 1 || game.StackSize() != 1 || a.Life() != 20 || b.Life() != 20 {
				t.Fatal("draw or damage timing before priority")
			}
		}
		return mage.PriorityAction{Type: mage.PriorityPass}
	})
	g.RunStepWithPriority(core.Draw)
	if first || a.Life() != 17 || b.Life() != 17 {
		t.Fatalf("native Clock first=%t life=%d/%d stack=%d counters=%v", first, a.Life(), b.Life(), g.StackSize(), g.FindPermanent(p.ID()).Counters)
	}
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.Draw)
	if len(b.Hand()) != 1 || a.Life() != 17 || b.Life() != 17 {
		t.Fatal("opponent draw triggered Clock")
	}
}

func TestRONCounterInformationHostAndEvent(t *testing.T) {
	for _, context := range []string{"host", "event"} {
		t.Run(context, func(t *testing.T) {
			g, a, b := ronGame()
			subject := "HostPermanent"
			rule := `Activated(PayMana([]),ActionList([GainLife(` + strings.ReplaceAll(counterInfoValue, "ThisPermanent", subject) + `)]))`
			var f mage.CardFactory
			if context == "host" {
				src := `Card(Name:"Host probe",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(AnyPermanent),` + rule + `])`
				nodes, err := Parse(strings.NewReader(src), "host-probe.ron")
				if err != nil {
					t.Fatal(err)
				}
				f, err = Compile(nodes[0])
				if err != nil {
					t.Fatal(err)
				}
			} else {
				subject = "Trigger_ThatPermanent"
				rule = `TriggerA(WhenAPermanentBecomesTapped(IsCardtype(Artifact)),ActionList([GainLife(` + strings.ReplaceAll(counterInfoValue, "ThisPermanent", subject) + `)]))`
				f = permanentScopeFactory(t, "Artifact", rule)
			}
			c := f()
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(c, a.PlayerID())
			victim := mage.NewArtifact("Host", "")
			victim.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(victim, b.PlayerID())
			g.MutablePermanent(p.ID()).Counters[core.Charge] = 4
			if context == "host" {
				g.Attach(source.ID(), p.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			} else {
				g.TapPermanent(g.FindPermanent(p.ID()))
				g.PutTriggersOnStack()
			}
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			g.ResolveStack()
			if a.Life() != 24 || b.Life() != 20 {
				t.Fatal("typed information reference lost")
			}
		})
	}
}

func TestRONCounterInformationReplacementOrder(t *testing.T) {
	for _, index := range []int{0, 1} {
		a := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("A"), index: index}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		value := strings.ReplaceAll(counterInfoValue, "ChargeCounter", "PTCounter(1,1)")
		c := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([PutCounters([ACounterOfTypeOnPermanent(PTCounter(1,1),ThisPermanent)]),GainLife(`+value+`)]))`)()
		p := g.PutOnBattlefield(c, a.PlayerID())
		g.AddReplacementEffect(&counterPlacementReplacement{source: p.ID()})
		g.AddCounterDoubler(p.ID(), core.P1P1, mage.AnyPermanent)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		want := 4 - index
		if g.FindPermanent(p.ID()).Counters[core.P1P1] != want || a.Life() != 20+want || a.choices != 1 {
			t.Fatal("value read before completed replacement pipeline")
		}
	}
}

func TestRONCounterInformationClockBatch(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Armageddon Clock"), a.PlayerID())
	g.MutablePermanent(p.ID()).Counters[core.Doom] = 3
	callbacks := 0
	g.SetOnDamageDealt(func(_, _ string, amount int, combat bool) {
		callbacks++
		if a.Life() != 17 || b.Life() != 17 || amount != 3 || combat {
			t.Error("damage was not one simultaneous batch")
		}
	})
	g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if callbacks != 2 {
		t.Fatal("wrong batch recipients")
	}
}
