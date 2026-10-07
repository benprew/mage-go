package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type counterPaymentClonePayer struct {
	mage.Player
	accept bool
}

func (p *counterPaymentClonePayer) ChooseMayAbility(string) bool { return p.accept }

func TestRONCounterPlacementPaymentChants(t *testing.T) {
	for _, tc := range []struct{ name, land string }{{"Thelon's Chant", "Swamp"}, {"Tourach's Chant", "Forest"}} {
		for _, state := range []string{"pay", "decline", "missing", "opponent", "phase", "source-leaves", "source-returned", "source-phased", "control", "copy", "doubled", "creature-control", "phase-cycle", "shroud", "wrong-land", "unknown-actor", "disabled", "prevent-damage"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: true}
				b := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("B")}, accept: state != "decline"}
				g := mage.NewGame(a, b)
				g.SetTurn(2)
				c := repositoryCard(t, "FEM", tc.name)
				if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
					t.Fatal(err)
				}
				c.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(c, a.PlayerID())
				owner := b.PlayerID()
				if state == "opponent" {
					owner = a.PlayerID()
				}
				victim := g.PutOnBattlefield(mage.NewCreature("Payment", "", 2, 5), owner)
				b.choice = victim.ID()
				if state == "phase" {
					g.PhaseOut(victim.ID())
				}
				if state == "missing" {
					g.RemoveFromBattlefield(victim)
				}
				if state == "shroud" {
					g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
				}
				if state == "disabled" {
					g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
				}
				landName := tc.land
				if state == "wrong-land" {
					landName = "Island"
				}
				land := repositoryCard(t, "LEB", landName)
				land.SetOwner(b.PlayerID())
				if state == "unknown-actor" {
					g.PutOnBattlefield(land, b.PlayerID())
				} else {
					g.PutOnBattlefieldBy(land, b.PlayerID(), b.PlayerID())
				}
				g.PutTriggersOnStack()
				if state == "wrong-land" || state == "unknown-actor" || state == "disabled" {
					if g.StackSize() != 0 || b.offers != 0 {
						t.Fatal("ineligible event")
					}
					return
				}
				if g.StackSize() != 1 || b.offers != 0 || b.choices != 0 || b.Life() != 20 {
					t.Fatal("event timing")
				}
				switch state {
				case "source-leaves", "source-returned":
					g.RemoveFromBattlefield(source)
					if state == "source-returned" {
						g.PutOnBattlefield(c, b.PlayerID())
					}
				case "source-phased":
					g.PhaseOut(source.ID())
				case "creature-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(victim.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy")
					}
				case "doubled":
					g.AddCounterDoubler(source.ID(), core.M1M1, mage.IsCreature)
				}
				if state == "prevent-damage" {
					b.accept = false
					g.AddPreventionShield(b.PlayerID(), 2)
				}
				clone := g.Clone()
				clone.SetPlayerAt(1, &counterPaymentClonePayer{clone.GetPlayer(b.PlayerID()), b.accept})
				g.ResolveStack()
				paid := state != "decline" && state != "missing" && state != "opponent" && state != "phase" && state != "creature-control" && state != "prevent-damage"
				want := 20
				if !paid {
					want = 17
					if state == "prevent-damage" {
						want = 19
					}
				}
				if b.Life() != want || a.Life() != 20 {
					t.Fatal("wrong event payer or damage branch")
				}
				if clone.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("clone changed")
				}
				clone.ResolveStack()
				if clone.GetPlayer(b.PlayerID()).Life() != want {
					t.Fatal("clone lost payment")
				}
				if paid {
					count := 1
					if state == "copy" || state == "doubled" {
						count = 2
					}
					if int(g.FindPermanent(victim.ID()).Counters[core.M1M1]) != count {
						t.Fatal("wrong placement")
					}
				}
			})
		}
	}
}

func TestRONCounterPlacementPaymentInvalid(t *testing.T) {
	cost := `PutCounters([ACounterOfTypeOnAPermanent(PTCounter(-1,-1),And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))]))])`
	for _, rule := range []string{
		`Activated(PayMana([]),ActionList([MayCost(` + cost + `),If(CostWasPaid,[GainLife(Integer(1))])]))`,
	} {
		permanentScopeFactory(t, "Artifact", rule)
	}
	for _, cost := range []string{
		`PutCounters([])`,
		`PutCounters()`,
		`PutCounters([ACounterOfTypeOnAPermanent(PTCounter(-1,-1),And([IsCardtype(Land),ControlledByAPlayer(SinglePlayer(You))]))])`,
		`PutCounters([ACounterOfTypeOnAPermanent(PTCounter(-1,-1),And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(Opponent))]))])`,
		`PutCounters([ACounterOfTypeOnAPermanent(PTCounter(-1,-1),And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(Trigger_ThatPlayer))]))])`,
		`PutCounters([ACounterOfTypeOnAPermanent(PTCounter(-1,-1),And([IsCardtype(Creature),IsCardtype(Creature)]))])`,
		`PutCounters([ACounterOfTypeOnAPermanent(PTCounter(-1,-1),And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You)),IsTapped]))])`,
		`PutCounters([ACounterOfTypeOnPermanent(PTCounter(-1,-1),ThisPermanent)])`,
		`PutCounters([NumberCountersOfTypeOnAPermanent(Integer(2),PTCounter(-1,-1),IsCardtype(Creature))])`,
		`PutCounters([ACounterOfTypeOnAPermanent(PTCounter(8,8),And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))]))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([MayCost(`+cost+`),If(CostWasPaid,[GainLife(Integer(1))])]))])`), "invalid-counter-payment.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", cost)
		}
	}
	for _, body := range []string{`MayCost(` + cost + `)`, `If(CostWasPaid,[GainLife(Integer(1))])`, `MayCost(` + cost + `),If(CostWasPaid,[] )`, `MayCost(` + cost + `),If(CostWasPaid,[GainLife(Integer(1))]),If(CostWasPaid,[GainLife(Integer(1))])`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`), "invalid-counter-payment.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONCounterPlacementPaymentCastingAndUpkeep(t *testing.T) {
	for _, tc := range []struct {
		name  string
		color core.Color
	}{{"Thelon's Chant", core.Green}, {"Tourach's Chant", core.Black}} {
		for _, pay := range []bool{false, true} {
			t.Run(tc.name+"/"+boolName(pay), func(t *testing.T) {
				a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: pay}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetTurn(2)
				g.SetStep(core.PrecombatMain)
				c := repositoryCard(t, "FEM", tc.name)
				c.SetOwner(a.PlayerID())
				a.AddToHand(c)
				a.ManaPool().Add(tc.color, 2)
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 || a.ManaPool().TotalMana() != 2 {
					t.Fatal("unpaid cast rollback")
				}
				a.ManaPool().Add(core.Colorless, 1)
				g.SetStep(core.Upkeep)
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil || a.ManaPool().TotalMana() != 3 {
					t.Fatal("casting timing")
				}
				g.SetStep(core.PrecombatMain)
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(c.ID()) != nil {
					t.Fatal("early entry")
				}
				g.ResolveStack()
				if g.FindPermanent(c.ID()) == nil || a.ManaPool().TotalMana() != 0 {
					t.Fatal("cast cost")
				}
				queueRONUpkeep(g, b)
				if g.StackSize() != 0 {
					t.Fatal("opponent upkeep")
				}
				a.ManaPool().Add(tc.color, 1)
				queueRONUpkeep(g, a)
				if g.StackSize() != 1 || a.offers != 0 {
					t.Fatal("upkeep timing")
				}
				g.ResolveStack()
				if (g.FindPermanent(c.ID()) != nil) != pay {
					t.Fatal("upkeep branch")
				}
				want := 1
				if pay {
					want = 0
				}
				if a.ManaPool().TotalMana() != want {
					t.Fatal("upkeep payment")
				}
			})
		}
	}
}

func TestRONCounterPlacementPaymentIndependentEvents(t *testing.T) {
	a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: true}
	b := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("B")}, accept: true}
	g := mage.NewGame(a, b)
	g.SetTurn(2)
	g.PutOnBattlefield(repositoryCard(t, "FEM", "Thelon's Chant"), a.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "FEM", "Thelon's Chant"), b.PlayerID())
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 6), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 6), b.PlayerID())
	a.choice = first.ID()
	b.choice = second.ID()
	for _, actor := range []mage.Player{a, b} {
		g.PutOnBattlefieldBy(repositoryCard(t, "LEB", "Swamp"), actor.PlayerID(), actor.PlayerID())
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatal("pending events")
	}
	g.ResolveStack()
	if first = g.FindPermanent(first.ID()); first.Counters[core.M1M1] != 2 {
		t.Fatal("first actor lost")
	}
	if second = g.FindPermanent(second.ID()); second.Counters[core.M1M1] != 2 {
		t.Fatal("second actor lost")
	}
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("payment binding crossed")
	}
}

func TestRONCounterPlacementPaymentInventory(t *testing.T) {
	files := fstest.MapFS{"FEM.ron": &fstest.MapFile{Data: append(append(damageReceivedRecord(t, "FEM", "Thelon's Chant"), '\n'), damageReceivedRecord(t, "FEM", "Tourach's Chant")...)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 2 || len(report.Issues) != 0 {
		t.Fatalf("report: %+v %v", report, err)
	}
	roles := map[string]string{"ACounterOfTypeOnAPermanent": "counter_payment_instruction", "PTCounter": "permanent_counter_type", "And": "permanent_filter", "SinglePlayer": "player_filter", "Trigger_ThatPlayer": "player_reference"}
	for name, role := range roles {
		found := false
		for _, s := range report.Signatures {
			if s.Name == name && strings.Contains(s.Context, "PutCounters") {
				found = true
				if s.Role != role {
					t.Fatalf("%s role=%s want=%s", name, s.Role, role)
				}
			}
		}
		if !found {
			t.Fatalf("missing %s", name)
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("complete declarations failed strict loading", err)
	}
}

func TestRONCounterPlacementPaymentNativeLand(t *testing.T) {
	for _, tc := range []struct{ name, land string }{{"Thelon's Chant", "Swamp"}, {"Tourach's Chant", "Forest"}} {
		for _, pay := range []bool{false, true} {
			t.Run(tc.name+"/"+boolName(pay), func(t *testing.T) {
				a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: true}
				b := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("B")}, accept: pay}
				g := mage.NewGame(a, b)
				g.SetTurn(2)
				g.SetStep(core.PrecombatMain)
				g.SetActivePlayerIndex(1)
				g.PutOnBattlefield(repositoryCard(t, "FEM", tc.name), a.PlayerID())
				victim := g.PutOnBattlefield(mage.NewCreature("Payment", "", 2, 4), b.PlayerID())
				b.choice = victim.ID()
				land := repositoryCard(t, "LEB", tc.land)
				land.SetOwner(b.PlayerID())
				b.AddToHand(land)
				if err := g.PlayLand(b.PlayerID(), land.ID()); err != nil {
					t.Fatal(err)
				}
				life, count := 17, 0
				if pay {
					life, count = 20, 1
				}
				if b.Life() != life || int(g.FindPermanent(victim.ID()).Counters[core.M1M1]) != count {
					t.Fatal("native land payment")
				}
				second := repositoryCard(t, "LEB", tc.land)
				second.SetOwner(b.PlayerID())
				b.AddToHand(second)
				if err := g.PlayLand(b.PlayerID(), second.ID()); err == nil || b.Life() != life || int(g.FindPermanent(victim.ID()).Counters[core.M1M1]) != count {
					t.Fatal("illegal land changed payment")
				}
			})
		}
	}
}
