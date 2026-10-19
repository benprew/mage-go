package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPacketDamageInstructionBlast(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		for _, state := range []string{"normal", "player", "overlap", "phase", "cycle", "return", "shroud", "protection", "prevention", "lethal", "clone", "copy"} {
			t.Run(set+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				toughness := 8
				if state == "lethal" {
					toughness = 1
				}
				p := g.PutOnBattlefield(mage.NewCreature("Animated land", "", 1, toughness), b.PlayerID())
				p.GrantBaseAttr(core.AttrIsLand)
				query, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
				watched := query.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
				target := p.ID()
				if state == "player" {
					target = b.PlayerID()
				}
				if state == "overlap" {
					target = a.PlayerID()
				}
				obj := castRONSpell(t, g, a, repositoryCard(t, set, "Psionic Blast"), target)
				switch state {
				case "phase":
					g.PhaseOut(p.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "return":
					g.RemoveFromBattlefield(p)
					g.PutOnBattlefield(p.Card, b.PlayerID())
				case "shroud":
					p.GrantBaseAttr(core.Shroud)
				case "protection":
					p.RuntimeAbilities = append(p.RuntimeAbilities, mage.ProtectionFromColor(core.Blue))
				case "prevention":
					g.AddPreventionShield(p.ID(), 2)
				case "copy":
					obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
				}
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, watched); certain || !known {
						t.Fatal("packet spell query is unknown or proves destruction")
					}
				}
				if a.Life() != 20 || b.Life() != 20 || p.Damage != 0 {
					t.Fatal("query changed damage state")
				}
				if state == "clone" {
					clone := g.Clone()
					if certain, known := clone.InspectSpellDestruction(clone.StackPeek(), watched); certain || !known {
						t.Fatal("clone query differs")
					}
					clone.ResolveTopOfStack()
					if clone.GetPlayer(a.PlayerID()).Life() != 18 || clone.FindPermanent(p.ID()).Damage != 4 || a.Life() != 20 || p.Damage != 0 {
						t.Fatal("clone damage state crossed")
					}
				}
				g.ResolveTopOfStack()
				invalid := state == "phase" || state == "return" || state == "shroud" || state == "protection"
				wantA, wantB, wantDamage := 18, 20, 4
				if invalid {
					wantA = 20
					wantDamage = 0
				}
				if state == "player" {
					wantB = 16
					wantDamage = 0
				}
				if state == "overlap" {
					wantA = 14
					wantDamage = 0
				}
				if state == "prevention" {
					wantDamage = 2
				}
				if state == "copy" {
					wantA = 20
					wantB = 18
				}
				if a.Life() != wantA || b.Life() != wantB {
					t.Fatalf("life=%d/%d want=%d/%d", a.Life(), b.Life(), wantA, wantB)
				}
				if state == "lethal" {
					if g.FindPermanent(p.ID()) != nil {
						t.Fatal("lethal damage did not cause death")
					}
				} else if state != "return" {
					current := g.FindPermanentIncludingPhased(p.ID())
					if current == nil || current.Damage != wantDamage {
						t.Fatalf("current permanent=%v want damage=%d", current, wantDamage)
					}
				}
			})
		}
	}
}

func TestRONPacketDamageInstructionBindings(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([AnyTarget,AnyTarget],ActionList([SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(2),Ref_AnyTarget1),DamageToRecipients(Integer(1),Ref_AnyTarget2),DamageToRecipients(Integer(1),Player(You))])])))`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 9), b.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 9), b.PlayerID())
	first := castRONSpell(t, g, a, factory(), p.ID(), q.ID())
	second := castRONSpell(t, g, a, factory(), q.ID(), p.ID())
	for _, obj := range []*mage.StackObject{first, second} {
		groups, ok := mage.InspectPacketDamageInstructions(mage.Sequence(obj.Effects...))
		if !ok || len(groups) != 1 || len(groups[0].Packets) != 3 {
			t.Fatal("packet lowering lost group")
		}
		if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
			t.Fatal("independent packet bindings unknown")
		}
	}
	p.GrantBaseAttr(core.Shroud)
	g.ResolveTopOfStack()
	if p.Damage != 0 || q.Damage != 2 || a.Life() != 19 {
		t.Fatal("partial legality or second instance differs")
	}
	g.ResolveTopOfStack()
	if p.Damage != 0 || q.Damage != 3 || a.Life() != 18 {
		t.Fatal("first instance lost bindings")
	}
}

func TestRONPacketDamageInstructionInvalid(t *testing.T) {
	for _, action := range []string{
		`SpellDealsMultipleDamage(ThisSpell,[])`,
		`SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(1),Player(Ref_TargetPlayer))])`,
		`SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(1),Ref_AnyTarget2)])`,
		`SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(1),Player(You()))])`,
		`SpellDealsMultipleDamage(ThisPermanent,[DamageToRecipients(Integer(1),Player(You))])`,
		`SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(1),EachPlayer(AnyPlayer))])`,
	} {
		_, err := Compile(nameScopeNode(t, `Card(Name:"Invalid packet",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([AnyTarget],ActionList([`+action+`])))])`))
		if err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}

func TestRONPacketDamageInstructionPlayer(t *testing.T) {
	for _, action := range []string{`SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(2),Player(Ref_TargetPlayer)),DamageToRecipients(Integer(1),Player(You))])`, `SpellDealsDamage(ThisSpell,Integer(2),Player(Ref_TargetPlayer))`} {
		factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(Opponent)],ActionList([`+action+`])))`)
		g, a, b := ronGame()
		obj := castRONSpell(t, g, a, factory(), b.PlayerID())
		if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
			t.Fatal("native player damage unknown")
		}
		g.ResolveTopOfStack()
		if b.Life() != 18 {
			t.Fatal("wrong player damaged")
		}
	}
}
