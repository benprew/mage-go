package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDrawnCardReturnTimingAndCopies(t *testing.T) {
	for _, state := range []string{"normal", "phase", "copy", "clone", "counter", "two-libraries"} {
		t.Run(state, func(t *testing.T) {
			a := &ronDrawnReturnPlayer{BasePlayer: mage.NewBasePlayer("A"), decisions: []bool{true, false, false, true, false, false}}
			b := &ronDrawnReturnPlayer{BasePlayer: mage.NewBasePlayer("B"), decisions: []bool{true, false, false}}
			g := mage.NewGame(a, b)
			g.SetTurn(2)
			g.SetActivePlayerIndex(0)
			for range 8 {
				a.AddToLibrary(mage.NewLand("A card"))
				b.AddToLibrary(mage.NewLand("B card"))
			}
			source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Sylvan Library"), a.PlayerID())
			if state == "two-libraries" {
				g.PutOnBattlefield(repositoryCard(t, "LEG", "Sylvan Library"), a.PlayerID())
			}
			first := true
			g.SetOnPriority(func(game *mage.Game, _ int, _ bool) mage.PriorityAction {
				if first {
					first = false
					if len(a.Hand()) != 1 || a.choices != 0 {
						t.Fatal("ability resolved before normal draw")
					}
					switch state {
					case "phase":
						game.PhaseOut(source.ID())
					case "copy":
						if game.CopyStackObjectDirect(game.StackPeek(), b.PlayerID(), false) == nil {
							t.Fatal("copy failed")
						}
					case "clone":
						cloned := game.Clone()
						cloned.ResolveStack()
						if len(cloned.GetPlayer(a.PlayerID()).Hand()) != 1 || a.choices != 0 {
							t.Fatal("pending clone shared decision")
						}
					case "counter":
						game.CounterStackObject(game.StackPeek().ID)
					}
				}
				return mage.PriorityAction{Type: mage.PriorityPass}
			})
			g.RunStepWithPriority(core.Draw)
			wantChoices := 3
			if state == "counter" {
				wantChoices = 0
			}
			if state == "two-libraries" {
				wantChoices = 6
			}
			if first || len(a.Hand()) != 1 || a.Life() != 20 || a.choices != wantChoices {
				t.Fatal("timing, per-trigger decisions or put-back", len(a.Hand()), a.choices)
			}
			wantBChoices := 0
			if state == "copy" {
				wantBChoices = 3
			}
			if b.choices != wantBChoices || len(b.Hand()) != 0 {
				t.Fatal("copy player scope", b.choices, len(b.Hand()))
			}
		})
	}
}

func TestRONDrawnCardReturnReplacedDraws(t *testing.T) {
	for _, earlier := range []int{0, 1, 2} {
		a := &ronDrawnReturnPlayer{BasePlayer: mage.NewBasePlayer("A"), decisions: []bool{true, false, false}}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		a.AddToHand(mage.NewLand("Not drawn"))
		for range 5 {
			a.AddToLibrary(mage.NewLand("Draw"))
		}
		for range earlier {
			g.PlayerDrawCard(a)
		}
		replacement := permanentScopeFactory(t, "Enchantment", `ReplaceWouldDraw(APlayerWouldDrawExceptFirstDrawStepDraw(AnyPlayer),[PlayerAction(WouldDrawACard_ThatPlayer,GainLife(Integer(1)))])`)()
		g.PutOnBattlefield(replacement, b.PlayerID())
		g.PutOnBattlefield(repositoryCard(t, "LEG", "Sylvan Library"), a.PlayerID())
		g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: a.PlayerID()})
		g.PutTriggersOnStack()
		g.ResolveStack()
		if len(a.Hand()) != 1 || len(a.Library()) != 5 || a.Life() != 22 || a.choices != 1+earlier {
			t.Fatal("earlier draws or replaced draws", earlier, len(a.Hand()), len(a.Library()), a.Life(), a.choices)
		}
	}
}

func TestRONDrawnCardReturnLoadInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Sylvan Library"`) {
			record = line
		}
	}
	files := fstest.MapFS{"library.ron": {Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("full record load", err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	found := 0
	for _, signature := range report.Signatures {
		if signature.Role == "value" && signature.Name == "Integer" {
			found += signature.Occurrences
		}
	}
	if found != 3 {
		t.Fatal("draw and return operands lack value roles", found)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(strings.ReplaceAll(record, `"Sylvan Library"`, `"Invalid Library"`), `Integer(4)`, `Integer(-1)`))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("non-atomic rejection")
	}
}

func TestRONDrawnCardReturnPlayerScopes(t *testing.T) {
	for _, body := range []string{
		`PlayerAction(You,PayLifeOrPutCardsDrawnThisTurnOnTopOfLibrary(Integer(1),Integer(4)))`,
		`PlayerActions(You,[PlayerActions(ActionPlayer,[PayLifeOrPutCardsDrawnThisTurnOnTopOfLibrary(Integer(1),Integer(4))])])`,
		`PlayerAction(Ref_TargetPlayer,PayLifeOrPutCardsDrawnThisTurnOnTopOfLibrary(Integer(1),Integer(4)))`,
	} {
		g, a, b := ronGame()
		for _, p := range []*mage.BasePlayer{a, b} {
			p.AddToLibrary(mage.NewLand("Card"))
			g.PlayerDrawCard(p)
		}
		card := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([`+body+`])))`)()
		addRONHandCard(a, card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		wantA, wantB := 0, 1
		if strings.Contains(body, "Ref_TargetPlayer") {
			wantA, wantB = 1, 0
		}
		if len(a.Hand()) != wantA || len(b.Hand()) != wantB {
			t.Fatal("wrong player scope", body, len(a.Hand()), len(b.Hand()))
		}
	}
}
