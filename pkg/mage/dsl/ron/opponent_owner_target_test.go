package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func opponentOwnerFragment(t *testing.T) mage.CardFactory {
	t.Helper()
	record := staticBaseRecord(t, "ATQ", "Bronze Tablet")
	rule := record.Fields["Rules"].Args[2]
	nodes, err := Parse(strings.NewReader(`ActionList([TapPermanent(Ref_TargetPermanent),GainLife(Integer(1))])`), "owner-body.ron")
	if err != nil {
		t.Fatal(err)
	}
	rule.Args[1].Args[1] = nodes[0]
	rules := record.Fields["Rules"]
	rules.Args = []Node{rule}
	record.Fields["Rules"] = rules
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Bronze Tablet fragment", factory()); err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONOpponentOwnerTarget(t *testing.T) {
	for _, state := range []string{"normal", "control", "owner", "return", "phase", "cycle", "shroud", "source-return", "copy", "copy-opponent"} {
		t.Run(state, func(t *testing.T) {
			factory := opponentOwnerFragment(t)
			_, a, b := ronGame()
			g, err := mage.NewGameWithAnte(a, b, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			victimCard := mage.NewArtifact("Opponent owned", "")
			victimCard.SetOwner(b.PlayerID())
			victim := g.PutOnBattlefield(victimCard, a.PlayerID())
			ownCard := mage.NewArtifact("Own stolen", "")
			ownCard.SetOwner(a.PlayerID())
			own := g.PutOnBattlefield(ownCard, b.PlayerID())
			tokenCard := mage.NewArtifact("Opponent token", "")
			tokenCard.SetOwner(b.PlayerID())
			token := g.PutOnBattlefield(tokenCard, b.PlayerID())
			g.MutablePermanent(token.ID()).IsToken = true
			activate := func(id uuid.UUID) error {
				return g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{id})
			}
			if err := activate(victim.ID()); err == nil {
				t.Fatal("unpaid activation succeeded")
			}
			if g.StackSize() != 0 || source.Tapped {
				t.Fatal("failed payment changed state")
			}
			a.ManaPool().Add(core.Colorless, 4)
			for _, id := range []uuid.UUID{own.ID(), token.ID()} {
				if err := activate(id); err == nil {
					t.Fatal("invalid owner or token target accepted")
				}
				if a.ManaPool().TotalMana() != 4 || source.Tapped || g.StackSize() != 0 {
					t.Fatal("invalid target spent cost")
				}
			}
			if err := activate(victim.ID()); err != nil {
				t.Fatal(err)
			}
			if !source.Tapped || a.ManaPool().TotalMana() != 0 || victim.Tapped || a.Life() != 20 {
				t.Fatal("activation timing or cost")
			}
			legal := true
			switch state {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: own.ID(), TargetID: victim.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "owner":
				if err := g.ChangeOwner(victim.ID(), a.PlayerID()); err != nil {
					t.Fatal(err)
				}
				legal = false
			case "return":
				g.RemoveFromBattlefield(victim)
				g.PutOnBattlefield(victimCard, a.PlayerID())
				legal = false
			case "phase":
				g.PhaseOut(victim.ID())
				legal = false
			case "cycle":
				g.PhaseIn(g.PhaseOut(victim.ID()))
			case "shroud":
				g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
				legal = false
			case "source-return":
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(card, b.PlayerID())
			case "copy-opponent":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			if clone.FindPermanentIncludingPhased(victim.ID()).Tapped || clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("resolution changed clone")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				want := 20
				if legal {
					want++
				}
				if state == "copy" {
					want++
				}
				if game.GetPlayer(a.PlayerID()).Life() != want || game.GetPlayer(b.PlayerID()).Life() != 20 || game.FindPermanentIncludingPhased(victim.ID()).Tapped != legal {
					t.Fatal("owner legality or exact visit changed")
				}
			}
		})
	}
}

func TestRONOpponentOwnerTargetInvalid(t *testing.T) {
	for _, filter := range []string{`OwnedByAPlayer`, `OwnedByAPlayer()`, `OwnedByAPlayer(Opponent())`, `OwnedByAPlayer(Opponent,Opponent)`, `OwnedByAPlayer(You)`, `OwnedByAPlayer(SinglePlayer(Opponent))`, `OwnedByAPlayer(SinglePlayer(Ref_TargetPlayer))`, `Or([OwnedByAPlayer(Opponent),IsCardtype(Artifact)])`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),Targeted([TargetPermanent(` + filter + `)],ActionList([TapPermanent(Ref_TargetPermanent)])))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-owner-target.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", filter)
		}
	}
}

func TestRONOpponentOwnerTargetRemainingGap(t *testing.T) {
	_, err := Compile(staticBaseRecord(t, "ATQ", "Bronze Tablet"))
	if err == nil || (!strings.Contains(err.Error(), "unsupported two-branch payment cost (PayLife)") &&
		!strings.Contains(err.Error(), "unbound or incompatible player reference (OwnerOfPermanent)")) {
		t.Fatalf("next gap: %v", err)
	}
	t.Log(err)
}

func TestRONOpponentOwnerTargetInstances(t *testing.T) {
	factory := opponentOwnerFragment(t)
	g, a, b := ronGame()
	for _, pair := range [][2]mage.Player{{a, b}, {b, a}} {
		actor, opponent := pair[0], pair[1]
		card := factory()
		card.SetOwner(actor.PlayerID())
		source := g.PutOnBattlefield(card, actor.PlayerID())
		victimCard := mage.NewArtifact("Separate target", "")
		victimCard.SetOwner(opponent.PlayerID())
		victim := g.PutOnBattlefield(victimCard, actor.PlayerID())
		actor.ManaPool().Add(core.Colorless, 4)
		if err := g.ActivateAbilityByIndex(actor.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
			t.Fatal(err)
		}
	}
	clone := g.Clone()
	g.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("instance resolution changed clone")
	}
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		if game.GetPlayer(a.PlayerID()).Life() != 21 || game.GetPlayer(b.PlayerID()).Life() != 21 {
			t.Fatal("factory bindings crossed")
		}
	}
}
