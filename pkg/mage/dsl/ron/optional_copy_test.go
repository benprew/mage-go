package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONOptionalCopyGameplay(t *testing.T) {
	for _, change := range []string{"accept", "no exceptions", "decline", "illegal", "target phase", "target return", "target phase cycle", "source return", "source phase", "source phase cycle", "source control", "clone", "stack copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			chooser := &ronEntryCopyPlayer{Player: a, accept: true}
			g.SetPlayerAt(0, chooser)
			donor := g.PutOnBattlefield(repositoryCard(t, "LEA", "Air Elemental"), b.PlayerID())
			modification := "CopyEffects([KeepColor])"
			if change == "no exceptions" {
				modification = "NoCopyEffects"
			}
			factory := textCopyFactory(t, "Copier", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([MayAction(CreatePermanentLayerEffect(ThisPermanent,[IsACopyOfPermanent(Ref_TargetPermanent,`+modification+`)])),GainLife(Integer(1))])))`)
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{donor.ID()}); err != nil {
				t.Fatal(err)
			}
			if chooser.offers != 0 || p.Name() != "Copier" {
				t.Fatal("copy choice occurred before resolution")
			}
			wantCopy, wantOffer, wantLife := true, 1, 21
			switch change {
			case "decline":
				chooser.accept, wantCopy = false, false
			case "illegal":
				donor.GrantBaseAttr(core.Shroud)
				wantCopy, wantOffer, wantLife = false, 0, 20
			case "target phase":
				g.PhaseOut(donor.ID())
				wantCopy, wantOffer, wantLife = false, 0, 20
			case "target return":
				g.BouncePermanentToHand(donor)
				card := b.Hand()[0]
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
				wantCopy, wantOffer, wantLife = false, 0, 20
			case "target phase cycle":
				g.PhaseIn(g.PhaseOut(donor.ID()))
			case "source return":
				g.BouncePermanentToHand(p)
				card := a.Hand()[0]
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
				wantCopy, wantOffer = false, 0
			case "source phase":
				g.PhaseOut(p.ID())
				wantCopy, wantOffer = false, 0
			case "source phase cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "source control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: donor.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "clone":
				clone := g.Clone()
				clone.SetPlayerAt(0, &ronEntryCopyPlayer{Player: clone.GetPlayer(a.PlayerID()), accept: false})
				clone.ResolveStack()
				if clone.FindPermanent(p.ID()).Name() != "Copier" || g.FindPermanent(p.ID()).Name() != "Copier" || a.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 21 {
					t.Fatal("clone changed original or ignored its decision")
				}
			case "stack copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("stack copy failed")
				}
				wantOffer, wantLife = 2, 22
			}
			g.ResolveStack()
			p = g.FindPermanentIncludingPhased(p.ID())
			if (p.Name() == "Air Elemental") != wantCopy || chooser.offers != wantOffer || a.Life() != wantLife {
				t.Fatalf("copy=%v offers=%d life=%d; want %v/%d/%d", p.Name() == "Air Elemental", chooser.offers, a.Life(), wantCopy, wantOffer, wantLife)
			}
		})
	}
}

func TestRONOptionalCopyInvalid(t *testing.T) {
	valid := `MayAction(CreatePermanentLayerEffect(ThisPermanent,[IsACopyOfPermanent(Ref_TargetPermanent,NoCopyEffects)]))`
	for _, action := range []string{
		`MayAction`, `MayAction()`, `MayAction(TapPermanent(ThisPermanent))`,
		`MayAction(CreatePermanentLayerEffect(ThisPermanent,[AddCardtype(Creature)]))`,
		`MayAction(CreatePermanentLayerEffect(ThisPermanent,[IsACopyOfPermanent(ThisPermanent,NoCopyEffects),AddCardtype(Creature)]))`,
		`MayAction(CreatePermanentLayerEffect())`,
		strings.Replace(valid, "NoCopyEffects", "CopyEffects([KeepColor()])", 1),
		strings.Replace(valid, "NoCopyEffects", "CopyEffects([AddAbilityVariable(ThisAbility)])", 1),
		strings.Replace(valid, "Ref_TargetPermanent", "Ref_TargetPermanent_2", 1),
		strings.Replace(valid, "ThisPermanent", "HostPermanent", 1),
		strings.Replace(valid, "NoCopyEffects", "CopyEffects([])", 1),
	} {
		assertConditionalRejected(t, `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([`+action+`])))`)
	}
	for _, ctx := range []scope{{permanent: true, enteringSource: true}, {permanent: true, drawReplacementBody: true}} {
		nodes, err := Parse(strings.NewReader(`MayAction(CreatePermanentLayerEffect(ThisPermanent,[IsACopyOfPermanent(ThisPermanent,NoCopyEffects)]))`), "context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileEffect(nodes[0], ctx); err == nil {
			t.Fatal("replacement body accepted optional copy")
		}
	}
}

func TestRONOptionalCopyVesuvanClause(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := repositoryRecord(t, set, "Vesuvan Doppelganger", set)
			if _, err := Compile(record); err == nil {
				t.Fatal("ThisAbility must remain rejected")
			}
			rewrite := func(n Node) Node { return n }
			rewrite = func(n Node) Node {
				if n.Name == "CopyEffects" && len(n.Args) == 1 && len(n.Args[0].Args) == 2 && n.Args[0].Args[1].Name == "AddAbilityVariable" {
					n.Args[0].Args = n.Args[0].Args[:1]
				}
				for i, child := range n.Args {
					n.Args[i] = rewrite(child)
				}
				for key, child := range n.Fields {
					n.Fields[key] = rewrite(child)
				}
				return n
			}
			factory, err := Compile(rewrite(record))
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			donor := g.PutOnBattlefield(repositoryCard(t, "LEA", "Grizzly Bears"), b.PlayerID())
			next := g.PutOnBattlefield(repositoryCard(t, "LEA", "Air Elemental"), b.PlayerID())
			chooser := &copySpellTargetPlayer{ronEntryCopyPlayer: ronEntryCopyPlayer{Player: a, accept: true}, target: next.ID()}
			chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return donor }
			g.SetPlayerAt(0, chooser)
			card := factory()
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 2)
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			chooser.accept = false
			queueRONUpkeep(g, a)
			if g.StackSize() != 1 || chooser.offers != 1 {
				t.Fatal("upkeep made optional choice")
			}
			g.ResolveStack()
			if g.FindPermanent(card.ID()).Name() != "Grizzly Bears" || chooser.offers != 2 {
				t.Fatal("decline lost the copied rule")
			}
			chooser.accept = true
			queueRONUpkeep(g, a)
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p.Name() != "Air Elemental" || !p.HasAttr(core.Flying) || len(p.Colors()) != 1 || p.Colors()[0] != core.Blue || chooser.offers != 3 {
				t.Fatal("optional copy clause failed")
			}
		})
	}
}
