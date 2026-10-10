package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func enterRONUpkeepKeywordSpell(t *testing.T, g *mage.Game, a *mage.BasePlayer, card mage.Card) *mage.Permanent {
	t.Helper()
	player := g.GetPlayer(a.PlayerID())
	defer g.SetPlayerAt(0, player)
	obj := castRONSpell(t, g, a, card)
	castUpkeepSpellWords(t, g, a, obj, "LEA", "Sleight of Mind", "Green", "Blue")
	castUpkeepSpellWords(t, g, a, obj, "LEA", "Magical Hack", "Swamp", "Forest")
	word, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	if err := g.RevisePrintedSpellText(obj.ID, word); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil {
		t.Fatal("word consumer resolved the subject early")
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if p == nil || p.Card.Owner() != card.Owner() || p.Card.IsToken() {
		t.Fatal("continued entry lost physical identity")
	}
	return p
}

func TestRONTextUpkeepSpellKeywordsRecords(t *testing.T) {
	testRONTextUpkeepKeywordRecords(t, enterRONUpkeepKeywordSpell)
}

func TestRONTextUpkeepSpellKeywordsWall(t *testing.T) {
	testRONTextUpkeepKeywordWall(t, enterRONUpkeepKeywordSpell)
}

func TestRONTextUpkeepSpellKeywordsLifecycle(t *testing.T) {
	factory := textCopyFactory(t, "Keyword Karma", `Flying,Trample,Flying,TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),ActionList([PermanentDealsDamage(ThisPermanent,TheNumberOfPermanentsOnTheBattlefield(And([IsLandType(Swamp),ControlledByAPlayer(SinglePlayer(Trigger_ThatPlayer))])),Player(Trigger_ThatPlayer))]))`)
	testRONTextUpkeepSpellLifecycle(t, factory(), core.Flying, core.Trample)
}

func TestRONTextUpkeepSpellKeywordsIdentity(t *testing.T) {
	factory := textCopyFactory(t, "Controller keywords", `Flying,Trample,TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([PermanentDealsDamage(ThisPermanent,TheNumberOfPermanentsOnTheBattlefield(IsLandType(Swamp)),Player(You))]))`)
	g, a, b := ronGame()
	card := factory()
	obj := castRONSpell(t, g, a, card)
	castUpkeepSpellWords(t, g, a, obj, "LEA", "Magical Hack", "Swamp", "Forest")
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), a.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	queueRONUpkeep(g, a)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.ApplyContinuousEffects()
	phased := g.PhaseOut(p.ID())
	queueRONUpkeep(g, b)
	if g.StackSize() != 1 {
		t.Fatal("phased source detected upkeep")
	}
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 20 {
		t.Fatal("pending body lost detection controller")
	}
	g.PhaseIn(phased)
	current := g.FindPermanent(p.ID())
	ground := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 1, 1), a.PlayerID())
	if mage.CanBlock(ground, current, g) || !current.HasKeyword(core.Trample) {
		t.Fatal("continued text lost keyword mechanics")
	}
	queueRONUpkeep(g, a)
	if g.StackSize() != 0 {
		t.Fatal("former controller triggered upkeep")
	}
	queueRONUpkeep(g, b)
	g.AddPreventionShield(b.PlayerID(), 1)
	g.BouncePermanentToHand(current)
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, a.PlayerID())
	g.ResolveStack()
	if b.Life() != 19 || !returned.HasKeyword(core.Flying) {
		t.Fatal("pending departure lost damage, prevention or printed return")
	}
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if a.Life() != 17 {
		t.Fatal("return did not use printed Swamp")
	}
}

func TestRONTextUpkeepSpellKeywordsInventory(t *testing.T) {
	testRONTextUpkeepKeywordInventory(t, true)
}

func TestRONTextUpkeepSpellKeywordsPayments(t *testing.T) {
	testRONTextUpkeepSpellPayments(t, "Flying,Trample,")
}

func TestRONTextUpkeepSpellKeywordsTargets(t *testing.T) {
	testRONTextUpkeepSpellPendingTargets(t, "Flying,Trample,")
}

func TestRONTextUpkeepSpellKeywordsBoundaries(t *testing.T) {
	TestRONTextUpkeepKeywordsBoundaries(t)
	g, a, _ := ronGame()
	word, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	for _, sibling := range []string{`Landwalk(IsLandType(Swamp))`, `Protection(FromColor(Colors([Red])))`, `Activated(PayMana([]),ActionList([GainLife(Integer(1))]))`, `AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent)])`} {
		eventPlayer := "You"
		if sibling == `AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent)])` {
			eventPlayer = "TheChosenPlayer"
		}
		card := textCopyFactory(t, "Foreign spell sibling", sibling+`,Flying,TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(`+eventPlayer+`)),ActionList([GainLife(Integer(1))]))`)()
		obj := castRONSpell(t, g, a, card)
		if err := g.RevisePrintedSpellText(obj.ID, word); err == nil {
			t.Fatal("unsupported spell sibling accepted", sibling)
		}
		g.CounterSpellOnStack(obj.SourceID)
	}
}
