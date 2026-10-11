package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONAPNAPPermanentChoices(t *testing.T) {
	for _, active := range []int{0, 1} {
		t.Run(string(rune('0'+active)), func(t *testing.T) {
			body := `SimultaneousPermanentChoices([PlayerAction(You,ChooseAPermanent(And([IsCardtype(Artifact),ControlledByAPlayer(SinglePlayer(You))]))),PlayerAction(Ref_TargetPlayer,ChooseAPermanent(And([IsCardtype(Artifact),ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer))])))]),ExchangeControl(ThePermanentChosenByPlayerThisWay(You),ThePermanentChosenByPlayerThisWay(Ref_TargetPlayer))`
			f := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([`+body+`])))`)
			g, baseA, baseB := ronGame()
			g.SetActivePlayerIndex(active)
			var order []string
			a := &ronPermanentChoicePlayer{Player: baseA, choose: func(ps []*mage.Permanent) *mage.Permanent { order = append(order, "a"); return ps[len(ps)-1] }}
			b := &ronPermanentChoicePlayer{Player: baseB, choose: func(ps []*mage.Permanent) *mage.Permanent { order = append(order, "b"); return ps[len(ps)-1] }}
			g.SetPlayerAt(0, a)
			g.SetPlayerAt(1, b)
			chosen := make([]*mage.Permanent, 0, 2)
			for _, p := range []mage.Player{a, b} {
				g.PutOnBattlefield(mage.NewArtifact("First", ""), p.PlayerID())
				chosen = append(chosen, g.PutOnBattlefield(mage.NewArtifact("Last", ""), p.PlayerID()))
			}
			c := f()
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			g.ResolveStack()
			want := "a,b"
			if active == 1 {
				want = "b,a"
			}
			if strings.Join(order, ",") != want {
				t.Fatal("wrong decision order", order)
			}
			if g.FindPermanent(chosen[0].ID()).ControllerID() != b.PlayerID() || g.FindPermanent(chosen[1].ID()).ControllerID() != a.PlayerID() {
				t.Fatal("exchange lost grouped bindings")
			}
			clone.ResolveStack()
		})
	}
}

func TestRONAPNAPPermanentChoicesJuxtapose(t *testing.T) {
	g, baseA, baseB := ronGame()
	g.SetActivePlayerIndex(1)
	var order []string
	a := &ronPermanentChoicePlayer{Player: baseA, choose: func(ps []*mage.Permanent) *mage.Permanent { order = append(order, "a"); return ps[len(ps)-1] }}
	b := &ronPermanentChoicePlayer{Player: baseB, choose: func(ps []*mage.Permanent) *mage.Permanent { order = append(order, "b"); return ps[len(ps)-1] }}
	g.SetPlayerAt(0, a)
	g.SetPlayerAt(1, b)
	for _, p := range []mage.Player{a, b} {
		for range 2 {
			g.PutOnBattlefield(mage.NewCreature("Tie", "{3}", 1, 4), p.PlayerID())
			g.PutOnBattlefield(mage.NewArtifact("Tie artifact", "{2}"), p.PlayerID())
		}
	}
	c := juxtaposeFactory(t)()
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	a.ManaPool().Add(core.Blue, 1)
	a.ManaPool().Add(core.Colorless, 3)
	g.SetStep(core.PrecombatMain)
	// A sorcery needs permission when its caster is not the active player.
	g.PutOnBattlefield(mage.NewEnchantment("Flash permission", "", mage.WithStaticAbility(mage.CastAsThoughHadFlash(mage.CardFilter{}))), a.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if strings.Join(order, ",") != "b,a,b,a" {
		t.Fatal("Juxtapose still uses caster-first choices", order)
	}
}

func TestRONAPNAPPermanentChoicesInvalid(t *testing.T) {
	group := `SimultaneousPermanentChoices([PlayerAction(You,ChooseAPermanent(IsCardtype(Artifact)))])`
	permanentScopeFactory(t, "Instant", `SpellActions(ActionList([ChooseAnAction([[`+group+`,TapPermanent(ThePermanentChosenByPlayerThisWay(You))],[GainLife(Integer(1))]])]))`)
	for _, body := range []string{
		`SimultaneousPermanentChoices`,
		`SimultaneousPermanentChoices([])`,
		`SimultaneousPermanentChoices([ChooseAPermanent(IsCardtype(Artifact))])`,
		`SimultaneousPermanentChoices([PlayerAction(You,DrawACard)])`,
		`SimultaneousPermanentChoices([PlayerAction(You,ChooseAPermanent())])`,
		`SimultaneousPermanentChoices([PlayerAction(You,ChooseAPermanent(IsCardtype(Artifact),You))])`,
		`SimultaneousPermanentChoices([PlayerAction(TheChosenPlayer,ChooseAPermanent(IsCardtype(Artifact)))])`,
		`SimultaneousPermanentChoices([PlayerAction(You,ChooseAPermanent(IsCardtype(Artifact)))],You)`,
		`TapPermanent(ThePermanentChosenByPlayerThisWay(You)),` + group,
		`ChooseAnAction([[` + group + `],[GainLife(Integer(1))]]),TapPermanent(ThePermanentChosenByPlayerThisWay(You))`,
		group + `,TapPermanent(TheChosenPermanent)`,
		`SimultaneousPermanentChoices([PlayerAction(You,ChooseAPermanent(IsCardtype(Artifact))),PlayerAction(ControllerOfPermanent(ThePermanentChosenByPlayerThisWay(You)),ChooseAPermanent(IsCardtype(Land)))])`,
	} {
		ns, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[],Rules:[SpellActions(ActionList([`+body+`]))])`), "invalid")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(ns[0]); err == nil {
			t.Fatal("invalid group accepted", body)
		}
	}
}

func TestRONAPNAPPermanentChoicesDelayAndCopies(t *testing.T) {
	for _, change := range []string{"none", "phase", "cycle", "return"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			subjectCard := mage.NewArtifact("Subject", "")
			subjectCard.SetOwner(b.PlayerID())
			subject := g.PutOnBattlefield(subjectCard, b.PlayerID())
			f := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),Targeted([TargetPlayer(AnyPlayer)],ActionList([SimultaneousPermanentChoices([PlayerAction(Ref_TargetPlayer,ChooseAPermanent(And([IsCardtype(Artifact),ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer))])))]),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([TapPermanent(ThePermanentChosenByPlayerThisWay(Ref_TargetPlayer))]))])))`)
			source := g.PutOnBattlefield(f(), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err == nil || g.StackSize() != 0 {
				t.Fatal("failed payment queued a choice")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
			g.ResolveStack()
			if subject.Tapped {
				t.Fatal("delay ran early")
			}
			clone := g.Clone()
			g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			switch change {
			case "phase":
				g.PhaseOut(subject.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(subject.ID()))
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(subject.ID()))
				b.RemoveFromHand(subject.ID())
				g.PutOnBattlefield(subjectCard, b.PlayerID())
			}
			for _, game := range []*mage.Game{g, clone} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				want := game == clone || change == "none" || change == "cycle"
				if game.FindPermanentIncludingPhased(subject.ID()).Tapped != want {
					t.Fatal("group delay lost exact visit")
				}
			}
		})
	}
}

func TestRONAPNAPPermanentChoicesWaitingSource(t *testing.T) {
	g, base, b := ronGame()
	var order []string
	a := &ronPermanentChoicePlayer{Player: base, choose: func(ps []*mage.Permanent) *mage.Permanent { order = append(order, "a"); return ps[len(ps)-1] }}
	g.SetPlayerAt(0, a)
	for range 2 {
		g.PutOnBattlefield(mage.NewCreature("Candidate", "", 1, 4), a.PlayerID())
	}
	f := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([SimultaneousPermanentChoices([PlayerAction(ControllerOfPermanent(ThisPermanent),ChooseAPermanent(IsCardtype(Creature)))]),TapPermanent(ThePermanentChosenByPlayerThisWay(You))]))]))`)
	source := g.PutOnBattlefield(f(), a.PlayerID())
	saved := g.CaptureDamageSource(source.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.BouncePermanentToHand(source)
	a.RemoveFromHand(source.ID())
	g.PutOnBattlefield(source.Card, b.PlayerID())
	for range 2 {
		sources := g.WaitingDelayedPermanentSources()
		if len(sources) != 1 || !sources[0].SameObject(saved) || len(order) != 0 {
			t.Fatal("waiting query lost source or made a choice")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if strings.Join(order, ",") != "a" || len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("delay followed returned source controller", order)
	}
}
