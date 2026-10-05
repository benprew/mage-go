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

var upkeepPaymentCards = []struct {
	set, name string
	colors    []core.Color
}{
	{"LEB", "Force of Nature", []core.Color{core.Green, core.Green, core.Green, core.Green}},
	{"LEB", "Conversion", []core.Color{core.White, core.White}},
	{"LEB", "Phantasmal Forces", []core.Color{core.Blue}},
	{"ARN", "Junún Efreet", []core.Color{core.Black, core.Black}},
	{"DRK", "Sunken City", []core.Color{core.Blue, core.Blue}},
	{"LEG", "Nicol Bolas", []core.Color{core.Blue, core.Black, core.Red}},
	{"LEG", "Chromium", []core.Color{core.White, core.Blue, core.Black}},
	{"LEG", "Palladia-Mors", []core.Color{core.Red, core.Green, core.White}},
	{"LEG", "Vaevictis Asmadi", []core.Color{core.Black, core.Red, core.Green}},
}

func queueRONUpkeep(g *mage.Game, p mage.Player) {
	g.SetStep(core.Upkeep)
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: p.PlayerID()})
	g.PutTriggersOnStack()
}

func TestRONUpkeepPaymentRealCards(t *testing.T) {
	for _, tc := range upkeepPaymentCards {
		for _, choice := range []string{"pay", "decline", "unaffordable", "wrong-color", "mana-sources", "departed", "returned", "phased", "control", "prevent"} {
			t.Run(tc.name+"/"+choice, func(t *testing.T) {
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: choice != "decline"}
				b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
				g := mage.NewGame(a, b)
				g.SetTurn(2)
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				queueRONUpkeep(g, b)
				if !g.GetStack().IsEmpty() {
					t.Fatal("opponent upkeep triggered payment")
				}
				queueRONUpkeep(g, a)
				if g.GetStack().Size() != 1 || a.offers != 0 || a.Life() != 20 {
					t.Fatal("payment did not wait for resolution")
				}
				paid := choice == "pay" || choice == "mana-sources"
				var lands []*mage.Permanent
				switch choice {
				case "mana-sources", "wrong-color":
					for _, color := range tc.colors {
						name := map[core.Color]string{core.Green: "Forest", core.Blue: "Island", core.Black: "Swamp", core.Red: "Mountain", core.White: "Plains"}[color]
						if choice == "wrong-color" {
							name = "Forest"
							if color == core.Green {
								name = "Island"
							}
						}
						land := repositoryCard(t, "LEB", name)
						land.SetOwner(a.PlayerID())
						lands = append(lands, g.PutOnBattlefield(land, a.PlayerID()))
					}
				case "pay", "decline":
					for _, color := range tc.colors {
						a.ManaPool().Add(color, 1)
					}
				}
				if choice == "departed" || choice == "returned" {
					g.DestroyPermanent(source)
				}
				if choice == "returned" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
				if choice == "phased" {
					g.PhaseOut(source.ID())
				}
				if choice == "control" {
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), source.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					for _, color := range tc.colors {
						b.ManaPool().Add(color, 1)
					}
				}
				if choice == "prevent" {
					g.AddPreventionShield(a.PlayerID(), 3)
				}
				g.ResolveTopOfStack()
				life := 20
				if tc.name == "Force of Nature" && !paid {
					life = 12
					if choice == "prevent" {
						life = 15
					}
				}
				if g.GetPlayer(a.PlayerID()).Life() != life || g.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatalf("wrong damage recipient/outcome: %d/%d", g.GetPlayer(a.PlayerID()).Life(), g.GetPlayer(b.PlayerID()).Life())
				}
				present := paid || tc.name == "Force of Nature" || choice == "returned" || choice == "control"
				if choice == "departed" || choice == "phased" {
					present = false
				}
				if (g.FindPermanent(source.ID()) != nil) != present {
					t.Fatal("sacrifice used the wrong source incarnation")
				}
				offers := 0
				if choice == "pay" || choice == "decline" || choice == "mana-sources" {
					offers = 1
				}
				if a.offers != offers || b.offers != 0 {
					t.Fatalf("payment offers %d/%d", a.offers, b.offers)
				}
				for _, land := range lands {
					if g.FindPermanent(land.ID()).Tapped != paid {
						t.Fatal("mana payment or rollback is wrong")
					}
				}
				mana := 0
				if choice == "decline" {
					mana = len(tc.colors)
				}
				if g.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != mana {
					t.Fatal("payment consumed the wrong mana")
				}
			})
		}
	}
}

func TestRONUpkeepPaymentInvalid(t *testing.T) {
	for _, actions := range []string{
		`MayCost`, `MayCost(PayMana([]), You)`, `MayCost(PayMana([]))`,
		`MayCost(PayMana([])), GainLife(Integer(1)), Unless(CostWasPaid, [GainLife(Integer(1))])`,
		`MayCost(PayMana([ManaCostX])), Unless(CostWasPaid, [GainLife(Integer(1))])`,
		`MayCost(PayLife(Integer(1))), Unless(CostWasPaid, [GainLife(Integer(1))])`,
		`MayCost(PayMana([])), If(CostWasPaid, [GainLife(Integer(1))]), If(CostWasPaid, [GainLife(Integer(1))])`,
		`MayCost(PayMana([])), Unless(CostWasPaid(), [GainLife(Integer(1))])`,
		`MayCost(PayMana([])), Unless(CostWasPaid, [])`,
		`MayCost(PayMana([])), Unless(CostWasPaid, [TapPermanent(Ref_TargetPermanent)])`,
		`Unless(CostWasPaid, [GainLife(Integer(1))])`,
		`MayCost(PayMana([])), Unless(CostWasPaid, [GainLife(Integer(1))]), Unless(CostWasPaid, [GainLife(Integer(1))])`,
	} {
		src := fmt.Sprintf(`Card(Name: "Bad", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)), ActionList([%s]))])`, actions)
		nodes, err := Parse(strings.NewReader(src), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted invalid payment: %s", actions)
		}
	}
}

func TestRONUpkeepPaymentInventory(t *testing.T) {
	src := `Card(Name: "Inventory", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)), ActionList([MayCost(PayMana([])), Unless(CostWasPaid, [PermanentDealsDamage(ThisPermanent, Integer(8), Player(You))])]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{"PayMana": "cost", "Player": "recipient", "You": "player_reference"}
	for name, role := range roles {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s role for %s", role, name)
		}
	}
}

func TestRONUpkeepPaymentInstancesAndClones(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
	g := mage.NewGame(a, b)
	for _, p := range []mage.Player{a, b} {
		card := repositoryCard(t, "LEB", "Force of Nature")
		card.SetOwner(p.PlayerID())
		g.PutOnBattlefield(card, p.PlayerID())
	}
	queueRONUpkeep(g, a)
	queueRONUpkeep(g, b)
	for range 4 {
		a.ManaPool().Add(core.Green, 1)

	}
	clone := g.Clone()
	g.ResolveStack()
	if g.GetPlayer(a.PlayerID()).Life() != 20 || g.GetPlayer(b.PlayerID()).Life() != 12 {
		t.Fatal("independent trigger instances used the wrong payer")
	}
	if clone.GetPlayer(b.PlayerID()).Life() != 20 || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 4 {
		t.Fatal("resolution changed clone state")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 12 || clone.GetPlayer(b.PlayerID()).Life() != 12 || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 4 {
		t.Fatal("clone lost payment context")
	}
}

func TestRONUpkeepPaymentOtherRules(t *testing.T) {
	g, a, b := ronGame()
	forces := repositoryCard(t, "LEB", "Phantasmal Forces")
	forces.SetOwner(b.PlayerID())
	blue := g.PutOnBattlefield(forces, b.PlayerID())
	ownCard := repositoryCard(t, "LEB", "Phantasmal Forces")
	ownCard.SetOwner(a.PlayerID())
	ownBlue := g.PutOnBattlefield(ownCard, a.PlayerID())
	city := repositoryCard(t, "DRK", "Sunken City")
	city.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(city, a.PlayerID())
	if blue.CurrentPower(g) != 5 || blue.CurrentToughness(g) != 2 || ownBlue.CurrentPower(g) != 5 || !blue.HasKeyword(core.Flying) {
		t.Fatal("city boost or flying rule is missing")
	}
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if g.FindPermanent(source.ID()) != nil || g.FindPermanent(blue.ID()).CurrentPower(g) != 4 || g.FindPermanent(ownBlue.ID()) != nil {
		t.Fatal("city boost survived unpaid sacrifice")
	}
	for _, tc := range upkeepPaymentCards {
		if tc.name == "Sunken City" || tc.name == "Phantasmal Forces" || tc.name == "Conversion" {
			continue
		}
		card := repositoryCard(t, tc.set, tc.name)
		card.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		keyword := core.Flying
		if tc.name == "Force of Nature" {
			keyword = core.Trample
		}
		if !p.HasKeyword(keyword) {
			t.Fatalf("missing keyword on %s", tc.name)
		}
		if tc.name == "Palladia-Mors" && !p.HasKeyword(core.Trample) {
			t.Fatal("missing trample")
		}
		if tc.name == "Vaevictis Asmadi" {
			for i, color := range []core.Color{core.Black, core.Red, core.Green} {
				a.ManaPool().Add(color, 1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), i+1, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
			}
			if g.FindPermanent(p.ID()).CurrentPower(g) != 10 {
				t.Fatal("colored pump abilities are missing")
			}
		}
	}
}

func TestRONUpkeepPaymentZeroAndNested(t *testing.T) {
	for _, accept := range []bool{true, false} {
		factory := permanentScopeFactory(t, "Artifact", `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)), ActionList([MayCost(PayMana([])), Unless(CostWasPaid, [MayCost(PayMana([])), Unless(CostWasPaid, [GainLife(Integer(3))])]), MayCost(PayMana([])), Unless(CostWasPaid, [GainLife(Integer(2))])]))`)
		first, second := factory(), factory()
		if first.Abilities()[0].(*mage.GenericTriggered).Effects()[0] == second.Abilities()[0].(*mage.GenericTriggered).Effects()[0] {
			t.Fatal("factory shared payment effects")
		}
		a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: accept}
		g := mage.NewGame(a, mage.NewBasePlayer("B"))
		first.SetOwner(a.PlayerID())
		g.PutOnBattlefield(first, a.PlayerID())
		queueRONUpkeep(g, a)
		g.ResolveStack()
		life, offers := 25, 3
		if accept {
			life, offers = 20, 2
		}
		if a.Life() != life || a.offers != offers {
			t.Fatalf("nested results or zero choice: life=%d offers=%d", a.Life(), a.offers)
		}
	}
}

func TestRONUpkeepDamageAdditionalCards(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"ARN", "Juzám Djinn"}, {"ARN", "Serendib Efreet"}, {"DRK", "Elves of Deep Shadow"}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			g.SetTurn(2)
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			if tc.name == "Elves of Deep Shadow" {
				source.RevokeBaseAttr(core.AttrSummonSick)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if !g.GetStack().IsEmpty() || a.ManaPool().TotalMana() != 1 || !g.FindPermanent(source.ID()).Tapped {
					t.Fatal("mana ability did not resolve with tap cost")
				}
			} else {
				queueRONUpkeep(g, b)
				if !g.GetStack().IsEmpty() {
					t.Fatal("wrong upkeep triggered damage")
				}
				queueRONUpkeep(g, a)
				if a.Life() != 20 {
					t.Fatal("damage occurred before resolution")
				}
				g.ResolveStack()
				if tc.name == "Serendib Efreet" && !g.FindPermanent(source.ID()).HasKeyword(core.Flying) {
					t.Fatal("flying rule is missing")
				}
			}
			if g.GetPlayer(a.PlayerID()).Life() != 19 || g.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("self damage used wrong recipient")
			}
		})
	}
}

func TestRONUpkeepDamageLKI(t *testing.T) {
	for _, change := range []string{"leave", "return", "phase", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			var events []core.GameEvent
			watcher := mage.NewEnchantment("Observer", "", mage.WithAbility(mage.NewTriggered(core.EvtDamageDealt, false, mage.DrawCards(mage.Fixed(0))).SetCondition(func(e *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
				events = append(events, *e)
				return false
			})))
			g.PutOnBattlefield(watcher, b.PlayerID())
			card := repositoryCard(t, "LEB", "Force of Nature")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			old := g.CaptureDamageSource(source.ID()).Incarnation()
			queueRONUpkeep(g, a)
			switch change {
			case "leave", "return":
				g.DestroyPermanent(source)
				if change == "return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "control":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), source.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			if len(events) != 1 || events[0].Amount != 8 || events[0].SourceID != source.ID() || events[0].SourceIncarnation != old || events[0].TargetID != a.PlayerID() {
				t.Fatalf("wrong damage source or recipient: %+v", events)
			}
		})
	}
}

func TestRONUpkeepDamageInvalid(t *testing.T) {
	for _, action := range []string{
		`PermanentDealsDamage(ThisPermanent, Integer(1), Player(You, You))`,
		`PermanentDealsDamage(ThisPermanent, Integer(1), Player(You()))`,
		`PermanentDealsDamage(ThisPermanent, Integer(1), Player(Ref_TargetPlayer))`,
		`PermanentDealsDamage(Ref_TargetPermanent, Integer(1), Player(You))`,
		`PermanentDealsDamage(ThisPermanent, Integer(1), Player(ThisPermanent))`,
	} {
		src := fmt.Sprintf(`Card(Name: "Bad", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(PayMana([]), ActionList([%s]))])`, action)
		nodes, err := Parse(strings.NewReader(src), "bad-damage.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted invalid damage: %s", action)
		}
	}
}
