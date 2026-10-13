package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTemporaryControlPreacher(t *testing.T) {
	for _, change := range []string{"none", "untap-before", "phase-before", "return-before", "target-return", "target-phase", "shroud", "control-before", "untap-after", "phase-after", "control-after", "abilities-after", "copy", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := opponentTargetGame()
			card := repositoryCard(t, "DRK", "Preacher")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			victimCard := mage.NewCreature("Victim", "", 2, 4)
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			mine := g.PutOnBattlefield(mage.NewCreature("Mine", "", 2, 4), a.PlayerID())
			b.requested = []uuid.UUID{victim.ID()}
			source.GrantBaseAttr(core.AttrSummonSick)
			if g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{mine.ID()}) == nil || source.Tapped || g.StackSize() != 0 {
				t.Fatal("sick activation did not roll back")
			}
			source.RevokeBaseAttr(core.AttrSummonSick)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{mine.ID()}); err != nil {
				t.Fatal(err)
			}
			if g.StackPeek().Targets[0] != victim.ID() || victim.ControllerID() != b.PlayerID() || !source.Tapped {
				t.Fatal("opponent choice or timing")
			}
			switch change {
			case "untap-before":
				g.UntapPermanent(g.FindPermanent(source.ID()))
			case "phase-before":
				g.PhaseOut(source.ID())
			case "return-before":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.TapPermanent(g.PutOnBattlefield(card, a.PlayerID()))
			case "target-return":
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(victimCard, b.PlayerID())
			case "target-phase":
				g.PhaseOut(victim.ID())
			case "shroud":
				g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
			case "control-before":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
			}
			if change == "clone" {
				c := g.Clone()
				c.ResolveStack()
				if c.FindPermanent(victim.ID()).ControllerID() != a.PlayerID() || g.FindPermanent(victim.ID()).ControllerID() != b.PlayerID() {
					t.Fatal("pending clone isolation")
				}
			}
			g.ResolveStack()
			active := change != "untap-before" && change != "phase-before" && change != "return-before" && change != "target-return" && change != "target-phase" && change != "shroud"
			want := b.PlayerID()
			if active {
				want = a.PlayerID()
			}
			if g.FindPermanentIncludingPhased(victim.ID()).ControllerID() != want {
				t.Fatal("resolution duration or legality")
			}
			switch change {
			case "untap-after":
				g.UntapPermanent(g.FindPermanent(source.ID()))
				active = false
			case "phase-after":
				g.PhaseIn(g.PhaseOut(source.ID()))
				active = false
			case "control-after":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "abilities-after":
				g.GetEffects().Add(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
			}
			g.ApplyContinuousEffects()
			want = b.PlayerID()
			if active {
				want = a.PlayerID()
			}
			if g.FindPermanentIncludingPhased(victim.ID()).ControllerID() != want {
				t.Fatal("installed duration")
			}
		})
	}
}

func TestRONTemporaryControlInvalid(t *testing.T) {
	for _, duration := range []string{"UntilEndOfTurn()", "UntilEndOfTurn(Integer(1))", "ForAsLongAsPermanentRemainsTapped(Ref_TargetPermanent)", "ForAsLongAsPermanentRemainsTapped(ThisPermanent())", "ForAsLongAsPermanentRemainsTapped(ThisPermanent,ThisPermanent)", "UntilEndOfCombat"} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([GainControlOfPermanentUntil(Ref_TargetPermanent,` + duration + `)])))])`
		nodes, err := Parse(strings.NewReader(src), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid duration accepted", duration)
		}
	}
}

func TestRONTemporaryControlDisharmony(t *testing.T) {
	for _, change := range []string{"none", "phase", "return", "not-attacking", "shroud", "phase-cycle", "copy", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Disharmony")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			victimCard := mage.NewCreature("Attacker", "", 3, 5)
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			g.GetCombat().AddAttacker(victim.ID(), a.PlayerID())
			g.TapPermanent(victim)
			a.ManaPool().Add(core.Red, 1)
			a.ManaPool().Add(core.Colorless, 2)
			if g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim.ID()}, 0) == nil || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 3 {
				t.Fatal("casting window rollback")
			}
			g.SetStep(core.DeclareAttackers)
			a.ManaPool().Clear()
			if g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim.ID()}, 0) == nil || len(a.Hand()) != 1 {
				t.Fatal("unpaid cast succeeded")
			}
			a.ManaPool().Add(core.Red, 1)
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if victim.ControllerID() != b.PlayerID() || !victim.Tapped || !g.GetCombat().IsAttacking(victim.ID()) {
				t.Fatal("actions ran at announcement")
			}
			switch change {
			case "phase":
				g.PhaseOut(victim.ID())
			case "return":
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(victimCard, b.PlayerID())
			case "not-attacking":
				g.GetCombat().RemoveFromCombat(victim.ID())
			case "shroud":
				g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(victim.ID()))
			case "copy":
				g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
			}
			active := change == "none" || change == "copy" || change == "clone"
			if change == "clone" {
				c := g.Clone()
				c.ResolveStack()
				if c.FindPermanent(victim.ID()).ControllerID() != a.PlayerID() || g.FindPermanent(victim.ID()).ControllerID() != b.PlayerID() {
					t.Fatal("pending clone isolation")
				}
			}
			g.ResolveStack()
			want := b.PlayerID()
			if active {
				want = a.PlayerID()
			}
			live := g.FindPermanentIncludingPhased(victim.ID())
			if live.ControllerID() != want {
				t.Fatal("target legality or control")
			}
			if active && (live.Tapped || g.GetCombat().IsAttacking(victim.ID())) {
				t.Fatal("untap or combat removal")
			}
			g.RunStepWithPriority(core.Cleanup)
			if g.FindPermanentIncludingPhased(victim.ID()).ControllerID() != b.PlayerID() {
				t.Fatal("cleanup did not end control")
			}
		})
	}
}

func TestRONTemporaryControlPreacherCastingAndUntap(t *testing.T) {
	for _, choice := range []string{"decline", "accept", "stun", "target-gone"} {
		t.Run(choice, func(t *testing.T) {
			a := &ronStoragePlayer{ronCounterPayer: &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A")}, untap: choice == "accept" || choice == "stun"}
			b := &ronTargetChooser{BasePlayer: mage.NewBasePlayer("B")}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			card := repositoryCard(t, "DRK", "Preacher")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.White, 1)
			if g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0) == nil || a.ManaPool().TotalMana() != 1 || len(a.Hand()) != 1 {
				t.Fatal("failed cast spent resources")
			}
			a.ManaPool().Add(core.White, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			if source == nil || source.CurrentPower(g) != 1 || source.CurrentToughness(g) != 1 || !source.HasAttr(core.AttrMayNotUntap) || !source.HasAttr(core.AttrSummonSick) {
				t.Fatal("entry metadata or untap permission")
			}
			source.RevokeBaseAttr(core.AttrSummonSick)
			victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 3), b.PlayerID())
			b.requested = []uuid.UUID{victim.ID()}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if choice == "stun" {
				g.MutablePermanent(source.ID()).AddCounter(core.Stun, 1)
			}
			if choice == "target-gone" {
				g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
			}
			g.RunStepWithPriority(core.Untap)
			if a.untapCalls != 1 || g.FindPermanent(source.ID()).Tapped != (choice != "accept") {
				t.Fatal("optional untap or stun")
			}
			if live := g.FindPermanent(victim.ID()); live != nil {
				want := a.PlayerID()
				if choice == "accept" {
					want = b.PlayerID()
				}
				if live.ControllerID() != want {
					t.Fatal("untap duration result")
				}
			}
		})
	}
}

func TestRONTemporaryControlContextsAndCopies(t *testing.T) {
	for _, duration := range []string{"UntilEndOfTurn", "ForAsLongAsPermanentRemainsTapped(ThisPermanent)"} {
		nodes, err := Parse(strings.NewReader(`GainControlOfPermanentUntil(ThisPermanent,`+duration+`)`), "context.ron")
		if err != nil {
			t.Fatal(err)
		}
		for _, ctx := range []scope{{permanent: true, enteringSource: true}, {permanent: true, drawReplacementBody: true}} {
			if _, err := compileSourceControl(nodes[0], ctx); err == nil {
				t.Fatal("invalid context accepted")
			}
		}
	}
	a := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.DeclareAttackers)
	card := repositoryCard(t, "LEG", "Disharmony")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 4), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 4), b.PlayerID())
	for _, p := range []*mage.Permanent{first, second} {
		g.GetCombat().AddAttacker(p.ID(), a.PlayerID())
		g.TapPermanent(p)
	}
	a.ManaPool().Add(core.Red, 1)
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	a.targets = []uuid.UUID{second.ID()}
	original := g.StackPeek()
	copied := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if copied == nil || copied.Targets[0] != second.ID() || original.Targets[0] != first.ID() {
		t.Fatal("copy targets shared")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(second.ID()).ControllerID() != a.PlayerID() || g.FindPermanent(first.ID()).ControllerID() != b.PlayerID() {
		t.Fatal("copy changed original subject")
	}
	g.ResolveStack()
	clone := g.Clone()
	g.RunStepWithPriority(core.Cleanup)
	if g.FindPermanent(first.ID()).ControllerID() != b.PlayerID() || g.FindPermanent(second.ID()).ControllerID() != b.PlayerID() || clone.FindPermanent(first.ID()).ControllerID() != a.PlayerID() {
		t.Fatal("cleanup or installed clone isolation")
	}
}

func TestRONTemporaryControlDelayed(t *testing.T) {
	for _, duration := range []string{"UntilEndOfTurn", "ForAsLongAsPermanentRemainsTapped(ThisPermanent)"} {
		g, a, b := ronGame()
		factory := permanentScopeFactory(t, "Artifact", `Activated(TapPermanent(ThisPermanent),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([GainControlOfPermanentUntil(ThisPermanent,`+duration+`)]))]))`)
		card := factory()
		source := g.PutOnBattlefield(card, a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		for range 2 {
			reads := g.WaitingDelayedPermanentSources()
			if len(reads) != 1 || reads[0].ID() != source.ID() {
				t.Fatal("missing waiting source")
			}
		}
		g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
		g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
		g.ResolveStack()
		if g.FindPermanent(source.ID()).ControllerID() != a.PlayerID() {
			t.Fatal("delayed body lost controller")
		}
		if duration == "UntilEndOfTurn" {
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
		} else {
			g.UntapPermanent(g.FindPermanent(source.ID()))
		}
		if g.FindPermanent(source.ID()).ControllerID() != b.PlayerID() {
			t.Fatal("delayed duration did not expire")
		}
	}
}
