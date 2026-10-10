package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextGlobalGrantSpellLifecycle(t *testing.T) {
	g, a, b := ronGame()
	g.SetStep(core.PrecombatMain)
	card := repositoryCard(t, "LEA", "Goblin King")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 10)
	own := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mons's Goblin Raiders"), a.PlayerID())
	other := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mons's Goblin Raiders"), b.PlayerID())
	unrelated := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	first, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.ForestLandType)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, first); err != nil {
		t.Fatal(err)
	}
	check := func(game *mage.Game, walk core.Attr, active bool) {
		t.Helper()
		for _, p := range []*mage.Permanent{own, other} {
			power := 1
			if active {
				power++
			}
			staticPTCheck(t, game, p, power, power)
			for _, kw := range []core.Attr{core.Mountainwalk, core.Forestwalk, core.Swampwalk} {
				if game.FindPermanent(p.ID()).HasKeyword(kw) != (active && kw == walk) {
					t.Fatal("grant and P/T do not agree")
				}
			}
		}
		staticPTCheck(t, game, unrelated, 1, 1)
	}
	check(g, core.Forestwalk, false)
	g.ResolveStack()
	check(g, core.Forestwalk, true)
	source := g.FindPermanent(card.ID())
	id := textStaticID(t, source)
	forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	if !mage.HasLandwalkEvasion(g.FindPermanent(own.ID()), b.PlayerID(), g) {
		t.Fatal("continued landwalk did not prevent blocking")
	}
	phasedLand := g.PhaseOut(forest.ID())
	g.ApplyContinuousEffects()
	if mage.HasLandwalkEvasion(g.FindPermanent(own.ID()), b.PlayerID(), g) {
		t.Fatal("phased land enabled landwalk")
	}
	g.PhaseIn(phasedLand)
	second, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.SwampLandType)
	if err := g.AddPrintedStaticGrantTextEffect(source, id, core.EndOfTurn, second); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
		check(g, core.Swampwalk, true)
	}
	clone := g.Clone()
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	check(clone, core.Forestwalk, true)
	check(g, core.Swampwalk, true)
	g.AddContinuousEffect(mage.RemoveAllAbilities(card.ID(), core.EndOfTurn))
	check(g, core.Swampwalk, false)
	phased := g.PhaseOut(card.ID())
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	check(g, core.Forestwalk, true)
	if textStaticID(t, g.FindPermanent(card.ID())) != id {
		t.Fatal("restoration changed declaration identity")
	}
	phased = g.PhaseOut(card.ID())
	g.ApplyContinuousEffects()
	check(g, core.Forestwalk, false)
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	check(g, core.Forestwalk, true)
	values, err := g.CaptureCopiableValues(g.FindPermanent(card.ID()))
	if err != nil {
		t.Fatal(err)
	}
	printed, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	printedSource := g.PutOnBattlefield(printed, b.PlayerID())
	if !g.FindPermanent(own.ID()).HasKeyword(core.Mountainwalk) {
		t.Fatal("word changes entered copiable values")
	}
	g.RemoveFromBattlefield(g.FindPermanent(printedSource.ID()))
	check(g, core.Forestwalk, true)
	g.BouncePermanentToHand(g.FindPermanent(own.ID()))
	lki := g.LKI(own.ID())
	if lki == nil || lki.Power != 2 || !lki.Snapshot.HasKeyword(core.Forestwalk) {
		t.Fatal("recipient LKI lost continued grant and P/T")
	}
	g.BouncePermanentToHand(g.FindPermanent(card.ID()))
	sourceLKI := g.LKI(card.ID())
	if sourceLKI == nil {
		t.Fatal("source departure lost LKI")
	}
	staticPTCheck(t, g, other, 1, 1)
	lkiSource := g.PutOnBattlefield(sourceLKI.Snapshot.Card.Copy(), b.PlayerID())
	if !g.FindPermanent(other.ID()).HasKeyword(core.Forestwalk) || g.FindPermanent(other.ID()).HasKeyword(core.Mountainwalk) {
		t.Fatal("source LKI lost continued donor rules")
	}
	g.RemoveFromBattlefield(g.FindPermanent(lkiSource.ID()))
	if err := g.AddPrintedStaticGrantTextEffect(source, id, core.Indefinite, second); err == nil {
		t.Fatal("departed source accepted")
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !g.FindPermanent(other.ID()).HasKeyword(core.Mountainwalk) || g.FindPermanent(other.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("recast did not use printed grant")
	}
}

func TestRONTextGlobalGrantSpellCopiesAndMembership(t *testing.T) {
	g, a, b := ronGame()
	g.SetStep(core.PrecombatMain)
	card := repositoryCard(t, "DRK", "Hidden Path")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Green, 10)
	green := g.PutOnBattlefield(textColorCard(core.Green), a.PlayerID())
	blue := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	first, _ := mage.ReplaceColorWord(core.Green, core.Blue)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, first); err != nil {
		t.Fatal(err)
	}
	copyObj := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	second, _ := mage.ReplaceColorWord(core.Blue, core.Red)
	if err := g.RevisePrintedSpellText(copyObj.ID, second); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	third, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := clone.RevisePrintedSpellText(copyObj.ID, third); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	clone.ResolveStack()
	for _, tc := range []struct {
		game    *mage.Game
		yes, no *mage.Permanent
	}{{g, red, green}, {clone, green, red}} {
		if !tc.game.FindPermanent(tc.yes.ID()).HasKeyword(core.Forestwalk) || tc.game.FindPermanent(tc.no.ID()).HasKeyword(core.Forestwalk) || !tc.game.FindPermanent(blue.ID()).HasKeyword(core.Forestwalk) {
			t.Fatal("copies shared semantic rules")
		}
	}
	token := g.FindPermanent(copyObj.SourceID)
	if token == nil || !token.Card.IsToken() || token.ControllerID() != b.PlayerID() || textStaticID(t, token) == textStaticID(t, g.FindPermanent(card.ID())) {
		t.Fatal("copy entry lost separate token or declaration identity")
	}
	g.RemoveFromBattlefield(token)
	if len(b.Graveyard()) != 0 || g.FindPermanent(red.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("copy departure retained grant or physical card")
	}
	late := g.PutOnBattlefield(textColorCard(core.Blue), a.PlayerID())
	if !g.FindPermanent(late.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("late recipient missed continued grant")
	}
	phased := g.PhaseOut(late.ID())
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	if !g.FindPermanent(late.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("returned phased recipient missed grant")
	}
}

func TestRONTextGlobalGrantSpellSharedSubjects(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `EachPermanentLayerEffect(And([IsColor(Red),IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You)),DoesntHaveAbility(Flying)]),[AdjustPT(1,2),AddAbility([Flying,Haste])])`)
	g, a, b := ronGame()
	g.SetStep(core.PrecombatMain)
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	own := g.PutOnBattlefield(textColorCard(core.Green), a.PlayerID())
	other := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	check := func(p *mage.Permanent, active bool) {
		t.Helper()
		power, toughness := 1, 1
		if active {
			power++
			toughness += 2
		}
		staticPTCheck(t, g, p, power, toughness)
		for _, kw := range []core.Attr{core.Flying, core.Haste} {
			if g.FindPermanent(p.ID()).HasKeyword(kw) != active {
				t.Fatal("grants changed shared subject selection")
			}
		}
	}
	for range 3 {
		g.ApplyContinuousEffects()
		check(own, true)
		check(other, false)
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	check(own, false)
	check(other, true)
}
