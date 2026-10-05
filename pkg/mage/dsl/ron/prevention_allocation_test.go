package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronPreventionAllocationPlayer struct {
	*mage.BasePlayer
	choices int
}

func (p *ronPreventionAllocationPlayer) ChooseNumber(minimum, maximum int, reason string) int {
	p.choices++
	return 1
}

func TestRONPreventionAllocationCombat(t *testing.T) {
	a := mage.NewBasePlayer("A")
	b := &ronPreventionAllocationPlayer{BasePlayer: mage.NewBasePlayer("B")}
	g := mage.NewGame(a, b)
	first := g.PutOnBattlefield(repositoryCard(t, "LEB", "Savannah Lions"), a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEB", "War Mammoth"), a.PlayerID())
	first.RevokeBaseAttr(core.AttrSummonSick)
	second.RevokeBaseAttr(core.AttrSummonSick)
	g.AddPreventionShield(b.PlayerID(), 3)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{first.ID(), second.ID()})
	g.ExecuteCombatDamage()
	if b.Life() != 18 || b.choices != 1 {
		t.Fatalf("life=%d choices=%d", b.Life(), b.choices)
	}
	source1 := g.CaptureDamageSource(first.ID())
	source2 := g.CaptureDamageSource(second.ID())
	if !g.HasDealtDamageToPlayerFromSource(source1, b.PlayerID()) || !g.HasDealtDamageToPlayerFromSource(source2, b.PlayerID()) {
		t.Fatal("split prevention erased one source outcome")
	}
}

func TestRONPreventionAllocationKeepsDeclarationsStrict(t *testing.T) {
	for _, action := range []string{
		`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(3),Ref_AnyTarget),[PreventThatDamage])`,
		`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(3),Permanent(Ref_TargetPlayer)),[PreventThatDamage])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Unsupported",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+action+`]))])`), "unsupported.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("engine prerequisite authorized an unsupported declaration")
		}
	}
}

func TestRONPreventionAllocationSpellRollbackAndClone(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEB", "Hurricane")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 3); err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("unpaid casting changed state")
	}
	g.AddPreventionShield(a.PlayerID(), 2)
	g.AddPreventionShield(b.PlayerID(), 4)
	a.ManaPool().Add(core.Green, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 3); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 0 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("payment or premature damage")
	}
	clone := g.Clone()
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("resolution or clone isolation")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 19 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("pending clone lost prevention")
	}
	g.DealDamageToPlayer(b, 2, uuid.New())
	if b.Life() != 19 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("shield consumption leaked into clone")
	}
}
