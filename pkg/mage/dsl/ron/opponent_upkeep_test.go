package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const opponentUpkeepRule = `TriggerA(AtTheBeginningOfAPlayersUpkeep(Opponent),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(2))),GainLife(Integer(1))]))`

func TestRONOpponentUpkeepGameplay(t *testing.T) {
	registry, err := LoadFS(fstest.MapFS{"upkeep.ron": {Data: []byte(colorEntryFixture(opponentUpkeepRule))}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	card, err := registry.CreateCard("Color entry")
	if err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	queueRONUpkeep(g, a)
	if g.StackSize() != 0 {
		t.Fatal("controller upkeep triggered")
	}
	queueRONUpkeep(g, b)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	queueRONUpkeep(g, b)
	queueRONUpkeep(g, a)
	if g.StackSize() != 2 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("upkeep detection or timing")
	}
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, b.PlayerID())
	clone := g.Clone()
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 23 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("event captures or clone isolation")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 23 || clone.GetPlayer(b.PlayerID()).Life() != 23 {
		t.Fatal("clone lost pending event captures")
	}
}

func TestRONOpponentUpkeepPsychicAllergy(t *testing.T) {
	for _, scenario := range []string{"normal", "control", "depart", "return", "phase", "cycle", "removed", "prevent", "copy", "late-member"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			chooser := &ronColorEntryPlayer{Player: a, answer: core.Red}
			g.SetPlayerAt(0, chooser)
			card := repositoryCard(t, "DRK", "Psychic Allergy")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 5)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if chooser.calls != 0 {
				t.Fatal("entry choice at announcement")
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			if source == nil || source.ChosenColor != core.Red || chooser.calls != 1 {
				t.Fatal("entry choice missing")
			}
			for _, controller := range []mage.Player{a, b, b} {
				g.PutOnBattlefield(mage.NewCreature("Red", "{R}", 2, 2), controller.PlayerID())
			}
			g.PutOnBattlefield(mage.NewCreature("Blue", "{U}", 2, 2), b.PlayerID())
			token := mage.NewCreature("Token", "{R}", 2, 2)
			g.PutOnBattlefield(token, b.PlayerID()).IsToken = true
			if scenario == "phase" {
				g.PhaseOut(source.ID())
			}
			if scenario == "cycle" {
				g.PhaseIn(g.PhaseOut(source.ID()))
			}
			if scenario == "removed" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
			}
			queueRONUpkeep(g, b)
			if scenario == "phase" || scenario == "removed" {
				if g.StackSize() != 0 {
					t.Fatal("inactive source triggered")
				}
				return
			}
			if g.StackSize() != 1 || b.Life() != 20 {
				t.Fatal("damage ran before resolution")
			}
			want := 18
			switch scenario {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "depart", "return":
				g.BouncePermanentToHand(source)
				if scenario == "return" {
					chooser.answer = core.Blue
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "prevent":
				g.AddPreventionShield(b.PlayerID(), 1)
				want = 19
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				want = 16
			case "late-member":
				g.PutOnBattlefield(mage.NewEnchantment("Late red", "{R}"), b.PlayerID())
				want = 17
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != want || clone.GetPlayer(b.PlayerID()).Life() != want {
				t.Fatalf("life %d/%d, want 20/%d", a.Life(), b.Life(), want)
			}
		})
	}
}

func TestRONOpponentUpkeepInvalid(t *testing.T) {
	for _, selector := range []string{`Opponent()`, `SinglePlayer(Opponent)`, `AnyPlayer(Opponent)`, `Opponent,AnyPlayer`, `Trigger_ThatPlayer`, `SinglePlayer(Trigger_ThatPlayer)`} {
		src := colorEntryFixture(strings.Replace(opponentUpkeepRule, "Upkeep(Opponent)", "Upkeep("+selector+")", 1))
		if _, err := LoadFS(fstest.MapFS{"invalid.ron": {Data: []byte(src)}}, "*.ron"); err == nil {
			t.Errorf("accepted selector %s", selector)
		}
	}
	for _, rule := range []string{
		strings.Replace(opponentUpkeepRule, "GainLife(Integer(1))", "TapPermanent(Trigger_ThatPermanent)", 1),
		opponentUpkeepRule + `,TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([TapPermanent(Trigger_ThatPermanent)]))`,
		`SpellActions(ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))`,
		strings.Replace(opponentUpkeepRule, "Upkeep", "EndStep", 1),
		strings.Replace(opponentUpkeepRule, "Upkeep", "DrawStep", 1),
		`TriggerI(AtTheBeginningOfAPlayersUpkeep(Opponent),PlayerPassesFilter(Trigger_ThatPlayer,ControlsA(IsCardtype(Land))),ActionList([GainLife(Integer(1))]))`,
	} {
		if _, err := LoadFS(fstest.MapFS{"invalid.ron": {Data: []byte(colorEntryFixture(rule))}}, "*.ron"); err == nil {
			t.Errorf("accepted unbound reference: %s", rule)
		}
	}
}

func TestRONOpponentUpkeepCollectionAndTextBoundary(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Psychic Allergy"`) {
			record = line
			break
		}
	}
	if record == "" {
		t.Fatal("missing complete record")
	}
	files := fstest.MapFS{"allergy.ron": {Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.CreateCard("Psychic Allergy"); err != nil {
		t.Fatal(err)
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(colorEntryFixture(strings.Replace(opponentUpkeepRule, "Opponent", "Opponent()", 1)))}
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("mixed invalid collection accepted")
	}
	if _, err := registry.CreateCard("Psychic Allergy"); err != nil {
		t.Fatal("failed collection changed the registry")
	}
	factory := permanentScopeFactory(t, "Artifact", opponentUpkeepRule)
	g, a, b := ronGame()
	first := g.PutOnBattlefield(factory(), a.PlayerID())
	g.PutOnBattlefield(factory(), b.PlayerID())
	queueRONUpkeep(g, b)
	queueRONUpkeep(g, a)
	if g.StackSize() != 2 {
		t.Fatal("factory instances share event context")
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 23 {
		t.Fatal("factory instances share captures")
	}
	change, err := mage.ReplaceColorWord(core.Red, core.Blue)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedUpkeepObjectTextEffect(first, core.EndOfTurn, change); err == nil {
		t.Fatal("opponent event opted into whole-object text revision")
	}
	for _, ability := range first.RuntimeAbilities {
		if _, ok := ability.(*mage.GenericTriggered); !ok {
			continue
		}
		if err := g.RevisePrintedUpkeepTrigger(first, ability.AbilityID(), change); err == nil {
			t.Fatal("opponent event opted into upkeep text revision")
		}
	}
}

func TestRONOpponentUpkeepPsychicAllergyPayment(t *testing.T) {
	for _, scenario := range []string{"pay", "decline", "one", "wrong-type", "phased-island", "control", "depart", "return", "pending-phase", "regenerate", "indestructible"} {
		t.Run(scenario, func(t *testing.T) {
			a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: scenario != "decline" && scenario != "return" && scenario != "regenerate" && scenario != "indestructible"}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			card := repositoryCard(t, "DRK", "Psychic Allergy")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			lands := make([]*mage.Permanent, 0, 2)
			for i := range 2 {
				name := "Island"
				if scenario == "wrong-type" && i == 1 {
					name = "Forest"
				}
				land := repositoryCard(t, "LEB", name)
				land.SetOwner(b.PlayerID())
				lands = append(lands, g.PutOnBattlefield(land, a.PlayerID()))
			}
			if scenario == "one" {
				g.BouncePermanentToHand(lands[1])
			}
			if scenario == "phased-island" {
				g.PhaseOut(lands[1].ID())
			}
			queueRONUpkeep(g, a)
			if g.StackSize() != 1 || a.offers != 0 || len(b.Graveyard()) != 0 {
				t.Fatal("payment before resolution")
			}
			switch scenario {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "depart", "return":
				g.BouncePermanentToHand(source)
				if scenario == "return" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "pending-phase":
				g.PhaseOut(source.ID())
			case "regenerate":
				g.AddRegenerationShield(source.ID())
			case "indestructible":
				g.FindPermanent(source.ID()).GrantBaseAttr(core.Indestructible)
			}
			g.ResolveStack()
			paid := scenario == "pay" || scenario == "control" || scenario == "depart" || scenario == "pending-phase"
			wantSacrificed := 0
			if paid {
				wantSacrificed = 2
			}
			if len(b.Graveyard()) != wantSacrificed {
				t.Fatal("payment used wrong controller, type or count, or spent partial resources")
			}
			present := paid && scenario != "depart" && scenario != "pending-phase" || scenario == "return" || scenario == "regenerate" || scenario == "indestructible"
			if (g.FindPermanent(source.ID()) != nil) != present {
				t.Fatal("unpaid destruction or exact source identity")
			}
			if scenario == "regenerate" && !g.FindPermanent(source.ID()).Tapped {
				t.Fatal("unpaid destruction did not use regeneration")
			}
			if !paid && scenario != "one" && scenario != "phased-island" && g.FindPermanent(lands[0].ID()) == nil {
				t.Fatal("failed payment spent a land")
			}
		})
	}
}
