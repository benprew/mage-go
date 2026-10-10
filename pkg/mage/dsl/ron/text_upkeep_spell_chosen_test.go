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

func TestRONTextUpkeepSpellChosenRecords(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
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
			if name != "Black Vise" && name != "The Rack" {
				continue
			}
			counts[name]++
			t.Run(filepath.Base(file)+"/"+name, func(t *testing.T) {
				factory, err := Compile(record)
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				card := factory()
				obj := castRONSpell(t, g, a, card)
				castUpkeepSpellWords(t, g, a, obj, "LEA", "Sleight of Mind", "Red", "Blue")
				castUpkeepSpellWords(t, g, a, obj, "LEA", "Magical Hack", "Swamp", "Forest")
				word, _ := mage.ReplaceColorWord(core.Red, core.Green)
				if err := g.RevisePrintedSpellText(obj.ID, word); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(card.ID()) != nil {
					t.Fatal("entry happened before resolution")
				}
				copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
				if copied == nil {
					t.Fatal("copy failed")
				}
				clone := g.Clone()
				g.ResolveStack()
				clone.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					p, token := game.FindPermanent(card.ID()), game.FindPermanent(copied.SourceID)
					if p == nil || token == nil || p.ChosenPlayer != b.PlayerID() || token.ChosenPlayer != a.PlayerID() || !token.Card.IsToken() {
						t.Fatal("entry choices or token identity lost")
					}
					if p.Card.Owner() != a.PlayerID() || p.Card.IsToken() {
						t.Fatal("physical identity lost")
					}
					if err := game.AddPrintedUpkeepObjectTextEffect(p, core.Indefinite, word); err != nil {
						t.Fatal(err)
					}
					game.RemoveFromBattlefield(token)
				}
				fillRONHand(b, 4)
				if name == "The Rack" {
					for _, held := range b.Hand() {
						b.RemoveFromHand(held.ID())
					}
				}
				queueRONUpkeep(g, a)
				if g.StackSize() != 0 {
					t.Fatal("unchosen upkeep triggered")
				}
				queueRONUpkeep(g, b)
				if g.StackSize() != 1 || b.Life() != 20 {
					t.Fatal("chosen upkeep timing lost")
				}
				if name == "Black Vise" {
					fillRONHand(b, 3)
				} else {
					fillRONHand(b, 1)
				}
				p := g.FindPermanent(card.ID())
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.ApplyContinuousEffects()
				phased := g.PhaseOut(p.ID())
				queueRONUpkeep(g, b)
				if g.StackSize() != 1 {
					t.Fatal("phased source detected upkeep")
				}
				g.ResolveStack()
				want := 17
				if name == "The Rack" {
					want = 18
				}
				if b.Life() != want || a.Life() != 20 {
					t.Fatal("resolution hand size or saved event player lost")
				}
				g.PhaseIn(phased)
				p = g.FindPermanent(card.ID())
				if p.ChosenPlayer != b.PlayerID() {
					t.Fatal("control or phasing changed entry choice")
				}
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.EndOfTurn))
				queueRONUpkeep(g, b)
				if g.StackSize() != 0 {
					t.Fatal("removed rule triggered")
				}
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				queueRONUpkeep(g, b)
				g.AddPreventionShield(b.PlayerID(), 1)
				stale := g.FindPermanent(p.ID())
				g.BouncePermanentToHand(stale)
				if g.LKI(p.ID()) == nil || a.Hand()[0] != card {
					t.Fatal("LKI or printed departure lost")
				}
				a.RemoveFromHand(card.ID())
				returned := g.PutOnBattlefield(card, b.PlayerID())
				if returned.ChosenPlayer != a.PlayerID() {
					t.Fatal("return reused the entry choice")
				}
				g.ResolveStack()
				damage := 2
				if name == "The Rack" {
					damage = 1
				}
				if b.Life() != want-damage {
					t.Fatal("pending departed body or prevention lost")
				}
				if err := g.AddPrintedUpkeepObjectTextEffect(stale, core.EndOfTurn, word); err == nil {
					t.Fatal("stale visit revised return")
				}
				queueRONUpkeep(g, b)
				if g.StackSize() != 0 {
					t.Fatal("returned source kept old choice")
				}
				queueRONUpkeep(g, a)
				if g.StackSize() != 1 {
					t.Fatal("returned source lost fresh choice")
				}
				if clone.GetPlayer(b.PlayerID()).Life() != 20 || clone.FindPermanent(card.ID()).ChosenPlayer != b.PlayerID() {
					t.Fatal("clone shared choice or actions")
				}
			})
		}
	}
	if !maps.Equal(counts, map[string]int{"Black Vise": 6, "The Rack": 2}) {
		t.Fatalf("unmapped records: %v", counts)
	}
}

func TestRONTextUpkeepSpellChosenLifecycle(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent)]),TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(TheChosenPlayer)),ActionList([PermanentDealsDamage(ThisPermanent,TheNumberOfPermanentsOnTheBattlefield(And([IsLandType(Swamp),ControlledByAPlayer(SinglePlayer(Trigger_ThatPlayer))])),Player(Trigger_ThatPlayer))]))`)
	g, a, b := ronGame()
	card := factory()
	obj := castRONSpell(t, g, a, card)
	castUpkeepSpellWords(t, g, a, obj, "LEA", "Magical Hack", "Swamp", "Forest")
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy failed")
	}
	second, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	if err := g.RevisePrintedSpellText(copied.ID, second); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	third, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.MountainLandType)
	if err := clone.RevisePrintedSpellText(copied.ID, third); err != nil {
		t.Fatal(err)
	}
	for _, game := range []*mage.Game{g, clone} {
		game.ResolveStack()
		for _, land := range []string{"Swamp", "Forest", "Forest", "Island", "Island", "Island", "Mountain"} {
			game.PutOnBattlefield(repositoryCard(t, "LEA", land), a.PlayerID())
			game.PutOnBattlefield(repositoryCard(t, "LEA", land), b.PlayerID())
		}
		queueRONUpkeep(game, a)
		game.ResolveStack()
	}
	if a.Life() != 17 || clone.GetPlayer(a.PlayerID()).Life() != 19 {
		t.Fatal("copies or clones shared text")
	}
	p := g.FindPermanent(card.ID())
	id := p.RuntimeAbilities[1].AbilityID()
	queueRONUpkeep(g, b)
	if err := g.AddPrintedUpkeepObjectTextEffect(p, core.EndOfTurn, second); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
	}
	pendingClone := g.Clone()
	g.ResolveStack()
	pendingClone.ResolveStack()
	if b.Life() != 18 || pendingClone.GetPlayer(b.PlayerID()).Life() != 18 {
		t.Fatal("pending body lost detection text")
	}
	queueRONUpkeep(g, b)
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	g.ResolveStack()
	if b.Life() != 12 || a.Life() != 17 {
		t.Fatal("trigger copy changed event player")
	}
	values, err := g.CaptureCopiableValues(p)
	if err != nil {
		t.Fatal(err)
	}
	printed, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	sibling := g.PutOnBattlefield(printed, b.PlayerID())
	if sibling.ChosenPlayer != a.PlayerID() {
		t.Fatal("copiable values kept choice")
	}
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if a.Life() != 13 {
		t.Fatal("copiable values kept ordinary text changes")
	}
	g.RemoveFromBattlefield(sibling)
	g.RemoveFromBattlefield(g.FindPermanent(copied.SourceID))
	g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.EndOfTurn))
	queueRONUpkeep(g, b)
	if g.StackSize() != 0 {
		t.Fatal("removed rule triggered")
	}
	phased := g.PhaseOut(p.ID())
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	p = g.FindPermanent(p.ID())
	if p.ChosenPlayer != b.PlayerID() || p.RuntimeAbilities[1].AbilityID() != id {
		t.Fatal("restoration changed choice or declaration")
	}
	queueRONUpkeep(g, b)
	g.BouncePermanentToHand(p)
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, b.PlayerID())
	g.ResolveStack()
	if b.Life() != 10 || returned.ChosenPlayer != a.PlayerID() {
		t.Fatal("departure lost baseline or return choice")
	}
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if a.Life() != 12 {
		t.Fatal("return retained ordinary changes")
	}
	if pendingClone.FindPermanent(card.ID()).ChosenPlayer != b.PlayerID() {
		t.Fatal("clone choice followed return")
	}
}

func TestRONTextUpkeepSpellChosenBoundaries(t *testing.T) {
	choice := `AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent)])`
	trigger := `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(TheChosenPlayer)),ActionList([GainLife(Integer(1))]))`
	for _, rules := range []string{trigger, choice + "," + choice + "," + trigger, strings.ReplaceAll(choice, "Opponent", "AnyPlayer") + "," + trigger, strings.ReplaceAll(choice, "ThisPermanent", "HostPermanent") + "," + trigger} {
		records, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rules+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(records[0]); err == nil {
			t.Fatal("invalid declaration accepted", rules)
		}
	}
	for _, rules := range []string{choice + "," + trigger + `,Activated(PayMana([]),ActionList([GainLife(Integer(1))]))`, choice + "," + trigger + `,Protection(FromColor(Colors([Red])))`} {
		factory := permanentScopeFactory(t, "Artifact", rules)
		g, a, _ := ronGame()
		obj := castRONSpell(t, g, a, factory())
		word, _ := mage.ReplaceColorWord(core.Red, core.Blue)
		if err := g.RevisePrintedSpellText(obj.ID, word); err == nil {
			t.Fatal("foreign sibling context accepted")
		}
	}
}

func TestRONTextUpkeepSpellChosenRuleOrder(t *testing.T) {
	trigger := `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(TheChosenPlayer)),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))`
	choice := `AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent)])`
	for _, rules := range []string{trigger + "," + choice, `Flying,` + choice + `,` + trigger + `,Flying`} {
		factory := textCopyFactory(t, "Chosen order", rules)
		g, a, b := ronGame()
		obj := castRONSpell(t, g, a, factory())
		word, _ := mage.ReplaceColorWord(core.Red, core.Blue)
		if err := g.RevisePrintedSpellText(obj.ID, word); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		queueRONUpkeep(g, a)
		if g.StackSize() != 0 {
			t.Fatal("wrong event player")
		}
		queueRONUpkeep(g, b)
		g.ResolveStack()
		if b.Life() != 21 {
			t.Fatal("choice declaration order changed scope")
		}
		p := g.FindPermanent(obj.SourceID)
		if strings.Contains(rules, "Flying") && !p.HasKeyword(core.Flying) {
			t.Fatal("fixed keyword lost")
		}
	}
}

func TestRONTextUpkeepSpellChosenPayments(t *testing.T) {
	rule := `AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent)]),TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(TheChosenPlayer)),ActionList([PlayerMayCost(Trigger_ThatPlayer,PayMana([ManaCostG,ManaCostG])),Unless(CostWasPaid,[PermanentDealsDamage(ThisPermanent,Integer(3),Player(Trigger_ThatPlayer))])]))`
	for _, scenario := range []string{"paid", "declined", "wrong-color", "insufficient"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			payer := &ronPaymentPlayer{BasePlayer: b, accept: scenario != "declined"}
			g.SetPlayerAt(1, payer)
			obj := castRONSpell(t, g, a, permanentScopeFactory(t, "Artifact", rule)())
			word, _ := mage.ReplaceColorWord(core.Green, core.Blue)
			if err := g.RevisePrintedSpellText(obj.ID, word); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			color, amount := core.Green, 2
			if scenario == "wrong-color" {
				color = core.Blue
			}
			if scenario == "insufficient" {
				amount = 1
			}
			b.ManaPool().Add(color, amount)
			queueRONUpkeep(g, b)
			if payer.offers != 0 || b.ManaPool().TotalMana() != amount {
				t.Fatal("payment ran before resolution")
			}
			clone := g.Clone()
			ownMana := a.ManaPool().TotalMana()
			g.ResolveStack()
			if scenario == "paid" {
				if b.Life() != 20 || b.ManaPool().TotalMana() != 0 {
					t.Fatal("fixed mana payment changed")
				}
			} else if b.Life() != 17 || b.ManaPool().TotalMana() != amount {
				t.Fatal("unpaid resources changed")
			}
			if a.ManaPool().TotalMana() != ownMana || a.Life() != 20 {
				t.Fatal("payment used controller")
			}
			if clone.GetPlayer(b.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != amount {
				t.Fatal("clone shared payment")
			}
		})
	}
}

func TestRONTextUpkeepSpellChosenTargets(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent)]),TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(TheChosenPlayer)),Targeted([TargetPermanent(IsColor(Red))],ActionList([TapPermanent(Ref_TargetPermanent),PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))])))`)
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
	a.targets = []uuid.UUID{red.ID()}
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.AddPrintedUpkeepObjectTextEffect(p, core.Indefinite, change); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("pending target text changed")
	}
	clone := g.Clone()
	a.targets = []uuid.UUID{green.ID()}
	queueRONUpkeep(g, b)
	g.ResolveStack()
	if !g.FindPermanent(red.ID()).Tapped || !g.FindPermanent(green.ID()).Tapped || b.Life() != 22 {
		t.Fatal("independent target declarations lost")
	}
	clone.ResolveStack()
	if clone.FindPermanent(green.ID()).Tapped || clone.GetPlayer(b.PlayerID()).Life() != 21 {
		t.Fatal("clone shared target actions")
	}
	g.UntapPermanent(green)
	a.targets = []uuid.UUID{green.ID()}
	queueRONUpkeep(g, b)
	g.BouncePermanentToHand(green)
	b.RemoveFromHand(green.Card.ID())
	returned := g.PutOnBattlefield(green.Card, b.PlayerID())
	g.ResolveStack()
	if returned.Tapped || b.Life() != 22 {
		t.Fatal("returned target supplied prior visit")
	}
}
