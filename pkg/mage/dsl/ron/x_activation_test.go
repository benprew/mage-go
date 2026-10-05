package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func xActivationFactory(t *testing.T, cost, body string) func() mage.Card {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Variable",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(`+cost+`,`+body+`)])`), "x.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Variable", factory()); err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONXActivationPayment(t *testing.T) {
	for _, cost := range []string{`PayManaX([ManaCostX],ValueX)`, `PayManaX([ManaCostX,ManaCostX,ManaCostU,ManaCostGeneric(1)],ValueX)`, `And([PayManaX([ManaCostX],ValueX),PayManaX([ManaCostX],ValueX),PayMana([ManaCostU,ManaCostGeneric(1)])])`} {
		t.Run(cost, func(t *testing.T) {
			factory := xActivationFactory(t, cost, `ActionList([PlayerAction(You,GainLife(ValueX))])`)
			g, a, _ := ronGame()
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			g.SetXValue(3)
			required := 3
			if cost != `PayManaX([ManaCostX],ValueX)` {
				required = 8
				a.ManaPool().Add(core.Blue, 1)
			}
			a.ManaPool().Add(core.Colorless, required-1-a.ManaPool().TotalMana())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
				t.Fatal("underpayment accepted")
			}
			if a.ManaPool().TotalMana() != required-1 || g.StackSize() != 0 || a.Life() != 20 {
				t.Fatal("failed payment changed state")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || a.Life() != 20 || g.StackPeek().XValue != 3 {
				t.Fatal("wrong payment or announcement")
			}
			g.SetXValue(99)
			g.ResolveStack()
			if a.Life() != 23 {
				t.Fatal("resolution lost announced X")
			}
		})
	}
}

func TestRONXActivationCandelabra(t *testing.T) {
	for _, scenario := range []string{"normal", "zero", "unpaid", "count", "duplicate", "nonland", "shroud", "phase-source", "wrong-controller", "source-left", "source-control", "target-shroud", "target-left", "target-return", "target-phase", "target-cycle", "all-illegal", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "ATQ", "Candelabra of Tawnos")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid artifact cast")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil {
				t.Fatal("artifact entered before resolution")
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			land := mage.NewLand("Land")
			land.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(land, b.PlayerID())
			q := g.PutOnBattlefield(mage.NewLand("Other land"), a.PlayerID())
			r := g.PutOnBattlefield(mage.NewCreature("Nonland", "", 2, 2), a.PlayerID())
			for _, perm := range []*mage.Permanent{p, q, r} {
				g.TapPermanent(perm)
			}
			targets := []uuid.UUID{p.ID(), q.ID()}
			x := 2
			mana := 2
			payer := a
			switch scenario {
			case "zero":
				x = 0
				mana = 0
				targets = nil
			case "unpaid":
				mana = 1
			case "count":
				targets = targets[:1]
			case "duplicate":
				targets[1] = p.ID()
			case "nonland":
				targets[1] = r.ID()
			case "shroud":
				p.GrantBaseAttr(core.Shroud)
			case "phase-source":
				g.PhaseOut(source.ID())
			case "wrong-controller":
				payer = b
			}
			payer.ManaPool().Add(core.Colorless, mana)
			g.SetXValue(x)
			err := g.ActivateAbilityByIndex(payer.PlayerID(), source.ID(), 0, targets)
			illegal := scenario == "unpaid" || scenario == "count" || scenario == "duplicate" || scenario == "nonland" || scenario == "shroud" || scenario == "phase-source" || scenario == "wrong-controller"
			if illegal {
				if err == nil {
					t.Fatal("illegal activation accepted")
				}
				if payer.ManaPool().TotalMana() != mana || g.StackSize() != 0 || g.FindPermanentIncludingPhased(source.ID()).Tapped {
					t.Fatal("failed activation changed costs or stack")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(source.ID()).Tapped || payer.ManaPool().TotalMana() != 0 || !g.FindPermanent(p.ID()).Tapped || g.StackPeek().XValue != x {
				t.Fatal("costs or pre-resolution timing")
			}
			switch scenario {
			case "source-left":
				g.DestroyPermanent(source)
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: q.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "target-shroud":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
			case "target-left":
				g.RemoveFromBattlefield(p)
			case "target-return":
				g.RemoveFromBattlefield(p)
				g.TapPermanent(g.PutOnBattlefield(land, b.PlayerID()))
			case "target-phase":
				g.PhaseOut(p.ID())
			case "target-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "all-illegal":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
				g.MutablePermanent(q.ID()).GrantBaseAttr(core.Shroud)
			case "copy":
				copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
				if copied == nil || copied.XValue != 2 {
					t.Fatal("copy lost X")
				}
			}
			clone := g.Clone()
			g.SetXValue(77)
			g.ResolveStack()
			if !clone.FindPermanent(q.ID()).Tapped {
				t.Fatal("resolution changed clone")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if scenario == "target-phase" {
					game.PhaseIn([]uuid.UUID{p.ID()})
				}
				pIllegal := scenario == "target-shroud" || scenario == "target-return" || scenario == "target-phase" || scenario == "all-illegal" || scenario == "zero"
				if current := game.FindPermanent(p.ID()); current != nil && current.Tapped != pIllegal {
					t.Fatal("wrong target identity or legality")
				}
				if game.FindPermanent(q.ID()).Tapped != (scenario == "all-illegal" || scenario == "zero") || !game.FindPermanent(r.ID()).Tapped {
					t.Fatal("wrong target collection")
				}
			}
		})
	}
}

func TestRONXActivationInvalid(t *testing.T) {
	for _, cost := range []string{`PayManaX`, `PayManaX([ManaCostX])`, `PayManaX([ManaCostX],ValueX,Integer(1))`, `PayManaX([ManaCostX],ValueX())`, `PayManaX([ManaCostX],Integer(2))`, `PayManaX([ManaCostX],Ref_TargetPlayer)`, `PayManaX([],ValueX)`, `PayManaX([ManaCostGeneric(1)],ValueX)`, `PayManaX([ManaCostX,ManaCostGeneric(-1)],ValueX)`, `PayManaX(ManaCostX,ValueX)`, `PayMana([ManaCostX])`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(` + cost + `,ActionList([PlayerAction(You,GainLife(ValueX))]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", cost)
		}
	}
	for _, rule := range []string{`Activated(PayMana([]),Targeted([NumberTargetPermanents(ValueX,AnyPermanent)],ActionList([UntapEachPermanent(Ref_TargetPermanents)])))`, `Activated(PayMana([]),ActionList([PlayerAction(You,GainLife(ValueX))]))`, `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([MayCost(PayManaX([ManaCostX],ValueX)),If(CostWasPaid,[GainLife(Integer(1))])]))`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONXActivationInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayManaX([ManaCostX],ValueX),ActionList([UnknownAction]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range report.Signatures {
		if entry.Name == "ValueX" {
			found = true
			if entry.Role != "announcement_value" {
				t.Fatalf("role=%s", entry.Role)
			}
		}
	}
	if !found || report.Compiled != 0 || len(report.Issues) != 1 {
		t.Fatal("lost nested inventory or accepted unsupported body")
	}
}

func TestRONXActivationCandelabraAutotapAndReduction(t *testing.T) {
	for _, reduction := range []int{0, 2} {
		t.Run(string(rune('0'+reduction)), func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Candelabra of Tawnos"), a.PlayerID())
			if reduction > 0 {
				g.PutOnBattlefield(mage.NewEnchantment("Reducer", "", mage.WithStaticAbility(mage.FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(game *mage.Game, _ uuid.UUID) error {
					game.AddActivationCostReduction(source.ID(), reduction)
					return nil
				}))), a.PlayerID())
			}
			manaLand := g.PutOnBattlefield(mage.NewLand("Mana", mage.WithManaAbility(core.Green)), a.PlayerID())
			other := g.PutOnBattlefield(mage.NewLand("Other"), b.PlayerID())
			third := g.PutOnBattlefield(mage.NewLand("Third"), b.PlayerID())
			x := 1
			targets := []uuid.UUID{manaLand.ID()}
			if reduction > 0 {
				x = 3
				targets = append(targets, other.ID(), third.ID())
			}
			g.SetStep(core.DeclareBlockers)
			g.SetActivePlayerIndex(1)
			g.SetXValue(x)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(manaLand.ID()).Tapped || !g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
				t.Fatal("mana planning or reserved source payment")
			}
			g.ResolveStack()
			if g.FindPermanent(manaLand.ID()).Tapped || !g.FindPermanent(source.ID()).Tapped {
				t.Fatal("untapped target or tap cost identity")
			}
		})
	}
}

func TestRONXActivationInstancesAndCopies(t *testing.T) {
	g, a, b := ronGame()
	data, err := os.ReadFile("../../../../cards/mtgish/ATQ.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "ATQ.ron")
	if err != nil {
		t.Fatal(err)
	}
	var factory mage.CardFactory
	for _, node := range nodes {
		if node.Fields["Name"].String == "Candelabra of Tawnos" {
			factory, err = Compile(node)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if factory == nil {
		t.Fatal("missing complete record")
	}
	first := g.PutOnBattlefield(factory(), a.PlayerID())
	second := g.PutOnBattlefield(factory(), b.PlayerID())
	lands := make([]*mage.Permanent, 0, 4)
	for _, name := range []string{"First", "Second", "Third", "Fourth"} {
		p := g.PutOnBattlefield(mage.NewLand(name), a.PlayerID())
		g.TapPermanent(p)
		lands = append(lands, p)
	}
	a.ManaPool().Add(core.Colorless, 1)
	g.SetXValue(1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, []uuid.UUID{lands[0].ID()}); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	b.ManaPool().Add(core.Colorless, 2)
	g.SetXValue(2)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), second.ID(), 0, []uuid.UUID{lands[1].ID(), lands[2].ID()}); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	copied := g.CopyStackObjectDirect(original, b.PlayerID(), false)
	if copied == nil || copied.XValue != 1 || original.XValue != 1 || g.Stack().FindByID(clone.StackPeek().ID).XValue != 2 {
		t.Fatal("instance X overlap")
	}
	copied.Targets[0] = lands[3].ID()
	copied.TargetZones[lands[3].ID()] = core.ZoneBattlefield
	g.ResolveTopOfStack()
	if !g.FindPermanent(lands[0].ID()).Tapped || g.FindPermanent(lands[3].ID()).Tapped || !clone.FindPermanent(lands[3].ID()).Tapped {
		t.Fatal("copy retarget or clone identity")
	}
	g.ResolveStack()
	clone.ResolveStack()
	for i, p := range lands {
		if g.FindPermanent(p.ID()).Tapped || clone.FindPermanent(p.ID()).Tapped != (i == 3) {
			t.Fatal("pending instance target overlap")
		}
	}
}

func TestRONXActivationRegistry(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/ATQ.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Candelabra of Tawnos"`) {
			record = line
		}
	}
	registry, err := LoadFS(fstest.MapFS{"a.ron": {Data: []byte(record)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("full record not registered")
	}
}

func TestRONXActivationModifiersAndScope(t *testing.T) {
	src := `Card(Name:"Limited X",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[ActivatedWithModifiers(PayManaX([ManaCostX],ValueX),ActionList([PlayerAction(You,GainLife(ValueX))]),ActivateOnlyOnceEachTurn)])`
	nodes, err := Parse(strings.NewReader(src), "modifier.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Limited X", factory()); err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	a.ManaPool().Add(core.Colorless, 3)
	g.SetXValue(2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.SetXValue(1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("activation limit ignored")
	}
	if a.ManaPool().TotalMana() != 1 || g.StackSize() != 1 {
		t.Fatal("failed repeat changed payment or pending X")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("modifier or later announcement changed X")
	}
	invalid := `Card(Name:"Scope",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayManaX([ManaCostX],ValueX),ActionList([PlayerAction(You,GainLife(ValueX))])),Activated(PayMana([]),ActionList([PlayerAction(You,GainLife(ValueX))]))])`
	nodes, err = Parse(strings.NewReader(invalid), "scope.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("X declaration leaked between abilities")
	}
}
