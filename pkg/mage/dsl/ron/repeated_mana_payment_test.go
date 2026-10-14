package ron

import (
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronRepeatedPaymentClonePlayer struct{ mage.Player }

func (p *ronRepeatedPaymentClonePlayer) ChooseMayAbility(string) bool { return true }

func TestRONRepeatedManaPaymentCyclone(t *testing.T) {
	for _, scenario := range []string{"paid", "declined", "short", "wrong-color", "mana-source", "prevent"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario != "declined"}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			c := repositoryCard(t, "ARN", "Cyclone")
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast succeeded")
			}
			a.ManaPool().Add(core.Green, 2)
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(c.ID())
			g.MutablePermanent(p.ID()).Counters[core.Wind] = 1
			one := g.PutOnBattlefield(mage.NewCreature("One", "", 1, 4), a.PlayerID())
			two := g.PutOnBattlefield(mage.NewCreature("Two", "", 1, 4), b.PlayerID())
			switch scenario {
			case "wrong-color":
				a.ManaPool().Add(core.Blue, 2)
			case "mana-source":
				for range 2 {
					g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
				}
			default:
				a.ManaPool().Add(core.Green, 1)
				if scenario != "short" {
					a.ManaPool().Add(core.Green, 1)
				}
			}
			queueRONUpkeep(g, b)
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep triggered Cyclone")
			}
			queueRONUpkeep(g, a)
			if scenario == "prevent" {
				g.AddPreventionShield(a.PlayerID(), 1)
			}
			callbacks := 0
			g.SetOnDamageDealt(func(source, target string, amount int, combat bool) {
				callbacks++
				wantA := 18
				if scenario == "prevent" {
					wantA = 19
				}
				if source != "Cyclone" || combat || a.Life() != wantA || b.Life() != 18 || g.FindPermanent(one.ID()).Damage != 2 || g.FindPermanent(two.ID()).Damage != 2 {
					t.Errorf("incomplete Cyclone damage batch: %s -> %s", source, target)
				}
			})
			if g.FindPermanent(p.ID()).Counters[core.Wind] != 1 {
				t.Fatal("counter placed before resolution")
			}
			g.ResolveStack()
			paid := scenario == "paid" || scenario == "mana-source" || scenario == "prevent"
			wantA := 18
			if scenario == "prevent" {
				wantA = 19
			}
			if paid {
				if callbacks != 4 {
					t.Fatal("wrong damage observations")
				}
				if g.FindPermanent(p.ID()) == nil || g.FindPermanent(p.ID()).Counters[core.Wind] != 2 || a.Life() != wantA || b.Life() != 18 || g.FindPermanent(one.ID()).Damage != 2 || g.FindPermanent(two.ID()).Damage != 2 || a.ManaPool().TotalMana() != 0 {
					t.Fatal("wrong paid Cyclone result")
				}
			} else if g.FindPermanent(p.ID()) != nil || a.Life() != 20 || b.Life() != 20 || g.FindPermanent(one.ID()).Damage != 0 || g.FindPermanent(two.ID()).Damage != 0 {
				t.Fatal("unpaid branch did not sacrifice without damage")
			}
		})
	}
}

func TestRONRepeatedManaPaymentInvalid(t *testing.T) {
	pair := `MayCost(PayManaForEach([ManaCostG],Integer(2))),IfElse(CostWasPaid,[GainLife(Integer(3))],[GainLife(Integer(1))])`
	for _, actions := range []string{
		strings.Replace(pair, "[ManaCostG]", "[ManaCostX]", 1),
		strings.Replace(pair, "[ManaCostG]", "[ManaCostGeneric(-1)]", 1),
		strings.Replace(pair, "[ManaCostG]", "ManaCostG", 1),
		strings.Replace(pair, "Integer(2)", "Integer(-1)", 1),
		strings.Replace(pair, "Integer(2)", "TheTotalNumberOfCountersOfTypeAmongPermanents(WindCounter,SinglePermanent(Ref_TargetPermanent))", 1),
		strings.Replace(pair, "CostWasPaid", "CostWasPaid()", 1),
		strings.Replace(pair, "[GainLife(Integer(1))]", "[]", 1),
		strings.Replace(pair, "MayCost", "MustCost", 1),
		`MayCost(PayManaForEach([ManaCostG],Integer(2)))`,
		pair + `,If(CostWasPaid,[GainLife(Integer(99))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+actions+`]))])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", actions)
		}
	}
}

func TestRONRepeatedManaPaymentIdentity(t *testing.T) {
	for _, state := range []string{"live", "changed", "departed", "returned", "phased", "cycle", "control", "copy"} {
		t.Run(state, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([MayCost(PayManaForEach([ManaCostG],`+counterInfoValue+`)),IfElse(CostWasPaid,[GainLife(Integer(3))],[GainLife(Integer(1))])]))`)
			c := factory()
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(p.ID()).Counters[core.Charge] = 2
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			wantCost := 2
			switch state {
			case "changed":
				g.MutablePermanent(p.ID()).Counters[core.Charge] = 3
				wantCost = 3
			case "departed", "returned":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if state == "returned" {
					a.RemoveFromHand(c.ID())
					q := g.PutOnBattlefield(c, b.PlayerID())
					g.MutablePermanent(q.ID()).Counters[core.Charge] = 8
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
				g.MutablePermanent(p.ID()).Counters[core.Charge] = 3
				wantCost = 3
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: c.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			a.ManaPool().Add(core.Green, 3)
			b.ManaPool().Add(core.Green, 3)
			clone := g.Clone()
			clone.SetPlayerAt(0, &ronRepeatedPaymentClonePlayer{clone.GetPlayer(a.PlayerID())})
			clone.SetPlayerAt(1, &ronRepeatedPaymentClonePlayer{clone.GetPlayer(b.PlayerID())})
			g.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 3 {
				t.Fatal("clone shared payment")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if game.GetPlayer(a.PlayerID()).Life() != 23 || game.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 3-wantCost {
					t.Fatalf("payment used wrong visit, count or controller: life=%d mana=%d offers=%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(a.PlayerID()).ManaPool().TotalMana(), a.offers)
				}
				wantB, manaB := 20, 3
				if state == "copy" {
					wantB, manaB = 23, 1
				}
				if game.GetPlayer(b.PlayerID()).Life() != wantB || game.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != manaB {
					t.Fatal("copy or control changed payer frame")
				}
			}
		})
	}
}

func TestRONRepeatedManaPaymentBranches(t *testing.T) {
	for _, continuation := range []string{`If(CostWasPaid,[GainLife(Integer(3))])`, `Unless(CostWasntPaid,[GainLife(Integer(3))])`, `IfElse(CostWasntPaid,[GainLife(Integer(99))],[GainLife(Integer(3))])`} {
		a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		body := `PlayerMayCost(You,PayManaForEach([ManaCostG,ManaCostGeneric(1)],Integer(2))),` + continuation + `,MayCost(PayManaForEach([ManaCostW],Integer(1))),Unless(CostWasPaid,[GainLife(Integer(2))])`
		c := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([`+body+`]))`)()
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
		a.ManaPool().Add(core.Green, 2)
		a.ManaPool().Add(core.Colorless, 2)
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if a.Life() != 25 || a.ManaPool().TotalMana() != 0 || a.offers != 1 {
			t.Fatal("branch polarity or independent result failed")
		}
	}
}

func TestRONRepeatedManaPaymentDelayed(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	g := mage.NewGame(a, mage.NewBasePlayer("B"))
	c := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([MayCost(PayManaForEach([ManaCostG],`+counterInfoValue+`)),IfElse(CostWasPaid,[GainLife(Integer(3))],[GainLife(Integer(1))])]))]))`)()
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	g.MutablePermanent(p.ID()).Counters[core.Charge] = 2
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	a.ManaPool().Add(core.Green, 2)
	for range 2 {
		sources := g.WaitingDelayedPermanentSources()
		if len(sources) != 1 || a.offers != 0 || a.ManaPool().TotalMana() != 2 {
			t.Fatal("delayed query lost amount reference or offered payment")
		}
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	g.RunStepWithPriority(core.EndStep)
	if a.Life() != 23 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("delayed payment lost saved visit")
	}
}

func TestRONRepeatedManaPaymentStrictLoad(t *testing.T) {
	record := repositoryRecord(t, "ARN", "Cyclone", "cyclone.ron")
	data, err := os.ReadFile("../../../../cards/mtgish/ARN.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	source := strings.Split(string(data), "\n")[record.Position.Line-1]
	files := fstest.MapFS{"cyclone.ron": {Data: []byte(source)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("complete Cyclone load", err)
	}
	if _, err := Compile(record); err != nil {
		t.Fatal(err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	for name, role := range map[string]string{"PayManaForEach": "resolution_cost", "ManaCostG": "metadata", "TheTotalNumberOfCountersOfTypeAmongPermanents": "value"} {
		if !slices.ContainsFunc(report.Signatures, func(signature InventorySignature) bool { return signature.Name == name && signature.Role == role }) {
			t.Fatal("missing role", name, role)
		}
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(source, "PayManaForEach([ManaCostG]", "PayManaForEach([ManaCostX]", 1))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection changed registry")
	}
}

func TestRONRepeatedManaPaymentCycloneLifecycle(t *testing.T) {
	for _, state := range []string{"departed", "returned", "phased", "control", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			c := repositoryCard(t, "ARN", "Cyclone")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(p.ID()).Counters[core.Wind] = 2
			queueRONUpkeep(g, a)
			wantDamage, wantCost := 3, 3
			switch state {
			case "departed", "returned":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				wantDamage, wantCost = 2, 2
				if state == "returned" {
					a.RemoveFromHand(c.ID())
					q := g.PutOnBattlefield(c, b.PlayerID())
					g.MutablePermanent(q.ID()).Counters[core.Wind] = 9
				}
			case "phased":
				g.PhaseOut(p.ID())
				wantDamage, wantCost = 2, 2
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: c.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				wantDamage, wantCost = 7, 4
			}
			a.ManaPool().Add(core.Green, 4)
			b.ManaPool().Add(core.Green, 3)
			clone := g.Clone()
			clone.SetPlayerAt(0, &ronRepeatedPaymentClonePlayer{clone.GetPlayer(a.PlayerID())})
			clone.SetPlayerAt(1, &ronRepeatedPaymentClonePlayer{clone.GetPlayer(b.PlayerID())})
			g.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 4 {
				t.Fatal("cloned trigger shared outcome")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if game.GetPlayer(a.PlayerID()).Life() != 20-wantDamage || game.GetPlayer(b.PlayerID()).Life() != 20-wantDamage || game.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 4-wantCost {
					t.Fatal("Cyclone trigger lost exact source or payer")
				}
				if state == "returned" && game.FindPermanent(c.ID()).Counters[core.Wind] != 9 {
					t.Fatal("old trigger changed new source visit")
				}
				if state == "copy" && game.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 0 {
					t.Fatal("copy did not pay separate count")
				}
			}
		})
	}
}
