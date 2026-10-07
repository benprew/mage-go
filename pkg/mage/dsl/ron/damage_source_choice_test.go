package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronSourceChoicePlayer struct {
	*mage.BasePlayer
	calls  int
	source mage.DamageSource
}

func (p *ronSourceChoicePlayer) ChooseDamageSource(sources []mage.DamageSource, _ *mage.Game) int {
	p.calls++
	p.source = sources[0]
	return 0
}

func TestRONDamageSourceChoicePrerequisite(t *testing.T) {
	for _, scenario := range []string{"live", "departed", "returned", "phased", "clone", "copied-trigger"} {
		t.Run(scenario, func(t *testing.T) {
			_, a, b := ronGame()
			chooser := &ronSourceChoicePlayer{BasePlayer: b}
			g := mage.NewGame(a, chooser)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			card := repositoryCard(t, "LEA", "Prodigal Sorcerer")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			eventSource, chosen, amount := mage.BindDamageSource("event source"), mage.BindDamageSource("choice"), mage.BindNumber("event amount")
			fixture := mage.NewEnchantment("Choice fixture", "", mage.WithAbility(mage.WhenEvent(
				mage.ControllerIsDealtDamage(amount).WithSource(eventSource),
				mage.Sequence(mage.ChooseDamageSourceFrom(mage.You, []mage.DamageSourceRef{mage.FromDamageSource(eventSource)}, chosen),
					mage.DealDamageFrom(mage.FromDamageSource(chosen), mage.ActivePlayer, mage.Fixed(2))))))
			if err := mage.ValidateCardDefinition(fixture.Name(), fixture); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(fixture, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if chooser.calls != 0 || b.Life() != 20 {
				t.Fatal("choice ran before resolution")
			}
			g.ResolveTopOfStack()
			g.PutTriggersOnStack()
			if chooser.calls != 0 || b.Life() != 19 || g.StackSize() != 1 {
				t.Fatal("event did not retain separate resolution timing")
			}
			original := g.CaptureDamageSource(source.ID())
			switch scenario {
			case "departed", "returned":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				if scenario == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phased":
				g.PhaseOut(source.ID())
			case "copied-trigger":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("trigger copy failed")
				}
			}
			clone := g.Clone()
			if scenario == "clone" {
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != 18 {
					t.Fatal("clone did not use selected source")
				}
				if g.StackSize() != 1 || chooser.calls != 0 {
					t.Fatal("clone changed original pending choice")
				}
			}
			g.ResolveStack()
			want := 1
			if scenario == "copied-trigger" {
				want = 2
			}
			if chooser.calls != want || !chooser.source.SameObject(original) || !chooser.source.HasColor(core.Blue) || !chooser.source.HasType(core.TypeCreature) || chooser.source.Controller() != a.PlayerID() {
				t.Fatal("choice lost captured RON damage source")
			}
			if b.Life() != 19 || a.Life() != 20-2*want {
				t.Fatal("selected source continuation changed wrong life total")
			}
		})
	}
}
