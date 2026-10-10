package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func activationWord(t *testing.T, from, to core.Color) mage.TextChange {
	t.Helper()
	c, err := mage.ReplaceColorWord(from, to)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRONTextActivationLayerTargets(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Northern Paladin")
	p := g.PutOnBattlefield(card, a.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	id := p.RuntimeAbilities[0].AbilityID()
	black := g.PutOnBattlefield(textColorCard(core.Black), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	a.ManaPool().Add(core.White, 8)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{black.ID()}); err != nil {
		t.Fatal(err)
	}
	g.UntapPermanent(g.FindPermanent(p.ID()))
	if err := g.AddPrintedActivatedTextEffect(p, id, core.EndOfTurn, activationWord(t, core.Black, core.Green)); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{black.ID()}); err == nil {
		t.Fatal("old target filter remained")
	}
	if g.FindPermanent(p.ID()).Tapped {
		t.Fatal("illegal target paid tap cost")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{green.ID()}); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.RunStepWithPriority(core.Cleanup)
	g.ResolveStack()
	if g.FindPermanent(black.ID()) != nil || g.FindPermanent(green.ID()) != nil {
		t.Fatal("queued targets lost their text")
	}
	if clone.FindPermanent(black.ID()) == nil {
		t.Fatal("clone shared resolution")
	}
	clone.ResolveStack()
	if clone.FindPermanent(green.ID()) != nil {
		t.Fatal("clone lost revised body")
	}
	g.UntapPermanent(g.FindPermanent(p.ID()))
	black = g.PutOnBattlefield(textColorCard(core.Black), b.PlayerID())
	a.ManaPool().Add(core.White, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{black.ID()}); err != nil {
		t.Fatal("expiry did not restore target", err)
	}
}

func TestRONTextActivationLayerLimits(t *testing.T) {
	for _, modifier := range []string{`ActivateOnlyOnceEachTurn`, `ActivateNoMoreThanNumberTimesEachTurn(Integer(2))`} {
		t.Run(modifier, func(t *testing.T) {
			g, a, _ := ronGame()
			factory := permanentScopeFactory(t, "Artifact", `ActivatedWithModifiers(PayMana([ManaCostR]),ActionList([GainLife(Integer(1))]),`+modifier+`)`)
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			id := p.RuntimeAbilities[0].AbilityID()
			change := activationWord(t, core.Red, core.Green)
			if err := g.AddPrintedActivatedTextEffect(p, id, core.EndOfTurn, change); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
				t.Fatal("unpaid activation succeeded")
			}
			a.ManaPool().Add(core.Red, 5)
			count := 1
			if modifier != `ActivateOnlyOnceEachTurn` {
				count = 2
			}
			for range count {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				g.ApplyContinuousEffects()
			}
			clone := g.Clone()
			remover := g.PutOnBattlefield(mage.NewArtifact("Remover", ""), a.PlayerID())
			remove := mage.RemoveAllAbilities(p.ID(), core.WhileOnBattlefield)
			remove.SetSourceID(remover.ID())
			g.AddContinuousEffect(remove)
			if len(g.FindPermanent(p.ID()).RuntimeAbilities) != 0 {
				t.Fatal("removal failed")
			}
			g.GetEffects().Remove(remover.ID())
			g.ApplyContinuousEffects()
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
				t.Fatal("layer restoration reset limit")
			}
			if err := clone.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
				t.Fatal("clone reset limit")
			}
			remove = mage.RemoveAllAbilities(p.ID(), core.WhileOnBattlefield)
			remove.SetSourceID(remover.ID())
			g.AddContinuousEffect(remove)
			g.RunStepWithPriority(core.Cleanup)
			g.GetEffects().Remove(remover.ID())
			g.ApplyContinuousEffects()
			g.SetTurn(2)
			g.SetStep(core.PrecombatMain)
			a.ManaPool().Add(core.Red, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal("hidden limit survived cleanup", err)
			}
		})
	}
}

func TestRONTextActivationLayerHistory(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEA", "Dragon Whelp"}, {"PROMOS", "Nalathni Dragon"}, {"FEM", "Farrelite Priest"}, {"FEM", "Initiates of the Ebon Hand"}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, _ := ronGame()
			p := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
			id := p.RuntimeAbilities[len(p.RuntimeAbilities)-1].AbilityID()
			if err := g.AddPrintedActivatedTextEffect(p, id, core.EndOfTurn, activationWord(t, core.Red, core.Green)); err != nil {
				t.Fatal(err)
			}
			a.ManaPool().Add(core.Red, 8)
			a.ManaPool().Add(core.Colorless, 8)
			for range 3 {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				g.ApplyContinuousEffects()
			}
			phased := g.PhaseOut(p.ID())
			g.RunStepWithPriority(core.Cleanup)
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			g.SetTurn(2)
			g.SetStep(core.PrecombatMain)
			a.ManaPool().Add(core.Red, 8)
			a.ManaPool().Add(core.Colorless, 8)
			for range 4 {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				g.ApplyContinuousEffects()
			}
			if len(g.DelayedTriggers()) != 1 {
				t.Fatal("layer passes lost history")
			}
			g.RunStepWithPriority(core.EndStep)
			if g.FindPermanent(p.ID()) != nil {
				t.Fatal("history did not sacrifice source")
			}
		})
	}
}

func TestRONTextActivationLayerChoices(t *testing.T) {
	for _, tc := range []struct {
		name  string
		color core.Color
	}{{"Circle of Protection: Red", core.Red}, {"Circle of Protection: Blue", core.Blue}} {
		t.Run(tc.name, func(t *testing.T) {
			_, base, b := ronGame()
			a := &ronSourceChoicePlayer{BasePlayer: base}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			p := g.PutOnBattlefield(repositoryCard(t, "LEA", tc.name), a.PlayerID())
			id := p.RuntimeAbilities[0].AbilityID()
			old := g.PutOnBattlefield(textColorCard(tc.color), b.PlayerID())
			green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if err := g.AddPrintedActivatedTextEffect(p, id, core.Indefinite, activationWord(t, tc.color, core.Green)); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if a.source.ID() != old.ID() {
				t.Fatal("pending choice used revised text")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if a.source.ID() != green.ID() {
				t.Fatal("new choice lost revised text")
			}
			g.DealDamageToPlayerFromSource(a, 2, g.CaptureDamageSource(old.ID()))
			g.DealDamageToPlayerFromSource(a, 3, g.CaptureDamageSource(green.ID()))
			if a.Life() != 20 {
				t.Fatal("independent choice shields failed")
			}
			g.DealDamageToPlayerFromSource(a, 1, g.CaptureDamageSource(green.ID()))
			if a.Life() != 19 {
				t.Fatal("next event shield repeated")
			}
		})
	}
}

func TestRONTextActivationLayerBoosts(t *testing.T) {
	for _, name := range []string{"Firebreathing", "Shivan Dragon"} {
		t.Run(name, func(t *testing.T) {
			g, a, _ := ronGame()
			p := g.PutOnBattlefield(repositoryCard(t, "LEA", name), a.PlayerID())
			subject := p
			if name == "Firebreathing" {
				subject = g.PutOnBattlefield(repositoryCard(t, "LEA", "Wall of Wood"), a.PlayerID())
				g.Attach(p.ID(), subject.ID())
			}
			index := len(p.RuntimeAbilities) - 1
			id := p.RuntimeAbilities[index].AbilityID()
			power := subject.CurrentPower(g)
			if err := g.AddPrintedActivatedTextEffect(p, id, core.EndOfTurn, activationWord(t, core.Red, core.Green)); err != nil {
				t.Fatal(err)
			}
			a.ManaPool().Add(core.Green, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
				t.Fatal("text changed mana symbol")
			}
			a.ManaPool().Add(core.Red, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(subject.ID()).CurrentPower(g) != power+1 {
				t.Fatal("revised boost lost subject")
			}
			if name == "Shivan Dragon" && !g.FindPermanent(p.ID()).HasAttr(core.Flying) {
				t.Fatal("sibling keyword changed")
			}
			g.RunStepWithPriority(core.Cleanup)
			if g.FindPermanent(subject.ID()).CurrentPower(g) != power {
				t.Fatal("boost did not expire")
			}
		})
	}
}

func TestRONTextActivationLayerOrderingLifetime(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsColor(Red))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`)
	g, a, b := ronGame()
	card := factory()
	p := g.PutOnBattlefield(card, a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	blue := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
	if err := g.AddPrintedActivatedTextEffect(p, id, core.EndOfTurn, activationWord(t, core.Red, core.Green)); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedActivatedTextEffect(p, id, core.Indefinite, activationWord(t, core.Green, core.Blue)); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
	}
	if err := g.RevisePrintedActivatedAbility(p, id, activationWord(t, core.Blue, core.White)); err == nil {
		t.Fatal("direct revision replaced layered text")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{blue.ID()}); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	phased := g.PhaseOut(p.ID())
	g.RunStepWithPriority(core.Cleanup)
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{blue.ID()}); err == nil {
		t.Fatal("expiry lost baseline")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{red.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	clone.ResolveStack()
	stale := g.FindPermanent(p.ID())
	g.BouncePermanentToHand(stale)
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.AddPrintedActivatedTextEffect(stale, id, core.EndOfTurn, activationWord(t, core.Red, core.Green)); err == nil {
		t.Fatal("returned source inherited text")
	}
	red = g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), returned.ID(), 0, []uuid.UUID{red.ID()}); err != nil {
		t.Fatal("return lost printed rule", err)
	}
}

func TestRONTextActivationLayerBoundaries(t *testing.T) {
	g, a, _ := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([GainLife(Integer(1))]))`)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	other := g.PutOnBattlefield(factory(), a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	change := activationWord(t, core.Red, core.Green)
	for _, tc := range []struct {
		id       uuid.UUID
		duration core.Duration
		changes  []mage.TextChange
	}{{other.RuntimeAbilities[0].AbilityID(), core.Indefinite, []mage.TextChange{change}}, {id, core.WhileOnBattlefield, []mage.TextChange{change}}, {id, core.Indefinite, nil}, {id, core.Indefinite, []mage.TextChange{{}}}} {
		if err := g.AddPrintedActivatedTextEffect(p, tc.id, tc.duration, tc.changes...); err == nil {
			t.Fatal("invalid installation accepted")
		}
	}
	phased := g.PhaseOut(p.ID())
	if err := g.AddPrintedActivatedTextEffect(p, id, core.Indefinite, change); err == nil {
		t.Fatal("phased source accepted")
	}
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	g.MutablePermanent(p.ID()).FaceDown = true
	if err := g.AddPrintedActivatedTextEffect(p, id, core.Indefinite, change); err == nil {
		t.Fatal("face-down source accepted")
	}
	for _, name := range []string{"Magical Hack", "Sleight of Mind"} {
		if _, err := Compile(hostUntapRecord(t, "LEA", name)); err == nil {
			t.Fatal("partial text spell accepted")
		}
	}
}

func TestRONTextActivationLayerBaselineControllerAndLKI(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsColor(Red))],ActionList([DestroyPermanent(Ref_TargetPermanent)]))),Activated(PayMana([]),ActionList([GainLife(Integer(2))]))`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	if err := g.RevisePrintedActivatedAbility(p, id, activationWord(t, core.Red, core.Green)); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedActivatedTextEffect(p, id, core.EndOfTurn, activationWord(t, core.Green, core.Blue)); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	clone.ApplyContinuousEffects()
	if g.FindPermanent(p.ID()).RuntimeAbilities[0].Controller() != a.PlayerID() || clone.FindPermanent(p.ID()).RuntimeAbilities[0].Controller() != b.PlayerID() {
		t.Fatal("controller state leaked between clones")
	}
	if err := clone.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 1, nil); err != nil {
		t.Fatal(err)
	}
	clone.ResolveStack()
	if clone.GetPlayer(b.PlayerID()).Life() != 22 || b.Life() != 20 {
		t.Fatal("sibling action or clone lost controller")
	}
	g.RunStepWithPriority(core.Cleanup)
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{green.ID()}); err != nil {
		t.Fatal("expiry lost prior direct baseline", err)
	}
	stale := clone.FindPermanent(p.ID())
	old := stale.RuntimeAbilities[0]
	clone.BouncePermanentToHand(stale)
	if clone.LKIAbilities(p.ID())[0] != old {
		t.Fatal("departure lost revised LKI")
	}
}

func TestRONTextActivationLayerTimingAndResults(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `ActivatedWithModifiers(PayMana([ManaCostR]),Targeted([TargetPermanent(IsColor(Red))],ActionList([DrawACard,RevealTheCardDrawnThisWay,DiscardTheCardDrawnThisWay,DestroyPermanent(Ref_TargetPermanent)])),ActivateOnlyAsASorcery)`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	if err := g.AddPrintedActivatedTextEffect(p, id, core.Indefinite, activationWord(t, core.Red, core.Green)); err != nil {
		t.Fatal(err)
	}
	a.ManaPool().Add(core.Red, 2)
	g.SetStep(core.Upkeep)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{green.ID()}); err == nil {
		t.Fatal("text reset timing restriction")
	}
	if a.ManaPool().TotalMana() != 2 {
		t.Fatal("timing failure paid cost")
	}
	g.SetStep(core.PrecombatMain)
	a.SetLibrary([]mage.Card{repositoryCard(t, "LEA", "Forest"), repositoryCard(t, "LEA", "Island")})
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{green.ID()}); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		player := game.GetPlayer(a.PlayerID())
		if len(player.Hand()) != 0 || len(player.Graveyard()) != 1 || len(game.PublicReveals()) != 1 || game.FindPermanent(green.ID()) != nil {
			t.Fatal("cloned result frame lost bindings")
		}
	}
}

func TestRONTextActivationLayerModal(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Modal_ChooseOne([Targeted([TargetPermanent(IsColor(Red))],ActionList([DestroyPermanent(Ref_TargetPermanent)])),ActionList([GainLife(Integer(2))])]))`)
	g, a, b := ronModalGame(0)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	a.targets = []uuid.UUID{red.ID()}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedActivatedTextEffect(p, id, core.EndOfTurn, activationWord(t, core.Red, core.Green)); err != nil {
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
		t.Fatal("pending modes lost their independent text")
	}
}

func TestRONTextActivationLayerLandWords(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),Targeted([TargetPermanent(IsLandType(Forest))],ActionList([TapPermanent(Ref_TargetPermanent)])))`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	island := g.PutOnBattlefield(mage.NewLand("Nonbasic Island", mage.WithSubTypes("Island")), b.PlayerID())
	change, err := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedActivatedTextEffect(p, id, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{forest.ID()}); err == nil {
		t.Fatal("old land type remained")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{island.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !g.FindPermanent(island.ID()).Tapped || g.FindPermanent(forest.ID()).Tapped {
		t.Fatal("land word lost exact subject")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{forest.ID()}); err != nil {
		t.Fatal("land word expiry lost baseline", err)
	}
	g.ResolveStack()
	if !g.FindPermanent(forest.ID()).Tapped {
		t.Fatal("restored land target did not tap")
	}
}
