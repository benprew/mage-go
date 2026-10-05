package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPreventionSourcePrerequisite(t *testing.T) {
	for _, name := range []string{"Prodigal Sorcerer", "Grapeshot Catapult"} {
		for _, change := range []string{"departed", "returned", "phased", "copy"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				set := "LEB"
				if name == "Grapeshot Catapult" {
					set = "ATQ"
				}
				card := repositoryCard(t, set, name)
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 10, mage.WithKeyword(core.Flying)), b.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err == nil || g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
					t.Fatal("summoning sickness rejection changed payment state")
				}
				g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
					t.Fatal(err)
				}
				if !g.FindPermanent(source.ID()).Tapped || victim.Damage != 0 {
					t.Fatal("payment or early damage")
				}
				if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				if change == "phased" {
					g.PhaseOut(source.ID())
				} else {
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					if change == "returned" {
						g.PutOnBattlefield(card, b.PlayerID())
					}
				}
				filter := mage.NewPermanentFilter("original controller", func(p *mage.Permanent, _ *mage.Game) bool {
					return p.HasType(core.TypeCreature) && p.ControllerID() == a.PlayerID()
				})
				g.AddDamagePreventionRule(mage.WithFrom(filter), mage.WithTo(mage.IsID(victim.ID())))
				clone := g.Clone()
				g.ResolveStack()
				if g.FindPermanent(victim.ID()).Damage != 0 {
					t.Fatal("RON damage lost source incarnation in prevention")
				}
				if clone.FindPermanent(victim.ID()).Damage != 0 {
					t.Fatal("resolution changed clone")
				}
				clone.ResolveStack()
				if clone.FindPermanent(victim.ID()).Damage != 0 {
					t.Fatal("clone resolution lost source")
				}
			})
		}
	}
}

func TestRONPreventionSourcePrerequisiteInvalid(t *testing.T) {
	for _, rule := range []string{
		`Activated(TapPermanent(ThisPermanent),ActionList([PermanentDealsDamage(Ref_TargetPermanent,Integer(1),Player(You))]))`,
		`Activated(TapPermanent(ThisPermanent),Targeted([TargetPlayer(AnyPlayer)],ActionList([PermanentDealsDamage(Ref_TargetPlayer,Integer(1),Player(You))])))`,
		`Activated(TapPermanent(ThisPermanent),ActionList([PermanentDealsDamage(ThisPermanent(),Integer(1),Player(You))]))`,
	} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rule + `])`
		nodes, err := Parse(strings.NewReader(source), "invalid-prevention-source.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted invalid source declaration: %s", rule)
		}
	}
}
