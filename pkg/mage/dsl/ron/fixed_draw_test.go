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

var fixedDrawCards = []struct {
	set, name     string
	colors        []core.Color
	life          int
	tap, targeted bool
}{
	{"LEB", "Jayemdae Tome", []core.Color{core.Colorless, core.Colorless, core.Colorless, core.Colorless}, 0, true, false},
	{"LEG", "Xira Arien", []core.Color{core.Black, core.Red, core.Green}, 0, true, true},
	{"DRK", "Book of Rass", []core.Color{core.Colorless, core.Colorless}, 2, false, false},
	{"LEG", "Greed", []core.Color{core.Black}, 2, false, false},
}

func TestRONFixedDrawRealActivations(t *testing.T) {
	for _, tc := range fixedDrawCards {
		for _, change := range []string{"none", "departed", "returned", "control", "phased"} {
			t.Run(tc.name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				fillRONLibrary(t, a, 3)
				fillRONLibrary(t, b, 3)
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(b.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				g.SetTurn(3)
				source.RevokeBaseAttr(core.AttrSummonSick)
				g.ApplyContinuousEffects()
				for _, color := range tc.colors {
					a.ManaPool().Add(color, 1)
				}
				var targets []uuid.UUID
				if tc.targeted {
					targets = []uuid.UUID{b.PlayerID()}
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err != nil {
					t.Fatal(err)
				}
				if len(a.Hand()) != 0 || len(b.Hand()) != 0 || a.Life() != 20-tc.life || a.ManaPool().TotalMana() != 0 || g.GetStack().Size() != 1 {
					t.Fatal("activation did not pay before drawing")
				}
				if g.FindPermanent(source.ID()).Tapped != tc.tap {
					t.Fatal("wrong tap payment")
				}
				if tc.targeted && !g.FindPermanent(source.ID()).HasKeyword(core.Flying) {
					t.Fatal("Xira lost flying")
				}
				switch change {
				case "departed", "returned":
					g.DestroyPermanent(g.FindPermanent(source.ID()))
					if change == "returned" {
						g.PutOnBattlefield(card, b.PlayerID())
					}
				case "control":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), source.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "phased":
					g.PhaseOut(source.ID())
				}
				clone := g.Clone()
				g.ResolveStack()
				wantA, wantB := 1, 0
				if tc.targeted {
					wantA, wantB = 0, 1
				}
				if len(a.Hand()) != wantA || len(b.Hand()) != wantB {
					t.Fatal("draw followed source owner or new controller")
				}
				if len(clone.GetPlayer(a.PlayerID()).Hand()) != 0 || len(clone.GetPlayer(b.PlayerID()).Hand()) != 0 {
					t.Fatal("resolution changed clone")
				}
				clone.ResolveStack()
				if len(clone.GetPlayer(a.PlayerID()).Hand()) != wantA || len(clone.GetPlayer(b.PlayerID()).Hand()) != wantB {
					t.Fatal("clone lost draw context")
				}
				if a.Life() != 20-tc.life || b.Life() != 20 {
					t.Fatal("resolution paid life again or charged wrong player")
				}
			})
		}
	}
}

func TestRONFixedDrawPaymentRollback(t *testing.T) {
	for _, tc := range fixedDrawCards {
		for _, failure := range []string{"mana", "life", "target", "tapped", "phased", "sick"} {
			if failure == "life" && tc.life == 0 || failure == "target" && !tc.targeted || failure == "tapped" && !tc.tap || failure == "sick" && !tc.targeted {
				continue
			}
			t.Run(tc.name+"/"+failure, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				if failure != "sick" {
					g.SetTurn(3)
					source.RevokeBaseAttr(core.AttrSummonSick)
					g.ApplyContinuousEffects()
				}
				if failure == "life" {
					g.PlayerLoseLife(a, 19)
				}
				if failure != "mana" {
					for _, color := range tc.colors {
						a.ManaPool().Add(color, 1)
					}
				}
				var targets []uuid.UUID
				if tc.targeted {
					targets = []uuid.UUID{b.PlayerID()}
				}
				if failure == "target" {
					targets[0] = uuid.New()
				}
				if failure == "tapped" {
					g.TapPermanent(source)
				}
				var phased []uuid.UUID
				if failure == "phased" {
					phased = g.PhaseOut(source.ID())
				}
				life, mana := a.Life(), a.ManaPool().TotalMana()
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err == nil {
					t.Fatal("illegal activation succeeded")
				}
				if a.Life() != life || a.ManaPool().TotalMana() != mana || !g.GetStack().IsEmpty() || len(a.Hand()) != 0 || len(b.Hand()) != 0 {
					t.Fatal("failed activation changed live state")
				}
				g.PhaseIn(phased)
				if g.FindPermanent(source.ID()).Tapped != (failure == "tapped") {
					t.Fatal("failed activation changed tap state")
				}
			})
		}
	}
}

func TestRONFixedDrawLifeAndManaSources(t *testing.T) {
	for _, tc := range fixedDrawCards {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			fillRONLibrary(t, a, 2)
			fillRONLibrary(t, b, 2)
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.SetTurn(3)
			source.RevokeBaseAttr(core.AttrSummonSick)
			g.ApplyContinuousEffects()
			var lands []*mage.Permanent
			for _, color := range tc.colors {
				name := map[core.Color]string{core.Black: "Swamp", core.Red: "Mountain", core.Green: "Forest", core.Colorless: "Forest"}[color]
				land := repositoryCard(t, "LEB", name)
				land.SetOwner(a.PlayerID())
				lands = append(lands, g.PutOnBattlefield(land, a.PlayerID()))
			}
			if tc.life > 0 {
				g.PlayerLoseLife(a, 17)
			}
			g.AddPreventionShield(a.PlayerID(), 100)
			var targets []uuid.UUID
			if tc.targeted {
				targets = []uuid.UUID{a.PlayerID()}
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err != nil {
				t.Fatal(err)
			}
			for _, land := range lands {
				if !g.FindPermanent(land.ID()).Tapped {
					t.Fatal("mana source was not paid")
				}
			}
			if tc.life > 0 && a.Life() != 1 {
				t.Fatal("damage prevention changed life payment")
			}
			g.ResolveStack()
			if len(a.Hand()) != 1 || len(b.Hand()) != 0 {
				t.Fatal("wrong draw recipient")
			}
		})
	}
}

func TestRONFixedDrawReplacementAndEmptyLibrary(t *testing.T) {
	for _, replace := range []bool{false, true} {
		t.Run(fmt.Sprint(replace), func(t *testing.T) {
			g, a, _ := ronGame()
			card := repositoryCard(t, "LEG", "Greed")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			count := 0
			if replace {
				g.AddEmptyLibraryDrawReplacement(source.ID(), a.PlayerID(), func(_ *mage.Game, _ uuid.UUID) { count++ })
			}
			a.ManaPool().Add(core.Black, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if a.Life() != 18 || len(a.Hand()) != 0 || a.IsAlive() != replace || count != map[bool]int{false: 0, true: 1}[replace] {
				t.Fatal("draw bypassed replacement or empty-library service")
			}
		})
	}
}

func TestRONFixedDrawXiraInstances(t *testing.T) {
	g, a, b := ronGame()
	fillRONLibrary(t, a, 3)
	fillRONLibrary(t, b, 3)
	first, second := repositoryCard(t, "LEG", "Xira Arien"), repositoryCard(t, "LEG", "Xira Arien")
	if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("instances share targets")
	}
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	p, q := g.PutOnBattlefield(first, a.PlayerID()), g.PutOnBattlefield(second, b.PlayerID())
	g.SetTurn(3)
	p.RevokeBaseAttr(core.AttrSummonSick)
	q.RevokeBaseAttr(core.AttrSummonSick)
	g.ApplyContinuousEffects()
	for _, player := range []*mage.BasePlayer{a, b} {
		for _, color := range []core.Color{core.Black, core.Red, core.Green} {
			player.ManaPool().Add(color, 1)
		}
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), q.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(a.Hand()) != 1 || len(b.Hand()) != 1 {
		t.Fatal("instances lost distinct target bindings")
	}
}

func TestRONFixedLifeStandingStones(t *testing.T) {
	for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green} {
		t.Run(fmt.Sprint(color), func(t *testing.T) {
			a := &ronFixedManaPlayer{BasePlayer: mage.NewBasePlayer("A"), color: color}
			g := mage.NewGame(a, mage.NewBasePlayer("B"))
			card := repositoryCard(t, "DRK", "Standing Stones")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			if !mage.IsActivatedManaAbility(card.Abilities()[0]) {
				t.Fatal("mana classification lost")
			}
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if !g.GetStack().IsEmpty() || a.Life() != 19 || a.ManaPool().Count(color) != 1 || !g.FindPermanent(source.ID()).Tapped {
				t.Fatalf("mana ability: stack=%d life=%d pool=%v tapped=%v", g.GetStack().Size(), a.Life(), a.ManaPool().SnapshotPool(), g.FindPermanent(source.ID()).Tapped)
			}
		})
	}
}

func TestRONFixedDrawInvalidDeclarations(t *testing.T) {
	for _, rule := range []string{
		`Activated(PayMana([]), ActionList([DrawACard()]))`,
		`Activated(PayMana([]), ActionList([DrawACard(Integer(1))]))`,
		`Activated(PayMana([]), ActionList([PlayerAction(Ref_TargetPlayer, DrawACard)]))`,
		`Activated(PayMana([]), Targeted([TargetPermanent(AnyPermanent)], ActionList([PlayerAction(Ref_TargetPermanent, DrawACard)])))`,
		`Activated(PayLife(Integer(-1)), ActionList([DrawACard]))`,
		`Activated(PayLife(ValueX), ActionList([DrawACard]))`,
		`Activated(PayLife(2), ActionList([DrawACard]))`,
		`Activated(PayLife(Integer()), ActionList([DrawACard]))`,
		`Activated(PayLife(Integer(1), Integer(2)), ActionList([DrawACard]))`,
		`Activated(PayMana([]), ActionList([MayCost(PayLife(ValueX)), Unless(CostWasPaid, [DrawACard])]))`,
	} {
		src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [%s])`, rule)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", rule)
		}
	}
}

func TestRONFixedLifeInventory(t *testing.T) {
	report, err := InspectFS(fstest.MapFS{"life.ron": {Data: []byte(`Card(Name: "Life", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(PayLife(Integer(2)), ActionList([DrawACard]))])`)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range report.Signatures {
		if entry.Name == "Integer" {
			if entry.Role != "value" {
				t.Fatalf("life amount role = %s", entry.Role)
			}
			return
		}
	}
	t.Fatal("life amount missing")
}

type ronFixedManaPlayer struct {
	*mage.BasePlayer
	color core.Color
}

func (p *ronFixedManaPlayer) ChooseManaColor(string) core.Color { return p.color }

func TestRONFixedDrawManaPrerequisiteRegressions(t *testing.T) {
	for _, tc := range []struct {
		set, name    string
		amount, cost int
		sacrifice    bool
	}{
		{"LEB", "Birds of Paradise", 1, 0, false},
		{"LEB", "Black Lotus", 3, 0, true},
		{"LEB", "Celestial Prism", 1, 2, false},
		{"FEM", "Implements of Sacrifice", 2, 1, true},
	} {
		for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green} {
			t.Run(tc.name+"/"+fmt.Sprint(color), func(t *testing.T) {
				a := &ronFixedManaPlayer{BasePlayer: mage.NewBasePlayer("A"), color: color}
				g := mage.NewGame(a, mage.NewBasePlayer("B"))
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				source := g.PutOnBattlefield(card, a.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				g.ApplyContinuousEffects()
				a.ManaPool().Add(core.Colorless, tc.cost)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().Count(color) != tc.amount || a.ManaPool().TotalMana() != tc.amount || !g.GetStack().IsEmpty() {
					t.Fatal("mana did not use one chosen color")
				}
				if (g.FindPermanent(source.ID()) == nil) != tc.sacrifice {
					t.Fatal("wrong source sacrifice")
				}
				if !tc.sacrifice && !g.FindPermanent(source.ID()).Tapped {
					t.Fatal("source was not tapped")
				}
				if tc.name == "Birds of Paradise" && !g.FindPermanent(source.ID()).HasKeyword(core.Flying) {
					t.Fatal("Birds lost flying")
				}
			})
		}
	}
}

func TestRONFixedLifeBoundariesAndCombinedRollback(t *testing.T) {
	for _, tc := range []struct {
		cost       string
		life, want int
		legal      bool
	}{
		{`PayLife(Integer(0))`, 0, 0, true},
		{`PayLife(Integer(2))`, 2, 0, true},
		{`And([PayMana([ManaCostGeneric(1)]), TapPermanent(ThisPermanent), PayLife(Integer(2)), PayLife(Integer(2))])`, 3, 3, false},
	} {
		t.Run(tc.cost, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `Activated(`+tc.cost+`, ActionList([DrawACard]))`)
			first, second := factory(), factory()
			if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
				t.Fatal(err)
			}
			if first.Abilities()[0].(*mage.ActionDefinition).Costs()[0] == second.Abilities()[0].(*mage.ActionDefinition).Costs()[0] {
				t.Fatal("factories share life costs")
			}
			g, a, _ := ronGame()
			g.PlayerLoseLife(a, 20-tc.life)
			first.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(first, a.PlayerID())
			land := repositoryCard(t, "LEB", "Forest")
			land.SetOwner(a.PlayerID())
			manaSource := g.PutOnBattlefield(land, a.PlayerID())
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
			if (err == nil) != tc.legal || a.Life() != tc.want {
				t.Fatalf("payment: err=%v life=%d", err, a.Life())
			}
			if !tc.legal && (g.FindPermanent(source.ID()).Tapped || g.FindPermanent(manaSource.ID()).Tapped || a.ManaPool().TotalMana() != 0 || !g.GetStack().IsEmpty()) {
				t.Fatal("total life failure consumed other costs")
			}
		})
	}
}

func TestRONFixedDrawSharedContextsAndCopies(t *testing.T) {
	for _, trigger := range []bool{false, true} {
		t.Run(fmt.Sprint(trigger), func(t *testing.T) {
			typeline, rule := "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)], ActionList([PlayerAction(Ref_TargetPlayer, DrawACard), PlayerAction(You, DrawACard)])))`
			if trigger {
				typeline, rule = "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)), ActionList([DrawACard]))`
			}
			src := fmt.Sprintf(`Card(Name: "Draw context", Typeline: (Supertypes: [], Cardtypes: [%s], Subtypes: []), Rules: [%s])`, typeline, rule)
			nodes, err := Parse(strings.NewReader(src), "draw.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			first, second := factory(), factory()
			for _, c := range []mage.Card{first, second} {
				if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
					t.Fatal(err)
				}
			}
			g, a, b := ronGame()
			fillRONLibrary(t, a, 6)
			fillRONLibrary(t, b, 6)
			first.SetOwner(a.PlayerID())
			if trigger {
				g.PutOnBattlefield(first, a.PlayerID())
			} else {
				if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
					t.Fatal("factory shares targets")
				}
				a.AddToHand(first)
				if err := g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
					t.Fatal(err)
				}
				if copied := g.CopySpellOnStack(first.ID(), b.PlayerID(), true); copied == nil || copied.Targets[0] != a.PlayerID() {
					t.Fatal("copy did not retarget")
				}
			}
			if len(a.Hand()) != 0 || len(b.Hand()) != 0 {
				t.Fatal("draw ran before resolution")
			}
			g.ResolveStack()
			want := 2
			if trigger {
				want = 1
			}
			wantB := 2
			if trigger {
				wantB = 0
			}
			if len(a.Hand()) != want || len(b.Hand()) != wantB {
				t.Fatalf("scopes/copies drew %d/%d", len(a.Hand()), len(b.Hand()))
			}
		})
	}
}
