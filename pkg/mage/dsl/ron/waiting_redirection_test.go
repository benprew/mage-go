package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingRedirectionReferences(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEA", "Veteran Bodyguard"}, {"ATQ", "Martyrs of Korlis"}} {
		for _, state := range []string{"live", "tap", "phase", "phase-cycle", "depart", "return", "remove-abilities", "control", "damage", "cleanup"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				subject := g.PutOnBattlefield(card, a.PlayerID())
				saved := g.CaptureDamageSource(subject.ID())
				if got := g.WaitingRedirectionPermanentSources(); len(got) != 1 || !got[0].SameObject(saved) {
					t.Fatal("missing RON destination")
				}
				baseline := g.Clone()
				want := 1
				switch state {
				case "tap":
					g.TapPermanent(subject)
					want = 0
				case "phase":
					g.PhaseOut(subject.ID())
					want = 0
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(subject.ID()))
				case "depart", "return":
					g.RemoveFromBattlefield(subject)
					want = 0
					if state == "return" {
						returned := g.PutOnBattlefield(card, b.PlayerID())
						saved = g.CaptureDamageSource(returned.ID())
						want = 1
					}
				case "remove-abilities":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), subject.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					want = 0
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: subject.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "damage":
					source := g.PutOnBattlefield(mage.NewCreature("Artifact source", "", 1, 20, mage.WithKeyword(core.AttrIsArtifact)), b.PlayerID())
					source.RevokeBaseAttr(core.AttrSummonSick)
					g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{source.ID()})
					g.ExecuteBlockers(nil)
					g.DealDamageToPlayer(a, 2, source.ID())
					if a.Life() != 20 || g.FindPermanent(subject.ID()).Damage != 2 {
						t.Fatal("redirection failed")
					}
				case "cleanup":
					g.GetEffects().ClearReplacementsEndOfTurn()
				}
				for _, game := range []*mage.Game{g, g.Clone()} {
					got := game.WaitingRedirectionPermanentSources()
					if len(got) != want {
						t.Fatalf("subjects=%d want=%d", len(got), want)
					}
					if want == 0 {
						continue
					}
					if !got[0].SameObject(saved) {
						t.Fatal("wrong incarnation")
					}
					binding := mage.BindDamageSource("destination")
					effect := mage.Sequence(mage.ChooseDamageSourceFromWaitingRedirectionReferences(mage.You, mage.DamageSourceHasColor(core.White), binding), mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(1)))
					if err := mage.ApplyEffect(game, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					if game.GetPlayer(a.PlayerID()).Life() != 19 {
						t.Fatal("chosen destination could not deal damage")
					}
				}
				if len(baseline.WaitingRedirectionPermanentSources()) != 1 {
					t.Fatal("mutation changed clone")
				}
			})
		}
	}
}
