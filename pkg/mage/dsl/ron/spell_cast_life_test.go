package ron

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

var spellCastLifeCards = []struct {
	set, name, spell string
	paid             bool
}{
	{"LEB", "Crystal Rod", "Ancestral Recall", true},
	{"LEB", "Iron Star", "Lightning Bolt", true},
	{"LEB", "Ivory Cup", "Savannah Lions", true},
	{"LEB", "Throne of Bone", "Dark Ritual", true},
	{"LEB", "Wooden Sphere", "Llanowar Elves", true},
	{"ATQ", "Urza's Chalice", "Sol Ring", true},
	{"LEG", "Sol'kanar the Swamp King", "Dark Ritual", false},
}

func TestRONSpellCastLifeRealCards(t *testing.T) {
	for _, tc := range spellCastLifeCards {
		for _, scenario := range []string{"pay", "decline", "unaffordable", "mana-source", "rollback", "departed", "returned", "control", "phased", "phased-before", "other-caster", "nonmatching", "copy", "countered", "failed-cast"} {
			t.Run(tc.name+"/"+scenario, func(t *testing.T) {
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario != "decline"}
				b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(b.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				if tc.name == "Sol'kanar the Swamp King" && !source.HasKeyword(core.Swampwalk) {
					t.Fatal("missing swampwalk")
				}
				if scenario == "phased-before" {
					g.PhaseOut(source.ID())
				}
				caster := b.BasePlayer
				if scenario == "other-caster" {
					caster = a.BasePlayer
				}
				spell := repositoryCard(t, "LEB", tc.spell)
				if scenario == "nonmatching" {
					name := "Dark Ritual"
					if tc.spell == name {
						name = "Lightning Bolt"
					}
					spell = repositoryCard(t, "LEB", name)
				}
				spell.SetOwner(caster.PlayerID())
				caster.AddToHand(spell)
				if scenario != "failed-cast" {
					for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green, core.Colorless} {
						caster.ManaPool().Add(color, 20)
					}
				}
				var targetIDs []uuid.UUID
				if len(spell.Abilities()) > 0 && len(spell.Abilities()[0].(*mage.ActionDefinition).Targets()) > 0 {
					targetIDs = []uuid.UUID{source.ID()}
					if tc.spell == "Ancestral Recall" || spell.Name() == "Lightning Bolt" {
						targetIDs = []uuid.UUID{b.PlayerID()}
					}
				}
				if caster.PlayerID() == b.PlayerID() {
					g.SetActivePlayerIndex(1)
				}
				err := g.CastSpellByID(caster.PlayerID(), spell.ID(), targetIDs, 0)
				g.PutTriggersOnStack()
				if scenario == "failed-cast" {
					if err == nil || !g.GetStack().IsEmpty() || a.offers != 0 {
						t.Fatal("failed cast triggered")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				matching := scenario != "nonmatching" && scenario != "phased-before"
				wantStack := 1
				if matching {
					wantStack++
				}
				if g.GetStack().Size() != wantStack || a.offers != 0 || b.offers != 0 || a.Life() != 20 {
					t.Fatalf("announcement: stack=%d offers=%d/%d life=%d", g.GetStack().Size(), a.offers, b.offers, a.Life())
				}
				if !matching {
					return
				}
				a.ManaPool().Clear()
				b.ManaPool().Clear()
				b.ManaPool().Add(core.Colorless, 5)
				var land *mage.Permanent
				if scenario == "mana-source" || scenario == "rollback" {
					c := repositoryCard(t, "LEB", "Forest")
					c.SetOwner(a.PlayerID())
					land = g.PutOnBattlefield(c, a.PlayerID())
					if scenario == "rollback" {
						g.TapPermanent(land)
					}
				} else if scenario != "unaffordable" {
					a.ManaPool().Add(core.Colorless, 1)
				}
				switch scenario {
				case "departed", "returned":
					g.DestroyPermanent(g.FindPermanent(source.ID()))
					if scenario == "returned" {
						g.PutOnBattlefield(card, b.PlayerID())
					}
				case "control":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), source.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "phased":
					g.PhaseOut(source.ID())
				case "copy":
					if g.CopySpellOnStack(spell.ID(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
					if g.GetStack().Size() != 3 {
						t.Fatal("copy triggered cast ability")
					}
					g.GetStack().RemoveByID(g.StackPeek().ID)
				case "countered":
					for _, object := range g.GetStack().Objects() {
						if object.Card != nil && object.SourceID == spell.ID() {
							g.GetStack().RemoveByID(object.ID)
							break
						}
					}
				}
				cloned := g.Clone()
				cloned.SetPlayerAt(0, &ronPaymentPlayer{BasePlayer: cloned.GetPlayer(a.PlayerID()).(*mage.SearchPlayer).BasePlayer, accept: a.accept})
				g.ResolveTopOfStack()
				gain := !tc.paid || scenario != "decline" && scenario != "unaffordable" && scenario != "rollback"
				wantLife := 20
				if gain {
					wantLife++
				}
				liveA, liveB := g.GetPlayer(a.PlayerID()), g.GetPlayer(b.PlayerID())
				if liveA.Life() != wantLife || liveB.Life() != 20 || liveB.ManaPool().TotalMana() != 5 {
					t.Fatal("wrong payment or recipient")
				}
				offers := 0
				if tc.paid && scenario != "unaffordable" && scenario != "rollback" {
					offers = 1
				}
				if a.offers != offers || b.offers != 0 {
					t.Fatal("wrong resolution choice")
				}
				if tc.paid && gain {
					if liveA.ManaPool().TotalMana() != 0 {
						t.Fatal("wrong payment amount")
					}
					if land != nil && !g.FindPermanent(land.ID()).Tapped {
						t.Fatal("mana source not used")
					}
				}
				if scenario == "decline" && liveA.ManaPool().TotalMana() != 1 {
					t.Fatal("refusal consumed mana")
				}
				if cloned.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("clone changed with original")
				}
				cloned.ResolveTopOfStack()
				if cloned.GetPlayer(a.PlayerID()).Life() != wantLife {
					t.Fatal("clone lost trigger context")
				}
			})
		}
	}
}

func TestRONSpellCastLifeInvalid(t *testing.T) {
	for _, rule := range []string{
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer), ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer(), AnySpell), ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPlayerCastsASpell(SinglePlayer(You), AnySpell), ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer, SingleSpell(ThisSpell)), ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer, AnySpell), ActionList([PlayerAction(Trigger_ThatPlayer, GainLife(Integer(1)))]))`,
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer, AnySpell), ActionList([CounterSpell(Trigger_ThatSpell())]))`,
		`Activated(PayMana([]), ActionList([If(CostWasPaid, [GainLife(Integer(1))])]))`,
		`Activated(PayMana([]), ActionList([MayCost(PayMana([])), If(CostWasPaid(), [GainLife(Integer(1))])]))`,
		`Activated(PayMana([]), ActionList([MayCost(PayMana([])), If(CostWasPaid, [])]))`,
		`Activated(PayMana([]), ActionList([MayCost(PayMana([])), GainLife(Integer(1)), If(CostWasPaid, [GainLife(Integer(1))])]))`,
		`Activated(PayMana([]), ActionList([MayCost(PayMana([ManaCostX])), If(CostWasPaid, [GainLife(Integer(1))])]))`,
		`Activated(PayMana([]), ActionList([MayCost(PayLife(ValueX)), If(CostWasPaid, [GainLife(Integer(1))])]))`,
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

func TestRONSpellCastLifeInventory(t *testing.T) {
	src := `Card(Name: "Life", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [TriggerA(WhenAPlayerCastsASpell(AnyPlayer, IsColor(Blue)), ActionList([MayCost(PayMana([])), If(CostWasPaid, [GainLife(Integer(1))])]))])`
	report, err := InspectFS(fstest.MapFS{"life.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"AnyPlayer": "player_filter", "IsColor": "spell_filter", "CostWasPaid": "condition", "GainLife": "action"} {
		if !slices.ContainsFunc(report.Signatures, func(s InventorySignature) bool { return s.Name == name && s.Role == role }) {
			t.Fatalf("missing %s role for %s", role, name)
		}
	}
}

func TestRONSpellCastLifeIndependentInstances(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	for _, player := range []*ronPaymentPlayer{a, b} {
		for range 2 {
			card := repositoryCard(t, "LEB", "Throne of Bone")
			card.SetOwner(player.PlayerID())
			g.PutOnBattlefield(card, player.PlayerID())
		}
	}
	spell := repositoryCard(t, "LEG", "Xira Arien")
	castRONSpell(t, g, a.BasePlayer, spell)
	g.PutTriggersOnStack()
	if g.GetStack().Size() != 5 {
		t.Fatal("multicolor black spell did not trigger all sources")
	}
	a.ManaPool().Clear()
	b.ManaPool().Clear()
	a.ManaPool().Add(core.Colorless, 1)
	b.ManaPool().Add(core.Colorless, 2)
	for range 4 {
		g.ResolveTopOfStack()
	}
	if g.GetPlayer(a.PlayerID()).Life() != 21 || g.GetPlayer(b.PlayerID()).Life() != 22 || a.offers != 1 || b.offers != 2 {
		t.Fatal("instances shared payment or player")
	}
}

func TestRONSpellCastLifePaymentContexts(t *testing.T) {
	for _, tc := range []struct {
		actions            string
		accept             bool
		mana, life, offers int
	}{
		{`MayCost(PayMana([])), If(CostWasPaid, [GainLife(Integer(1))])`, true, 0, 21, 1},
		{`MayCost(PayMana([])), If(CostWasPaid, [GainLife(Integer(1))])`, false, 0, 20, 1},
		{`MayCost(PayMana([ManaCostGeneric(1)])), If(CostWasPaid, [GainLife(Integer(1)), MayCost(PayMana([ManaCostGeneric(1)])), If(CostWasPaid, [GainLife(Integer(8))])]), MayCost(PayMana([ManaCostGeneric(1)])), If(CostWasPaid, [GainLife(Integer(4))])`, true, 1, 21, 1},
		{`MayCost(PayMana([ManaCostG, ManaCostU])), If(CostWasPaid, [GainLife(Integer(1))])`, true, 0, 20, 0},
		{`PlayerMayCost(Ref_TargetPlayer, PayMana([ManaCostGeneric(1)])), If(CostWasPaid, [PlayerAction(Ref_TargetPlayer, DrawACard)])`, true, 1, 20, 1},
	} {
		t.Run(tc.actions+fmt.Sprint(tc.accept), func(t *testing.T) {
			targeted := strings.Contains(tc.actions, "Ref_TargetPlayer")
			actions := `ActionList([` + tc.actions + `])`
			if targeted {
				actions = `Targeted([TargetPlayer(AnyPlayer)], ` + actions + `)`
			}
			factory := permanentScopeFactory(t, "Instant", `SpellActions(`+actions+`)`)
			first, second := factory(), factory()
			if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
				t.Fatal(err)
			}
			if first.Abilities()[0].(*mage.ActionDefinition).Effects()[0] == second.Abilities()[0].(*mage.ActionDefinition).Effects()[0] {
				t.Fatal("shared payment effects")
			}
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: tc.accept}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: tc.accept}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			payer := a
			var targets []uuid.UUID
			if targeted {
				payer = b
				targets = []uuid.UUID{b.PlayerID()}
			}
			fillRONLibrary(t, b.BasePlayer, 2)
			payer.ManaPool().Add(core.Colorless, tc.mana)
			land := repositoryCard(t, "LEB", "Forest")
			land.SetOwner(payer.PlayerID())
			source := g.PutOnBattlefield(land, payer.PlayerID())
			if !strings.Contains(tc.actions, "ManaCostG,") {
				g.TapPermanent(source)
			}
			first.SetOwner(a.PlayerID())
			a.AddToHand(first)
			if err := g.CastSpellByID(a.PlayerID(), first.ID(), targets, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveTopOfStack()
			if g.GetPlayer(payer.PlayerID()).Life() != tc.life || payer.offers != tc.offers {
				t.Fatalf("paid branch result or payer incorrect: life=%d offers=%d", g.GetPlayer(payer.PlayerID()).Life(), payer.offers)
			}
			if targeted && (len(g.GetPlayer(b.PlayerID()).Hand()) != 1 || len(g.GetPlayer(a.PlayerID()).Hand()) != 0 || a.offers != 0) {
				t.Fatal("wrong paid target binding")
			}
			if strings.Contains(tc.actions, "ManaCostG,") && (g.FindPermanent(source.ID()).Tapped || g.GetPlayer(payer.PlayerID()).ManaPool().TotalMana() != 0) {
				t.Fatal("failed colored payment consumed source")
			}
		})
	}
}

func TestRONSpellCastLifeSharedContexts(t *testing.T) {
	for _, context := range []string{"activation", "upkeep", "entry"} {
		t.Run(context, func(t *testing.T) {
			actions := `ActionList([MayCost(PayMana([])), If(CostWasPaid, [GainLife(Integer(1))])])`
			rule := `Activated(PayMana([]), ` + actions + `)`
			if context == "upkeep" {
				rule = `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)), ` + actions + `)`
			}
			if context == "entry" {
				rule = `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)), ` + actions + `)`
			}
			factory := permanentScopeFactory(t, "Artifact", rule)
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			g := mage.NewGame(a, mage.NewBasePlayer("B"))
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			switch context {
			case "activation":
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			case "upkeep":
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			}
			if a.offers != 0 {
				t.Fatal("payment before resolution")
			}
			g.ResolveStack()
			if g.GetPlayer(a.PlayerID()).Life() != 21 || a.offers != 1 {
				t.Fatal("shared paid context failed")
			}
		})
	}
}

func TestRONSpellCastLifeFilterCombinations(t *testing.T) {
	for _, tc := range []struct {
		filter  string
		matches bool
	}{
		{`AnySpell`, true},
		{`And([IsColor(Black), IsCardtype(Creature)])`, true},
		{`And([IsColor(Black), IsCardtype(Artifact)])`, false},
		{`Or([IsColor(White), IsColor(Green)])`, true},
		{`IsColor(White)`, false},
	} {
		t.Run(tc.filter, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPlayerCastsASpell(AnyPlayer, `+tc.filter+`), ActionList([GainLife(Integer(1))]))`)
			g, a, b := ronGame()
			source := factory()
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			castRONSpell(t, g, a, repositoryCard(t, "LEG", "Xira Arien"))
			g.PutTriggersOnStack()
			wantStack := 1
			if tc.matches {
				wantStack++
			}
			if g.GetStack().Size() != wantStack {
				t.Fatal("wrong event filter")
			}
			if tc.matches {
				g.ResolveTopOfStack()
				if a.Life() != 21 || b.Life() != 20 {
					t.Fatal("wrong event recipient")
				}
			}
		})
	}
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Sol'kanar the Swamp King")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	if mage.HasLandwalkEvasion(source, b.PlayerID(), g) {
		t.Fatal("swampwalk without swamp")
	}
	swamp := repositoryCard(t, "LEB", "Swamp")
	swamp.SetOwner(b.PlayerID())
	g.PutOnBattlefield(swamp, b.PlayerID())
	if !mage.HasLandwalkEvasion(source, b.PlayerID(), g) {
		t.Fatal("swampwalk lost")
	}
}
