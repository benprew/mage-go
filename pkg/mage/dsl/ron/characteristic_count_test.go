package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

var characteristicCards = []struct {
	set, name, member string
	power             int
}{
	{"LEG", "Dakkon Blackblade", "Forest", -1},
	{"LEB", "Keldon Warlord", "Llanowar Elves", -1},
	{"LEB", "Nightmare", "Swamp", -1},
	{"LEB", "Plague Rats", "Plague Rats", -1},
	{"DRK", "People of the Woods", "Forest", 1},
}

func checkRONCountPT(t *testing.T, g *mage.Game, c mage.Card, power, toughness int) {
	t.Helper()
	g.ApplyContinuousEffects()
	p, q := g.CardPowerToughness(c)
	if p != power || q != toughness {
		t.Fatalf("%s P/T %d/%d, want %d/%d", c.Name(), p, q, power, toughness)
	}
}

func TestRONCharacteristicCountGameplay(t *testing.T) {
	for _, tc := range characteristicCards {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			member := repositoryCard(t, "LEB", tc.member)
			member.SetOwner(a.PlayerID())
			own := g.PutOnBattlefield(member, a.PlayerID())
			opposingCard := repositoryCard(t, "LEB", tc.member)
			opposingCard.SetOwner(b.PlayerID())
			opposing := g.PutOnBattlefield(opposingCard, b.PlayerID())
			c := repositoryCard(t, tc.set, tc.name)
			c.SetOwner(a.PlayerID())
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			count := 1
			if tc.name == "Plague Rats" {
				count = 2
			}
			check := func(game *mage.Game, n int) {
				t.Helper()
				p := n
				if tc.power >= 0 {
					p = tc.power
				}
				checkRONCountPT(t, game, c, p, n)
			}
			check(g, count)
			a.AddToHand(c)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			if g.StackSize() != 0 || len(a.Hand()) != 1 {
				t.Fatal("failed cast changed state")
			}
			for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green, core.Colorless} {
				a.ManaPool().Add(color, 10)
			}
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(c.ID()) != nil {
				t.Fatal("entered before resolution")
			}
			check(g, count)
			pending := g.Clone()
			g.ResolveStack()
			if tc.name == "Keldon Warlord" || tc.name == "Plague Rats" {
				count++
			}
			check(g, count)
			pending.ResolveStack()
			check(pending, count)
			if tc.name == "Nightmare" && !g.FindPermanent(c.ID()).HasKeyword(core.Flying) {
				t.Fatal("flying lost")
			}
			if tc.name == "Dakkon Blackblade" && !c.HasSuperType(core.SuperLegendary) {
				t.Fatal("legendary lost")
			}
			clone := g.Clone()
			clone.PhaseOut(own.ID())
			check(clone, count-1)
			check(g, count)
			g.PhaseOut(own.ID())
			check(g, count-1)
			g.PhaseIn([]uuid.UUID{own.ID()})
			check(g, count)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: uuid.New(), TargetID: opposing.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			if tc.name != "Plague Rats" {
				count++
			}
			check(g, count)
			source := g.FindPermanent(c.ID())
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: uuid.New(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
			otherCount := 0
			if tc.name == "Keldon Warlord" {
				otherCount = 1
			}
			if tc.name == "Plague Rats" {
				otherCount = count
			}
			check(g, otherCount)
			g.BouncePermanentToHand(g.FindPermanent(c.ID()))
			if len(a.Hand()) != 1 {
				t.Fatal("return did not use owner")
			}
			ownerCount := 2
			check(g, ownerCount)
			a.RemoveFromHand(c.ID())
			g.PutOnBattlefield(c, a.PlayerID())
			if tc.name == "Keldon Warlord" || tc.name == "Plague Rats" {
				ownerCount++
			}
			check(g, ownerCount)
		})
	}
}

func TestRONCharacteristicCountReprints(t *testing.T) {
	files, err := os.ReadDir("../../../../cards/mtgish")
	if err != nil {
		t.Fatal(err)
	}
	subset := fstest.MapFS{}
	names := map[string]bool{}
	records := 0
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".ron") {
			continue
		}
		data, err := os.ReadFile("../../../../cards/mtgish/" + file.Name())
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), file.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range nodes {
			for _, tc := range characteristicCards {
				if n.Fields["Name"].String != tc.name {
					continue
				}
				factory, err := Compile(n)
				if err != nil {
					t.Fatal(err)
				}
				first, second := factory(), factory()
				if first.ID() == second.ID() {
					t.Fatal("factory identity shared")
				}
				if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
					t.Fatal(err)
				}
				names[first.Name()] = true
				records++
				for line := range strings.SplitSeq(string(data), "\n") {
					if strings.Contains(line, `Name: "`+tc.name+`"`) {
						subset[fmt.Sprintf("%03d.ron", records)] = &fstest.MapFile{Data: []byte(line)}
					}
				}
			}
		}
	}
	if records != 20 || len(names) != 5 {
		t.Fatalf("records/names %d/%d", records, len(names))
	}
	registry, err := LoadFS(subset, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 5 {
		t.Fatal("reprints not canonical")
	}
}

func TestRONCharacteristicCountInvalid(t *testing.T) {
	count := `TheNumberOfPermanentsOnTheBattlefield(And([IsCardtype(Land),ControlledByAPlayer(SinglePlayer(You))]))`
	for _, rules := range []string{
		`CDA_Power()`, `CDA_Power(ValueX)`, `CDA_Toughness(Ref_TargetPermanent)`, `CDA_Toughness(TheNumberOfPermanentsOnTheBattlefield(And([])))`,
		`CDA_Power(` + count + `)`, `CDA_Toughness(TheNumberOfPermanentsOnTheBattlefield(IsCardtype(Land)))`,
		`CDA_Power(` + count + `),CDA_Power(` + count + `),CDA_Toughness(` + count + `)`,
		`CDA_Toughness(` + count + `),CDA_Toughness(` + count + `)`,
		`CDA_Power(` + count + `),CDA_Toughness(TheNumberOfPermanentsOnTheBattlefield(IsCardtype(Creature)))`,
		`CDA_Toughness(TheNumberOfPermanentsOnTheBattlefield(Other(ThisPermanent)))`,
		`CDA_Toughness(TheNumberOfPermanentsOnTheBattlefield(And([PowerIs(EqualTo(Integer(1))),ControlledByAPlayer(SinglePlayer(You))])))`,
		`CDA_Toughness(TheNumberOfPermanentsOnTheBattlefield(IsNamed(NamedCard(""))))`,
		`CDA_Toughness(TheNumberOfPermanentsOnTheBattlefield(ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer))))`,
		`CDA_Toughness(TheNumberOfPermanentsOnTheBattlefield(Or([IsCardtype(Land),ControlledByAPlayer(SinglePlayer(You))])))`,
		`Activated(PayMana([ManaCostG]),ActionList([CDA_Toughness(` + count + `)]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:0),Rules:[`+rules+`])`), "invalid-cda.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rules)
		}
	}
	for _, typ := range []string{"Artifact", "Instant"} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[`+typ+`],Subtypes:[]),Rules:[CDA_Toughness(`+count+`)])`), "invalid-cda.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("noncreature CDA")
		}
	}
}

func TestRONCharacteristicCountMembership(t *testing.T) {
	t.Run("WallsAndTypes", func(t *testing.T) {
		g, a, b := ronGame()
		c := repositoryCard(t, "LEB", "Keldon Warlord")
		c.SetOwner(a.PlayerID())
		g.PutOnBattlefield(c, a.PlayerID())
		wall := g.PutOnBattlefield(repositoryCard(t, "LEB", "Wall of Wood"), a.PlayerID())
		g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), b.PlayerID())
		g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
		checkRONCountPT(t, g, c, 1, 1)
		member := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
		checkRONCountPT(t, g, c, 2, 2)
		g.AddContinuousEffect(mage.AddSubtypes(member.ID(), mage.SubtypeCreature, core.EndOfTurn, "Wall"))
		checkRONCountPT(t, g, c, 1, 1)
		g.RemoveFromBattlefield(wall)
		checkRONCountPT(t, g, c, 1, 1)
	})
	t.Run("Names", func(t *testing.T) {
		g, a, b := ronGame()
		c := repositoryCard(t, "LEB", "Plague Rats")
		c.SetOwner(a.PlayerID())
		g.PutOnBattlefield(c, a.PlayerID())
		g.PutOnBattlefield(mage.NewLand("Plague Rats"), a.PlayerID())
		g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
		checkRONCountPT(t, g, c, 1, 1)
		copyCard := c.Copy()
		copyCard.SetOwner(b.PlayerID())
		g.PutOnBattlefield(copyCard, b.PlayerID())
		checkRONCountPT(t, g, c, 2, 2)
		checkRONCountPT(t, g, copyCard, 2, 2)
		g.PhaseOut(copyCard.ID())
		checkRONCountPT(t, g, c, 1, 1)
	})
	for _, tc := range characteristicCards {
		if tc.member != "Forest" && tc.member != "Swamp" {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, tc.set, tc.name)
			c.SetOwner(a.PlayerID())
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Swamp"), a.PlayerID())
			dual := g.PutOnBattlefield(repositoryCard(t, "LEB", "Bayou"), a.PlayerID())
			g.PutOnBattlefield(repositoryCard(t, "LEB", tc.member), b.PlayerID())
			n := 2
			if tc.name == "Dakkon Blackblade" {
				n = 3
			}
			p := n
			if tc.power >= 0 {
				p = tc.power
			}
			checkRONCountPT(t, g, c, p, n)
			g.PhaseOut(dual.ID())
			n--
			p = n
			if tc.power >= 0 {
				p = tc.power
			}
			checkRONCountPT(t, g, c, p, n)
		})
	}
}

func TestRONCharacteristicCountZonesLayersAndLKI(t *testing.T) {
	for _, tc := range characteristicCards {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			for range 2 {
				g.PutOnBattlefield(repositoryCard(t, "LEB", tc.member), a.PlayerID())
			}
			c := repositoryCard(t, tc.set, tc.name)
			c.SetOwner(a.PlayerID())
			power := 2
			if tc.power >= 0 {
				power = tc.power
			}
			checkRONCountPT(t, g, c, power, 2)
			a.AddToLibrary(c)
			checkRONCountPT(t, g, c, power, 2)
			a.SetLibrary(nil)
			a.AddToGraveyard(c)
			checkRONCountPT(t, g, c, power, 2)
			if _, ok := g.MoveFromGraveyard(a.PlayerID(), c.ID(), core.ZoneExile); !ok {
				t.Fatal("graveyard removal")
			}
			g.ExileCard(c, c.ID())
			checkRONCountPT(t, g, c, power, 2)
			g.RemoveFromExile(c.ID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			n := 2
			if tc.name == "Keldon Warlord" || tc.name == "Plague Rats" {
				n++
			}
			power = n
			if tc.power >= 0 {
				power = tc.power
			}
			checkRONCountPT(t, g, c, power, n)
			copied := repositoryCard(t, "LEB", "Llanowar Elves")
			copied.CloneFrom(c)
			copied.SetOwner(b.PlayerID())
			copiedPower, copiedToughness := 0, 0
			if tc.name == "Plague Rats" {
				copiedPower, copiedToughness = n, n
			}
			if tc.power >= 0 {
				copiedPower = tc.power
			}
			checkRONCountPT(t, g, copied, copiedPower, copiedToughness)
			g.AddCountersWithReplacement(g.FindPermanent(p.ID()), core.P1P1, 1, p.ID(), false)
			checkRONCountPT(t, g, c, power+1, n+1)
			ids := g.PhaseOut(p.ID())
			g.PhaseIn(ids)
			checkRONCountPT(t, g, c, power+1, n+1)
			g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.EndOfTurn))
			printedPower := max(tc.power, 0)
			checkRONCountPT(t, g, c, printedPower+1, 1)
			g.RunStepWithPriority(core.Cleanup)
			checkRONCountPT(t, g, c, power+1, n+1)
			g.SetStep(core.PrecombatMain)
			spell := repositoryCard(t, "LEB", "Giant Growth")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.Green, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			checkRONCountPT(t, g, c, power+1, n+1)
			g.ResolveStack()
			checkRONCountPT(t, g, c, power+4, n+4)
			exile := repositoryCard(t, "LEB", "Swords to Plowshares")
			exile.SetOwner(b.PlayerID())
			b.AddToHand(exile)
			b.ManaPool().Add(core.White, 1)
			if err := g.CastSpellByID(b.PlayerID(), exile.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			g.ResolveStack()
			if g.FindPermanent(p.ID()) != nil || g.FindExiledCard(c.ID()) == nil || a.Life() != 20+power+4 {
				t.Fatal("exile lost power/controller LKI")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 20+power+4 {
				t.Fatal("cloned LKI")
			}
			checkRONCountPT(t, g, c, func() int {
				if tc.power >= 0 {
					return tc.power
				}
				return 2
			}(), 2)
		})
	}
}

func TestRONCharacteristicCountSpellController(t *testing.T) {
	g, a, b := ronGame()
	for range 2 {
		g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
	}
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
	c := repositoryCard(t, "LEG", "Dakkon Blackblade")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Colorless} {
		a.ManaPool().Add(color, 10)
	}
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.StackPeek().Controller = b.PlayerID()
	checkRONCountPT(t, g, c, 1, 1)
	copied := g.CopySpellOnStack(c.ID(), a.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	checkRONCountPT(t, g, copied.Card, 2, 2)
	clone := g.Clone()
	g.ResolveStack()
	checkRONCountPT(t, clone, c, 1, 1)
	g.ApplyContinuousEffects()
	checkRONCountPT(t, g, c, 1, 1)
}

func TestRONCharacteristicCountInventoryRoles(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "People of the Woods"`) {
			record = line
		}
	}
	report, err := InspectFS(fstest.MapFS{"cda.ron": {Data: []byte(record)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{"CDA_Toughness": "rule", "TheNumberOfPermanentsOnTheBattlefield": "characteristic_count_value", "And": "permanent_filter", "IsLandType": "permanent_filter", "ControlledByAPlayer": "permanent_filter", "SinglePlayer": "player_filter", "You": "player_reference"}
	found := map[string]bool{}
	for _, entry := range report.Signatures {
		if role, ok := roles[entry.Name]; ok {
			found[entry.Name] = true
			if entry.Role != role {
				t.Fatalf("%s role %s, want %s", entry.Name, entry.Role, role)
			}
		}
	}
	if len(found) != len(roles) || report.Compiled != 1 || len(report.Issues) != 0 {
		t.Fatal("missing roles or incomplete card")
	}
}

func TestRONCharacteristicCountPairOrderAndInstances(t *testing.T) {
	value := `TheNumberOfPermanentsOnTheBattlefield(And([Or([IsLandType(Forest),IsLandType(Swamp)]),ControlledByAPlayer(SinglePlayer(You))]))`
	src := `Card(Name:"Count",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:0,Toughness:0),Rules:[CDA_Toughness(` + value + `),CDA_Power(` + value + `)])`
	nodes, err := Parse(strings.NewReader(src), "order.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Bayou"), a.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Swamp"), b.PlayerID())
	first, second := factory(), factory()
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	checkRONCountPT(t, g, first, 1, 1)
	checkRONCountPT(t, g, second, 2, 2)
	clone := g.Clone()
	clone.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
	checkRONCountPT(t, clone, first, 2, 2)
	checkRONCountPT(t, g, first, 1, 1)
	checkRONCountPT(t, g, second, 2, 2)
}
