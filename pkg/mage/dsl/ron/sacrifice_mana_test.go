package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSacrificeManaPriest(t *testing.T) {
	a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
	b := mage.NewBasePlayer("Bob")
	g := mage.NewGame(a, b)
	source := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Priest of Yawgmoth"), a.PlayerID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	paid := g.PutOnBattlefield(mage.NewArtifact("Paid", "{X}{3}{U}"), a.PlayerID())
	a.choice = paid.ID()
	if !mage.IsActivatedManaAbility(source.Card.Abilities()[0]) {
		t.Fatal("not a mana ability")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != 0 || a.ManaPool().Count(core.Black) != 4 || g.FindPermanent(paid.ID()) != nil || !g.FindPermanent(source.ID()).Tapped {
		t.Fatal("payment or immediate mana incorrect")
	}
}

func TestRONSacrificeManaMetamorphosis(t *testing.T) {
	a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
	b := mage.NewBasePlayer("Bob")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	spell := repositoryCard(t, "ARN", "Metamorphosis")
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	paid := mage.NewCreature("Paid", "{X}{3}{U}", 1, 1)
	paid.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(paid, a.PlayerID())
	a.choice = p.ID()
	a.ManaPool().Add(core.Green, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 0 || g.StackSize() != 1 || len(b.Graveyard()) != 1 {
		t.Fatal("cast timing")
	}
	b.RemoveFromGraveyard(paid.ID())
	g.PutOnBattlefield(paid, b.PlayerID())
	copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	g.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
		t.Fatal("clone changed")
	}
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		for _, id := range []mage.Player{game.GetPlayer(a.PlayerID()), game.GetPlayer(b.PlayerID())} {
			pool := id.ManaPool()
			creature := core.SpellPaymentContext{IsCreature: true}
			noncreature := core.SpellPaymentContext{}
			cost := core.ManaCost{Generic: 5}
			if pool.TotalMana() != 5 || !pool.CanPay(cost, &creature) || pool.CanPay(cost, &noncreature) || pool.CanPay(cost, nil) {
				t.Fatal("saved value, copy controller, or restriction lost")
			}
		}
	}
}

func TestRONSacrificeManaInvalid(t *testing.T) {
	for _, body := range []string{
		`AddManaRepeatedWithModifiers(AnyManaColor,Integer(1),CanOnlySpendOnSpells(IsCardtype(Land)))`,
		`AddManaRepeatedWithModifiers(AnyManaColor,Integer(1),CanOnlySpendOnSpells())`,
		`AddManaRepeatedWithModifiers(AnyManaColor,Integer(1),CanOnlySpendOnSpells(IsCardtype(Creature)),Integer(1))`,
		`AddManaRepeated(ManaProduceB,ManaValueOfTheSacrificedPermanent)`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(TapPermanent(ThisPermanent),ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONSacrificeManaPriestLegality(t *testing.T) {
	for _, scenario := range []string{"sick", "phased-source", "phased-payment", "opponent", "wrong-type", "invalid-choice", "zero", "self"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
			b := mage.NewBasePlayer("Bob")
			g := mage.NewGame(a, b)
			card := repositoryCard(t, "ATQ", "Priest of Yawgmoth")
			if scenario == "self" {
				card.AddType(core.TypeArtifact)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			if scenario != "sick" {
				source.RevokeBaseAttr(core.AttrSummonSick)
			}
			owner := a.PlayerID()
			if scenario == "opponent" {
				owner = b.PlayerID()
			}
			cost := "{4}"
			if scenario == "zero" {
				cost = ""
			}
			payment := mage.NewArtifact("Payment", cost)
			if scenario == "wrong-type" {
				payment = mage.NewCreature("Wrong type", cost, 1, 1)
			}
			paid := g.PutOnBattlefield(payment, owner)
			a.choice = paid.ID()
			if scenario == "self" {
				a.choice = source.ID()
			}
			if scenario == "phased-source" {
				g.PhaseOut(source.ID())
			}
			if scenario == "phased-payment" {
				g.PhaseOut(paid.ID())
			}
			if scenario == "invalid-choice" {
				a.invalid = g.PutOnBattlefield(mage.NewCreature("Invalid", "", 1, 1), a.PlayerID())
			}
			clone := g.Clone()
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
			valid := scenario == "zero" || scenario == "self"
			if (err == nil) != valid {
				t.Fatalf("activation error=%v", err)
			}
			if g.StackSize() != 0 {
				t.Fatal("mana ability used stack")
			}
			if !valid {
				if g.FindPermanentIncludingPhased(source.ID()).Tapped || g.FindPermanentIncludingPhased(paid.ID()) == nil || a.ManaPool().TotalMana() != 0 || len(a.Graveyard()) != 0 {
					t.Fatal("failed payment changed state")
				}
			} else if scenario == "self" {
				if g.FindPermanent(source.ID()) != nil || a.ManaPool().Count(core.Black) != 2 {
					t.Fatal("self sacrifice lost mana value or controller")
				}
			} else if g.FindPermanent(paid.ID()) != nil || a.ManaPool().TotalMana() != 0 {
				t.Fatal("zero value payment failed")
			}
			if clone.FindPermanentIncludingPhased(source.ID()).Tapped || clone.FindPermanentIncludingPhased(paid.ID()) == nil || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
				t.Fatal("activation changed clone")
			}
		})
	}
}

func TestRONSacrificeManaMetamorphosisRollback(t *testing.T) {
	for _, scenario := range []string{"mana", "wrong-type", "phased", "opponent", "invalid-choice", "timing"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
			b := mage.NewBasePlayer("Bob")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			spell := repositoryCard(t, "ARN", "Metamorphosis")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			paid := mage.NewCreature("Payment", "{3}", 1, 1)
			if scenario == "wrong-type" {
				paid = mage.NewArtifact("Wrong type", "{3}")
			}
			controller := a.PlayerID()
			if scenario == "opponent" {
				controller = b.PlayerID()
			}
			p := g.PutOnBattlefield(paid, controller)
			a.choice = p.ID()
			if scenario == "phased" {
				g.PhaseOut(p.ID())
			}
			if scenario == "invalid-choice" {
				a.invalid = g.PutOnBattlefield(mage.NewArtifact("Invalid", ""), a.PlayerID())
			}
			if scenario != "mana" {
				a.ManaPool().Add(core.Green, 1)
			}
			if scenario == "timing" {
				g.SetStep(core.Upkeep)
			}
			mana := a.ManaPool().TotalMana()
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err == nil {
				t.Fatal("invalid cast accepted")
			}
			if g.FindPermanentIncludingPhased(p.ID()) == nil || a.ManaPool().TotalMana() != mana || len(a.Hand()) != 1 || len(a.Graveyard()) != 0 || g.StackSize() != 0 {
				t.Fatal("failed cast changed resources")
			}
		})
	}
}

func TestRONSacrificeManaActualSpending(t *testing.T) {
	for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green} {
		a := &ronSacrificeManaChooser{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}, color: color}
		b := mage.NewBasePlayer("Bob")
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		spell := repositoryCard(t, "ARN", "Metamorphosis")
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		p := g.PutOnBattlefield(mage.NewCreature("Paid", "{3}", 1, 1), a.PlayerID())
		a.choice = p.ID()
		a.ManaPool().Add(core.Green, 1)
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		if a.calls != 0 {
			t.Fatal("color chosen before resolution")
		}
		g.ResolveStack()
		if a.calls != 1 || a.ManaPool().Count(color) != 4 {
			t.Fatal("full amount must use one chosen color")
		}
		noncreature := mage.NewArtifact("Noncreature", "{4}")
		noncreature.SetOwner(a.PlayerID())
		a.AddToHand(noncreature)
		if err := g.CastSpellByID(a.PlayerID(), noncreature.ID(), nil, 0); err == nil || a.ManaPool().TotalMana() != 4 {
			t.Fatal("noncreature cast used restricted mana")
		}
		source := g.PutOnBattlefield(mage.NewArtifact("Activation", "", mage.WithAbility(mage.Activate(mage.Costs(mage.ManaCostOf("{1}")), mage.GainLifeFor(mage.You, mage.Fixed(1))))), a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || a.ManaPool().TotalMana() != 4 {
			t.Fatal("activation used restricted mana")
		}
		creature := mage.NewCreature("Creature", "{4}", 2, 2)
		creature.SetOwner(a.PlayerID())
		a.AddToHand(creature)
		if err := g.CastSpellByID(a.PlayerID(), creature.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		if a.ManaPool().TotalMana() != 0 {
			t.Fatal("creature payment left mana")
		}
		g.ResolveStack()
		if g.FindPermanent(creature.ID()) == nil {
			t.Fatal("creature did not resolve")
		}
	}
}

type ronSacrificeManaChooser struct {
	*ronSacrificeChooser
	color core.Color
	calls int
}

func (p *ronSacrificeManaChooser) ChooseManaColor(string) core.Color { p.calls++; return p.color }

func TestRONSacrificeManaRegistryAndInventory(t *testing.T) {
	files := fstest.MapFS{}
	for _, tc := range []struct{ set, name string }{{"ATQ", "Priest of Yawgmoth"}, {"ARN", "Metamorphosis"}} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "`+tc.name+`"`) {
				files[tc.set+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("missing real record")
	}
	for _, name := range registry.RegisteredCardNames() {
		for range 2 {
			card, err := registry.CreateCard(name)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition(name, card); err != nil {
				t.Fatal(err)
			}
		}
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 2 || len(report.Issues) != 0 {
		t.Fatal("supported records rejected")
	}
	for name, role := range map[string]string{"CanOnlySpendOnSpells": "mana_spending_restriction", "ManaValueOfTheSacrificedPermanent": "value"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing role %s", role)
		}
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[UnknownRule])`)}
	if partial, err := LoadFS(files, "*.ron"); partial != nil || err == nil {
		t.Fatal("partial registry published")
	}
	if _, err := registry.CreateCard("Metamorphosis"); err != nil {
		t.Fatal("accepted registry changed")
	}
}

func TestRONSacrificeManaPendingValues(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/ARN.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "ARN.ron")
	if err != nil {
		t.Fatal(err)
	}
	var factory mage.CardFactory
	for _, node := range nodes {
		if node.Fields["Name"].String == "Metamorphosis" {
			factory, err = Compile(node)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if factory == nil {
		t.Fatal("record missing")
	}
	a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
	b := mage.NewBasePlayer("Bob")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.PutOnBattlefield(mage.NewEnchantment("Flash permission", "", mage.WithStaticAbility(mage.CastAsThoughHadFlash(mage.CardFilter{}))), a.PlayerID())
	a.ManaPool().Add(core.Green, 2)
	for _, cost := range []string{"{1}", "{4}"} {
		spell := factory()
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		paid := g.PutOnBattlefield(mage.NewCreature("Payment", cost, 1, 1), a.PlayerID())
		a.choice = paid.ID()
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		if game.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 7 || game.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 5 {
			t.Fatal("factory instances or copied payments shared a value")
		}
	}
}
