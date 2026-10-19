package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONExiledOwnershipIndependent(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(AnyPermanent)],ActionList([Exile([Permanent(ThisPermanent),Permanent(Ref_TargetPermanent)],[]),AcquireOwnershipOfCard(TheFirstCardExiledThisWay,OwnerOfPermanent(Ref_TargetPermanent)),AcquireOwnershipOfCard(TheSecondCardExiledThisWay,You)])))`)
	for _, state := range []string{"normal", "absent", "returned", "phased", "stolen", "clone", "copy"} {
		t.Run(state, func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g, _ := mage.NewGameWithAnte(a, b, nil, nil)
			source := factory()
			source.SetOwner(a.PlayerID())
			if state == "stolen" {
				source.SetOwner(b.PlayerID())
			}
			p := g.PutOnBattlefield(source, a.PlayerID())
			target := mage.NewArtifact("target", "")
			target.SetOwner(b.PlayerID())
			q := g.PutOnBattlefield(target, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "absent", "returned":
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(source.ID())
					g.PutOnBattlefield(source, a.PlayerID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			want := a.PlayerID()
			if state == "copy" {
				want = b.PlayerID()
			}
			if g.FindCardAnywhere(target.ID()).Owner() != want {
				t.Fatal("independent target assignment failed")
			}
			if clone.FindCardAnywhere(target.ID()).Owner() != b.PlayerID() {
				t.Fatal("ownership leaked into clone")
			}
			clone.ResolveStack()
			if clone.FindCardAnywhere(target.ID()).Owner() != want {
				t.Fatal("clone lost assignment")
			}
			if state == "normal" || state == "stolen" || state == "clone" {
				if g.FindCardAnywhere(source.ID()).Owner() != b.PlayerID() {
					t.Fatal("source assignment failed")
				}
			}
		})
	}
}

func TestRONExiledOwnershipInvalid(t *testing.T) {
	producer := `Exile([Permanent(ThisPermanent),Permanent(ThisPermanent)],[])`
	for _, body := range []string{
		`AcquireOwnershipOfCard(TheFirstCardExiledThisWay,You)`,
		producer + `,AcquireOwnershipOfCard(TheFirstCardExiledThisWay(),You)`,
		producer + `,AcquireOwnershipOfCard(TheFirstCardExiledThisWay,Ref_TargetPlayer)`,
		producer + `,AcquireOwnershipOfCard(TheOtherPermanentsExiledThisWay,You)`,
		producer + `,AcquireOwnershipOfCard(TheFirstCardExiledThisWay)`,
		`If(PermanentPassesFilter(ThisPermanent,IsTapped),[` + producer + `]),AcquireOwnershipOfCard(TheFirstCardExiledThisWay,You)`,
	} {
		record := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`)
		if _, err := Compile(record); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONExiledOwnershipBronzeTablet(t *testing.T) {
	for _, state := range []string{"decline", "paid", "short", "absent", "returned", "phased", "stolen", "illegal", "target-return", "phase-cycle", "control-change", "clone", "failed-cost"} {
		t.Run(state, func(t *testing.T) {
			a := mage.NewBasePlayer("A")
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: state == "paid" || state == "short"}
			if state == "short" {
				b.SetLife(9)
			}
			g, _ := mage.NewGameWithAnte(a, b, nil, nil)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			card := repositoryCard(t, "ATQ", "Bronze Tablet")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Colorless, 6)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p == nil || !p.Tapped {
				t.Fatal("entry must be tapped")
			}
			g.UntapPermanent(p)
			if state == "stolen" {
				if err := g.ChangeOwner(card.ID(), b.PlayerID()); err != nil {
					t.Fatal(err)
				}
			}
			other := mage.NewArtifact("other", "")
			other.SetOwner(b.PlayerID())
			q := g.PutOnBattlefield(other, b.PlayerID())
			mana := 4
			if state == "failed-cost" {
				mana = 3
			}
			a.ManaPool().Add(core.Colorless, mana)
			err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()})
			if state == "failed-cost" {
				if err == nil || p.Tapped || g.StackSize() != 0 || a.ManaPool().TotalMana() != 3 {
					t.Fatal("failed cost mutated state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			switch state {
			case "absent", "returned":
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseOut(q.ID())
				g.PhaseIn([]uuid.UUID{q.ID()})
			case "control-change":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: q.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "illegal":
				g.PhaseOut(q.ID())
			case "target-return":
				g.BouncePermanentToHand(q)
				b.RemoveFromHand(other.ID())
				g.PutOnBattlefield(other, b.PlayerID())
			}
			clone := g.Clone()
			g.ResolveStack()
			if state == "illegal" || state == "target-return" {
				if b.offers != 0 || g.FindPermanent(p.ID()) == nil || len(g.GetExile()) != 0 {
					t.Fatal("illegal target resolved")
				}
				return
			}
			if state == "paid" {
				if b.Life() != 10 || len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != card.ID() || g.FindCardAnywhere(other.ID()).Owner() != b.PlayerID() {
					t.Fatal("paid branch failed")
				}
			} else if g.FindCardAnywhere(other.ID()).Owner() != a.PlayerID() {
				t.Fatal("unpaid branch must acquire target independently")
			}
			if state == "decline" || state == "stolen" || state == "short" || state == "clone" {
				if state != "paid" && g.FindCardAnywhere(card.ID()).Owner() != b.PlayerID() {
					t.Fatal("source owner assignment failed")
				}
			}
			if clone.FindCardAnywhere(other.ID()).Owner() != b.PlayerID() {
				t.Fatal("clone shared owner")
			}
			if state == "clone" {
				clone.ResolveStack()
				if clone.FindCardAnywhere(other.ID()).Owner() != a.PlayerID() {
					t.Fatal("clone failed")
				}
			}
		})
	}
}
