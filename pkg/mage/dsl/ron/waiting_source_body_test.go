package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingSourceBodies(t *testing.T) {
	for _, tc := range []struct {
		set, name         string
		activations, cost int
		color             core.Color
	}{
		{"LEB", "Dragon Whelp", 4, 1, core.Red}, {"PROMOS", "Nalathni Dragon", 4, 1, core.Red},
		{"ATQ", "Rocket Launcher", 1, 2, core.Colorless}, {"ATQ", "Rakalite", 1, 2, core.Colorless},
	} {
		for _, change := range []string{"live", "returned", "phased", "cycle", "departed"} {
			t.Run(tc.name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				g.SetTurn(2)
				g.SetStep(core.PrecombatMain)
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				p := g.PutOnBattlefield(card, a.PlayerID())
				original := g.CaptureDamageSource(p.ID())
				g.SetTurn(3)
				var targets []uuid.UUID
				if tc.set == "ATQ" {
					targets = []uuid.UUID{b.PlayerID()}
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, targets); err == nil || g.StackSize() != 0 || len(g.DelayedTriggers()) != 0 {
					t.Fatal("unpaid activation changed state")
				}
				a.ManaPool().Add(tc.color, tc.activations*tc.cost)
				for i := 0; i < tc.activations; i++ {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, targets); err != nil {
						t.Fatal(err)
					}
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("source entered waiting category early")
					}
					g.ResolveStack()
				}
				switch change {
				case "departed", "returned":
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					if change == "returned" {
						a.RemoveFromHand(card.ID())
						g.PutOnBattlefield(card, b.PlayerID())
					}
				case "phased", "cycle":
					ids := g.PhaseOut(p.ID())
					if change == "cycle" {
						g.PhaseIn(ids)
					}
				}
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(original) {
					t.Fatal("real RON delay lost source body reference")
				}
				result := mage.BindDamageSource("source body choice")
				effect := mage.Sequence(mage.ChooseDamageSourceFromWaitingDelayedBindings(mage.You, mage.DamageSourceFilter{}, result), mage.DealDamageFrom(mage.FromDamageSource(result), mage.You, mage.Fixed(1)))
				if err := mage.ApplyEffect(g, effect, a.PlayerID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if a.Life() != 19 {
					t.Fatal("shared choice did not consume reference")
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{g, clone} {
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					if len(game.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("detected delay still waiting")
					}
					game.PutTriggersOnStack()
					game.ResolveStack()
					survives := change == "returned" || change == "phased"
					if (game.FindPermanentIncludingPhased(p.ID()) != nil) != survives {
						t.Fatal("native delayed action identity")
					}
					if tc.name == "Rakalite" && change == "live" && len(game.GetPlayer(a.PlayerID()).Hand()) != 1 {
						t.Fatal("return did not use owner")
					}
				}
			})
		}
	}
}

func TestRONWaitingSourceBodyInvalid(t *testing.T) {
	for _, body := range []string{
		"DestroyPermanent(ThisPermanent())", "SacrificePermanent(Trigger_ThatPermanent)",
		"PutPermanentIntoItsOwnersHand(HostPermanent)", "DestroyPermanent(ThisPermanent,Extra)",
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid delayed source declaration accepted", body)
		}
	}
}
