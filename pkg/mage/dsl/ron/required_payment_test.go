package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONRequiredPaymentLordOfThePit(t *testing.T) {
	for _, situation := range []string{"pay", "missing", "self", "opponent", "phased", "return", "departed", "control", "clone", "copy", "prevent", "source-phased", "disabled"} {
		t.Run(situation, func(t *testing.T) {
			a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: false}
			b := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("B")}, accept: false}
			g := mage.NewGame(a, b)
			c := repositoryCard(t, "LEB", "Lord of the Pit")
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			if !p.HasAttr(core.Flying) || !p.HasAttr(core.Trample) || p.CurrentPower(g) != 7 || p.CurrentToughness(g) != 7 {
				t.Fatal("metadata")
			}
			var victim *mage.Permanent
			if situation != "missing" && situation != "self" && situation != "return" && situation != "prevent" {
				owner := a.PlayerID()
				if situation == "opponent" {
					owner = b.PlayerID()
				}
				victim = g.PutOnBattlefield(mage.NewCreature("Payment", "", 1, 1), owner)
				a.choice = victim.ID()
				g.AddRegenerationShield(victim.ID())
				if situation == "phased" {
					g.PhaseOut(victim.ID())
				}
			}
			if situation == "disabled" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
			}
			queueRONUpkeep(g, b)
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep")
			}
			queueRONUpkeep(g, a)
			if situation == "disabled" {
				if g.StackSize() != 0 {
					t.Fatal("removed ability triggered")
				}
				return
			}
			if g.StackSize() != 1 || a.choices != 0 || a.offers != 0 {
				t.Fatal("early payment")
			}
			switch situation {
			case "return":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(c.ID())
				p = g.PutOnBattlefield(c, a.PlayerID())
				a.choice = p.ID()
			case "departed":
				g.BouncePermanentToHand(p)
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
			case "source-phased":
				g.PhaseOut(p.ID())
			case "prevent":
				g.AddPreventionShield(a.PlayerID(), 2)
			}
			clone := g.Clone()
			if situation == "clone" {
				clone.ResolveStack()
				if g.FindPermanent(victim.ID()) == nil || g.StackSize() != 1 {
					t.Fatal("clone changed original")
				}
			}
			g.ResolveStack()
			want := 20
			if situation == "missing" || situation == "self" || situation == "opponent" || situation == "phased" {
				want = 13
			}
			if situation == "prevent" {
				want = 15
			}
			if a.Life() != want || a.offers != 0 || b.offers != 0 {
				t.Fatalf("life=%d offers=%d/%d", a.Life(), a.offers, b.offers)
			}
			if situation == "copy" && b.Life() != 13 {
				t.Fatal("copy payer")
			}
			if situation == "return" && g.FindPermanent(p.ID()) != nil {
				t.Fatal("returned source must be a different, eligible payment")
			}
			if victim != nil && situation != "phased" && situation != "opponent" && g.FindPermanent(victim.ID()) != nil {
				t.Fatal("forced sacrifice not paid")
			}
			if situation == "self" && g.FindPermanent(p.ID()) == nil {
				t.Fatal("source sacrificed itself")
			}
		})
	}
}

func TestRONRequiredPaymentScopes(t *testing.T) {
	for _, tc := range []struct {
		condition string
		life      int
	}{
		{`If(CostWasPaid,[GainLife(Integer(3))])`, 23},
		{`Unless(CostWasntPaid,[GainLife(Integer(3))])`, 23},
		{`If(CostWasntPaid,[GainLife(Integer(3))])`, 20},
		{`Unless(CostWasPaid,[GainLife(Integer(3))])`, 20},
	} {
		g, a, _ := ronGame()
		c := permanentScopeFactory(t, "Sorcery", `SpellActions(ActionList([MustCost(SacrificeAPermanent(IsCardtype(Creature))),`+tc.condition+`]))`)()
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
		victim := g.PutOnBattlefield(mage.NewCreature("Payment", "", 1, 1), a.PlayerID())
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(victim.ID()) == nil {
			t.Fatal("payment before resolution")
		}
		g.ResolveStack()
		if a.Life() != tc.life || g.FindPermanent(victim.ID()) != nil {
			t.Fatal("branch polarity")
		}
	}
	g, a, _ := ronGame()
	c := permanentScopeFactory(t, "Sorcery", `SpellActions(ActionList([MustCost(SacrificeAPermanent(IsCardtype(Creature))),If(CostWasPaid,[MustCost(SacrificeAPermanent(IsCardtype(Creature))),Unless(CostWasPaid,[GainLife(Integer(2))]),GainLife(Integer(3))])]))`)()
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	g.PutOnBattlefield(mage.NewCreature("Only", "", 1, 1), a.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 25 {
		t.Fatal("nested results interfered")
	}
}

func TestRONRequiredPaymentInvalid(t *testing.T) {
	for _, actions := range []string{
		`MustCost()`, `MustCost(SacrificeAPermanent(IsCardtype(Creature)))`,
		`MustCost(SacrificeAPermanent(Other(Ref_TargetPermanent))),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(SacrificeAPermanent(Other(ThisPermanent()))),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(SacrificeAPermanent(Or([Other(ThisPermanent),IsCardtype(Creature)]))),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(PayMana([])),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(DestroyPermanent(ThisPermanent)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(SacrificeAPermanent(And([]))),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(SacrificeAPermanent(IsCardtype(Creature))),If(CostWasPaid,[])`,
		`MustCost(SacrificeAPermanent(IsCardtype(Creature))),GainLife(Integer(1)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`If(CostWasPaid,[GainLife(Integer(1))])`,
	} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + actions + `]))])`
		nodes, err := Parse(strings.NewReader(source), "invalid-required.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", actions)
		}
	}
}

func TestRONRequiredPaymentAllReprintsAndNativeCasting(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, a, _ := ronGame()
			card := repositoryCard(t, set, "Lord of the Pit")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Black, 3)
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 || a.ManaPool().TotalMana() != 6 {
				t.Fatal("cast rollback")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil {
				t.Fatal("early entry")
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p == nil || p.CurrentPower(g) != 7 || !p.HasAttr(core.Flying) || !p.HasAttr(core.Trample) {
				t.Fatal("entry")
			}
			victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 1), a.PlayerID())
			g.RunStepWithPriority(core.Upkeep)
			if g.FindPermanent(victim.ID()) != nil || a.Life() != 20 || g.StackSize() != 0 {
				t.Fatal("native upkeep payment")
			}
			g.RunStepWithPriority(core.Upkeep)
			if a.Life() != 13 || g.FindPermanent(p.ID()) == nil {
				t.Fatal("native upkeep damage")
			}
		})
	}
}

func TestRONRequiredPaymentInvalidChoice(t *testing.T) {
	for _, stale := range []bool{false, true} {
		a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}}
		g := mage.NewGame(a, mage.NewBasePlayer("B"))
		card := repositoryCard(t, "LEB", "Lord of the Pit")
		p := g.PutOnBattlefield(card, a.PlayerID())
		victimCard := mage.NewCreature("Payment", "", 1, 1)
		victim := g.PutOnBattlefield(victimCard, a.PlayerID())
		a.invalid = p
		if stale {
			a.invalid = victim
			g.BouncePermanentToHand(victim)
			a.RemoveFromHand(victimCard.ID())
			victim = g.PutOnBattlefield(victimCard, a.PlayerID())
		}
		queueRONUpkeep(g, a)
		g.ResolveStack()
		if a.Life() != 13 || g.FindPermanent(victim.ID()) == nil || g.FindPermanent(p.ID()) == nil || a.offers != 0 {
			t.Fatal("invalid payment did not roll back")
		}
	}
}

func TestRONRequiredPaymentInventoryAndStrictLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEB.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEB.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record Node
	for _, n := range nodes {
		if n.Fields["Name"].String == "Lord of the Pit" {
			record = n
		}
	}
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	first, second := factory(), factory()
	if first.ID() == second.ID() {
		t.Fatal("factory identity shared")
	}
	var source string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Lord of the Pit"`) {
			source = line
		}
	}
	registry, err := LoadFS(fstest.MapFS{
		"first.ron":  &fstest.MapFile{Data: []byte(source)},
		"second.ron": &fstest.MapFile{Data: []byte(source)},
	}, "*.ron")
	if err != nil || registry == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("reprint registry", err)
	}
	if err := LoadInto(registry, fstest.MapFS{"invalid.ron": &fstest.MapFile{Data: []byte(source + "\n" + `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([MustCost(PayMana([])),If(CostWasPaid,[GainLife(Integer(1))])]))])`)}}, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection replaced the registry")
	}
	report, err := InspectFS(fstest.MapFS{"test.ron": &fstest.MapFile{Data: []byte(`Card(Name:"Synthetic",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([MustCost(SacrificeAPermanent(IsCardtype(Creature))),Unless(CostWasPaid,[GainLife(Integer(1))])]))])`)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "SacrificeAPermanent" && signature.Role == "resolution_cost" {
			found = true
		}
	}
	if !found {
		t.Fatal("mandatory cost semantic role")
	}
}
