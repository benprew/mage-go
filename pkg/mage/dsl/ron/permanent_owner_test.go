package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPermanentOwner(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phased", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			factory := ronDelayedCombatFactory(t, `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PlayerAction(OwnerOfPermanent(ThisPermanent),GainLife(Integer(3)))]))]))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			switch state {
			case "departed", "returned":
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					card.SetOwner(b.PlayerID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			}
			for range 2 {
				refs := g.WaitingDelayedPermanentSources()
				if len(refs) != 1 || !refs[0].SameObject(saved) {
					t.Fatal("waiting owner read lost exact source")
				}
				if a.Life() != 20 || b.Life() != 20 {
					t.Fatal("waiting query changed life")
				}
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				if game.GetPlayer(a.PlayerID()).Life() != 23 || game.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("owner action used controller or returned visit")
				}
			}
			if a.Life() != 23 {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestRONPermanentOwnerInvalid(t *testing.T) {
	for _, ref := range []string{`OwnerOfPermanent`, `OwnerOfPermanent()`, `OwnerOfPermanent(ThisPermanent,ThisPermanent)`, `OwnerOfPermanent(You)`, `OwnerOfPermanent(Ref_TargetPermanent)`, `OwnerOfPermanent(HostPermanent)`, `OwnerOfPermanent(Trigger_ThatPermanent)`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([PlayerAction(` + ref + `,GainLife(Integer(1)))]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-owner.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", ref)
		}
	}
}

func TestRONPermanentOwnerBronzeTabletPaymentFragment(t *testing.T) {
	for _, accept := range []bool{false, true} {
		t.Run(fmt.Sprint(accept), func(t *testing.T) {
			record := staticBaseRecord(t, "ATQ", "Bronze Tablet")
			targeted := record.Fields["Rules"].Args[2].Args[1]
			payment := targeted.Args[1].Args[0].Args[1]
			parseAction := func(src string) Node {
				nodes, err := Parse(strings.NewReader(src), "owner-fragment.ron")
				if err != nil {
					t.Fatal(err)
				}
				return nodes[0]
			}
			targeted = parseAction(`Targeted([TargetPermanent(AnyPermanent)],ActionList([]))`)
			targeted.Args[1].Args[0].Args = []Node{
				parseAction(`Exile([Permanent(Ref_TargetPermanent)],[])`),
				payment,
				parseAction(`If(CostWasPaid,[PlayerAction(You,GainLife(Integer(1)))])`),
			}
			activation := parseAction(`Activated(PayMana([]),ActionList([]))`)
			activation.Args[1] = targeted
			rules := record.Fields["Rules"]
			rules.Args = []Node{activation}
			record.Fields["Rules"] = rules
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: accept}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			card := factory()
			card.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(card, b.PlayerID())
			subject := mage.NewArtifact("Payment subject", "")
			subject.SetOwner(a.PlayerID())
			target := g.PutOnBattlefield(subject, b.PlayerID())
			if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if a.offers != 0 || a.Life() != 20 {
				t.Fatal("payment happened at activation")
			}
			clone := g.Clone()
			g.ResolveStack()
			wantA, wantB := 20, 20
			if accept {
				wantA, wantB = 10, 21
			}
			if a.Life() != wantA || b.Life() != wantB || a.offers != 1 || g.FindPermanent(target.ID()) != nil {
				t.Fatal("exiled target's owner did not receive the payment choice")
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("payment changed clone")
			}
		})
	}
}
