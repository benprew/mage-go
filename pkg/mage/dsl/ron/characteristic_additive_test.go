package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCharacteristicAdditiveAvenger(t *testing.T) {
	g, a, b := ronGame()
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Sol Ring"), a.PlayerID())
	firstCard := repositoryCard(t, "LEB", "Sol Ring")
	firstCard.SetOwner(b.PlayerID())
	first := g.PutOnBattlefield(firstCard, b.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEB", "Sol Ring"), b.PlayerID())
	c := repositoryCard(t, "ATQ", "Gaea's Avenger")
	c.SetOwner(a.PlayerID())
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	checkRONCountPT(t, g, c, 3, 3)
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast")
	}
	if len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("failed cast changed state")
	}
	a.ManaPool().Add(core.Green, 3)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	checkRONCountPT(t, g, c, 3, 3)
	if g.FindPermanent(c.ID()) != nil {
		t.Fatal("entry before resolution")
	}
	clone := g.Clone()
	g.ResolveStack()
	checkRONCountPT(t, g, c, 3, 3)
	clone.ResolveStack()
	checkRONCountPT(t, clone, c, 3, 3)
	ids := g.PhaseOut(first.ID())
	checkRONCountPT(t, g, c, 2, 2)
	g.PhaseIn(ids)
	checkRONCountPT(t, g, c, 3, 3)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: uuid.New(), TargetID: second.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	checkRONCountPT(t, g, c, 2, 2)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: uuid.New(), TargetID: c.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	checkRONCountPT(t, g, c, 3, 3)
	g.DestroyPermanent(g.FindPermanent(first.ID()))
	checkRONCountPT(t, g, c, 3, 3)
	g.BouncePermanentToHand(g.FindPermanent(c.ID()))
	checkRONCountPT(t, g, c, 1, 1)
	a.RemoveFromHand(c.ID())
	g.PutOnBattlefield(c, a.PlayerID())
	checkRONCountPT(t, g, c, 1, 1)
}

func TestRONCharacteristicAdditiveCopiesAndLKI(t *testing.T) {
	g, a, b := ronGame()
	for range 2 {
		g.PutOnBattlefield(repositoryCard(t, "LEB", "Sol Ring"), b.PlayerID())
	}
	c := repositoryCard(t, "ATQ", "Gaea's Avenger")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	a.ManaPool().Add(core.Green, 3)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.StackPeek().Controller = b.PlayerID()
	checkRONCountPT(t, g, c, 1, 1)
	copySpell := g.CopySpellOnStack(c.ID(), a.PlayerID(), false)
	if copySpell == nil {
		t.Fatal("missing copied spell")
	}
	checkRONCountPT(t, g, copySpell.Card, 3, 3)
	g.ResolveStack()
	checkRONCountPT(t, g, c, 1, 1)
	checkRONCountPT(t, g, copySpell.Card, 3, 3)
	physicalCopy := repositoryCard(t, "LEB", "Llanowar Elves")
	physicalCopy.CloneFrom(c)
	physicalCopy.SetOwner(a.PlayerID())
	checkRONCountPT(t, g, physicalCopy, 3, 3)
	p := g.PutOnBattlefield(physicalCopy, a.PlayerID())
	g.AddCountersWithReplacement(p, core.P1P1, 1, p.ID(), false)
	checkRONCountPT(t, g, physicalCopy, 4, 4)
	g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.EndOfTurn))
	checkRONCountPT(t, g, physicalCopy, 2, 2)
	g.RunStepWithPriority(core.Cleanup)
	checkRONCountPT(t, g, physicalCopy, 4, 4)
	g.SetStep(core.PrecombatMain)
	for _, name := range []string{"Giant Growth", "Swords to Plowshares"} {
		spell := repositoryCard(t, "LEB", name)
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		a.ManaPool().Add(core.Green, 1)
		a.ManaPool().Add(core.White, 1)
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
			t.Fatal(err)
		}
		clone := g.Clone()
		g.ResolveStack()
		clone.ResolveStack()
		if name == "Giant Growth" {
			checkRONCountPT(t, g, physicalCopy, 7, 7)
			checkRONCountPT(t, clone, physicalCopy, 7, 7)
		} else if a.Life() != 27 || clone.GetPlayer(a.PlayerID()).Life() != 27 || g.FindExiledCard(physicalCopy.ID()) == nil {
			t.Fatal("departure lost additive power LKI")
		}
	}
	checkRONCountPT(t, g, physicalCopy, 3, 3)
}

func additiveRONSource(value string) string {
	return `Card(Name:"Sum",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[CDA_Power(` + value + `),CDA_Toughness(` + value + `)])`
}

func TestRONCharacteristicAdditiveDeclarations(t *testing.T) {
	for _, filter := range []string{
		`IsCardtype(Artifact)`,
		`And([IsCardtype(Artifact),ControlledByAPlayer(SinglePlayer(You))])`,
		`And([ControlledByAPlayer(Opponent),Or([IsCardtype(Artifact),IsCardtype(Land)])])`,
	} {
		src := additiveRONSource(`Plus(Integer(-1),TheNumberOfPermanentsOnTheBattlefield(` + filter + `))`)
		registry, err := LoadFS(fstest.MapFS{"sum.ron": {Data: []byte(src)}}, "*.ron")
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		g.PutOnBattlefield(mage.NewArtifact("Own", ""), a.PlayerID())
		g.PutOnBattlefield(mage.NewArtifact("Opposing", ""), b.PlayerID())
		g.PutOnBattlefield(mage.NewLand("Opposing land"), b.PlayerID())
		c, err := registry.CreateCard("Sum")
		if err != nil {
			t.Fatal(err)
		}
		c.SetOwner(a.PlayerID())
		want := 1
		if strings.Contains(filter, "SinglePlayer") {
			want = 0
		}
		checkRONCountPT(t, g, c, want, want)
		g.PutOnBattlefield(c, a.PlayerID())
		checkRONCountPT(t, g, c, want, want)
	}
}

func TestRONCharacteristicAdditiveInvalid(t *testing.T) {
	count := `TheNumberOfPermanentsOnTheBattlefield(IsCardtype(Artifact))`
	for _, value := range []string{
		`Plus`, `Plus(Integer(1))`, `Plus(Integer(1),` + count + `,Integer(2))`,
		`Plus(ValueX,` + count + `)`, `Plus(Integer(1),ValueX)`,
		`Plus(Integer(1),Plus(Integer(1),` + count + `))`, `Plus(` + count + `,Integer(1))`,
		`Plus(Integer(1),TheNumberOfPermanentsOnTheBattlefield(And([])))`,
		`Plus(Integer(1),TheNumberOfPermanentsOnTheBattlefield(Other(ThisPermanent)))`,
		`Plus(Integer(1),TheNumberOfPermanentsOnTheBattlefield(ControlledByAPlayer(Ref_TargetPlayer)))`,
		`Plus(Integer(1),TheNumberOfPermanentsOnTheBattlefield(ControlledByAPlayer(Opponent())))`,
		`Plus(Integer(1),TheNumberOfPermanentsOnTheBattlefield(Or([IsCardtype(Artifact),ControlledByAPlayer(Opponent)])))`,
		`Plus(Integer(1),TheNumberOfPermanentsOnTheBattlefield(And([ControlledByAPlayer(Opponent),ControlledByAPlayer(SinglePlayer(You))])))`,
	} {
		nodes, err := Parse(strings.NewReader(additiveRONSource(value)), "invalid-sum.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
	value := `Plus(Integer(1),` + count + `)`
	for _, src := range []string{
		strings.ReplaceAll(additiveRONSource(value), "[Creature]", "[Artifact]"),
		strings.ReplaceAll(additiveRONSource(value), "[Creature]", "[Instant]"),
		strings.Replace(additiveRONSource(value), `CDA_Power(`+value+`),`, "", 1),
		strings.Replace(additiveRONSource(value), `,CDA_Toughness(`+value+`)`, "", 1),
		strings.Replace(additiveRONSource(value), `CDA_Power(`+value+`)`, `CDA_Power(`+value+`),CDA_Power(`+value+`)`, 1),
		strings.Replace(additiveRONSource(value), `CDA_Toughness(`+value+`)`, `CDA_Toughness(Plus(Integer(2),`+count+`))`, 1),
		strings.Replace(additiveRONSource(value), `CDA_Power(`+value+`)`, `Activated(PayMana([]),ActionList([CDA_Power(`+value+`)]))`, 1),
	} {
		nodes, err := Parse(strings.NewReader(src), "invalid-sum.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", src)
		}
	}
}

func TestRONCharacteristicAdditiveInstancesAndMembership(t *testing.T) {
	g, a, b := ronGame()
	first := repositoryCard(t, "ATQ", "Gaea's Avenger")
	second := repositoryCard(t, "ATQ", "Gaea's Avenger")
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	member := repositoryCard(t, "LEB", "Llanowar Elves")
	member.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(member, b.PlayerID())
	checkRONCountPT(t, g, first, 1, 1)
	g.AddContinuousEffect(mage.TargetEffect(core.LayerType, core.EndOfTurn, p.ID(), func(game *mage.Game, p *mage.Permanent) error {
		game.GrantAttr(p.ID(), core.AttrIsArtifact)
		return nil
	}))
	checkRONCountPT(t, g, first, 2, 2)
	checkRONCountPT(t, g, second, 1, 1)
	clone := g.Clone()
	clone.PhaseOut(p.ID())
	checkRONCountPT(t, clone, first, 1, 1)
	checkRONCountPT(t, g, first, 2, 2)
	g.RunStepWithPriority(core.Cleanup)
	checkRONCountPT(t, g, first, 1, 1)
}
