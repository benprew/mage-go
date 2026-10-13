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

func TestRONPublicVisibilityCards(t *testing.T) {
	for _, name := range []string{"Field of Dreams", "Revelation"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", name)
			if err := mage.ValidateCardDefinition(name, card); err != nil {
				t.Fatal(err)
			}
			if !card.HasSuperType(core.SuperWorld) {
				t.Fatal("missing world supertype")
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("unpaid casting changed state")
			}
			color := core.Blue
			if name == "Revelation" {
				color = core.Green
			}
			a.ManaPool().Add(color, 1)
			check := func(game *mage.Game, want bool) {
				t.Helper()
				for _, player := range []mage.Player{a, b} {
					got := game.IsTopCardRevealed(player.PlayerID())
					if name == "Revelation" {
						got = game.IsHandRevealed(player.PlayerID())
					}
					if got != want {
						t.Fatal("wrong public visibility")
					}
				}
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			check(g, false)
			g.ResolveStack()
			check(g, true)
			clone := g.Clone()
			check(clone, true)
			g.DestroyPermanent(g.FindPermanent(card.ID()))
			g.ApplyContinuousEffects()
			check(g, false)
			check(clone, true)
		})
	}
}

func TestRONPublicVisibilityInvalid(t *testing.T) {
	for _, rule := range []string{
		`EachPlayerEffect(AnyPlayer,[PlaysWithHandRevealed()])`,
		`EachPlayerEffect(AnyPlayer,[PlaysWithTopOfLibraryRevealed(Integer(1))])`,
		`EachPlayerEffect(AnyPlayer(),[PlaysWithHandRevealed])`,
		`EachPlayerEffect(SinglePlayer(You),[PlaysWithHandRevealed])`,
		`EachPlayerEffect(AnyPlayer,[PlaysWithHandRevealed,Unknown])`,
		`PlayerEffect(You,[PlaysWithHandRevealed])`,
		`Activated(PayMana([]),ActionList([PlaysWithHandRevealed]))`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[%s])`, rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONPublicVisibilityLifecycle(t *testing.T) {
	for _, name := range []string{"Field of Dreams", "Revelation"} {
		for _, change := range []string{"phase", "control", "suppression", "return", "world"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "LEG", name)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				visible := func() bool {
					g.ApplyContinuousEffects()
					if name == "Revelation" {
						return g.IsHandRevealed(a.PlayerID()) && g.IsHandRevealed(b.PlayerID())
					}
					return g.IsTopCardRevealed(a.PlayerID()) && g.IsTopCardRevealed(b.PlayerID())
				}
				if !visible() {
					t.Fatal("missing initial visibility")
				}
				switch change {
				case "phase":
					g.PhaseOut(p.ID())
					if visible() {
						t.Fatal("phased source exposed cards")
					}
					g.PhaseIn([]uuid.UUID{p.ID()})
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "suppression":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					if visible() {
						t.Fatal("removed ability exposed cards")
					}
					g.RunStepWithPriority(core.Cleanup)
				case "return":
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					if visible() {
						t.Fatal("departed source exposed cards")
					}
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				case "world":
					otherName := "Revelation"
					if name == "Revelation" {
						otherName = "Field of Dreams"
					}
					g.PutOnBattlefield(repositoryCard(t, "LEG", otherName), b.PlayerID())
					g.CheckStateBasedActions()
					if g.FindPermanent(p.ID()) != nil || visible() {
						t.Fatal("World rule retained old visibility")
					}
					return
				}
				if !visible() {
					t.Fatal("visibility not restored or followed controller")
				}
			})
		}
	}
}

func TestRONPublicVisibilityContents(t *testing.T) {
	g, a, b := ronGame()
	a.SetLibrary([]mage.Card{mage.NewLand("First"), mage.NewLand("Second")})
	b.SetLibrary([]mage.Card{mage.NewLand("Opponent")})
	g.PutOnBattlefield(repositoryCard(t, "LEG", "Field of Dreams"), a.PlayerID())
	g.ApplyContinuousEffects()
	check := func(player mage.Player, name string) {
		t.Helper()
		if !g.IsTopCardRevealed(player.PlayerID()) {
			t.Fatal("top concealed")
		}
		cards := g.RevealTopN(player, 1)
		if len(cards) != 1 || cards[0].Name() != name {
			t.Fatal("stale top contents")
		}
	}
	check(a, "First")
	check(b, "Opponent")
	g.PlayerDrawCard(a)
	check(a, "Second")
	if len(a.Hand()) != 1 || a.Hand()[0].Name() != "First" {
		t.Fatal("draw changed by visibility")
	}
	g.PlayerDrawCard(a)
	if len(g.RevealTopN(a, 1)) != 0 {
		t.Fatal("empty library exposes card")
	}
	a.SetLibrary([]mage.Card{mage.NewLand("New")})
	check(a, "New")
	if g.IsHandRevealed(a.PlayerID()) {
		t.Fatal("library reveal also exposed hand")
	}
}

func TestRONPublicVisibilityInventoryAndAtomicLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(data), "\n")
	var records []string
	for _, n := range nodes {
		if n.Fields["Name"].String == "Revelation" || n.Fields["Name"].String == "Field of Dreams" {
			records = append(records, lines[n.Position.Line-1])
		}
	}
	if len(records) != 2 {
		t.Fatal("missing complete source records")
	}
	files := fstest.MapFS{"visibility.ron": {Data: []byte(strings.Join(records, "\n"))}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 2 || report.UniqueCards != 2 || len(report.Issues) != 0 {
		t.Fatal("wrong compilation counts")
	}
	for _, name := range []string{"PlaysWithHandRevealed", "PlaysWithTopOfLibraryRevealed"} {
		found := false
		for _, s := range report.Signatures {
			if s.Name == name && s.Role == "player_rule_operation" {
				found = true
			}
		}
		if !found {
			t.Fatal("missing semantic role", name)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(records[0], "AnyPlayer", "AnyPlayer()", 1))}
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("invalid collection installed")
	}
	if len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("strict rejection changed registry")
	}
}
