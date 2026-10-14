package ron

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLastDrawnCostRing(t *testing.T) {
	for _, set := range []string{"ARN", "3ED"} {
		for _, change := range []string{"normal", "depart", "control", "phase", "return", "copy", "counter", "library-cost"} {
			t.Run(set+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				ring := repositoryCard(t, set, "Jandor's Ring")
				ring.SetOwner(a.PlayerID())
				a.AddToHand(ring)
				a.ManaPool().Add(core.Colorless, 6)
				if err := g.CastSpellByID(a.PlayerID(), ring.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				source := g.FindPermanent(ring.ID())
				first := mage.NewLand("First")
				first.SetOwner(a.PlayerID())
				a.AddToLibrary(first)
				for range 3 {
					a.AddToLibrary(mage.NewLand("Next"))
					b.AddToLibrary(mage.NewLand("Foreign"))
				}
				g.PlayerDrawCard(a)
				if change == "library-cost" {
					g.PutOnBattlefield(mage.NewArtifact("Effect discard rule", "", mage.WithStaticAbility(mage.DiscardToLibraryReplacement())), a.PlayerID())
				}
				a.ManaPool().Add(core.Colorless, 2)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				grave := 1
				if len(a.Hand()) != 0 || len(a.Graveyard()) != grave || a.ManaPool().TotalMana() != 0 || !g.FindPermanent(source.ID()).Tapped || g.StackSize() != 1 {
					t.Fatal("cost or timing")
				}
				want, foreign := 1, 0
				switch change {
				case "depart":
					g.DestroyPermanent(g.FindPermanent(source.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "phase":
					g.PhaseOut(source.ID())
				case "return":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					g.PutOnBattlefield(ring, b.PlayerID())
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy")
					}
					foreign = 1
				case "counter":
					g.GetStack().RemoveByID(g.StackPeek().ID)
					want = 0
				}
				clone := g.Clone()
				g.ResolveStack()
				if len(a.Hand()) != want || len(b.Hand()) != foreign || a.ManaPool().TotalMana() != 0 {
					t.Fatal("resolution lost controller or repeated payment")
				}
				if len(clone.GetPlayer(a.PlayerID()).Hand()) != 0 {
					t.Fatal("clone shared resolution")
				}
				clone.ResolveStack()
				if len(clone.GetPlayer(a.PlayerID()).Hand()) != want || len(clone.GetPlayer(b.PlayerID()).Hand()) != foreign {
					t.Fatal("clone lost body")
				}
			})
		}
	}
}

func TestRONLastDrawnCostRollback(t *testing.T) {
	for _, state := range []string{"none", "added", "returned", "latest-gone", "unpaid", "tapped", "phased", "wrong-controller", "reset"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "ARN", "Jandor's Ring"), a.PlayerID())
			first := mage.NewLand("First")
			first.SetOwner(a.PlayerID())
			if state == "added" {
				a.AddToHand(first)
			} else if state != "none" {
				a.AddToLibrary(first)
				g.PlayerDrawCard(a)
			}
			switch state {
			case "returned":
				a.RemoveFromHand(first.ID())
				a.AddToHand(first)
			case "latest-gone":
				latest := mage.NewLand("Latest")
				a.AddToLibrary(latest)
				g.PlayerDrawCard(a)
				a.RemoveFromHand(latest.ID())
			case "tapped":
				g.TapPermanent(source)
			case "phased":
				g.PhaseOut(source.ID())
			case "reset":
				g.RunStepWithPriority(core.Cleanup)
			}
			mana := 2
			if state == "unpaid" {
				mana = 1
			}
			a.ManaPool().Add(core.Colorless, mana)
			payer := a.PlayerID()
			if state == "wrong-controller" {
				payer = b.PlayerID()
			}
			hand, tapped := len(a.Hand()), g.FindPermanentIncludingPhased(source.ID()).Tapped
			if err := g.ActivateAbilityByIndex(payer, source.ID(), 0, nil); err == nil {
				t.Fatal("invalid activation accepted")
			}
			if len(a.Hand()) != hand || len(a.Graveyard()) != 0 || a.ManaPool().TotalMana() != mana || g.StackSize() != 0 || g.FindPermanentIncludingPhased(source.ID()).Tapped != tapped {
				t.Fatal("failed payment changed resources")
			}
		})
	}
}

func TestRONLastDrawnCostInvalid(t *testing.T) {
	for _, cost := range []string{"DiscardCard", "DiscardCard()", "DiscardCard(TheLastCardDrawnThisTurn,ThisPermanent)", "DiscardCard(TheLastCardDrawnThisTurn())", "DiscardCard(ThisGraveyardCard)", "DiscardCard(Ref_TargetPermanent)"} {
		text := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(` + cost + `,ActionList([DrawACard]))])`
		nodes, err := Parse(strings.NewReader(text), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", cost)
		}
	}
	for _, rule := range []string{
		`SpellActions(ActionList([DiscardCard(TheLastCardDrawnThisTurn)]))`,
		`SpellActions(ActionList([MayCost(DiscardCard(TheLastCardDrawnThisTurn)),If(CostWasPaid,[DrawACard])]))`,
		`AsPermanentEnters(ThisPermanent,[DiscardCard(TheLastCardDrawnThisTurn)])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid context",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONLastDrawnCostAtomicLoad(t *testing.T) {
	files := fstest.MapFS{}
	for _, set := range []string{"ARN", "3ED"} {
		data, err := fs.ReadFile(repositoryFixtures.files, set+".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "Jandor's Ring"`) {
				files[set+".ron"] = &fstest.MapFile{Data: []byte(line)}
				break
			}
		}
	}
	if len(files) != 2 {
		t.Fatal("missing reprints")
	}
	registry := mage.NewRegistry()
	registry.Register("Retained", func() mage.Card { return mage.NewLand("Retained") })
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(strings.ReplaceAll(string(files["ARN.ron"].Data), `"Jandor's Ring"`, `"Bad"`), "TheLastCardDrawnThisTurn", "TheLastCardDrawnThisTurn()"))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Retained") || registry.CardRegistered("Jandor's Ring") {
		t.Fatal("partial collection published", err)
	}
	delete(files, "bad.ron")
	if err := LoadInto(registry, files, "*.ron"); err != nil || !registry.CardRegistered("Jandor's Ring") {
		t.Fatal("reprints failed to load", err)
	}
}

func TestRONLastDrawnCostInstances(t *testing.T) {
	factory, err := Compile(repositoryRecord(t, "ARN", "Jandor's Ring", "ring.ron"))
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	one := g.PutOnBattlefield(factory(), a.PlayerID())
	two := g.PutOnBattlefield(factory(), b.PlayerID())
	first, second := mage.NewLand("A draw"), mage.NewLand("B draw")
	a.AddToLibrary(first)
	b.AddToLibrary(second)
	g.PlayerDrawCard(a)
	g.PlayerDrawCard(b)
	a.AddToLibrary(mage.NewLand("A next"))
	b.AddToLibrary(mage.NewLand("B next"))
	a.ManaPool().Add(core.Colorless, 2)
	b.ManaPool().Add(core.Colorless, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), one.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), two.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.Graveyard()[0].ID() != first.ID() || b.Graveyard()[0].ID() != second.ID() {
		t.Fatal("instances shared payment")
	}
	g.ResolveStack()
	if len(a.Hand()) != 1 || len(b.Hand()) != 1 || a.LastDrawnCardID() != a.Hand()[0].ID() || b.LastDrawnCardID() != b.Hand()[0].ID() {
		t.Fatal("instances shared draw history")
	}
	g.UntapPermanent(g.FindPermanent(one.ID()))
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), one.ID(), 0, nil); err != nil {
		t.Fatal("new draw cannot pay", err)
	}
	if len(a.Graveyard()) != 2 {
		t.Fatal("second activation used old visit")
	}
}
