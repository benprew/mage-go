package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCollectionDamageInstructionRealCards(t *testing.T) {
	for _, tc := range []struct {
		name, set string
		amount    int
	}{
		{"Earthquake", "LEA", 3}, {"Earthquake", "LEB", 3}, {"Earthquake", "2ED", 3}, {"Earthquake", "3ED", 3}, {"Earthquake", "CED", 3}, {"Earthquake", "CEI", 3},
		{"Hurricane", "LEA", 3}, {"Hurricane", "LEB", 3}, {"Hurricane", "2ED", 3}, {"Hurricane", "3ED", 3}, {"Hurricane", "CED", 3}, {"Hurricane", "CEI", 3},
		{"Inferno", "DRK", 6}, {"Sandstorm", "ARN", 1},
	} {
		for _, state := range []string{"normal", "flying", "phase", "cycle", "prevention", "lethal", "clone", "copy"} {
			t.Run(tc.name+"/"+tc.set+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				toughness := 10
				if state == "lethal" {
					toughness = 1
				}
				p := g.PutOnBattlefield(mage.NewCreature("Animated land", "", 1, toughness, mage.WithKeyword(core.Shroud)), b.PlayerID())
				p.GrantBaseAttr(core.AttrIsLand)
				if tc.name == "Hurricane" {
					p.GrantBaseAttr(core.Flying)
				}
				if tc.name == "Sandstorm" {
					g.GetCombat().AddAttacker(p.ID(), a.PlayerID())
				}
				query, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
				watched := query.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				a.ManaPool().Add(core.Red, 20)
				a.ManaPool().Add(core.Green, 20)
				x := 0
				if tc.name == "Earthquake" || tc.name == "Hurricane" {
					x = tc.amount
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, x); err != nil {
					t.Fatal(err)
				}
				obj := g.StackPeek()
				switch state {
				case "flying":
					if tc.name == "Hurricane" {
						p.RevokeBaseAttr(core.Flying)
					} else {
						p.GrantBaseAttr(core.Flying)
					}
				case "phase":
					g.PhaseOut(p.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "prevention":
					g.AddPreventionShield(p.ID(), 1)
				case "copy":
					obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
				}
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, watched); certain || !known {
						t.Fatal("complete collection damage tree has no pure result")
					}
				}
				if p.Damage != 0 || a.Life() != 20 || b.Life() != 20 {
					t.Fatal("inspection changed damage or life")
				}
				if state == "clone" {
					clone := g.Clone()
					if certain, known := clone.InspectSpellDestruction(clone.StackPeek(), watched); certain || !known {
						t.Fatal("clone lost damage policy")
					}
					clone.ResolveTopOfStack()
					if clone.FindPermanent(p.ID()).Damage != tc.amount || p.Damage != 0 {
						t.Fatal("clone crossed damage state")
					}
				}
				g.ResolveTopOfStack()
				want := tc.amount
				if state == "phase" || state == "flying" && (tc.name == "Earthquake" || tc.name == "Hurricane") || state == "cycle" && tc.name == "Sandstorm" {
					want = 0
				}
				if state == "prevention" {
					want--
				}
				if state == "lethal" {
					if g.FindPermanent(p.ID()) != nil {
						t.Fatal("lethal damage did not cause state-based death")
					}
				} else if got := g.FindPermanentIncludingPhased(p.ID()).Damage; got != want {
					t.Fatalf("damage=%d want=%d", got, want)
				}
				life := 20 - tc.amount
				if tc.name == "Sandstorm" {
					life = 20
				}
				if a.Life() != life || b.Life() != life {
					t.Fatal("wrong player recipients")
				}
			})
		}
	}
}

func TestRONCollectionDamageInstructionInvalid(t *testing.T) {
	for _, recipients := range []string{`MultipleRecipients([])`, `MultipleRecipients([EachPermanent(IsCardtype(Creature))])`, `MultipleRecipients([EachPermanent(IsCardtype(Creature)),EachPlayer(AnyPlayer),EachPlayer(AnyPlayer)])`, `MultipleRecipients([EachPermanent(HasAbility(Flying())),EachPlayer(AnyPlayer)])`, `MultipleRecipients([EachPermanent(IsCardtype(Creature)),EachPlayer(Ref_TargetPlayer)])`, `EachPermanent(HasAbility(UnknownKeyword))`} {
		_, err := Compile(nameScopeNode(t, `Card(Name:"Invalid collection",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([SpellDealsDamage(ThisSpell,Integer(2),`+recipients+`)]))])`))
		if err == nil {
			t.Fatalf("accepted %s", recipients)
		}
	}
}
