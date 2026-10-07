package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONExilePaymentNecropolis(t *testing.T) {
	for _, change := range []string{"none", "depart", "return", "phase", "phase-cycle", "control", "exile-departure"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "Necropolis")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 {
				t.Fatal("unpaid creature cast")
			}
			a.ManaPool().Add(core.Colorless, 5)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			if source.CanDeclareAsAttacker(g) {
				t.Fatal("defender attacked")
			}
			if !source.HasKeyword(core.Defender) || source.CurrentPower(g) != 0 || source.CurrentToughness(g) != 1 {
				t.Fatal("metadata or defender")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || g.StackSize() != 0 {
				t.Fatal("empty payment accepted")
			}
			noncreature := mage.NewLand("Not a creature")
			noncreature.SetOwner(a.PlayerID())
			a.AddToGraveyard(noncreature)
			foreign := mage.NewCreature("Foreign", "{9}", 1, 1)
			foreign.SetOwner(b.PlayerID())
			b.AddToGraveyard(foreign)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
				t.Fatal("wrong card or graveyard paid")
			}
			for _, cost := range []string{"{2}", "{5}"} {
				c := mage.NewCreature("Payment", cost, 1, 1)
				c.SetOwner(a.PlayerID())
				a.AddToGraveyard(c)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			}
			if len(a.Graveyard()) != 1 || len(b.Graveyard()) != 1 || g.FindPermanent(source.ID()).Counters[core.P0P1] != 0 {
				t.Fatal("payment timing or scope")
			}
			if g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			switch change {
			case "depart":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
			case "phase":
				g.PhaseOut(source.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "exile-departure":
				if len(g.RemoveExiledCardBySource(source.ID())) != 2 {
					t.Fatal("exiled cards did not leave")
				}
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				p := game.FindPermanentIncludingPhased(source.ID())
				if change == "depart" {
					if p != nil {
						t.Fatal("absent source restored")
					}
					continue
				}
				want := 9
				if change == "return" || change == "phase" {
					want = 0
				}
				if int(p.Counters[core.P0P1]) != want {
					t.Fatalf("counters=%d want %d", p.Counters[core.P0P1], want)
				}
			}
		})
	}
}

func TestRONExilePaymentCane(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "ATQ", "Feldon's Cane")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	for _, p := range []*mage.BasePlayer{a, b} {
		c := mage.NewLand("Graveyard")
		c.SetOwner(p.PlayerID())
		p.AddToGraveyard(c)
	}
	source := g.FindPermanent(card.ID())
	g.TapPermanent(source)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || g.FindPermanent(source.ID()) == nil {
		t.Fatal("tap rollback")
	}
	g.UntapPermanent(g.FindPermanent(source.ID()))
	if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("foreign activation")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(source.ID()) != nil || len(a.Graveyard()) != 1 || len(a.Library()) != 0 {
		t.Fatal("exile cost or early shuffle")
	}
	clone := g.Clone()
	g.ResolveStack()
	if len(a.Library()) != 1 || len(a.Graveyard()) != 0 || len(b.Graveyard()) != 1 || len(clone.GetPlayer(a.PlayerID()).Graveyard()) != 1 {
		t.Fatal("shuffle scope or clone isolation")
	}
	clone.ResolveStack()
	if len(clone.GetPlayer(a.PlayerID()).Library()) != 1 {
		t.Fatal("clone resolution")
	}
}

func TestRONExilePaymentInvalid(t *testing.T) {
	for _, cost := range []string{
		`Exile`, `Exile()`, `Exile([],[])`, `Exile(Permanent(ThisPermanent),[])`,
		`Exile([ACardInGraveyards(And([]))],[])`,
		`Exile([ACardInGraveyards(And([InAPlayersGraveyard(SinglePlayer(You))]))],[])`,
		`Exile([Permanent(HostPermanent)],[])`, `Exile([Permanent(ThisPermanent())],[])`,
		`Exile([Permanent(ThisPermanent),Permanent(ThisPermanent)],[])`, `Exile([Permanent(ThisPermanent)],[DrawNumberCards(Integer(1))])`,
		`Exile([ACardInGraveyards(IsCardtype(Creature))],[])`,
		`Exile([ACardInGraveyards(And([IsCardtype(Creature),InAPlayersGraveyard(SinglePlayer(Opponent))]))],[])`,
		`And([Exile([ACardInGraveyards(And([IsCardtype(Creature),InAPlayersGraveyard(SinglePlayer(You))]))],[]),Exile([ACardInGraveyards(And([IsCardtype(Creature),InAPlayersGraveyard(SinglePlayer(You))]))],[])])`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(` + cost + `,ActionList([GainLife(ManaValueOfExiled(TheCardExiledThisWay))]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", cost)
		}
	}
	assertConditionalRejected(t, `Activated(PayMana([]),ActionList([GainLife(ManaValueOfExiled(TheCardExiledThisWay))]))`)
}

const ronGraveyardExileCost = `Exile([ACardInGraveyards(And([IsCardtype(Creature),InAPlayersGraveyard(SinglePlayer(You))]))],[])`

type ronExileChoicePlayer struct {
	*mage.BasePlayer
	choice mage.Card
	seen   int
}

func (p *ronExileChoicePlayer) ChooseCardFromGraveyard(cards []mage.Card, _ string, _ mage.GameReader) mage.Card {
	p.seen++
	if p.choice != nil {
		return p.choice
	}
	return cards[len(cards)-1]
}

func TestRONExilePaymentChoiceAndRollback(t *testing.T) {
	for _, mode := range []string{"choice", "invalid", "rollback", "two-instances"} {
		t.Run(mode, func(t *testing.T) {
			a := &ronExileChoicePlayer{BasePlayer: mage.NewBasePlayer("A")}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			rule := `Activated(` + ronGraveyardExileCost + `,ActionList([GainLife(ManaValueOfExiled(TheCardExiledThisWay))]))`
			if mode == "rollback" {
				rule = `Activated(And([` + ronGraveyardExileCost + `,PayLife(Integer(25))]),ActionList([GainLife(ManaValueOfExiled(TheCardExiledThisWay))]))`
			}
			factory := permanentScopeFactory(t, "Artifact", rule)
			c := factory()
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(c, a.PlayerID())
			cards := []mage.Card{mage.NewCreature("Two", "{2}", 1, 1), mage.NewCreature("Five", "{5}", 1, 1)}
			for _, card := range cards {
				card.SetOwner(a.PlayerID())
				a.AddToGraveyard(card)
			}
			if mode == "invalid" {
				a.choice = mage.NewCreature("Foreign", "{9}", 1, 1)
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
			if mode == "invalid" || mode == "rollback" {
				if err == nil || a.Life() != 20 || len(a.Graveyard()) != 2 || len(g.GetExile()) != 0 || g.StackSize() != 0 {
					t.Fatal("failed total payment changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if g.FindExiledCard(cards[1].ID()) == nil || g.FindExiledCard(cards[0].ID()) != nil || a.seen != 1 || a.Life() != 20 {
				t.Fatal("locked choice or timing")
			}
			want := 25
			if mode == "two-instances" {
				second := g.PutOnBattlefield(factory(), a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), second.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
				want = 27
			}
			clone := g.Clone()
			g.ResolveStack()
			if a.Life() != want || (mode == "two-instances" && b.Life() != 22) || clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("instance results, controller, or clone")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != want {
				t.Fatal("clone saved values")
			}
		})
	}
}

func TestRONExilePaymentScopeAndInventory(t *testing.T) {
	for _, rule := range []string{
		`Activated(` + ronGraveyardExileCost + `,ActionList([])),Activated(PayMana([]),ActionList([GainLife(ManaValueOfExiled(TheCardExiledThisWay))]))`,
		`Activated(Exile([Permanent(ThisPermanent)],[]),ActionList([GainLife(ManaValueOfExiled(TheCardExiledThisWay))]))`,
		`Activated(` + ronGraveyardExileCost + `,ActionList([GainLife(ManaValueOfExiled(TheCardExiledThisWay()))]))`,
		`Activated(` + ronGraveyardExileCost + `,ActionList([GainLife(ManaValueOfExiled(TheCardExiledThisWay,TheCardExiledThisWay))]))`,
		`Activated(` + ronGraveyardExileCost + `,ActionList([ShuffleGraveyardIntoLibrary()]))`,
		`SpellActions(ActionList([Exile([Permanent(ThisPermanent)],[])]))`,
	} {
		assertConditionalRejected(t, rule)
	}
	files := fstest.MapFS{}
	for _, entry := range []struct{ set, name string }{{"ATQ", "Feldon's Cane"}, {"DRK", "Necropolis"}} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + entry.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "`+entry.name+`"`) {
				files[entry.set+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	if len(files) != 2 {
		t.Fatal("complete records missing")
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || registry == nil {
		t.Fatal(err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct{ name, role string }{{"ACardInGraveyards", "exile_payment_object"}, {"InAPlayersGraveyard", "card_filter"}, {"TheCardExiledThisWay", "exile_payment_reference"}} {
		found := false
		for _, s := range report.Signatures {
			if s.Name == want.name && s.Role == want.role {
				found = true
			}
		}
		if !found {
			t.Fatal("missing role", want)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Rules:[Unsupported])`)}
	if r, err := LoadFS(files, "*.ron"); err == nil || r != nil {
		t.Fatal("partial registry")
	}
}

func TestRONExilePaymentCaneCopyController(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "ATQ", "Feldon's Cane")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	for _, p := range []*mage.BasePlayer{a, b} {
		c := mage.NewLand("Graveyard")
		c.SetOwner(p.PlayerID())
		p.AddToGraveyard(c)
	}
	phase := g.PhaseOut(source.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || g.StackSize() != 0 {
		t.Fatal("phased source paid")
	}
	g.PhaseIn(phase)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindExiledCard(card.ID()) == nil {
		t.Fatal("source not exiled")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy")
	}
	g.ResolveStack()
	if len(a.Graveyard()) != 0 || len(b.Graveyard()) != 0 || len(a.Library()) != 1 || len(b.Library()) != 1 {
		t.Fatal("copied controller")
	}
}

func TestRONExilePaymentModesAndSourceRollback(t *testing.T) {
	for _, mode := range []int{0, 1} {
		g, a, _ := ronModalGame(mode)
		factory := permanentScopeFactory(t, "Artifact", `Activated(`+ronGraveyardExileCost+`,Modal_ChooseOne([ActionList([GainLife(ManaValueOfExiled(TheCardExiledThisWay))]),ActionList([DrawNumberCards(ManaValueOfExiled(TheCardExiledThisWay))])]))`)
		source := g.PutOnBattlefield(factory(), a.PlayerID())
		c := mage.NewCreature("Payment", "{2}", 1, 1)
		c.SetOwner(a.PlayerID())
		a.AddToGraveyard(c)
		fillRONLibrary(t, a.BasePlayer, 3)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if len(a.Hand()) != 0 || a.Life() != 20 {
			t.Fatal("mode resolved during payment")
		}
		g.ResolveStack()
		if mode == 0 && (a.Life() != 22 || len(a.Hand()) != 0) || mode == 1 && (a.Life() != 20 || len(a.Hand()) != 2) {
			t.Fatal("mode lost payment binding")
		}
	}
	g, a, _ := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `Activated(And([TapPermanent(ThisPermanent),Exile([Permanent(ThisPermanent)],[]),PayLife(Integer(25))]),ActionList([ShuffleGraveyardIntoLibrary]))`)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || g.FindPermanent(source.ID()) == nil || g.FindPermanent(source.ID()).Tapped || len(g.GetExile()) != 0 || a.Life() != 20 || g.StackSize() != 0 {
		t.Fatal("source exile total rollback")
	}
}
