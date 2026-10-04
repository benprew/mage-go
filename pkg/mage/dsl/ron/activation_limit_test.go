package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONActivationLimitRealCards(t *testing.T) {
	for _, tc := range []struct {
		set, name             string
		color                 core.Color
		max, power, toughness int
	}{
		{"DRK", "Fire Drake", core.Red, 1, 1, 2},
		{"LEG", "Vampire Bats", core.Black, 2, 0, 1},
	} {
		for _, change := range []string{"none", "control", "phase", "return", "leave"} {
			t.Run(tc.name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				c := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
					t.Fatal(err)
				}
				c.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(c, a.PlayerID())
				id := p.ID()
				if !p.HasAttr(core.Flying) || p.CurrentPower(g) != tc.power || p.CurrentToughness(g) != tc.toughness {
					t.Fatal("wrong characteristics")
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), id, 0, nil); err == nil {
					t.Fatal("unpaid activation succeeded")
				}
				a.ManaPool().Add(tc.color, tc.max+1)
				for range tc.max {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), id, 0, nil); err != nil {
						t.Fatal(err)
					}
				}
				if g.FindPermanent(id).CurrentPower(g) != tc.power {
					t.Fatal("bonus started before resolution")
				}
				pending := g.Clone()
				payer := a
				switch change {
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: id, ControllerID: b.PlayerID(), Duration: core.Indefinite})
					b.ManaPool().Add(tc.color, 1)
					payer = b
				case "phase":
					g.PhaseIn(g.PhaseOut(id))
				case "return":
					g.BouncePermanentToHand(g.FindPermanent(id))
					a.RemoveFromHand(c.ID())
					id = g.PutOnBattlefield(c, a.PlayerID()).ID()
				case "leave":
					g.DestroyPermanent(g.FindPermanent(id))
				}
				err := g.ActivateAbilityByIndex(payer.PlayerID(), id, 0, nil)
				if change == "return" {
					if err != nil {
						t.Fatal("new object retained limit", err)
					}
				} else if err == nil {
					t.Fatal("limit or source absence ignored")
				}
				if change != "return" && payer.ManaPool().TotalMana() != 1 {
					t.Fatal("rejected activation consumed mana")
				}
				g.ResolveStack()
				want := tc.power + tc.max
				if change == "return" {
					want = tc.power + 1
				}
				if change != "leave" && g.FindPermanent(id).CurrentPower(g) != want {
					t.Fatal("bonus lost source identity")
				}
				pending.ResolveStack()
				if pending.FindPermanent(p.ID()).CurrentPower(pending) != tc.power+tc.max {
					t.Fatal("clone lost pending bonus")
				}
				if err := pending.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
					t.Fatal("clone lost use count")
				}
				g.RunStepWithPriority(core.Cleanup)
				if change == "leave" {
					return
				}
				if g.FindPermanent(id).CurrentPower(g) != tc.power {
					t.Fatal("bonus survived cleanup")
				}
				if pending.FindPermanent(p.ID()).CurrentPower(pending) != tc.power+tc.max {
					t.Fatal("cleanup changed clone")
				}
				g.SetActivePlayerIndex(1)
				g.SetTurn(2)
				g.SetStep(core.Upkeep)
				payer.ManaPool().Add(tc.color, 1)
				if err := g.ActivateAbilityByIndex(payer.PlayerID(), id, 0, nil); err != nil {
					t.Fatal("next turn retained count", err)
				}
				g.ResolveStack()
				if g.FindPermanent(id).CurrentPower(g) != tc.power+1 {
					t.Fatal("opponent turn activation lost bonus")
				}
			})
		}
	}
}

func TestRONActivationLimitInstancesCopiesAndClone(t *testing.T) {
	g, a, _ := ronGame()
	c := repositoryCard(t, "LEG", "Vampire Bats")
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	other := repositoryCard(t, "LEG", "Vampire Bats")
	other.SetOwner(a.PlayerID())
	q := g.PutOnBattlefield(other, a.PlayerID())
	if c.Abilities()[0] == other.Abilities()[0] {
		t.Fatal("factory shared ability")
	}
	a.ManaPool().Add(core.Black, 6)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	fork := g.Clone()
	if err := fork.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := fork.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
		t.Fatal("fork exceeded limit")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal("fork or copy consumed original count", err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
		t.Fatal("copy reset count")
	}
	for range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err != nil {
			t.Fatal("instances shared count", err)
		}
	}
	g.ResolveStack()
	if g.FindPermanent(p.ID()).CurrentPower(g) != 3 || g.FindPermanent(q.ID()).CurrentPower(g) != 2 {
		t.Fatal("copy or instance bonus wrong")
	}
	fork.ResolveStack()
	if fork.FindPermanent(p.ID()).CurrentPower(fork) != 2 || fork.FindPermanent(q.ID()).CurrentPower(fork) != 0 {
		t.Fatal("clone shared resolved effects")
	}
}

func TestRONActivationLimitConjunctionAndRollback(t *testing.T) {
	for _, filter := range []string{"IsTheirTurn", "IsTheirUpkeep"} {
		rule := fmt.Sprintf(`ActivatedWithModifiers(And([PayMana([ManaCostR]), TapPermanent(ThisPermanent)]), Targeted([TargetPermanent(AnyPermanent)], ActionList([TapPermanent(Ref_TargetPermanent)])), And([ActivateOnlyOnceEachTurn, ActivateOnlyIf(PlayerPassesFilter(You, %s))]))`, filter)
		factory := permanentScopeFactory(t, "Artifact", rule)
		g, a, b := ronGame()
		c := factory()
		c.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(c, a.PlayerID())
		target := g.PutOnBattlefield(mage.NewArtifact("Target", ""), b.PlayerID())
		a.ManaPool().Add(core.Red, 2)
		g.SetStep(core.Upkeep)
		g.SetActivePlayerIndex(1)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{target.ID()}); err == nil {
			t.Fatal("wrong turn accepted")
		}
		g.SetActivePlayerIndex(0)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{uuid.New()}); err == nil {
			t.Fatal("invalid target accepted")
		}
		if g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 2 || !g.GetStack().IsEmpty() {
			t.Fatal("failed activation changed state")
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
			t.Fatal(err)
		}
		g.UntapPermanent(g.FindPermanent(p.ID()))
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{target.ID()}); err == nil {
			t.Fatal("untap reset limit")
		}
		g.SetStep(core.EndStep)
		g.ResolveStack()
		if !g.FindPermanent(target.ID()).Tapped {
			t.Fatal("timing repeated at resolution")
		}
	}
}

func TestRONActivationLimitInvalid(t *testing.T) {
	for _, modifier := range []string{
		`ActivateOnlyOnceEachTurn()`, `ActivateNoMoreThanNumberTimesEachTurn`,
		`ActivateNoMoreThanNumberTimesEachTurn(Integer(0))`, `ActivateNoMoreThanNumberTimesEachTurn(Integer(-1))`,
		`ActivateNoMoreThanNumberTimesEachTurn(Integer(2), Integer(3))`, `ActivateNoMoreThanNumberTimesEachTurn(Integer())`,
		`ActivateNoMoreThanNumberTimesEachTurn(2)`, `ActivateNoMoreThanNumberTimesEachTurn(ValueX)`,
		`And([ActivateNoMoreThanNumberTimesEachTurn(Integer(1)), And([ActivateNoMoreThanNumberTimesEachTurn(Integer(2))])])`, `And([])`, `And(ActivateOnlyOnceEachTurn)`, `And([ActivateOnlyOnceEachTurn, UnknownModifier])`,
	} {
		src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [ActivatedWithModifiers(PayMana([]), ActionList([GainLife(Integer(1))]), %s)])`, modifier)
		nodes, err := Parse(strings.NewReader(src), "invalid-limit.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", modifier)
		}
	}
}

func TestRONActivationLimitManaAndInventory(t *testing.T) {
	rule := `ActivatedWithModifiers(TapPermanent(ThisPermanent), ActionList([AddMana(ManaProduceU)]), And([ActivateOnlyIf(PlayerPassesFilter(You, IsTheirUpkeep)), ActivateNoMoreThanNumberTimesEachTurn(Integer(1))]))`
	factory := permanentScopeFactory(t, "Land", rule)
	g, a, _ := ronGame()
	g.SetStep(core.Upkeep)
	c := factory()
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	for range 2 {
		if !g.CanAfford(a.PlayerID(), core.ParseManaCost("{U}"), nil) {
			t.Fatal("query consumed count")
		}
	}
	fork := g.Clone()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if !g.GetStack().IsEmpty() || a.ManaPool().TotalMana() != 1 {
		t.Fatal("mana ability used stack")
	}
	g.UntapPermanent(g.FindPermanent(p.ID()))
	if g.CanAfford(a.PlayerID(), core.ParseManaCost("{U}{U}"), nil) {
		t.Fatal("solver ignored limit")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
		t.Fatal("mana limit ignored")
	}
	if !fork.CanAfford(a.PlayerID(), core.ParseManaCost("{U}"), nil) {
		t.Fatal("original consumed clone limit")
	}
	if err := fork.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 1 {
		t.Fatal("clone changed original mana")
	}
	g.RunStepWithPriority(core.Cleanup)
	g.SetStep(core.Upkeep)
	if !g.CanAfford(a.PlayerID(), core.ParseManaCost("{U}"), nil) {
		t.Fatal("cleanup retained limit")
	}
	nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name: "Inventory", Typeline: (Supertypes: [], Cardtypes: [Land], Subtypes: []), Rules: [%s])`, rule)), "limit.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Inventory", "Card", "card", "collection", true, entries)
	for name, role := range map[string]string{"And": "activation_modifier", "ActivateNoMoreThanNumberTimesEachTurn": "activation_modifier", "Integer": "activation_limit_value"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s lacks %s role", name, role)
		}
	}
}

func TestRONActivationLimitPhasedReset(t *testing.T) {
	g, a, _ := ronGame()
	c := repositoryCard(t, "DRK", "Fire Drake")
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	a.ManaPool().Add(core.Red, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	ids := g.PhaseOut(p.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
		t.Fatal("phased source activated")
	}
	fork := g.Clone()
	g.RunStepWithPriority(core.Cleanup)
	g.PhaseIn(ids)
	g.SetStep(core.Upkeep)
	a.ManaPool().Add(core.Red, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal("phased source retained previous turn limit", err)
	}
	fork.PhaseIn(ids)
	fork.GetPlayer(a.PlayerID()).ManaPool().Add(core.Red, 1)
	if err := fork.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
		t.Fatal("cleanup reset clone limit")
	}
}

func TestRONActivationLimitNestedAndContexts(t *testing.T) {
	modifier := `And([ActivateNoMoreThanNumberTimesEachTurn(Integer(3)), And([ActivateOnlyOnceEachTurn, ActivateOnlyIf(PlayerPassesFilter(You, IsTheirTurn))])])`
	rule := fmt.Sprintf(`ActivatedWithModifiers(PayMana([]), ActionList([GainLife(Integer(1))]), %s)`, modifier)
	factory := permanentScopeFactory(t, "Artifact", rule)
	g, a, _ := ronGame()
	c := factory()
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
		t.Fatal("numbered maximum overwrote once-only limit")
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("nested modifiers lost action")
	}
	for _, kind := range []string{"Instant", "Sorcery"} {
		src := fmt.Sprintf(`Card(Name: "Invalid context", Typeline: (Supertypes: [], Cardtypes: [%s], Subtypes: []), Rules: [%s])`, kind, rule)
		nodes, err := Parse(strings.NewReader(src), "limit-context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted activation on spell card")
		}
	}
}
