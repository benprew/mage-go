package ron

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronHandContinuationPlayer struct {
	*ronPaymentPlayer
	beforeOffer func()
}

func (p *ronHandContinuationPlayer) ChooseMayAbility(text string) bool {
	if p.beforeOffer != nil {
		p.beforeOffer()
	}
	return p.ronPaymentPlayer.ChooseMayAbility(text)
}

func TestRONGraveyardHandContinuationPuppetMaster(t *testing.T) {
	for _, state := range []string{"pay", "decline", "unaffordable", "missing-host", "returned-host", "missing-aura", "returned-aura", "token", "copy", "clone", "phased", "simultaneous"} {
		t.Run(state, func(t *testing.T) {
			a := &ronHandContinuationPlayer{ronPaymentPlayer: &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: state != "decline"}}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			card := repositoryCard(t, "LEG", "Puppet Master")
			card.SetOwner(b.PlayerID())
			aura := g.PutOnBattlefield(card, a.PlayerID())
			hostCard := mage.NewCreature("Host", "", 2, 2)
			if state == "token" {
				hostCard = mage.NewToken("Token", 1, 1, []core.CardType{core.TypeCreature}, nil)
			}
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, a.PlayerID())
			g.Attach(aura.ID(), host.ID())
			a.beforeOffer = func() {
				if !slices.ContainsFunc(b.Hand(), func(c mage.Card) bool { return c.ID() == hostCard.ID() }) {
					t.Fatal("payment offered before host reached owner hand")
				}
			}
			if state != "unaffordable" {
				a.ManaPool().Add(core.Blue, 3)
			}
			if state == "phased" {
				g.PhaseOut(aura.ID())
			}
			if state == "simultaneous" {
				g.DestroyPermanents([]*mage.Permanent{host, aura})
			} else {
				g.DestroyPermanent(host)
			}
			g.CheckStateBasedActions()
			g.PutTriggersOnStack()
			if a.offers != 0 {
				t.Fatal("payment before resolution")
			}
			switch state {
			case "missing-host", "returned-host", "missing-aura", "returned-aura":
				id := hostCard.ID()
				if state == "missing-aura" || state == "returned-aura" {
					id = card.ID()
				}
				moved, ok := g.MoveFromGraveyard(b.PlayerID(), id, core.ZoneHand)
				if !ok {
					t.Fatal("missing fixture")
				}
				if state == "returned-host" || state == "returned-aura" {
					b.AddToGraveyard(moved)
				} else {
					b.AddToHand(moved)
				}
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "clone":
				cloned := g.Clone()
				cloned.SetPlayerAt(0, &ronOptionalReturnClonePlayer{SearchPlayer: cloned.GetPlayer(a.PlayerID()).(*mage.SearchPlayer)})
				cloned.ResolveStack()
				if len(b.Hand()) != 0 || a.ManaPool().TotalMana() != 3 || len(cloned.GetPlayer(b.PlayerID()).Hand()) != 2 {
					t.Fatal("clone shared state")
				}
			}
			g.ResolveStack()
			offers := 1
			if state == "missing-host" || state == "returned-host" || state == "token" || state == "unaffordable" || state == "phased" {
				offers = 0
			}
			if a.offers != offers || b.offers != 0 {
				t.Fatalf("offers %d/%d want %d/0", a.offers, b.offers, offers)
			}
			wantHand := 2
			switch state {
			case "decline", "unaffordable", "missing-host", "returned-aura":
				wantHand = 1
			case "returned-host", "token", "phased":
				wantHand = 0
			}
			if len(b.Hand()) != wantHand || len(a.Hand()) != 0 {
				t.Fatalf("hand %d/%d want 0/%d", len(a.Hand()), len(b.Hand()), wantHand)
			}
			wantMana := 0
			if offers == 0 && state != "unaffordable" || state == "decline" {
				wantMana = 3
			}
			if a.ManaPool().TotalMana() != wantMana {
				t.Fatal("wrong payment")
			}
		})
	}
}

func TestRONGraveyardHandContinuationBranches(t *testing.T) {
	for _, branch := range []string{"If(CostWasPaid,[GainLife(Integer(2))])", "Unless(CostWasntPaid,[GainLife(Integer(2))])", "IfElse(CostWasPaid,[GainLife(Integer(2))],[GainLife(Integer(3))])", "IfElse(CostWasntPaid,[GainLife(Integer(3))],[GainLife(Integer(2))])"} {
		t.Run(branch, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPermanentDies(SinglePermanent(ThisPermanent)),ActionList([MustCost(PutGraveyardCardIntoHand(Trigger_ThatGraveyardCard)),`+branch+`]))`)
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(b.PlayerID())
			g.DestroyPermanent(g.PutOnBattlefield(card, a.PlayerID()))
			g.PutTriggersOnStack()
			g.ResolveStack()
			if a.Life() != 22 || len(b.Hand()) != 1 {
				t.Fatal("wrong result branch")
			}
		})
	}
}

func TestRONGraveyardHandContinuationInvalid(t *testing.T) {
	const move = `MustCost(PutGraveyardCardIntoHand(Trigger_ThatGraveyardCard))`
	for _, body := range []string{
		move,
		move + `,GainLife(Integer(1)),If(CostWasPaid,[GainLife(Integer(2))])`,
		move + `,If(CostWasPaid(),[GainLife(Integer(2))])`,
		move + `,If(CostWasPaid,[])`,
		move + `,IfElse(CostWasPaid,[GainLife(Integer(2))],[])`,
		move + `,If(CostWasPaid,[GainLife(Integer(2))]),If(CostWasPaid,[GainLife(Integer(2))])`,
		`MustCost(PutGraveyardCardIntoHand(You)),If(CostWasPaid,[GainLife(Integer(2))])`,
		`MustCost(PutGraveyardCardIntoHand(Trigger_ThatGraveyardCard())),If(CostWasPaid,[GainLife(Integer(2))])`,
		`MustCost(PutGraveyardCardIntoHand()),If(CostWasPaid,[GainLife(Integer(2))])`,
		`MustCost(PutGraveyardCardIntoHand(Trigger_ThatGraveyardCard,You)),If(CostWasPaid,[GainLife(Integer(2))])`,
		`MayCost(PutGraveyardCardIntoHand(Trigger_ThatGraveyardCard)),If(CostWasPaid,[GainLife(Integer(2))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(WhenAPermanentDies(SinglePermanent(ThisPermanent)),ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	nodes, err := Parse(strings.NewReader(`MustCost(PutGraveyardCardIntoHand(Trigger_ThatGraveyardCard)) If(CostWasPaid,[GainLife(Integer(1))])`), "context.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{}, {resolving: true, enteringSource: true}, {resolving: true, drawReplacementBody: true}} {
		if _, err := compileGraveyardHandContinuation(nodes[0], nodes[1], ctx); err == nil {
			t.Fatal("invalid context accepted")
		}
	}
}

func TestRONGraveyardHandContinuationNested(t *testing.T) {
	const move = `MustCost(PutGraveyardCardIntoHand(Trigger_ThatGraveyardCard))`
	for _, state := range []string{"present", "returned"} {
		factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPermanentDies(SinglePermanent(ThisPermanent)),ActionList([`+move+`,IfElse(CostWasPaid,[`+move+`,IfElse(CostWasPaid,[GainLife(Integer(100))],[GainLife(Integer(2))])],[GainLife(Integer(3))]),`+move+`,Unless(CostWasPaid,[GainLife(Integer(5))])]))`)
		g, a, b := ronGame()
		card := factory()
		card.SetOwner(b.PlayerID())
		g.DestroyPermanent(g.PutOnBattlefield(card, a.PlayerID()))
		g.PutTriggersOnStack()
		if state == "returned" {
			moved, ok := g.MoveFromGraveyard(b.PlayerID(), card.ID(), core.ZoneHand)
			if !ok {
				t.Fatal("missing fixture")
			}
			b.AddToGraveyard(moved)
		}
		g.ResolveStack()
		want := 27
		if state == "returned" {
			want = 28
		}
		if a.Life() != want {
			t.Fatalf("life %d want %d", a.Life(), want)
		}
	}
}

func TestRONGraveyardHandContinuationPuppetCast(t *testing.T) {
	for _, state := range []string{"creature", "artifact", "shroud"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Puppet Master")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 3)
			hostCard := mage.NewCreature("Host", "", 2, 2)
			if state == "artifact" {
				hostCard = mage.NewArtifact("Host", "")
			}
			if state == "shroud" {
				hostCard = mage.NewCreature("Host", "", 2, 2, mage.WithKeyword(core.Shroud))
			}
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0)
			if state != "creature" {
				if err == nil || a.ManaPool().TotalMana() != 3 || len(a.Hand()) != 1 {
					t.Fatal("illegal enchantment target spent resources")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			aura := g.FindPermanent(card.ID())
			if aura == nil || aura.AttachedTo != host.ID() || a.ManaPool().TotalMana() != 0 {
				t.Fatal("Aura cast did not attach or pay UUU")
			}
		})
	}
}
