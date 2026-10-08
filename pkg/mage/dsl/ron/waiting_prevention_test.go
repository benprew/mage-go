package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingPreventionReferences(t *testing.T) {
	for _, name := range []string{"Samite Healer", "Healing Salve", "Alabaster Potion", "Indestructible Aura"} {
		for _, state := range []string{"live", "depart", "return", "phase", "cycle", "consume", "cleanup", "illegal"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				g, a, b := ronModalGame(1)
				set := "LEB"
				if name == "Alabaster Potion" || name == "Indestructible Aura" {
					set = "LEG"
				}
				card := repositoryCard(t, set, name)
				card.SetOwner(a.PlayerID())
				subjectCard := mage.NewCreature("Subject", "{R}", 1, 20)
				subject := g.PutOnBattlefield(subjectCard, b.PlayerID())
				saved := g.CaptureDamageSource(subject.ID())
				if name == "Samite Healer" {
					healer := g.PutOnBattlefield(card, a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), healer.ID(), 0, []uuid.UUID{subject.ID()}); err == nil {
						t.Fatal("sick activation succeeded")
					}
					g.MutablePermanent(healer.ID()).RevokeBaseAttr(core.AttrSummonSick)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), healer.ID(), 0, []uuid.UUID{subject.ID()}); err != nil {
						t.Fatal(err)
					}
				} else {
					a.AddToHand(card)
					x := 0
					if name == "Alabaster Potion" {
						x = 2
					}
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{subject.ID()}, x); err == nil {
						t.Fatal("unpaid cast succeeded")
					}
					a.ManaPool().Add(core.White, 4)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{subject.ID()}, x); err != nil {
						t.Fatal(err)
					}
				}
				if len(g.WaitingPreventionPermanentSources()) != 0 {
					t.Fatal("announced action supplied waiting reference")
				}
				if state == "illegal" {
					g.PhaseOut(subject.ID())
				}
				g.ResolveStack()
				switch state {
				case "depart", "return":
					g.RemoveFromBattlefield(g.FindPermanent(subject.ID()))
					if state == "return" {
						g.PutOnBattlefield(subjectCard, a.PlayerID())
					}
				case "phase":
					g.PhaseOut(subject.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(subject.ID()))
				case "consume":
					g.DealDamageToPermanent(g.FindPermanent(subject.ID()), 3, uuid.New())
				case "cleanup":
					g.GetEffects().ClearReplacementsEndOfTurn()
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{g, clone} {
					got := game.WaitingPreventionPermanentSources()
					want := 1
					if state == "cleanup" || state == "illegal" || (state == "consume" && name != "Indestructible Aura") {
						want = 0
					}
					if len(got) != want {
						t.Fatalf("sources=%d want=%d", len(got), want)
					}
					if want == 0 {
						continue
					}
					if !got[0].SameObject(saved) {
						t.Fatal("waiting reference lost recipient identity")
					}
					chosen := mage.BindDamageSource("waiting recipient")
					effect := mage.Sequence(mage.ChooseDamageSourceFromWaitingPreventionReferences(mage.You, mage.DamageSourceFilter{}, chosen), mage.DealDamageFrom(mage.FromDamageSource(chosen), mage.You, mage.Fixed(1)))
					if err := mage.ApplyEffect(game, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					if game.GetPlayer(a.PlayerID()).Life() != 19 {
						t.Fatal("saved absent recipient could not supply damage source")
					}
				}
			})
		}
	}
}
