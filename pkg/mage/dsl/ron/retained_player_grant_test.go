package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func retainedPlayerGrantRecord(t *testing.T) Node {
	t.Helper()
	n := repositoryRecord(t, "LEG", "Takklemaggot", "retained-grant.ron")
	rules := n.Fields["Rules"]
	modifications := &rules.Args[2].Args[1].Args[0].Args[1].Args[2].Args[1].Args[1]
	modifications.Args = modifications.Args[1:]
	n.Fields["Rules"] = rules
	return n
}

func TestRONRetainedPlayerGrant(t *testing.T) {
	for _, state := range []string{"normal", "control", "phase", "cycle", "return", "clone", "copy"} {
		t.Run(state, func(t *testing.T) {
			factory, err := Compile(retainedPlayerGrantRecord(t))
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			hostCard := mage.NewCreature("Host", "", 2, 2)
			hostCard.SetOwner(a.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			card := factory()
			card.SetOwner(a.PlayerID())
			aura := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(aura.ID(), host.ID())
			g.DestroyPermanent(host)
			g.CheckStateBasedActions()
			g.PutTriggersOnStack()
			if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("stack copy failed")
			}
			if state == "clone" {
				original := g
				g = g.Clone()
				g.ResolveStack()
				if original.FindPermanent(card.ID()) != nil {
					t.Fatal("pending clone changed original")
				}
			} else {
				g.ResolveStack()
			}
			p := g.FindPermanent(card.ID())
			if p == nil || p.HasSubType("Aura") {
				t.Fatal("fallback return failed")
			}
			want := 19
			switch state {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase":
				g.PhaseOut(p.ID())
				want = 20
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.BouncePermanentToHand(p)
				owner := g.GetPlayer(a.PlayerID())
				c, _ := owner.RemoveFromHand(card.ID())
				g.PutOnBattlefield(c, a.PlayerID())
				want = 20
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if g.GetPlayer(a.PlayerID()).Life() != 20 || g.GetPlayer(b.PlayerID()).Life() != want {
				t.Fatalf("grant followed wrong player: A=%d B=%d want B=%d", g.GetPlayer(a.PlayerID()).Life(), g.GetPlayer(b.PlayerID()).Life(), want)
			}
		})
	}
}

func TestRONRetainedPlayerGrantInvalid(t *testing.T) {
	const grant = `CreatePermanentLayerEffect(ThisPermanent,[AddAbility([TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(ControllerOfDeadPermanent)),ActionList([GainLife(Integer(1))]))])])`
	for _, body := range []string{
		grant,
		strings.Replace(grant, "ControllerOfDeadPermanent", "ControllerOfDeadPermanent()", 1),
		strings.Replace(grant, "ControllerOfDeadPermanent", "HostController", 1),
	} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("unbound player accepted: %s", body)
		}
	}
	n := retainedPlayerGrantRecord(t)
	rules := n.Fields["Rules"]
	grantNode := &rules.Args[2].Args[1].Args[0].Args[1].Args[2].Args[1]
	grantNode.Args[1].Args[0].Args[0].Args[0].Args[1] = nameScopeNode(t, `ActionList([PlayerAction(ControllerOfDeadPermanent,GainLife(Integer(1)))])`)
	n.Fields["Rules"] = rules
	if _, err := Compile(n); err == nil {
		t.Fatal("donor binding escaped into recipient body")
	}
}

func TestRONRetainedPlayerGrantRecipient(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreatePermanentLayerEffect(ThisPermanent,[AddAbility([TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(ControllerOfPermanent(ThisPermanent))),ActionList([GainLife(Integer(1))]))])])]))`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 21 {
		t.Fatal("recipient You or saved event player changed")
	}
}
