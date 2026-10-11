package ron

import (
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func juxtaposeFactory(t *testing.T) mage.CardFactory {
	t.Helper()
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEG")
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range nodes {
		if n.Fields["Name"].String == "Juxtapose" {
			f, err := Compile(n)
			if err != nil {
				t.Fatal(err)
			}
			return f
		}
	}
	t.Fatal("missing Juxtapose")
	return nil
}

func TestRONPlayerPermanentChoiceJuxtapose(t *testing.T) {
	for _, scenario := range []string{"normal", "empty-creatures", "empty-artifacts", "blocked", "phase", "self", "tie"} {
		t.Run(scenario, func(t *testing.T) {
			f := juxtaposeFactory(t)
			g, base, other := ronGame()
			order := []string{}
			a := &ronPermanentChoicePlayer{Player: base, choose: func(ps []*mage.Permanent) *mage.Permanent { order = append(order, "a"); return ps[len(ps)-1] }}
			b := &ronPermanentChoicePlayer{Player: other, choose: func(ps []*mage.Permanent) *mage.Permanent { order = append(order, "b"); return ps[len(ps)-1] }}
			g.SetPlayerAt(0, a)
			g.SetPlayerAt(1, b)
			var creatures, artifacts []*mage.Permanent
			for _, p := range []mage.Player{a, b} {
				if scenario != "empty-creatures" {
					g.PutOnBattlefield(mage.NewCreature("Low", "{1}", 1, 4), p.PlayerID())
					creatures = append(creatures, g.PutOnBattlefield(mage.NewCreature("High", "{3}", 1, 4, mage.WithKeyword(core.Shroud)), p.PlayerID()))
					if scenario == "tie" {
						creatures[len(creatures)-1] = g.PutOnBattlefield(mage.NewCreature("Tie", "{3}", 1, 4), p.PlayerID())
					}
				}
				if scenario != "empty-artifacts" {
					artifacts = append(artifacts, g.PutOnBattlefield(mage.NewArtifact("Artifact", "{2}"), p.PlayerID()))
				}
			}
			if scenario == "blocked" {
				g.MutablePermanent(creatures[0].ID()).GrantBaseAttr(core.AttrCantChangeControl)
			}
			if scenario == "phase" {
				g.PhaseOut(creatures[1].ID())
			}
			card := f()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 1)
			a.ManaPool().Add(core.Colorless, 3)
			target := b.PlayerID()
			if scenario == "self" {
				target = a.PlayerID()
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
				t.Fatal(err)
			}
			if len(order) != 0 {
				t.Fatal("choice before resolution")
			}
			clone := g.Clone()
			g.ResolveStack()
			for i, p := range artifacts {
				want := b.PlayerID()
				if i == 1 {
					want = a.PlayerID()
				}
				if scenario == "self" {
					want = a.PlayerID()
					if i == 1 {
						want = b.PlayerID()
					}
				}
				if g.FindPermanent(p.ID()).ControllerID() != want {
					t.Fatal("artifact round lost player bindings")
				}
			}
			if scenario != "empty-creatures" && scenario != "phase" {
				for i, p := range creatures {
					want := b.PlayerID()
					if i == 1 {
						want = a.PlayerID()
					}
					if scenario == "blocked" || scenario == "self" {
						want = a.PlayerID()
						if i == 1 {
							want = b.PlayerID()
						}
					}
					if g.FindPermanent(p.ID()).ControllerID() != want {
						t.Fatal("creature exchange incorrect")
					}
				}
			}
			if scenario == "tie" && strings.Join(order, ",") != "a,b" {
				t.Fatal("choices did not follow instruction order", order)
			}
			clone.ResolveStack()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			if f().Abilities()[0] == card.Abilities()[0] {
				t.Fatal("shared factory declarations")
			}
		})
	}
}

func TestRONPlayerPermanentChoiceInvalid(t *testing.T) {
	for _, body := range []string{
		`TapPermanent(ThePermanentChosenByPlayerThisWay(You)),ChooseAPermanent(AnyPermanent)`,
		`ChooseAPermanent(AnyPermanent),TapPermanent(ThePermanentChosenByPlayerThisWay)`,
		`ChooseAPermanent(AnyPermanent),TapPermanent(ThePermanentChosenByPlayerThisWay())`,
		`ChooseAPermanent(AnyPermanent),TapPermanent(ThePermanentChosenByPlayerThisWay(You,You))`,
		`ChooseAPermanent(AnyPermanent),TapPermanent(ThePermanentChosenByPlayerThisWay(Ref_TargetPermanent))`,
		`MayAction(ChooseAPermanent(AnyPermanent)),TapPermanent(ThePermanentChosenByPlayerThisWay(You))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", body)
		}
	}
}

func TestRONPlayerPermanentChoiceDelay(t *testing.T) {
	for _, change := range []string{"none", "phase", "cycle", "return"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			subjectCard := mage.NewArtifact("Chosen", "")
			subjectCard.SetOwner(b.PlayerID())
			subject := g.PutOnBattlefield(subjectCard, b.PlayerID())
			source := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,ChooseAPermanent(IsCardtype(Artifact))),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([TapPermanent(ThePermanentChosenByPlayerThisWay(Ref_TargetPlayer))]))])))`)(), a.PlayerID())
			targets := []uuid.UUID{b.PlayerID()}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err == nil || g.StackSize() != 0 {
				t.Fatal("failed payment queued choice")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(subject.ID()).Tapped {
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
					t.Fatal("delay lost exact visit or clone frame")
				}
			}
		})
	}
}

func TestRONPlayerPermanentChoiceAliasAndCopies(t *testing.T) {
	for _, self := range []bool{false, true} {
		t.Run(map[bool]string{false: "opponent", true: "self"}[self], func(t *testing.T) {
			g, a, b := ronGame()
			first := g.PutOnBattlefield(mage.NewArtifact("First", ""), a.PlayerID())
			second := g.PutOnBattlefield(mage.NewEnchantment("Second", ""), b.PlayerID())
			card := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([ChooseAPermanent(IsCardtype(Artifact)),PlayerAction(Ref_TargetPlayer,ChooseAPermanent(IsCardtype(Enchantment))),TapPermanent(ThePermanentChosenByPlayerThisWay(You)),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([TapPermanent(ThePermanentChosenByPlayerThisWay(Ref_TargetPlayer))]))])))`)()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			target := b.PlayerID()
			if self {
				target = a.PlayerID()
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
				t.Fatal(err)
			}
			g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
			g.ResolveStack()
			if g.FindPermanent(first.ID()).Tapped == self || g.FindPermanent(second.ID()).Tapped != self {
				t.Fatal("runtime player alias selected the wrong choice")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if !g.FindPermanent(second.ID()).Tapped {
				t.Fatal("copied delayed choice lost its player result")
			}
		})
	}
}

func TestRONPlayerPermanentChoiceMixedTypes(t *testing.T) {
	g, a, b := ronGame()
	first := g.PutOnBattlefield(mage.NewCreature("First", "{5}", 2, 4, mage.WithCardType(core.TypeArtifact)), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "{3}", 2, 4, mage.WithCardType(core.TypeArtifact)), b.PlayerID())
	other := g.PutOnBattlefield(mage.NewArtifact("Other", "{1}"), a.PlayerID())
	card := juxtaposeFactory(t)()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 1)
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(first.ID()).ControllerID() != a.PlayerID() || g.FindPermanent(second.ID()).ControllerID() != b.PlayerID() || g.FindPermanent(other.ID()).ControllerID() != a.PlayerID() {
		t.Fatal("artifact round did not read control after creature exchange")
	}
}
