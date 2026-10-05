package ron

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSacrificeNumberWarrens(t *testing.T) {
	for _, scenario := range []string{"pay", "one", "phased", "opponent", "mana", "departed", "copy", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "FEM", "Goblin Warrens")
			c.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(c, a.PlayerID())
			goblins := make([]*mage.Permanent, 0, 2)
			for range 2 {
				card := repositoryCard(t, "LEB", "Goblin Balloon Brigade")
				card.SetOwner(b.PlayerID())
				goblins = append(goblins, g.PutOnBattlefield(card, a.PlayerID()))
			}
			switch scenario {
			case "one":
				g.RemoveFromBattlefield(goblins[1])
			case "phased":
				g.PhaseOut(goblins[1].ID())
			case "opponent":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: goblins[1].ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			a.ManaPool().Add(core.Colorless, 2)
			if scenario != "mana" {
				a.ManaPool().Add(core.Red, 1)
			}
			before := g.Clone()
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
			if scenario == "one" || scenario == "phased" || scenario == "opponent" || scenario == "mana" {
				if err == nil || len(b.Graveyard()) != 0 || a.ManaPool().TotalMana() != before.GetPlayer(a.PlayerID()).ManaPool().TotalMana() || g.StackSize() != 0 {
					t.Fatal("failed activation spent resources")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || len(b.Graveyard()) != 2 || g.StackSize() != 1 || len(g.FilterBattlefield(mage.PermanentFilter{})) != 1 {
				t.Fatal("cost/effect timing")
			}
			if scenario == "departed" {
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			}
			if scenario == "copy" {
				g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
			}
			pendingCount := len(g.FilterBattlefield(mage.PermanentFilter{}))
			clone := g.Clone()
			g.ResolveStack()
			counts := map[bool]int{}
			for _, p := range g.FilterBattlefield(mage.PermanentFilter{}) {
				if p.IsToken {
					counts[p.ControllerID() == a.PlayerID()]++
					if p.CurrentPower(g) != 1 || p.CurrentToughness(g) != 1 || !slices.Contains(p.Colors(), core.Red) || !p.HasSubType("Goblin") {
						t.Fatal("wrong token")
					}
				}
			}
			if counts[true] != 3 || (scenario == "copy" && counts[false] != 3) {
				t.Fatal("wrong tokens/controller", counts)
			}
			if len(clone.FilterBattlefield(mage.PermanentFilter{})) != pendingCount {
				t.Fatal("resolution changed pending clone")
			}
			clone.ResolveStack()
			if len(clone.FilterBattlefield(mage.PermanentFilter{})) != len(g.FilterBattlefield(mage.PermanentFilter{})) {
				t.Fatal("pending clone lost creation")
			}
		})
	}
}

func TestRONSacrificeNumberMoldDemon(t *testing.T) {
	for _, scenario := range []string{"pay", "decline", "one", "wrong", "phased", "returned", "departed", "control"} {
		t.Run(scenario, func(t *testing.T) {
			a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: scenario != "decline"}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			lands := make([]*mage.Permanent, 0, 2)
			for range 2 {
				card := repositoryCard(t, "LEB", "Swamp")
				card.SetOwner(b.PlayerID())
				lands = append(lands, g.PutOnBattlefield(card, a.PlayerID()))
			}
			switch scenario {
			case "one":
				g.RemoveFromBattlefield(lands[1])
			case "wrong":
				g.RemoveFromBattlefield(lands[1])
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
			case "phased":
				g.PhaseOut(lands[1].ID())
			}
			card := repositoryCard(t, "LEG", "Mold Demon")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || a.offers != 0 || len(b.Graveyard()) != 0 {
				t.Fatal("payment before trigger resolution")
			}
			switch scenario {
			case "returned":
				g.BouncePermanentToHand(source)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "departed":
				g.BouncePermanentToHand(source)
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.ResolveTopOfStack()
			paid := scenario == "pay" || scenario == "returned" || scenario == "departed" || scenario == "control"
			want := 0
			if paid {
				want = 2
			}
			if len(b.Graveyard()) != want || (g.FindPermanent(source.ID()) != nil) != (paid && scenario != "departed") {
				t.Fatal("wrong payment or source identity")
			}
			if scenario == "returned" {
				g.ResolveStack()
				if g.FindPermanent(source.ID()) != nil {
					t.Fatal("new trigger reused old payment")
				}
			}
		})
	}
}

func TestRONSacrificeNumberInvalid(t *testing.T) {
	for _, cost := range []string{`SacrificeNumberPermanents`, `SacrificeNumberPermanents(Integer(2))`, `SacrificeNumberPermanents(Integer(2),IsCardtype(Creature),IsTapped)`, `SacrificeNumberPermanents(Integer(0),IsCardtype(Creature))`, `SacrificeNumberPermanents(Integer(-1),IsCardtype(Creature))`, `SacrificeNumberPermanents(ValueX,IsCardtype(Creature))`, `SacrificeNumberPermanents(Integer("2"),IsCardtype(Creature))`, `SacrificeNumberPermanents(Integer(2),And([]))`, `SacrificeNumberPermanents(Integer(2),SinglePermanent(Ref_TargetPermanent))`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(` + cost + `,ActionList([GainLife(Integer(1))]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid cost accepted", cost)
		}
	}
}

func TestRONSacrificeNumberInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([MayCost(SacrificeNumberPermanents(Integer(2),IsCardtype(Artifact))),If(CostWasPaid,[GainLife(Integer(1))])]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal(err, report.Issues)
	}
	for name, role := range map[string]string{"Integer": "value", "IsCardtype": "permanent_filter"} {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatal("missing inventory role", name, role)
		}
	}
}

func TestRONSacrificeNumberSelfAndExplicitPlayer(t *testing.T) {
	a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: true}
	b := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("B")}, accept: true}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	card := permanentScopeFactory(t, "Artifact", `Activated(SacrificeNumberPermanents(Integer(2),IsCardtype(Artifact)),ActionList([GainLife(Integer(2))]))`)()
	source := g.PutOnBattlefield(card, a.PlayerID())
	other := g.PutOnBattlefield(mage.NewArtifact("Other", ""), a.PlayerID())
	a.choice = source.ID()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(source.ID()) != nil || g.FindPermanent(other.ID()) != nil || a.Life() != 22 {
		t.Fatal("RON count excludes source")
	}
	spell := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerMayCost(Ref_TargetPlayer,SacrificeNumberPermanents(Integer(2),And([IsCardtype(Artifact),Or([IsTapped,IsCardtype(Creature)])]))),If(CostWasPaid,[GainLife(Integer(3))]) , MayCost(SacrificeNumberPermanents(Integer(2),IsCardtype(Artifact))),If(CostWasntPaid,[GainLife(Integer(1))])])))`)()
	for range 2 {
		p := g.PutOnBattlefield(mage.NewArtifact("Payment", ""), b.PlayerID())
		g.TapPermanent(p)
	}
	castRONSpell(t, g, a.BasePlayer, spell, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 26 || b.Life() != 20 || b.offers != 1 || a.offers != 0 {
		t.Fatal("explicit payer or independent results")
	}
}

func TestRONSacrificeNumberCasting(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"FEM", "Goblin Warrens"}, {"LEG", "Mold Demon"}} {
		g, a, _ := ronGame()
		card := repositoryCard(t, tc.set, tc.name)
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
			t.Fatal("unpaid cast accepted")
		}
		a.RemoveFromHand(card.ID())
		castRONSpell(t, g, a, card)
		g.ResolveStack()
		if tc.name == "Mold Demon" {
			g.PutTriggersOnStack()
			g.ResolveStack()
			if g.FindPermanent(card.ID()) != nil || len(a.Graveyard()) != 1 {
				t.Fatal("native entry payment did not run")
			}
		} else if g.FindPermanent(card.ID()) == nil {
			t.Fatal("paid enchantment failed entry")
		}
	}
}

func TestRONEnteringSourceElectricEel(t *testing.T) {
	for _, scenario := range []string{"normal", "leave", "return", "phase", "control", "prevent", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "Electric Eel")
			card.SetOwner(a.PlayerID())
			castRONSpell(t, g, a, card)
			g.ResolveTopOfStack()
			g.PutTriggersOnStack()
			if a.Life() != 20 || g.StackSize() != 1 {
				t.Fatal("entry damage timing")
			}
			switch scenario {
			case "leave", "return":
				g.BouncePermanentToHand(g.FindPermanent(card.ID()))
				if scenario == "return" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phase":
				g.PhaseOut(card.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "prevent":
				g.AddPreventionShield(a.PlayerID(), 1)
			case "copy":
				g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
			}
			clone := g.Clone()
			g.ResolveTopOfStack()
			if scenario == "copy" {
				if b.Life() != 19 || a.Life() != 20 {
					t.Fatal("copy damage controller")
				}
				g.ResolveTopOfStack()
			}
			want := 19
			if scenario == "prevent" {
				want = 20
			}
			if a.Life() != want {
				t.Fatal("entry damage lost original source/controller", a.Life())
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("entry damage changed clone")
			}
			if scenario != "normal" {
				return
			}
			a.ManaPool().Clear()
			a.ManaPool().Add(core.Red, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 1, nil); err == nil || a.ManaPool().TotalMana() != 1 {
				t.Fatal("unpaid boost spent mana")
			}
			a.ManaPool().Add(core.Red, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()).CurrentPower(g) != 1 || a.Life() != 19 {
				t.Fatal("boost before resolution")
			}
			g.ResolveStack()
			if g.FindPermanent(card.ID()).CurrentPower(g) != 3 || a.Life() != 18 {
				t.Fatal("boost or damage missing")
			}
			g.RunStepWithPriority(core.Cleanup)
			if g.FindPermanent(card.ID()).CurrentPower(g) != 1 {
				t.Fatal("boost survived cleanup")
			}
		})
	}
}

func TestRONEnteringSourceScopeRejected(t *testing.T) {
	for _, rule := range []string{`Activated(PayMana([]),ActionList([SacrificePermanent(ThatEnteringPermanent)]))`, `TriggerA(WhenAPermanentDies(SinglePermanent(ThisPermanent)),ActionList([SacrificePermanent(ThatEnteringPermanent)]))`, `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([SacrificePermanent(ThatEnteringPermanent())]))`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rule + `])`
		nodes, err := Parse(strings.NewReader(src), "scope.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("out-of-scope entering reference accepted")
		}
	}
}
