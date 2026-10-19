package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONRecipientCountInstructionTyphoon(t *testing.T) {
	for _, state := range []string{"normal", "zero", "phase", "cycle", "control", "return", "partial", "prevented", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Typhoon")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
				t.Fatal("unpaid cast changed state")
			}
			a.ManaPool().Add(core.Green, 1)
			a.ManaPool().Add(core.Colorless, 2)
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			damage := 0
			if state != "zero" {
				p := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
				damage = 2
				switch state {
				case "phase":
					g.PhaseOut(p.ID())
					damage = 1
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
					damage = 1
				case "return":
					g.BouncePermanentToHand(p)
					g.PutOnBattlefield(p.Card, b.PlayerID())
				case "partial":
					g.AddPreventionShield(b.PlayerID(), 1)
					damage = 1
				case "prevented":
					g.AddPreventionShield(b.PlayerID(), 2)
					damage = 0
				}
			}
			if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				for range 2 {
					if certain, known := game.InspectSpellDestruction(game.StackPeek(), nil); certain || !known {
						t.Fatalf("complete Typhoon query=%v/%v", certain, known)
					}
				}
				if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("query dealt damage")
				}
			}
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			wantA := 20
			if state == "copy" {
				wantA = 19
			}
			for _, game := range []*mage.Game{g, clone} {
				if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != 20-damage {
					t.Fatal("recipient, count, or damage incorrect")
				}
			}
		})
	}
}

func TestRONRecipientCountInstructionInvalid(t *testing.T) {
	for _, amount := range []string{
		`TheNumberOfPermanentsOnTheBattlefield()`,
		`TheNumberOfPermanentsOnTheBattlefield(ControlledByAPlayer(SinglePlayer(DealsDamage_ThatPlayer())))`,
		`TheNumberOfPermanentsOnTheBattlefield(ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer)))`,
		`TheNumberOfPermanentsOnTheBattlefield(And([IsLandType(Island()),ControlledByAPlayer(SinglePlayer(DealsDamage_ThatPlayer))]))`,
	} {
		source := `Card(Name:"Invalid recipient count",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([SpellDealsDamage(ThisSpell,` + amount + `,EachPlayer(Opponent))]))])`
		if _, err := Compile(nameScopeNode(t, source)); err == nil {
			t.Fatal("invalid count accepted", amount)
		}
	}
}

func TestRONRecipientCountInstructionBindings(t *testing.T) {
	action := nameScopeNode(t, `SpellDealsDamage(ThisSpell,TheNumberOfPermanentsOnTheBattlefield(ControlledByAPlayer(SinglePlayer(DealsDamage_ThatPlayer))),EachPlayer(AnyPlayer))`)
	plan, err := compilePlayerRecipientDamage(action, scope{resolving: true}, func(instanceBindings) mage.DamageSourceRef { return mage.This })
	if err != nil {
		t.Fatal(err)
	}
	one, ok := mage.InspectPlayerCollectionDamageInstruction(plan(instanceBindings{}))
	if !ok {
		t.Fatal("lost first native instruction")
	}
	two, ok := mage.InspectPlayerCollectionDamageInstruction(plan(instanceBindings{}))
	if !ok || one.Member == two.Member {
		t.Fatal("shared instance member")
	}
	if _, ok := mage.InspectRecipientPermanentCount(one.Amount, one.Member); !ok {
		t.Fatal("own member rejected")
	}
	if _, ok := mage.InspectRecipientPermanentCount(one.Amount, two.Member); ok {
		t.Fatal("foreign member accepted")
	}
}
