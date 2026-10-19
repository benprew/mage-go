package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPluralTargetQueryWinterBlast(t *testing.T) {
	for _, state := range []string{"normal", "one", "zero", "gain flying", "lose flying", "already tapped", "phase", "cycle", "return", "shroud", "all illegal", "copy", "clone", "prevention"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			g.SetStep(core.PrecombatMain)
			p := g.PutOnBattlefield(mage.NewCreature("Flyer", "", 1, 8, mage.WithKeyword(core.Flying)), b.PlayerID())
			q := g.PutOnBattlefield(mage.NewCreature("Ground", "", 1, 8), b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 8, mage.WithKeyword(core.Flying)), b.PlayerID())
			card := repositoryCard(t, "LEG", "Winter Blast")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			count := 2
			switch state {
			case "zero":
				count = 0
			case "one":
				count = 1
			}
			a.ManaPool().Add(core.Green, 1)
			a.ManaPool().Add(core.Colorless, count)
			ids := []uuid.UUID{p.ID(), q.ID()}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids[:count], count); err != nil {
				t.Fatal(err)
			}
			obj := g.StackPeek()
			wantP, wantQ := 2, 0
			tapP, tapQ := true, true
			switch state {
			case "one":
				tapQ = false
			case "zero":
				wantP = 0
				tapP, tapQ = false, false
			case "gain flying":
				q.GrantBaseAttr(core.Flying)
				wantQ = 2
			case "lose flying":
				p.RevokeBaseAttr(core.Flying)
				wantP = 0
			case "already tapped":
				g.TapPermanent(p)
			case "phase":
				g.PhaseOut(p.ID())
				wantP = 0
				tapP = false
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.RemoveFromBattlefield(p)
				p = g.PutOnBattlefield(p.Card, b.PlayerID())
				wantP = 0
				tapP = false
			case "shroud":
				p.GrantBaseAttr(core.Shroud)
				wantP = 0
				tapP = false
			case "all illegal":
				p.GrantBaseAttr(core.Shroud)
				q.GrantBaseAttr(core.Shroud)
				wantP = 0
				tapP, tapQ = false, false
			case "copy":
				obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
			case "clone":
				g = g.Clone()
				obj = g.StackPeek()
			case "prevention":
				g.AddPreventionShield(p.ID(), 1)
				wantP = 1
			}
			for range 2 {
				if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
					t.Fatal("complete Winter Blast query unknown")
				}
			}
			if g.FindPermanentIncludingPhased(p.ID()).Damage != 0 || g.FindPermanent(other.ID()).Tapped {
				t.Fatal("inspection changed state")
			}
			g.ResolveTopOfStack()
			if got := g.FindPermanentIncludingPhased(p.ID()); got.Damage != wantP || got.Tapped != tapP {
				t.Fatalf("flyer damage/tap=%d/%v want=%d/%v", got.Damage, got.Tapped, wantP, tapP)
			}
			if got := g.FindPermanent(q.ID()); got.Damage != wantQ || got.Tapped != tapQ {
				t.Fatal("ground selection differs")
			}
			if g.FindPermanent(other.ID()).Damage != 0 || g.FindPermanent(other.ID()).Tapped {
				t.Fatal("unannounced visit affected")
			}
			if state == "clone" && (p.Damage != 0 || p.Tapped) {
				t.Fatal("clone crossed state")
			}
		})
	}
}

func TestRONPluralTargetQueryInvalid(t *testing.T) {
	for _, body := range []string{
		`Targeted([NumberTargetPermanents(ValueX,IsCardtype(Creature))],ActionList([SpellDealsDamage(ThisSpell,Integer(2),EachPermanent(And([Ref_TargetPermanents,Ref_TargetPermanents_1])))]))`,
		`Targeted([NumberTargetPermanents(ValueX,IsCardtype(Creature))],ActionList([SpellDealsDamage(ThisSpell,Integer(2),EachPermanent(And([Ref_TargetPermanents,HasAbility(Flying())])))]))`,
		`ActionList([SpellDealsDamage(ThisSpell,Integer(2),EachPermanent(Ref_TargetPermanents))])`,
	} {
		node := nameScopeNode(t, `Card(Name:"Invalid plural",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),ManaCost:[ManaCostX],Rules:[SpellActions(`+body+`)])`)
		if _, err := Compile(node); err == nil {
			t.Fatalf("invalid plural declaration accepted: %s", body)
		}
	}
}
