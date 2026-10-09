package ron

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func copyColorFactory(t *testing.T, cost, indicator, modification string) mage.CardFactory {
	t.Helper()
	src := `Card(Name:"Color copy",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),ManaCost:` + cost + indicator + `,CardPT:(Power:0,Toughness:1),Rules:[AsPermanentEnters(ThisPermanent,[MayActions([EnterAsACopyOfAPermanent(IsCardtype(Creature),CopyEffects([` + modification + `]))])])])`
	nodes, err := Parse(strings.NewReader(src), "colors.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONCopyColorGameplay(t *testing.T) {
	for _, tc := range []struct {
		cost, indicator string
		colors          []core.Color
	}{
		{`[ManaCostU]`, "", []core.Color{core.Blue}},
		{`[ManaCostU,ManaCostW]`, "", []core.Color{core.White, core.Blue}},
		{`[]`, "", nil},
		{`[ManaCostU]`, `,ColorIndicator:[White,Red]`, []core.Color{core.White, core.Red}},
	} {
		g, a, b := ronGame()
		chooser := &ronEntryCopyPlayer{Player: a, accept: true}
		g.SetPlayerAt(0, chooser)
		source := g.PutOnBattlefield(repositoryCard(t, "LEB", "Birds of Paradise"), b.PlayerID())
		source.GrantBaseAttr(core.Shroud)
		factory := copyColorFactory(t, tc.cost, tc.indicator, `KeepColor,AddCardtypes([Enchantment])`)
		card := factory()
		p := g.PutOnBattlefield(card, a.PlayerID())
		if p == nil || p.Name() != "Birds of Paradise" || !p.HasAttr(core.Flying) || p.Card.ManaCost().Green != 1 || !p.HasType(core.TypeEnchantment) || !slices.Equal(p.Colors(), tc.colors) || chooser.offers != 1 {
			t.Fatal("entry values", p)
		}
		clone := g.Clone()
		clone.ChangeObjectColor(p.ID(), core.Black)
		if !slices.Equal(g.FindPermanent(p.ID()).Colors(), tc.colors) || !slices.Equal(clone.FindPermanent(p.ID()).Colors(), []core.Color{core.Black}) {
			t.Fatal("clone color isolation")
		}
		g.PhaseIn(g.PhaseOut(p.ID()))
		if chooser.offers != 1 || !slices.Equal(g.FindPermanent(p.ID()).Colors(), tc.colors) {
			t.Fatal("phase repeated copying")
		}
		p = g.FindPermanent(p.ID())
		g.ChangeObjectColor(p.ID(), core.Red)
		chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return g.FindPermanent(p.ID()) }
		next := g.PutOnBattlefield(repositoryCard(t, "LEB", "Clone"), a.PlayerID())
		if next == nil || !slices.Equal(next.Colors(), tc.colors) || !next.HasType(core.TypeEnchantment) {
			t.Fatal("subsequent copy lost retained colors")
		}
		g.BouncePermanentToHand(g.FindPermanent(p.ID()))
		if a.Hand()[0].Name() != "Color copy" || !slices.Equal(g.LKI(card.ID()).Snapshot.Colors(), []core.Color{core.Red}) {
			t.Fatal("printed departure or LKI")
		}
		returned := a.Hand()[0]
		a.RemoveFromHand(returned.ID())
		chooser.accept = false
		again := g.PutOnBattlefield(returned, a.PlayerID())
		if again == nil || again.Name() != "Color copy" || !slices.Equal(again.Colors(), tc.colors) || !slices.Equal(next.Colors(), tc.colors) {
			t.Fatal("return or donor independence")
		}
	}
}

func TestRONCopyColorInvalid(t *testing.T) {
	for _, form := range []string{`KeepColor()`, `KeepColor(Blue)`, `KeepColor([Blue])`, `CopyEffects([KeepColor])`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[AsPermanentEnters(ThisPermanent,[MayActions([EnterAsACopyOfAPermanent(IsCardtype(Creature),CopyEffects([` + form + `]))])])])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid keep color accepted", form)
		}
	}
}

func TestRONCopyColorDependencies(t *testing.T) {
	count := 0
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), set)
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range nodes {
			if n.Fields["Name"].String != "Vesuvan Doppelganger" {
				continue
			}
			count++
			if _, err := Compile(n); err == nil || !strings.Contains(err.Error(), "AddAbility") {
				t.Fatal("dependent card accepted or wrong blocker", err)
			}
		}
	}
	if count != 6 {
		t.Fatal("missing reprints", count)
	}
}

func TestRONCopyColorSpellTiming(t *testing.T) {
	g, a, b := ronGame()
	chooser := &ronEntryCopyPlayer{Player: a, accept: true}
	g.SetPlayerAt(0, chooser)
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Birds of Paradise"), b.PlayerID())
	factory := copyColorFactory(t, `[ManaCostU]`, "", `KeepColor,KeepColor`)
	card := factory()
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || chooser.offers != 0 {
		t.Fatal("failed cost changed copy choices")
	}
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if chooser.offers != 0 {
		t.Fatal("copy choice before resolution")
	}
	g.ChangeObjectColor(card.ID(), core.Red)
	clone := g.Clone()
	decline := &ronEntryCopyPlayer{Player: clone.GetPlayer(a.PlayerID()), accept: false}
	clone.SetPlayerAt(0, decline)
	g.ResolveStack()
	clone.ResolveStack()
	p := g.FindPermanent(card.ID())
	if p == nil || p.Name() != "Birds of Paradise" || !slices.Equal(p.Colors(), []core.Color{core.Red}) {
		t.Fatal("entering spell colors were not retained")
	}
	values, err := g.CaptureCopiableValues(p)
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := values.Build()
	if err != nil || !slices.Equal(g.EffectiveColors(p.ID()), []core.Color{core.Red}) {
		t.Fatal("saved spell color", err)
	}
	copied := g.PutOnBattlefield(rebuilt, a.PlayerID())
	if !slices.Equal(copied.Colors(), []core.Color{core.Blue}) {
		t.Fatal("ordinary spell color entered copiable values")
	}
	other := clone.FindPermanent(card.ID())
	if other == nil || other.Name() != "Color copy" || decline.offers != 1 {
		t.Fatal("cloned pending spell choice")
	}
	g.BouncePermanentToHand(p)
	if !slices.Equal(a.Hand()[0].ManaCost().Colors(), []core.Color{core.Blue}) {
		t.Fatal("printed color not restored on departure")
	}
}
