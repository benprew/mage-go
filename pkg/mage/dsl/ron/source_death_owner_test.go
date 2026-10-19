package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSourceDeathOwnerPersonalIncarnation(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, set, "Personal Incarnation")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast succeeded")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed cast changed state")
			}
			a.ManaPool().Add(core.White, 6)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			subject := g.FindPermanent(card.ID())
			if subject == nil || a.ManaPool().TotalMana() != 0 {
				t.Fatal("paid cast failed")
			}
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: subject.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			if err := g.ActivateAbilityByIndex(b.PlayerID(), subject.ID(), 0, nil); err == nil {
				t.Fatal("controller activated owner ability")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), subject.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.PutOnBattlefield(mage.NewCreature("Damage source", "", 1, 20), b.PlayerID())
			g.DealDamageToPermanent(subject, 2, source.ID())
			if a.Life() != 19 || b.Life() != 20 || subject.Damage != 1 {
				t.Fatal("activation lost owner or finite budget")
			}
			g.DestroyPermanent(subject)
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || g.StackPeek().Controller != b.PlayerID() {
				t.Fatal("death trigger lost controller")
			}
			a.SetLife(15)
			g.ResolveStack()
			if a.Life() != 7 || b.Life() != 20 {
				t.Fatalf("owner life %d/%d, want 7/20", a.Life(), b.Life())
			}
		})
	}
}

func TestRONSourceDeathOwnerLifetime(t *testing.T) {
	for _, change := range []string{"normal", "return", "phase", "phase-cycle", "exile", "bounce", "removed", "copy", "pending-clone", "exile-replacement", "regenerate"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEA", "Personal Incarnation")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			switch change {
			case "phase":
				g.PhaseOut(p.ID())
				g.DestroyPermanent(p)
			case "phase-cycle":
				g.PhaseOut(p.ID())
				g.PhaseIn([]uuid.UUID{p.ID()})
				g.DestroyPermanent(p)
			case "exile-replacement":
				g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), p.ID())
				g.DestroyPermanent(p)
			case "regenerate":
				g.AddRegenerationShield(p.ID())
				g.DestroyPermanent(p)
			case "exile":
				g.ExilePermanent(p)
			case "bounce":
				g.BouncePermanentToHand(p)
			case "removed":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				g.DestroyPermanent(p)
			default:
				g.DestroyPermanent(p)
			}
			wantCount := 1
			if change == "phase" || change == "exile" || change == "bounce" || change == "removed" || change == "exile-replacement" || change == "regenerate" {
				wantCount = 0
			}
			if a.Life() != 20 {
				t.Fatal("life loss before resolution")
			}
			clone := g.Clone()
			if change == "return" {
				a.RemoveFromGraveyard(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
				card.SetOwner(b.PlayerID())
			}
			g.PutTriggersOnStack()
			if g.StackSize() != wantCount {
				t.Fatalf("stack %d, want %d", g.StackSize(), wantCount)
			}
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			a.SetLife(15)
			clone.GetPlayer(a.PlayerID()).SetLife(11)
			g.ResolveStack()
			want := 15
			if wantCount == 1 {
				want = 7
			}
			if change == "copy" {
				want = 3
			}
			if a.Life() != want || b.Life() != 20 {
				t.Fatalf("life %d/%d, want %d/20", a.Life(), b.Life(), want)
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 11 {
				t.Fatal("original changed clone")
			}
			clone.PutTriggersOnStack()
			clone.ResolveStack()
			cloneWant := 11
			if wantCount == 1 {
				cloneWant = 5
			}
			if clone.GetPlayer(a.PlayerID()).Life() != cloneWant {
				t.Fatal("clone lost saved owner")
			}
		})
	}
}

func TestRONSourceDeathOwnerLifeTotals(t *testing.T) {
	for _, life := range []int{20, 15, 1, 0, -1, -10} {
		t.Run(fmt.Sprint(life), func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEA", "Personal Incarnation")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			g.DestroyPermanent(p)
			g.PutTriggersOnStack()
			a.SetLife(life)
			g.ResolveStack()
			want := life
			if life > 0 {
				want = life / 2
			}
			if a.Life() != want || b.Life() != 20 {
				t.Fatalf("life %d/%d, want %d/20", a.Life(), b.Life(), want)
			}
		})
	}
}

func TestRONSourceDeathOwnerSimultaneous(t *testing.T) {
	g, a, b := ronGame()
	factory, err := Compile(staticBaseRecord(t, "LEA", "Personal Incarnation"))
	if err != nil {
		t.Fatal(err)
	}
	subjects := make([]*mage.Permanent, 0, 2)
	for _, owner := range []mage.Player{a, b} {
		card := factory()
		card.SetOwner(owner.PlayerID())
		subjects = append(subjects, g.PutOnBattlefield(card, b.PlayerID()))
	}
	g.DestroyPermanents(subjects)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("missing independent deaths")
	}
	a.SetLife(15)
	b.SetLife(11)
	g.ResolveStack()
	if a.Life() != 7 || b.Life() != 5 {
		t.Fatalf("shared owner binding: %d/%d", a.Life(), b.Life())
	}
}

func TestRONSourceDeathOwnerInvalid(t *testing.T) {
	valid := `TriggerA(WhenAPermanentDies(SinglePermanent(ThisPermanent)),ActionList([PlayerAction(OwnerOfDeadPermanent,GainLife(Integer(1)))]))`
	for _, rule := range []string{
		strings.Replace(valid, "OwnerOfDeadPermanent,", "OwnerOfDeadPermanent(),", 1),
		strings.Replace(valid, "WhenAPermanentDies", "WhenAPermanentEntersTheBattlefield", 1),
		strings.Replace(valid, "ThisPermanent)", "ThisPermanent())", 1),
		strings.Replace(valid, "ThisPermanent", "Ref_TargetPermanent", 1),
		strings.Replace(valid, "OwnerOfDeadPermanent", "ControllerOfDeadPermanent", 1),
		strings.Replace(valid, "OwnerOfDeadPermanent", "ToughnessOfDeadPermanent", 1),
		valid + `,Activated(PayMana([]),ActionList([PlayerAction(OwnerOfDeadPermanent,GainLife(Integer(1)))]))`,
		`Activated(PayMana([]),ActionList([PlayerAction(OwnerOfDeadPermanent,GainLife(Integer(1)))]))`,
	} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`)
		if _, err := Compile(n); err == nil {
			t.Fatal("accepted invalid declaration", rule)
		}
	}
}
