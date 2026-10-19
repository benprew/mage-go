package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONResolutionTokenSetStangg(t *testing.T) {
	for _, action := range []string{"source-leaves", "token-leaves", "source-phased", "token-phased", "source-returns", "orphan", "copied", "token-control", "source-control", "abilities-removed", "source-phased-before", "copied-same"} {
		t.Run(action, func(t *testing.T) {
			factory, err := Compile(hostUntapRecord(t, "LEG", "Stangg"))
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			a.ManaPool().Add(core.Colorless, 4)
			a.ManaPool().Add(core.Red, 1)
			a.ManaPool().Add(core.Green, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveTopOfStack()
			g.PutTriggersOnStack()
			source := g.FindPermanent(card.ID())
			if source == nil || g.StackSize() != 1 {
				t.Fatal("entry trigger missing")
			}
			if action == "source-phased-before" {
				g.PhaseOut(source.ID())
			}
			if action == "copied-same" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			if action == "orphan" {
				g.BouncePermanentToHand(source)
			}
			if action == "copied" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.ResolveStack()
				tokens := game.FilterBattlefield(mage.IsToken)
				if action == "copied-same" {
					if game.FindPermanent(source.ID()) != nil || len(tokens) != 0 {
						t.Fatal("legend departure did not run both delayed links")
					}
					continue
				}
				want := 1
				if action == "copied" {
					want = 2
				}
				if len(tokens) != want {
					t.Fatalf("creation: %d", len(tokens))
				}
				switch action {
				case "source-phased-before":
					game.PhaseIn([]uuid.UUID{source.ID()})
					game.BouncePermanentToHand(game.FindPermanent(source.ID()))
				case "source-leaves":
					game.BouncePermanentToHand(game.FindPermanent(source.ID()))
				case "token-leaves":
					game.ExilePermanent(tokens[0])
				case "source-phased":
					game.PhaseOut(source.ID())
					game.ExilePermanent(tokens[0])
				case "token-phased":
					phase := game.PhaseOut(tokens[0].ID())
					game.ResolveStack()
					if game.FindPermanent(source.ID()) == nil {
						t.Fatal("phasing triggered sacrifice")
					}
					game.PhaseIn(phase)
					game.ExilePermanent(game.FindPermanent(tokens[0].ID()))
				case "source-returns":
					physical := game.FindPermanent(source.ID()).Card
					game.BouncePermanentToHand(game.FindPermanent(source.ID()))
					game.GetPlayer(a.PlayerID()).RemoveFromHand(card.ID())
					game.PutOnBattlefield(physical, b.PlayerID())
				case "orphan":
					game.PutOnBattlefield(factory(), b.PlayerID())
				case "token-control":
					game.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: tokens[0].ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					game.ExilePermanent(tokens[0])
				case "source-control":
					controllerSource := game.PutOnBattlefield(mage.NewArtifact("Control source", ""), b.PlayerID())
					game.AddControlEffect(mage.ControlEffectSpec{SourceID: controllerSource.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					game.ExilePermanent(tokens[0])
				case "abilities-removed":
					game.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
					game.ExilePermanent(tokens[0])
				case "copied":
					var other *mage.Permanent
					for _, p := range tokens {
						if p.ControllerID() == b.PlayerID() {
							other = p
						}
					}
					game.ExilePermanent(other)
					game.ResolveStack()
					if game.FindPermanent(source.ID()) == nil || len(game.FilterBattlefield(mage.IsToken)) != 1 {
						t.Fatal("copied controller sacrificed another player source or mixed results")
					}
					game.ExilePermanent(game.FilterBattlefield(mage.IsToken)[0])
				}
				game.ResolveStack()
				switch action {
				case "source-leaves", "token-leaves", "token-phased", "copied", "token-control", "abilities-removed", "source-phased-before":
					if game.FindPermanent(source.ID()) != nil || len(game.FilterBattlefield(mage.IsToken)) != 0 {
						t.Fatal("linked cleanup failed")
					}
				case "source-control":
					if game.FindPermanent(source.ID()) == nil || len(game.FilterBattlefield(mage.IsToken)) != 0 {
						t.Fatal("former controller sacrificed another player permanent")
					}
				case "source-phased":
					if game.FindPermanentIncludingPhased(source.ID()) == nil {
						t.Fatal("phased source sacrificed")
					}
				case "source-returns":
					if returned := game.FindPermanent(source.ID()); returned == nil || returned.ControllerID() != b.PlayerID() || len(game.FilterBattlefield(mage.IsToken)) != 1 {
						t.Fatal("returned source mixed token groups")
					}
				case "orphan":
					if len(game.FilterBattlefield(mage.IsToken)) != 2 {
						t.Fatal("absent source retroactively linked")
					}
				}
			}
		})
	}
}

func TestRONResolutionTokenSetInvalid(t *testing.T) {
	create := `CreateTokens([` + saprolingRON + `],[])`
	read := `CreateFutureTrigger(WhenAPermanentLeavesTheBattlefield(TheTokensCreatedThisWay),ActionList([GainLife(Integer(1))]))`
	for _, body := range []string{read, read + `,` + create, `If(PermanentPassesFilter(ThisPermanent,IsTapped),[` + create + `]),` + read, create + `,` + strings.ReplaceAll(read, "TheTokensCreatedThisWay", "TheTokensCreatedThisWay()"), create + `,` + strings.ReplaceAll(read, "TheTokensCreatedThisWay", "SinglePermanent(TheTokensCreatedThisWay)"), create + `,` + strings.ReplaceAll(read, "GainLife(Integer(1))", "SacrificePermanent(Trigger_ThatPermanent)")} {
		n := nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatal("accepted invalid result scope", body)
		}
	}
}

func TestRONResolutionTokenSetScopes(t *testing.T) {
	create := `CreateTokens([` + saprolingRON + `],[])`
	delay := `CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([Exile([EachPermanent(TheTokensCreatedThisWay)],[])]))`
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([`+create+`,`+delay+`,`+create+`]))`)
	g, a, _ := ronGame()
	first, second := g.PutOnBattlefield(factory(), a.PlayerID()), g.PutOnBattlefield(factory(), a.PlayerID())
	for _, p := range []*mage.Permanent{first, second} {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	if len(g.FilterBattlefield(mage.IsToken)) != 4 {
		t.Fatal("independent producers failed")
	}
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.ResolveStack()
		if len(game.FilterBattlefield(mage.IsToken)) != 2 {
			t.Fatal("delayed set included later or independent production")
		}
	}
}
