package ron

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONManaSpendingSunglasses(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, set, "Sunglasses of Urza")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p == nil || !card.HasType(core.TypeArtifact) || card.ManaCost().Generic != 3 {
				t.Fatal("metadata or entry")
			}
			a.ManaPool().Add(core.White, 1)
			b.ManaPool().Add(core.White, 1)
			cost := core.ManaCost{Red: 1}
			if !g.CanAfford(a.PlayerID(), cost, nil) || g.CanAfford(b.PlayerID(), cost, nil) {
				t.Fatal("controller scope")
			}
			clone := g.Clone()
			ids := g.PhaseOut(p.ID())
			if g.CanAfford(a.PlayerID(), cost, nil) {
				t.Fatal("phased rule active")
			}
			g.PhaseIn(ids)
			if !g.CanAfford(a.PlayerID(), cost, nil) {
				t.Fatal("phase-in lost rule")
			}
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			if g.CanAfford(a.PlayerID(), cost, nil) || !g.CanAfford(b.PlayerID(), cost, nil) {
				t.Fatal("control change")
			}
			g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.EndOfTurn))
			if g.CanAfford(b.PlayerID(), cost, nil) {
				t.Fatal("removed rule active")
			}
			g.RunStepWithPriority(core.Cleanup)
			b.ManaPool().Add(core.White, 1)
			if !g.CanAfford(b.PlayerID(), cost, nil) {
				t.Fatal("restored rule absent")
			}
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			if g.CanAfford(b.PlayerID(), cost, nil) {
				t.Fatal("departed rule active")
			}
			a.RemoveFromHand(card.ID())
			g.PutOnBattlefield(card, a.PlayerID())
			a.ManaPool().Add(core.White, 1)
			if !g.CanAfford(a.PlayerID(), cost, nil) || g.CanAfford(b.PlayerID(), cost, nil) {
				t.Fatal("returned rule")
			}
			if !clone.CanAfford(a.PlayerID(), cost, nil) || clone.CanAfford(b.PlayerID(), cost, nil) {
				t.Fatal("clone rule changed")
			}
		})
	}
}

func TestRONManaSpendingAutomatic(t *testing.T) {
	g, a, _ := ronGame()
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Sunglasses of Urza"), a.PlayerID())
	land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Plains"), a.PlayerID())
	spell := mage.NewSorcery("Red spell", "{R}", mage.WithAbility(mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1)))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if !g.CanAfford(a.PlayerID(), core.ManaCost{Red: 1}, nil) || g.FindPermanent(land.ID()).Tapped {
		t.Fatal("planning changed state")
	}
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(land.ID()).Tapped || a.ManaPool().LastDrainedColors[core.White] != 1 {
		t.Fatal("automatic payment")
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("spell resolution")
	}
	fail := mage.NewSorcery("Unpaid", "{R}{R}")
	fail.SetOwner(a.PlayerID())
	a.AddToHand(fail)
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), fail.ID(), nil, 0); err == nil || a.ManaPool().Count(core.White) != 1 || len(a.Hand()) != 1 {
		t.Fatal("failed payment changed resources")
	}
}

func TestRONManaSpendingInvalid(t *testing.T) {
	for _, rule := range []string{
		`PlayerEffect(You,[MaySpendColoredManaAsThoughItWereAnotherColor(White)])`,
		`PlayerEffect(You,[MaySpendColoredManaAsThoughItWereAnotherColor(White,Red,Blue)])`,
		`PlayerEffect(You,[MaySpendColoredManaAsThoughItWereAnotherColor(Colorless,Red)])`,
		`PlayerEffect(You,[MaySpendColoredManaAsThoughItWereAnotherColor(White,AnyColor)])`,
		`PlayerEffect(You,[MaySpendColoredManaAsThoughItWereAnotherColor(White(),Red)])`,
		`PlayerEffect(Opponent,[MaySpendColoredManaAsThoughItWereAnotherColor(White,Red)])`,
		`PlayerEffect(You(),[MaySpendColoredManaAsThoughItWereAnotherColor(White,Red)])`,
		`PlayerEffect(You,[MaySpendColoredManaAsThoughItWereAnotherColor(White,Red),Unknown])`,
		`SpellActions(ActionList([MaySpendColoredManaAsThoughItWereAnotherColor(White,Red)]))`,
		`AsPermanentEnters([MaySpendColoredManaAsThoughItWereAnotherColor(White,Red)])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONManaSpendingStrictLoad(t *testing.T) {
	valid := `Card(Name:"Permission",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[PlayerEffect(You,[MaySpendColoredManaAsThoughItWereAnotherColor(White,Red)])])`
	invalid := strings.ReplaceAll(valid, "White,Red", "White,Colorless")
	files := fstest.MapFS{"good.ron": &fstest.MapFile{Data: []byte(valid)}, "bad.ron": &fstest.MapFile{Data: []byte(invalid)}}
	registry := mage.NewRegistry()
	registry.Register("Retained", func() mage.Card { return mage.NewLand("Retained") })
	err := LoadInto(registry, files, "*.ron")
	var rejected *LoadError
	if !errors.As(err, &rejected) || len(rejected.Issues) != 1 || !registry.CardRegistered("Retained") || registry.CardRegistered("Permission") {
		t.Fatalf("atomic rejection: %v", err)
	}
	delete(files, "bad.ron")
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	for _, types := range []string{"Instant", "Sorcery"} {
		nodes, err := Parse(strings.NewReader(strings.ReplaceAll(valid, "[Artifact]", "["+types+"]")), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("spell static context accepted")
		}
	}
}

func TestRONManaSpendingCopies(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Sunglasses of Urza"), a.PlayerID())
	chooser := &ronEntryCopyPlayer{Player: b, accept: true}
	g.SetPlayerAt(1, chooser)
	copied := g.PutOnBattlefield(repositoryCard(t, "LEA", "Copy Artifact"), b.PlayerID())
	if copied == nil || copied.Name() != "Sunglasses of Urza" {
		t.Fatal("copy entry")
	}
	a.ManaPool().Add(core.White, 1)
	b.ManaPool().Add(core.White, 1)
	cost := core.ManaCost{Red: 1}
	if !g.CanAfford(a.PlayerID(), cost, nil) || !g.CanAfford(b.PlayerID(), cost, nil) {
		t.Fatal("copied permission")
	}
	g.BouncePermanentToHand(source)
	if g.CanAfford(a.PlayerID(), cost, nil) || !g.CanAfford(b.PlayerID(), cost, nil) {
		t.Fatal("copy followed donor")
	}
	g.AddContinuousEffect(mage.RemoveAllAbilities(copied.ID(), core.Indefinite))
	if g.CanAfford(b.PlayerID(), cost, nil) {
		t.Fatal("removed copy rule active")
	}
}
