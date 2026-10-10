package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func castActivatedObjectWords(t *testing.T, g *mage.Game, base mage.Player, subject *mage.Permanent, name, from, to string) {
	t.Helper()
	a := &intrinsicWordTester{Player: base, words: []string{from, to}}
	g.SetPlayerAt(0, a)
	spell := grantObjectConsumer(t, name)()
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{subject.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	if a.calls != 0 {
		t.Fatal("word choices ran at announcement")
	}
	g.ResolveStack()
	if a.calls != 2 {
		t.Fatal("word choices did not resolve")
	}
	g.SetPlayerAt(0, base)
}

func TestRONTextActivatedObjectRecords(t *testing.T) {
	for _, name := range []string{"Frozen Shade", "Granite Gargoyle", "Shivan Dragon", "Prodigal Sorcerer", "Northern Paladin", "Circle of Protection: Red"} {
		for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
			t.Run(set+"/"+name, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, set, name)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				p.RevokeBaseAttr(core.AttrSummonSick)
				id := p.RuntimeAbilities[0].AbilityID()
				from := "Red"
				if name == "Northern Paladin" {
					from = "Black"
				}
				castActivatedObjectWords(t, g, a, p, "Sleight of Mind", from, "Green")
				p = g.FindPermanent(p.ID())
				if p.RuntimeAbilities[0].AbilityID() != id {
					t.Fatal("object text changed activation identity")
				}
				var targets []uuid.UUID
				switch name {
				case "Northern Paladin":
					black := g.PutOnBattlefield(textColorCard(core.Black), b.PlayerID())
					green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
					a.ManaPool().Add(core.White, 2)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{black.ID()}); err == nil {
						t.Fatal("old target filter remains")
					}
					if g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 2 {
						t.Fatal("illegal target paid cost")
					}
					targets = []uuid.UUID{green.ID()}
				case "Prodigal Sorcerer":
					targets = []uuid.UUID{b.PlayerID()}
				case "Circle of Protection: Red":
					a.ManaPool().Add(core.Colorless, 1)
				default:
					color := core.Red
					if name == "Frozen Shade" {
						color = core.Black
					}
					a.ManaPool().Add(core.Green, 1)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
						t.Fatal("word change altered mana symbol")
					}
					a.ManaPool().Add(color, 1)
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, targets); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				switch name {
				case "Frozen Shade":
					staticPTCheck(t, g, p, 1, 2)
				case "Granite Gargoyle":
					staticPTCheck(t, g, p, 2, 3)
				case "Shivan Dragon":
					staticPTCheck(t, g, p, 6, 5)
				case "Prodigal Sorcerer":
					if b.Life() != 19 || !g.FindPermanent(p.ID()).Tapped {
						t.Fatal("damage activation failed")
					}
				case "Northern Paladin":
					if g.FindPermanent(targets[0]) != nil {
						t.Fatal("revised target was not destroyed")
					}
				}
				if (name == "Granite Gargoyle" || name == "Shivan Dragon") && !g.FindPermanent(p.ID()).HasKeyword(core.Flying) {
					t.Fatal("fixed flying lost")
				}
				values, err := g.CaptureCopiableValues(g.FindPermanent(p.ID()))
				if err != nil {
					t.Fatal(err)
				}
				if _, err := values.Build(); err != nil {
					t.Fatal(err)
				}
				clone := g.Clone()
				phased := g.PhaseOut(p.ID())
				g.ApplyContinuousEffects()
				g.PhaseIn(phased)
				g.ApplyContinuousEffects()
				if g.FindPermanent(p.ID()).RuntimeAbilities[0].AbilityID() != id || clone.FindPermanent(p.ID()).RuntimeAbilities[0].AbilityID() != id {
					t.Fatal("clone or phase cycle lost identity")
				}
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				switch name {
				case "Frozen Shade":
					staticPTCheck(t, g, p, 0, 1)
				case "Granite Gargoyle":
					staticPTCheck(t, g, p, 2, 2)
				case "Shivan Dragon":
					staticPTCheck(t, g, p, 5, 5)
				}
				old := g.FindPermanent(p.ID()).RuntimeAbilities[0]
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if g.LKIAbilities(p.ID())[0] != old {
					t.Fatal("departure lost revised LKI")
				}
				a.RemoveFromHand(card.ID())
				returned := g.PutOnBattlefield(card, a.PlayerID())
				returned.RevokeBaseAttr(core.AttrSummonSick)
				if name == "Northern Paladin" {
					black := g.PutOnBattlefield(textColorCard(core.Black), b.PlayerID())
					a.ManaPool().Add(core.White, 2)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), returned.ID(), 0, []uuid.UUID{black.ID()}); err != nil {
						t.Fatal("return lost printed target", err)
					}
				}
			})
		}
	}
}

func TestRONTextActivatedObjectSiblings(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `ActivatedWithModifiers(PayMana([ManaCostR]),Targeted([TargetPermanent(IsColor(Red))],ActionList([DestroyPermanent(Ref_TargetPermanent)])),ActivateOnlyOnceEachTurn),Activated(PayMana([]),Targeted([TargetPermanent(IsColor(Red))],ActionList([TapPermanent(Ref_TargetPermanent)])))`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	blue := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
	a.ManaPool().Add(core.Red, 3)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{red.ID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedActivatedObjectTextEffect(p, core.EndOfTurn, activationWord(t, core.Red, core.Green)); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedActivatedObjectTextEffect(g.FindPermanent(p.ID()), core.Indefinite, activationWord(t, core.Green, core.Blue)); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{blue.ID()}); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{blue.ID()}); err == nil {
		t.Fatal("object text reset activation limit")
	}
	clone := g.Clone()
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{green.ID()}); err == nil {
		t.Fatal("expiry retained expired first change")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{red.ID()}); err != nil {
		t.Fatal("expiry lost original baseline", err)
	}
	g.ResolveStack()
	clone.ResolveStack()
	if g.FindPermanent(blue.ID()) != nil || clone.FindPermanent(blue.ID()) != nil || !g.FindPermanent(red.ID()).Tapped || !clone.FindPermanent(red.ID()).Tapped {
		t.Fatal("pending declarations lost earlier words")
	}
	if g.FindPermanent(green.ID()) == nil {
		t.Fatal("unselected target changed")
	}
}

func TestRONTextActivatedObjectBoundaries(t *testing.T) {
	for _, name := range []string{"Firebreathing", "Gauntlet of Might", "Conversion", "Goblin King", "Green Ward"} {
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(repositoryCard(t, "LEA", name), a.PlayerID())
		if err := g.AddPrintedActivatedObjectTextEffect(p, core.Indefinite, activationWord(t, core.Red, core.Green)); err == nil {
			t.Fatalf("unsupported complete context accepted: %s", name)
		}
	}
	for _, body := range []string{
		`CreatePermanentLayerEffect(ThisPermanent,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheSecondChosenColorWord)])`,
		`ChooseTwoColorWords,CreatePermanentLayerEffect(ThisPermanent,[ReplaceLandTypeVariableWithNewLandTypeVariable(TheFirstChosenColorWord,TheSecondChosenColorWord)])`,
		`ChooseTwoColorWords,CreatePermanentLayerEffect(ThisPermanent,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheFirstChosenColorWord)])`,
		`ChooseTwoColorWords,CreatePermanentLayerEffect(ThisPermanent,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheSecondChosenColorWord),AdjustPT(1,1)])`,
	} {
		records, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Cardtypes:[Artifact]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		record := records[0]
		if _, err := Compile(record); err == nil {
			t.Fatal("invalid text declaration accepted")
		}
	}
}

func TestRONTextActivatedObjectTimingRemovalAndControl(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `ActivatedWithModifiers(PayMana([ManaCostR]),ActionList([GainLife(Integer(2))]),And([ActivateOnlyOnceEachTurn,ActivateOnlyAsASorcery]))`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := g.AddPrintedActivatedObjectTextEffect(p, core.Indefinite, activationWord(t, core.Red, core.Green)); err != nil {
		t.Fatal(err)
	}
	a.ManaPool().Add(core.Red, 2)
	g.SetStep(core.Upkeep)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
		t.Fatal("object text lost sorcery restriction")
	}
	if a.ManaPool().TotalMana() != 2 {
		t.Fatal("illegal timing paid cost")
	}
	g.SetStep(core.PrecombatMain)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	id := g.FindPermanent(p.ID()).RuntimeAbilities[0].AbilityID()
	clone := g.Clone()
	clone.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	clone.ApplyContinuousEffects()
	if clone.FindPermanent(p.ID()).RuntimeAbilities[0].Controller() != b.PlayerID() || g.FindPermanent(p.ID()).RuntimeAbilities[0].Controller() != a.PlayerID() {
		t.Fatal("cloned control change shared context")
	}
	g.ResolveStack()
	clone.ResolveStack()
	if a.Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("pending controller changed")
	}
	remover := g.PutOnBattlefield(mage.NewArtifact("Remover", ""), a.PlayerID())
	remove := mage.RemoveAllAbilities(p.ID(), core.WhileOnBattlefield)
	remove.SetSourceID(remover.ID())
	g.AddContinuousEffect(remove)
	if len(g.FindPermanent(p.ID()).RuntimeAbilities) != 0 {
		t.Fatal("removal failed")
	}
	g.GetEffects().Remove(remover.ID())
	g.ApplyContinuousEffects()
	if g.FindPermanent(p.ID()).RuntimeAbilities[0].AbilityID() != id {
		t.Fatal("restoration lost identity")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
		t.Fatal("restoration reset limit")
	}
	phased := g.PhaseOut(p.ID())
	g.RunStepWithPriority(core.Cleanup)
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	g.SetTurn(2)
	g.SetStep(core.PrecombatMain)
	a.ManaPool().Add(core.Red, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal("phased cleanup did not reset limit", err)
	}
}

func TestRONTextActivatedObjectPendingChoices(t *testing.T) {
	_, base, b := ronGame()
	a := &ronSourceChoicePlayer{BasePlayer: base}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	p := g.PutOnBattlefield(repositoryCard(t, "LEA", "Circle of Protection: Red"), a.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedActivatedObjectTextEffect(p, core.Indefinite, activationWord(t, core.Red, core.Green)); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.source.ID() != red.ID() {
		t.Fatal("pending choice used revised text")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.source.ID() != green.ID() {
		t.Fatal("new choice lost revised text")
	}
	g.DealDamageToPlayerFromSource(a, 2, g.CaptureDamageSource(red.ID()))
	g.DealDamageToPlayerFromSource(a, 3, g.CaptureDamageSource(green.ID()))
	if a.Life() != 20 {
		t.Fatal("independent shields failed")
	}
	g.DealDamageToPlayerFromSource(a, 1, g.CaptureDamageSource(green.ID()))
	if a.Life() != 19 {
		t.Fatal("shield repeated")
	}
}

func TestRONTextActivatedObjectLandWordsAndResults(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),Targeted([TargetPermanent(IsLandType(Forest))],ActionList([DrawACard,RevealTheCardDrawnThisWay,DiscardTheCardDrawnThisWay,TapPermanent(Ref_TargetPermanent)])))`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	islandCard := repositoryCard(t, "LEA", "Island")
	island := g.PutOnBattlefield(islandCard, b.PlayerID())
	castActivatedObjectWords(t, g, a, p, "Magical Hack", "Forest", "Island")
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{forest.ID()}); err == nil {
		t.Fatal("old land target remained")
	}
	a.SetLibrary([]mage.Card{repositoryCard(t, "LEA", "Forest"), repositoryCard(t, "LEA", "Island")})
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{island.ID()}); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.BouncePermanentToHand(g.FindPermanent(island.ID()))
	b.RemoveFromHand(islandCard.ID())
	returned := g.PutOnBattlefield(islandCard, b.PlayerID())
	g.ResolveStack()
	if g.FindPermanent(returned.ID()).Tapped || len(a.Graveyard()) != 1 {
		t.Fatal("returned target inherited visit or illegal body resolved")
	}
	clone.ResolveStack()
	player := clone.GetPlayer(a.PlayerID())
	if !clone.FindPermanent(island.ID()).Tapped || len(player.Hand()) != 0 || len(player.Graveyard()) != 2 || len(clone.PublicReveals()) != 1 {
		t.Fatal("clone lost independent target or result bindings")
	}
}

func TestRONTextActivatedObjectModal(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Modal_ChooseOne([Targeted([TargetPermanent(IsColor(Red))],ActionList([DestroyPermanent(Ref_TargetPermanent)])),ActionList([GainLife(Integer(2))])]))`)
	g, a, b := ronModalGame(0)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	a.targets = []uuid.UUID{red.ID()}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedActivatedObjectTextEffect(p, core.EndOfTurn, activationWord(t, core.Red, core.Green)); err != nil {
		t.Fatal(err)
	}
	a.targets = []uuid.UUID{green.ID()}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	a.mode = 1
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.ResolveStack()
	if a.Life() != 22 || g.FindPermanent(red.ID()) != nil || g.FindPermanent(green.ID()) != nil {
		t.Fatal("pending modal bodies lost their own text or selection")
	}
}
