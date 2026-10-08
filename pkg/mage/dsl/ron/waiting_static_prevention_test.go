package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingStaticPreventionReferences(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		aura      bool
	}{
		{"LEG", "Demonic Torment", true}, {"LEG", "Gaseous Form", true},
		{"LEG", "Enchanted Being", false}, {"ATQ", "Argothian Treefolk", false},
		{"LEG", "Wall of Vapor", false},
		{"ATQ", "Argothian Pixies", false}, {"ARN", "Desert Nomads", false},
		{"DRK", "Uncle Istvan", false},
	} {
		for _, state := range []string{"live", "reattach", "source-phase", "subject-phase", "phase-cycle", "depart", "return", "host-return", "remove-source", "remove-host", "control", "damage", "cleanup"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				hostCard := mage.NewCreature("Host", "{R}", 1, 20)
				host := g.PutOnBattlefield(hostCard, b.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				subject := source
				if tc.aura {
					g.Attach(source.ID(), host.ID())
					subject = host
				}
				saved := g.CaptureDamageSource(subject.ID())
				if got := g.WaitingPreventionPermanentSources(); len(got) != 1 || !got[0].SameObject(saved) {
					t.Fatalf("initial subjects=%d want exact subject", len(got))
				}
				baseline := g.Clone()
				want := 1
				switch state {
				case "reattach":
					if tc.aura {
						other := g.PutOnBattlefield(mage.NewCreature("Other", "{U}", 1, 20), a.PlayerID())
						g.Attach(source.ID(), other.ID())
						saved = g.CaptureDamageSource(other.ID())
					}
				case "source-phase":
					g.PhaseOut(source.ID())
					want = 0
				case "subject-phase":
					g.PhaseOut(subject.ID())
					want = 0
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(subject.ID()))
				case "depart", "return":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					want = 0
					if state == "return" {
						returned := g.PutOnBattlefield(card, b.PlayerID())
						if !tc.aura {
							saved = g.CaptureDamageSource(returned.ID())
							want = 1
						}
					}
				case "host-return":
					if tc.aura {
						g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
						g.PutOnBattlefield(hostCard, b.PlayerID())
						want = 0
					}
				case "remove-source", "remove-host":
					id := source.ID()
					if state == "remove-host" {
						id = subject.ID()
					}
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), id, a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					if state == "remove-source" || !tc.aura {
						want = 0
					}
				case "control":
					controller := b.PlayerID()
					if tc.aura {
						controller = a.PlayerID()
					}
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: host.ID(), TargetID: subject.ID(), ControllerID: controller, Duration: core.Indefinite})
				case "damage":
					artifact := g.PutOnBattlefield(mage.NewArtifact("Damage source", ""), b.PlayerID())
					g.DealDamageToPermanent(g.FindPermanent(subject.ID()), 1, artifact.ID())
				case "cleanup":
					g.GetEffects().ClearReplacementsEndOfTurn()
				}
				for _, game := range []*mage.Game{g, g.Clone()} {
					got := game.WaitingPreventionPermanentSources()
					if len(got) != want {
						t.Fatalf("subjects=%d want=%d", len(got), want)
					}
					if want == 0 {
						continue
					}
					if !got[0].SameObject(saved) {
						t.Fatal("query followed wrong subject incarnation")
					}
					if got[0].Controller() != game.CaptureDamageSource(saved.ID()).Controller() {
						t.Fatal("query did not read current controller")
					}
					binding := mage.BindDamageSource("static subject")
					effect := mage.Sequence(mage.ChooseDamageSourceFromWaitingPreventionReferences(mage.You, mage.DamageSourceFilter{}, binding), mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(1)))
					if err := mage.ApplyEffect(game, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					if game.GetPlayer(a.PlayerID()).Life() != 19 {
						t.Fatal("chosen subject could not supply damage")
					}
				}
				if len(baseline.WaitingPreventionPermanentSources()) != 1 {
					t.Fatal("mutation changed cloned rule")
				}
			})
		}
	}
}
