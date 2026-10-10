package ron

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextUpkeepSpellInventory(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]int{"Ivory Tower": 2, "Karma": 6, "Copper Tablet": 5, "Erhnam Djinn": 1, "Juzám Djinn": 1, "Gabriel Angelfire": 1, "Halfdane": 1, "Storm World": 1}
	counts := map[string]int{}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		records, err := Parse(strings.NewReader(string(data)), file)
		if err != nil {
			t.Fatal(err)
		}
		for _, record := range records {
			name := record.Fields["Name"].String
			if _, ok := expected[name]; !ok {
				continue
			}
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			g, a, _ := ronGame()
			obj := castRONSpell(t, g, a, factory())
			word, _ := mage.ReplaceColorWord(core.Red, core.Green)
			if err := g.RevisePrintedSpellText(obj.ID, word); err != nil {
				t.Fatal(name, err)
			}
			g.ResolveStack()
			if g.FindPermanent(obj.SourceID) == nil {
				t.Fatal("continued spell did not enter", name)
			}
			counts[name]++
		}
	}
	if !maps.Equal(counts, expected) {
		t.Fatalf("unmapped inventory: %v", counts)
	}
}

func castUpkeepSpellWords(t *testing.T, g *mage.Game, a *mage.BasePlayer, subject *mage.StackObject, set, name, from, to string) {
	t.Helper()
	chooser := &upkeepSpellWordTester{intrinsicWordTester: intrinsicWordTester{Player: a, words: []string{from, to}}}
	g.SetPlayerAt(0, chooser)
	castRONSpell(t, g, a, repositoryCard(t, set, name), subject.ID)
	if chooser.calls != 0 {
		t.Fatal("word choices ran at announcement")
	}
	g.ResolveTopOfStack()
	if chooser.calls != 2 {
		t.Fatal("word choices did not resolve")
	}
	g.SetPlayerAt(0, a)
}

type upkeepSpellWordTester struct{ intrinsicWordTester }

func (p *upkeepSpellWordTester) ChooseMode(options []string, prompt string) int {
	if len(options) == 2 {
		return 0
	}
	return p.intrinsicWordTester.ChooseMode(options, prompt)
}

func TestRONTextUpkeepSpellKarma(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, set, "Karma")
			own := castRONSpell(t, g, a, card)
			castUpkeepSpellWords(t, g, a, own, set, "Magical Hack", "Swamp", "Forest")
			if g.FindPermanent(card.ID()) != nil {
				t.Fatal("consumer resolved subject early")
			}
			g.ResolveStack()
			g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), b.PlayerID())
			g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
			g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
			queueRONUpkeep(g, b)
			if b.Life() != 20 || g.StackSize() != 1 {
				t.Fatal("upkeep acted before resolution")
			}
			g.ResolveStack()
			if b.Life() != 18 {
				t.Fatal("continued land word lost")
			}
			queueRONUpkeep(g, a)
			g.ResolveStack()
			if a.Life() != 20 {
				t.Fatal("event player binding lost")
			}
			g.BouncePermanentToHand(g.FindPermanent(card.ID()))
			if a.Hand()[0] != card || g.LKI(card.ID()) == nil {
				t.Fatal("printed departure lost")
			}
			g.SetStep(core.PrecombatMain)
			castRONSpell(t, g, a, card)
			g.ResolveStack()
			queueRONUpkeep(g, b)
			g.ResolveStack()
			if b.Life() != 17 {
				t.Fatal("recast did not use printed Swamp")
			}
		})
	}
}

func TestRONTextUpkeepSpellRecords(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"ATQ", "Ivory Tower"}, {"LEA", "Copper Tablet"}, {"ARN", "Juzám Djinn"}, {"LEG", "Storm World"}, {"LEG", "Gabriel Angelfire"}, {"LEG", "Halfdane"}, {"ARN", "Erhnam Djinn"}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			obj := castRONSpell(t, g, a, card)
			castUpkeepSpellWords(t, g, a, obj, "LEA", "Sleight of Mind", "Red", "Green")
			if tc.name == "Erhnam Djinn" {
				castUpkeepSpellWords(t, g, a, obj, "LEA", "Magical Hack", "Forest", "Island")
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			if source == nil {
				t.Fatal("continued entry failed")
			}
			victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 6), b.PlayerID())
			fillRONHand(a, 7)
			fillRONHand(b, 1)
			queueRONUpkeep(g, b)
			allUpkeeps := tc.name == "Copper Tablet" || tc.name == "Storm World"
			if (g.StackSize() == 1) != allUpkeeps {
				t.Fatal("incorrect opponent upkeep")
			}
			g.ResolveStack()
			wantB := 20
			if tc.name == "Copper Tablet" {
				wantB = 19
			}
			if tc.name == "Storm World" {
				wantB = 17
			}
			if b.Life() != wantB {
				t.Fatal("incorrect opponent damage")
			}
			queueRONUpkeep(g, a)
			if g.StackSize() != 1 || a.Life() != 20 {
				t.Fatal("controller upkeep timing lost")
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			wantA := 20
			if tc.name == "Ivory Tower" {
				wantA = 23
			}
			if tc.name == "Copper Tablet" || tc.name == "Juzám Djinn" {
				wantA = 19
			}
			if a.Life() != wantA || clone.GetPlayer(a.PlayerID()).Life() != wantA {
				t.Fatal("continued body or clone failed")
			}
			for _, game := range []*mage.Game{g, clone} {
				switch tc.name {
				case "Erhnam Djinn":
					v := game.FindPermanent(victim.ID())
					if !v.HasKeyword(core.Islandwalk) || v.HasKeyword(core.Forestwalk) {
						t.Fatal("continued landwalk word lost")
					}
				case "Halfdane":
					p := game.FindPermanent(source.ID())
					if p.CurrentPower(game) != 2 || p.CurrentToughness(game) != 6 {
						t.Fatal("continued target binding lost")
					}
				case "Gabriel Angelfire":
					if !game.FindPermanent(source.ID()).HasKeyword(core.Flying) {
						t.Fatal("continued chosen ability lost")
					}
				}
			}
			if tc.name == "Erhnam Djinn" || tc.name == "Halfdane" || tc.name == "Gabriel Angelfire" {
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.SetActivePlayerIndex(0)
				g.RunStepWithPriority(core.Upkeep)
				if tc.name == "Erhnam Djinn" && g.FindPermanent(victim.ID()).HasKeyword(core.Islandwalk) {
					t.Fatal("continued grant did not expire")
				}
			}
		})
	}
}

func TestRONTextUpkeepSpellLifecycle(t *testing.T) {
	testRONTextUpkeepSpellLifecycle(t, repositoryCard(t, "LEA", "Karma"))
}

func testRONTextUpkeepSpellLifecycle(t *testing.T, card mage.Card, keywords ...core.Attr) {
	g, a, b := ronGame()
	obj := castRONSpell(t, g, a, card)
	first, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.ForestLandType)
	second, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	third, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.MountainLandType)
	if err := g.RevisePrintedSpellText(obj.ID, first); err != nil {
		t.Fatal(err)
	}
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("spell copy failed")
	}
	if err := g.RevisePrintedSpellText(copied.ID, second); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if err := clone.RevisePrintedSpellText(copied.ID, third); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		for _, id := range []uuid.UUID{card.ID(), copied.SourceID} {
			p := game.FindPermanent(id)
			if p == nil {
				t.Fatal("continued permanent did not enter")
			}
			for _, keyword := range keywords {
				if !p.HasKeyword(keyword) {
					t.Fatal("continued entry lost fixed keyword")
				}
			}
		}
	}
	token := g.FindPermanent(copied.SourceID)
	if token == nil || !token.Card.IsToken() || token.ControllerID() != b.PlayerID() {
		t.Fatal("copy entry identity lost")
	}
	for _, game := range []*mage.Game{g, clone} {
		game.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
		game.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), a.PlayerID())
		game.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), a.PlayerID())
		game.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
	}
	if a.Life() != 17 || clone.GetPlayer(a.PlayerID()).Life() != 19 {
		t.Fatal("copy or clone shared word changes")
	}
	values, err := g.CaptureCopiableValues(token)
	if err != nil {
		t.Fatal(err)
	}
	printed, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	sibling := g.PutOnBattlefield(printed, a.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), a.PlayerID())
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if a.Life() != 13 {
		t.Fatal("copiable values retained ordinary changes")
	}
	g.RemoveFromBattlefield(g.FindPermanent(sibling.ID()))
	g.RemoveFromBattlefield(g.FindPermanent(token.ID()))
	p := g.FindPermanent(card.ID())
	id := p.RuntimeAbilities[0].AbilityID()
	queueRONUpkeep(g, a)
	if err := g.AddPrintedUpkeepObjectTextEffect(p, core.EndOfTurn, second); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
	}
	g.ResolveStack()
	if a.Life() != 12 {
		t.Fatal("pending body lost continued baseline")
	}
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if a.Life() != 10 {
		t.Fatal("later text layer did not use continued baseline")
	}
	g.AddContinuousEffect(mage.RemoveAllAbilities(card.ID(), core.EndOfTurn))
	for _, keyword := range keywords {
		if g.FindPermanent(card.ID()).HasKeyword(keyword) {
			t.Fatal("text layer restored a removed keyword")
		}
	}
	queueRONUpkeep(g, a)
	if g.StackSize() != 0 {
		t.Fatal("removed rule detected event")
	}
	phased := g.PhaseOut(card.ID())
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	p = g.FindPermanent(card.ID())
	for _, keyword := range keywords {
		if !p.HasKeyword(keyword) {
			t.Fatal("restoration lost fixed keyword")
		}
	}
	if p.RuntimeAbilities[0].AbilityID() != id {
		t.Fatal("restoration lost declaration identity")
	}
	queueRONUpkeep(g, a)
	g.BouncePermanentToHand(p)
	for _, keyword := range keywords {
		if !g.LKI(card.ID()).Snapshot.HasKeyword(keyword) {
			t.Fatal("departure lost keyword LKI")
		}
	}
	g.ResolveStack()
	if a.Life() != 9 || a.Hand()[0] != card || g.LKI(card.ID()) == nil {
		t.Fatal("pending departure lost continued text or printed card")
	}
	if err := g.AddPrintedUpkeepObjectTextEffect(p, core.EndOfTurn, second); err == nil {
		t.Fatal("departed visit revised")
	}
}

func TestRONTextUpkeepSpellBoundaries(t *testing.T) {
	g, a, _ := ronGame()
	word, _ := mage.ReplaceColorWord(core.Red, core.Green)
	for _, tc := range []struct{ set, name string }{{"LEA", "Black Vise"}, {"ATQ", "The Rack"}} {
		obj := castRONSpell(t, g, a, repositoryCard(t, tc.set, tc.name))
		if err := g.RevisePrintedSpellText(obj.ID, word); err != nil {
			t.Fatal("chosen-player spell context rejected", tc.name, err)
		}
		g.CounterSpellOnStack(obj.SourceID)
	}
	for _, rule := range []string{
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer()),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(Opponent)),ActionList([PlayerAction(Trigger_ThatSpell,GainLife(Integer(1)))]))`,
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),ActionList([If(CostWasPaid,[GainLife(Integer(1))])]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", rule)
		}
	}
}

func TestRONTextUpkeepSpellPendingTargets(t *testing.T) {
	testRONTextUpkeepSpellPendingTargets(t, "")
}

func testRONTextUpkeepSpellPendingTargets(t *testing.T, prefix string) {
	factory := permanentScopeFactory(t, "Artifact", prefix+`TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),Targeted([TargetPermanent(IsColor(Red))],ActionList([TapPermanent(Ref_TargetPermanent),PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))])))`)
	g, a, b := ronModalGame(0)
	g.SetActivePlayerIndex(0)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	obj := castRONSpell(t, g, a.BasePlayer, factory())
	word, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.ForestLandType)
	if err := g.RevisePrintedSpellText(obj.ID, word); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(obj.SourceID)
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	blue := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.AddPrintedUpkeepObjectTextEffect(p, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	a.targets = []uuid.UUID{red.ID()}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("pending target text changed")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	a.targets = []uuid.UUID{green.ID()}
	queueRONUpkeep(g, b)
	if g.StackSize() != 3 {
		t.Fatal("revised target not chosen")
	}
	clone := g.Clone()
	change, _ = mage.ReplaceColorWord(core.Green, core.Blue)
	if err := g.AddPrintedUpkeepObjectTextEffect(p, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	a.targets = []uuid.UUID{blue.ID()}
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if !g.FindPermanent(red.ID()).Tapped || !g.FindPermanent(green.ID()).Tapped || !g.FindPermanent(blue.ID()).Tapped || a.Life() != 23 || b.Life() != 21 {
		t.Fatalf("queued/copy facts: A=%d B=%d", a.Life(), b.Life())
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 21 || clone.FindPermanent(blue.ID()).Tapped {
		t.Fatal("clone shared later revision or actions")
	}
	g.UntapPermanent(blue)
	a.targets = []uuid.UUID{blue.ID()}
	queueRONUpkeep(g, b)
	g.BouncePermanentToHand(blue)
	b.RemoveFromHand(blue.Card.ID())
	returned := g.PutOnBattlefield(blue.Card, b.PlayerID())
	g.ResolveStack()
	if returned.Tapped || b.Life() != 21 {
		t.Fatal("illegal returned target permitted body")
	}
}

func TestRONTextUpkeepSpellPayments(t *testing.T) {
	testRONTextUpkeepSpellPayments(t, "")
}

func testRONTextUpkeepSpellPayments(t *testing.T, prefix string) {
	rule := prefix + `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([MayCost(PayMana([ManaCostG,ManaCostG])),Unless(CostWasPaid,[PermanentDealsDamage(ThisPermanent,Integer(3),Player(You))])]))`
	for _, scenario := range []string{"paid", "declined", "wrong-color", "insufficient"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario != "declined"}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			obj := castRONSpell(t, g, a.BasePlayer, permanentScopeFactory(t, "Artifact", rule)())
			word, _ := mage.ReplaceColorWord(core.Green, core.Blue)
			if err := g.RevisePrintedSpellText(obj.ID, word); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			a.ManaPool().Clear()
			p := g.FindPermanent(obj.SourceID)
			change, _ := mage.ReplaceColorWord(core.Green, core.Blue)
			if err := g.AddPrintedUpkeepObjectTextEffect(p, core.EndOfTurn, change); err != nil {
				t.Fatal(err)
			}
			color, amount := core.Green, 2
			if scenario == "wrong-color" {
				color = core.Blue
			}
			if scenario == "insufficient" {
				amount = 1
			}
			a.ManaPool().Add(color, amount)
			queueRONUpkeep(g, a)
			if a.offers != 0 || a.ManaPool().TotalMana() != amount {
				t.Fatal("pre-resolution payment")
			}
			clone := g.Clone()
			g.ResolveStack()
			if scenario == "paid" {
				if a.Life() != 20 || a.ManaPool().TotalMana() != 0 {
					t.Fatal("fixed mana symbols changed")
				}
			} else if a.Life() != 17 || a.ManaPool().TotalMana() != amount {
				t.Fatal("unpaid branch spent resources")
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != amount {
				t.Fatal("payment changed clone")
			}
		})
	}
}

func TestRONTextUpkeepSpellIdentity(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([PermanentDealsDamage(ThisPermanent,TheNumberOfPermanentsOnTheBattlefield(IsLandType(Swamp)),Player(You))]))`)
	g, a, b := ronGame()
	card := factory()
	card.SetOwner(a.PlayerID())
	obj := castRONSpell(t, g, a, card)
	word, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedSpellText(obj.ID, word); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(obj.SourceID)
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), a.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.ForestLandType)
	if err := g.AddPrintedUpkeepObjectTextEffect(p, core.Indefinite, change); err != nil {
		t.Fatal(err)
	}
	queueRONUpkeep(g, a)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.ApplyContinuousEffects()
	phased := g.PhaseOut(p.ID())
	if err := g.AddPrintedUpkeepObjectTextEffect(p, core.EndOfTurn, change); err == nil {
		t.Fatal("phased revision accepted")
	}
	queueRONUpkeep(g, b)
	if g.StackSize() != 1 {
		t.Fatal("phased source detected upkeep")
	}
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 20 {
		t.Fatal("pending controller changed")
	}
	g.PhaseIn(phased)
	queueRONUpkeep(g, a)
	if g.StackSize() != 0 {
		t.Fatal("former controller upkeep triggered")
	}
	queueRONUpkeep(g, b)
	g.ResolveStack()
	if b.Life() != 18 {
		t.Fatal("phase cycle lost text or control")
	}
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedUpkeepObjectTextEffect(g.FindPermanent(p.ID()), core.EndOfTurn, change); err == nil {
		t.Fatal("removed rule revised")
	}
	queueRONUpkeep(g, b)
	if g.StackSize() != 0 {
		t.Fatal("removed rule triggered")
	}
	g.RunStepWithPriority(core.Cleanup)
	g.ApplyContinuousEffects()
	queueRONUpkeep(g, b)
	g.ResolveStack()
	if b.Life() != 16 {
		t.Fatal("restoration lost revision")
	}
	stale := g.FindPermanent(p.ID())
	g.BouncePermanentToHand(stale)
	if lki := g.LKIAbilities(p.ID()); len(lki) != len(stale.RuntimeAbilities) || lki[0] != stale.RuntimeAbilities[0] {
		t.Fatal("departure lost the final upkeep rule")
	}
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, a.PlayerID())
	if err := g.AddPrintedUpkeepObjectTextEffect(stale, core.EndOfTurn, change); err == nil {
		t.Fatal("old incarnation revised return")
	}
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if a.Life() != 17 {
		t.Fatal("return did not use printed rules")
	}
}
