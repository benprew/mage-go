package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCopiableValuesGameplay(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "FEM", "Icatian Javelineers")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	values, err := g.CaptureCopiableValues(source)
	if err != nil {
		t.Fatal(err)
	}
	first, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	first.SetOwner(b.PlayerID())
	copied := g.PutOnBattlefield(first, b.PlayerID())
	copied.RevokeBaseAttr(core.AttrSummonSick)
	if copied.Tapped || copied.Counters[core.Javelin] != 1 || source.Counters[core.Javelin] != 0 {
		t.Fatal("snapshot transferred spent cost state")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), copied.ID(), 1, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	again, err := g.CaptureCopiableValues(copied)
	if err != nil {
		t.Fatal(err)
	}
	second, err := again.Build()
	if err != nil {
		t.Fatal(err)
	}
	third := g.PutOnBattlefield(second, a.PlayerID())
	if third.Counters[core.Javelin] != 1 {
		t.Fatal("copy of a copy lost entry declaration")
	}
	clone := g.Clone()
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 19 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("snapshot shared target or clone state")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 19 || clone.GetPlayer(b.PlayerID()).Life() != 19 {
		t.Fatal("clone lost reconstructed activations")
	}
}

func TestRONCopiableValuesLayersAndFaceDown(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), a.PlayerID())
	spell := repositoryCard(t, "LEB", "Giant Growth")
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Green, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{source.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if source.CurrentPower(g) != 7 {
		t.Fatal("growth fixture did not apply")
	}
	values, err := g.CaptureCopiableValues(source)
	if err != nil {
		t.Fatal(err)
	}
	card, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	copied := g.PutOnBattlefield(card, b.PlayerID())
	if copied.CurrentPower(g) != 4 || !copied.HasAttr(core.Flying) {
		t.Fatal("ordinary layer became copiable")
	}
	g.MutablePermanent(source.ID()).FaceDown = true
	hidden, err := g.CaptureCopiableValues(source)
	if err != nil {
		t.Fatal(err)
	}
	g.MutablePermanent(source.ID()).FaceDown = false
	card, err = hidden.Build()
	if err != nil {
		t.Fatal(err)
	}
	faceUpCopy := g.PutOnBattlefield(card, b.PlayerID())
	if faceUpCopy.FaceDown || faceUpCopy.Name() != "" || faceUpCopy.CurrentPower(g) != 2 || faceUpCopy.CurrentToughness(g) != 2 || faceUpCopy.HasAttr(core.Flying) || len(faceUpCopy.Colors()) != 0 {
		t.Fatal("face-down values or status copied incorrectly")
	}
}

func TestRONCopiableValuesAuraTargets(t *testing.T) {
	g, a, b := ronGame()
	aura := repositoryCard(t, "LEB", "Holy Strength")
	aura.SetOwner(a.PlayerID())
	target := g.PutOnBattlefield(repositoryCard(t, "LEB", "Wall of Wood"), a.PlayerID())
	a.AddToHand(aura)
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), aura.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	values, err := g.CaptureCopiableValues(g.FindPermanent(aura.ID()))
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	if fresh.CastTargets()[0].IsChosen() || fresh.CastTargets()[0] == aura.CastTargets()[0] {
		t.Fatal("snapshot kept attachment target state")
	}
	other := g.PutOnBattlefield(repositoryCard(t, "LEB", "Hill Giant"), b.PlayerID())
	fresh.SetOwner(a.PlayerID())
	a.AddToHand(fresh)
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), fresh.ID(), []uuid.UUID{other.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.ApplyContinuousEffects()
	if target.CurrentPower(g) != 1 || target.CurrentToughness(g) != 5 || other.CurrentPower(g) != 4 || other.CurrentToughness(g) != 5 {
		t.Fatal("snapshot shared aura attachment or layers")
	}
}

func TestRONCopiableValuesProvenanceAndInvalid(t *testing.T) {
	g, a, _ := ronGame()
	for _, name := range []string{"Birds of Paradise", "Wall of Wood"} {
		card := repositoryCard(t, "LEB", name)
		for _, instance := range []mage.Card{card, card.Copy()} {
			p := g.PutOnBattlefield(instance, a.PlayerID())
			values, err := g.CaptureCopiableValues(p)
			if err != nil {
				t.Fatal(err)
			}
			fresh, err := values.Build()
			if err != nil || fresh.Name() != name || fresh.Power() != card.Power() || fresh.Toughness() != card.Toughness() {
				t.Fatal("compiler did not retain copiable provenance", err)
			}
		}
	}
	for _, rule := range []string{
		`Activated(PayMana([]),ActionList([PlayerAction(TheChosenPlayer,GainLife(Integer(1)))]))`,
		`AsPermanentEnters(ThisPermanent,[MayActions([EnterAsACopyOfAPermanent(IsCardtype(Creature),CopyEffects([]))])])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid snapshot source",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid or unsupported copy declaration accepted")
		}
	}
}

func TestRONCopiableValuesManaCostPresence(t *testing.T) {
	g, a, _ := ronGame()
	for _, cost := range []string{"", "ManaCost:[],", "ManaCost:[ManaCostGeneric(0)],"} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Cost fixture",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),`+cost+`)`), "cost.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		values, err := g.CaptureCopiableValues(g.PutOnBattlefield(factory(), a.PlayerID()))
		if err != nil || values.HasManaCost() != (cost != "") {
			t.Fatal("compiler lost absent versus zero mana cost", err)
		}
		fresh, err := values.Build()
		if err != nil || fresh.(*mage.BaseCard).HasManaCost() != (cost != "") {
			t.Fatal("snapshot reconstruction lost mana cost presence", err)
		}
	}
}

func TestRONCopiableValuesColorIndicator(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Indicator fixture",Typeline:(Supertypes:[Legendary,Snow],Cardtypes:[Creature,Artifact],Subtypes:[Bird]),ManaCost:[ManaCostG],ColorIndicator:[Blue],CardPT:(Power:1,Toughness:2),Rules:[Flying])`), "indicator.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	values, err := g.CaptureCopiableValues(p)
	if err != nil {
		t.Fatal(err)
	}
	delete(nodes[0].Fields, "ManaCost")
	fresh, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	copied := mage.NewPermanent(fresh, a.PlayerID())
	if !fresh.HasSuperType(core.SuperLegendary) || !fresh.HasSuperType(core.SuperSnow) || !fresh.HasType(core.TypeArtifact) || !fresh.HasType(core.TypeCreature) || !fresh.HasSubType("Bird") || !copied.HasAttr(core.Flying) || len(copied.Colors()) != 1 || copied.Colors()[0] != core.Blue || !values.HasManaCost() {
		t.Fatal("compiler lost printed characteristics or captured the mutable AST")
	}
}
