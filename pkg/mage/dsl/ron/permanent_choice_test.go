package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronPermanentChoicePlayer struct {
	mage.Player
	choose func([]*mage.Permanent) *mage.Permanent
	calls  int
}

func (p *ronPermanentChoicePlayer) ChoosePermanent(candidates []*mage.Permanent, _ string, _ mage.GameReader) *mage.Permanent {
	p.calls++
	return p.choose(candidates)
}

func TestRONPermanentChoiceGameplay(t *testing.T) {
	for _, change := range []string{"none", "phase", "return", "late-entry", "invalid", "nil"} {
		t.Run(change, func(t *testing.T) {
			g, base, b := ronGame()
			a := &ronPermanentChoicePlayer{Player: base}
			g.SetPlayerAt(0, a)
			low := g.PutOnBattlefield(mage.NewCreature("Low", "{1}", 1, 1), b.PlayerID())
			first := g.PutOnBattlefield(mage.NewCreature("First", "{3}", 1, 1, mage.WithKeyword(core.Shroud)), b.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "{3}", 1, 1), b.PlayerID())
			a.choose = func(candidates []*mage.Permanent) *mage.Permanent {
				if change == "invalid" {
					return low
				}
				if change == "nil" {
					return nil
				}
				return candidates[len(candidates)-1]
			}
			factory := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([ChooseAPermanent(APermanentWithTheHighestManaValue(IsCardtype(Creature))),TapPermanent(TheChosenPermanent)]))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if a.calls != 0 {
				t.Fatal("choice occurred at announcement")
			}
			pending := g.Clone()
			clonePlayer := &ronPermanentChoicePlayer{Player: pending.GetPlayer(a.PlayerID()), choose: func(candidates []*mage.Permanent) *mage.Permanent { return candidates[0] }}
			pending.SetPlayerAt(0, clonePlayer)
			want := second.ID()
			switch change {
			case "phase":
				g.PhaseOut(second.ID())
				want = first.ID()
			case "return":
				second.Card.SetOwner(b.PlayerID())
				g.BouncePermanentToHand(g.FindPermanent(second.ID()))
				b.RemoveFromHand(second.ID())
				g.PutOnBattlefield(second.Card, b.PlayerID())
			case "late-entry":
				want = g.PutOnBattlefield(mage.NewCreature("Late", "{4}", 1, 1), b.PlayerID()).ID()
			case "invalid", "nil":
				want = first.ID()
			}
			g.ResolveStack()
			if !g.FindPermanent(want).Tapped || g.FindPermanent(low.ID()).Tapped {
				t.Fatal("incorrect resolution choice")
			}
			pending.ResolveStack()
			if !pending.FindPermanent(first.ID()).Tapped || pending.FindPermanent(second.ID()).Tapped {
				t.Fatal("pending clone shared a choice")
			}
			if change != "phase" && change != "invalid" && change != "nil" && g.FindPermanent(first.ID()).Tapped {
				t.Fatal("clone changed original")
			}
			if factory().Abilities()[0] == card.Abilities()[0] {
				t.Fatal("factory shared action state")
			}
		})
	}
}

func TestRONPermanentChoiceSequence(t *testing.T) {
	g, a, b := ronGame()
	first := g.PutOnBattlefield(mage.NewArtifact("Artifact", ""), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewEnchantment("Enchantment", ""), b.PlayerID())
	card := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([ChooseAPermanent(IsCardtype(Artifact)),TapPermanent(TheChosenPermanent),ChooseAPermanent(IsCardtype(Enchantment)),TapPermanent(TheChosenPermanent),ChooseAPermanent(IsCardtype(Land)),DestroyPermanent(TheChosenPermanent),GainLife(Integer(2))]))`)()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) == nil || !g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped || a.Life() != 22 {
		t.Fatal("separate choices or empty result failed")
	}
}

func TestRONPermanentChoiceExplicitPlayer(t *testing.T) {
	g, base, other := ronGame()
	a := &ronPermanentChoicePlayer{Player: base, choose: func(p []*mage.Permanent) *mage.Permanent { return p[0] }}
	b := &ronPermanentChoicePlayer{Player: other, choose: func(p []*mage.Permanent) *mage.Permanent { return p[1] }}
	g.SetPlayerAt(0, a)
	g.SetPlayerAt(1, b)
	first := g.PutOnBattlefield(mage.NewArtifact("First", ""), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewArtifact("Second", ""), b.PlayerID())
	card := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([ChooseAPermanent(IsCardtype(Artifact)),PlayerAction(Ref_TargetPlayer,ChooseAPermanent(IsCardtype(Artifact))),TapPermanent(TheChosenPermanent)])))`)()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.calls != 1 || b.calls != 1 || !g.FindPermanent(first.ID()).Tapped || g.FindPermanent(second.ID()).Tapped {
		t.Fatal("nested player choice escaped its scope")
	}
}

func TestRONPermanentChoiceActivation(t *testing.T) {
	g, base, b := ronGame()
	a := &ronPermanentChoicePlayer{Player: base, choose: func(candidates []*mage.Permanent) *mage.Permanent { return candidates[len(candidates)-1] }}
	g.SetPlayerAt(0, a)
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 1), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 1), b.PlayerID())
	source := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),ActionList([ChooseAPermanent(IsCardtype(Creature)),TapPermanent(TheChosenPermanent)]))`)(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || a.calls != 0 || g.StackSize() != 0 {
		t.Fatal("failed payment made a choice or queued an ability")
	}
	a.ManaPool().Add(core.Colorless, 2)
	for range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	if a.calls != 0 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("activation did not defer choice until resolution")
	}
	g.ResolveStack()
	if a.calls != 2 || !g.FindPermanent(second.ID()).Tapped || g.FindPermanent(first.ID()).Tapped {
		t.Fatal("pending activations shared a result or lost their body")
	}
}

func TestRONPermanentChoiceUnblocked(t *testing.T) {
	for _, blockersDeclared := range []bool{false, true} {
		g, a, b := ronGame()
		cards := []mage.Card{mage.NewCreature("Unblocked", "", 1, 4), mage.NewCreature("Blocked", "", 1, 4)}
		for _, card := range cards {
			card.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
		}
		g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{cards[0].ID(), cards[1].ID()})
		if blockersDeclared {
			blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 1, 4), a.PlayerID())
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: cards[1].ID()}})
			g.RemoveFromCombat(blocker.ID())
		}
		card := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([ChooseAPermanent(And([IsUnblocked,IsCardtype(Creature)])),PutPermanentIntoItsOwnersHand(TheChosenPermanent)]))`)()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if (g.FindPermanent(cards[0].ID()) == nil) != blockersDeclared || g.FindPermanent(cards[1].ID()) == nil {
			t.Fatal("unblocked selection ignored blocker timing or retained blocked status")
		}
	}
}

func TestRONPermanentChoiceDelayedBinding(t *testing.T) {
	for _, returned := range []bool{false, true} {
		g, a, b := ronGame()
		chosenCard := mage.NewCreature("Chosen", "{2}", 1, 4)
		chosenCard.SetOwner(a.PlayerID())
		chosen := g.PutOnBattlefield(chosenCard, a.PlayerID())
		card := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([ChooseAPermanent(IsCardtype(Creature)),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([TapPermanent(TheChosenPermanent)]))]))`)()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if g.FindPermanent(chosen.ID()).Tapped {
			t.Fatal("delayed instruction executed at installation")
		}
		clone := g.Clone()
		if returned {
			g.BouncePermanentToHand(chosen)
			a.RemoveFromHand(chosen.ID())
			g.PutOnBattlefield(chosenCard, b.PlayerID())
		} else {
			g.PhaseIn(g.PhaseOut(chosen.ID()))
		}
		for _, game := range []*mage.Game{g, clone} {
			game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			game.PutTriggersOnStack()
			game.ResolveStack()
			want := game == clone || !returned
			if game.FindPermanent(chosen.ID()).Tapped != want {
				t.Fatal("delayed capture lost incarnation identity or clone isolation")
			}
		}
	}
}

func TestRONPermanentChoiceInvalid(t *testing.T) {
	for _, body := range []string{
		`ChooseAPermanent`, `ChooseAPermanent()`, `ChooseAPermanent(AnyPermanent,AnyPermanent)`,
		`ChooseAPermanent(APermanentWithTheHighestManaValue)`, `ChooseAPermanent(APermanentWithTheHighestManaValue())`,
		`ChooseAPermanent(And([]))`, `ChooseAPermanent(TheChosenPermanent)`,
		`TapPermanent(TheChosenPermanent),ChooseAPermanent(AnyPermanent)`,
		`ChooseAPermanent(AnyPermanent),TapPermanent(TheChosenPermanent())`,
		`MayAction(ChooseAPermanent(AnyPermanent)),TapPermanent(TheChosenPermanent)`,
		`PlayerAction(Ref_TargetPlayer,ChooseAPermanent(AnyPermanent)),TapPermanent(TheChosenPermanent)`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	data := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([ChooseAPermanent(AnyPermanent)])),Activated(PayMana([]),ActionList([TapPermanent(TheChosenPermanent)]))])`
	nodes, err := Parse(strings.NewReader(data), "scope.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("choice escaped its ability")
	}
}

func TestRONPermanentChoiceRecordsAndInventory(t *testing.T) {
	count := 0
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI", "LEG"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), set)
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range nodes {
			name := n.Fields["Name"].String
			if name != "Forcefield" && name != "Juxtapose" && name != "Takklemaggot" {
				continue
			}
			count++
			_, err := Compile(n)
			if name == "Forcefield" {
				if err != nil {
					t.Fatalf("Forcefield: %v", err)
				}
				continue
			}
			blocker := ""
			if name == "Juxtapose" {
				if err != nil {
					t.Fatalf("Juxtapose: %v", err)
				}
				actions := n.Fields["Rules"].Args[0].Args[0].Args[1].Args[0].Args
				if len(actions) != 4 {
					t.Fatal("Juxtapose must choose and exchange creatures, then artifacts")
				}
				for i, category := range []string{"Creature", "Artifact"} {
					group := actions[2*i]
					if group.Name != "SimultaneousPermanentChoices" || len(group.Args) != 1 || len(group.Args[0].Args) != 2 {
						t.Fatal("Juxtapose must group both players' choices")
					}
					choices := []Node{group.Args[0].Args[0].Args[1], group.Args[0].Args[1].Args[1]}
					for _, choice := range choices {
						filter := choice.Args[0].Args[0].Args[0].Args[0]
						if filter.Name != "IsCardtype" || filter.Args[0].Name != category {
							t.Fatal("Juxtapose has the wrong exchange category")
						}
					}
					if actions[2*i+1].Name != "ExchangeControl" {
						t.Fatal("Juxtapose omitted an exchange")
					}
				}
				continue
			}
			if name == "Takklemaggot" {
				blocker = "unsupported lasting ability modification (LosesAbility)"
			}
			if err == nil || !strings.Contains(err.Error(), blocker) {
				t.Fatalf("%s: expected %s, got %v", name, blocker, err)
			}
		}
	}
	if count != 7 {
		t.Fatalf("records=%d", count)
	}
	src := `Card(Name:"Choice",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([ChooseAPermanent(APermanentWithTheHighestManaValue(IsCardtype(Creature))),TapPermanent(TheChosenPermanent)]))])`
	report, err := InspectFS(fstest.MapFS{"choice.ron": &fstest.MapFile{Data: []byte(src)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("choice inventory rejected", err)
	}
	for _, signature := range report.Signatures {
		if signature.Name == "IsCardtype" && signature.Role != "permanent_filter" {
			t.Fatal("wrong choice filter role", signature.Role)
		}
		if signature.Name == "TheChosenPermanent" && signature.Role != "permanent_reference" {
			t.Fatal("wrong choice reference role", signature.Role)
		}
	}
}
