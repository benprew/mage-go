package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONOrcGeneral(t *testing.T) {
	for _, payment := range []string{"Goblin", "Orc"} {
		for _, change := range []string{"none", "source-return", "source-left", "source-phase", "recipient-return", "phase", "control", "remove-abilities"} {
			t.Run(payment+"/"+change, func(t *testing.T) {
				a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				card := repositoryCard(t, "DRK", "Orc General")
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				add := func(kind string, player mage.Player) *mage.Permanent {
					c := mage.NewCreature(kind, "", 2, 2, mage.WithSubTypes(kind))
					c.SetOwner(player.PlayerID())
					return g.PutOnBattlefield(c, player.PlayerID())
				}
				victim := add(payment, a)
				a.choice = victim.ID()
				own := add("Orc", a)
				enemy := add("Orc", b)
				goblin := add("Goblin", a)
				phased := add("Orc", a)
				g.PhaseOut(phased.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(victim.ID()) != nil || !g.FindPermanent(source.ID()).Tapped || own.CurrentPower(g) != 2 {
					t.Fatal("payment or resolution timing")
				}
				early := add("Orc", b)
				if change == "source-return" || change == "source-left" {
					g.BouncePermanentToHand(g.FindPermanent(source.ID()))
					if change == "source-return" {
						a.RemoveFromHand(card.ID())
						source = g.PutOnBattlefield(card, a.PlayerID())
					}
				}
				if change == "source-phase" {
					g.PhaseOut(source.ID())
				}
				pending := g.Clone()
				g.ResolveStack()
				if pending.FindPermanent(own.ID()).CurrentPower(pending) != 2 {
					t.Fatal("resolution changed pending clone")
				}
				pending.ResolveStack()
				if pending.FindPermanent(own.ID()).CurrentPower(pending) != 3 {
					t.Fatal("clone lost binding")
				}
				late := add("Orc", a)
				check := func(p *mage.Permanent, want int) {
					t.Helper()
					live := g.FindPermanent(p.ID())
					if live.CurrentPower(g) != want || live.CurrentToughness(g) != want {
						t.Fatalf("%s P/T = %d/%d, want %d/%d", p.Card.Name(), live.CurrentPower(g), live.CurrentToughness(g), want, want)
					}
				}
				check(own, 3)
				check(enemy, 3)
				check(early, 3)
				check(goblin, 2)
				check(late, 2)
				wantSource := 2
				if change == "source-return" {
					wantSource = 3
				}
				if change == "source-phase" {
					g.PhaseIn([]uuid.UUID{source.ID()})
				}
				if change != "source-left" {
					check(source, wantSource)
				}
				switch change {
				case "recipient-return":
					c := own.Card
					g.BouncePermanentToHand(g.FindPermanent(own.ID()))
					a.RemoveFromHand(c.ID())
					own = g.PutOnBattlefield(c, a.PlayerID())
				case "phase":
					g.PhaseIn(g.PhaseOut(own.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: own.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "remove-abilities":
					g.AddContinuousEffect(mage.RemoveAllAbilities(own.ID(), core.EndOfTurn))
				}
				want := 3
				if change == "recipient-return" {
					want = 2
				}
				check(own, want)
				g.PhaseIn([]uuid.UUID{phased.ID()})
				check(phased, 2)
				g.RunStepWithPriority(core.Cleanup)
				check(own, 2)
				check(enemy, 2)
				if change != "source-left" {
					check(source, 2)
				}
			})
		}
	}
}

func TestRONOrcGeneralRollback(t *testing.T) {
	for _, reason := range []string{"self", "opponent", "wrong-type", "phased", "sick", "stale-choice"} {
		t.Run(reason, func(t *testing.T) {
			a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			c := repositoryCard(t, "DRK", "Orc General")
			c.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(c, a.PlayerID())
			if reason != "sick" {
				source.RevokeBaseAttr(core.AttrSummonSick)
			}
			player := mage.Player(a)
			if reason == "opponent" {
				player = b
			}
			kind := "Goblin"
			if reason == "wrong-type" {
				kind = "Human"
			}
			var victim *mage.Permanent
			if reason != "self" {
				v := mage.NewCreature("Payment", "", 1, 1, mage.WithSubTypes(kind))
				v.SetOwner(player.PlayerID())
				victim = g.PutOnBattlefield(v, player.PlayerID())
				a.choice = victim.ID()
				if reason == "phased" {
					g.PhaseOut(victim.ID())
				}
				if reason == "stale-choice" {
					a.invalid = victim
					g.BouncePermanentToHand(victim)
					player.RemoveFromHand(v.ID())
					victim = g.PutOnBattlefield(v, player.PlayerID())
				}
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
				t.Fatal("illegal payment accepted")
			}
			if g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 || len(a.Graveyard()) != 0 || len(b.Graveyard()) != 0 {
				t.Fatal("failed payment spent resources")
			}
			if victim != nil && reason != "phased" && g.FindPermanent(victim.ID()) == nil {
				t.Fatal("failed payment moved victim")
			}
		})
	}
}

func TestRONSourceExclusionInvalid(t *testing.T) {
	for _, filter := range []string{`Other`, `Other()`, `Other(ThisPermanent())`, `Other(HostPermanent)`, `Other(Ref_TargetPermanent)`, `Other(You)`, `Or([Other(ThisPermanent), IsCardtype(Creature)])`, `And([Other(ThisPermanent), Or([])])`} {
		for _, rule := range []string{
			fmt.Sprintf(`Activated(SacrificeAPermanent(%s),ActionList([GainLife(Integer(1))]))`, filter),
			fmt.Sprintf(`Activated(PayMana([ManaCostGeneric(1)]),ActionList([CreateEachPermanentLayerEffectUntil(%s,[AdjustPT(1,1)],UntilEndOfTurn)]))`, filter),
		} {
			nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[%s],CardPT:(Power:1,Toughness:1))`, rule)), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatalf("accepted %s", rule)
			}
		}
	}
}

func TestRONSourceExclusionInstancesAndCopy(t *testing.T) {
	a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	nodes, err := Parse(strings.NewReader(`Card(Name:"Source",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[Orc]),Rules:[Activated(SacrificeAPermanent(And([Other(ThisPermanent),IsCreatureType(Goblin)])),ActionList([CreateEachPermanentLayerEffectUntil(And([Other(ThisPermanent),IsCreatureType(Orc)]),[AdjustPT(1,1)],UntilEndOfTurn)]))],CardPT:(Power:2,Toughness:2))`), "instances.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	first := g.PutOnBattlefield(factory(), a.PlayerID())
	second := g.PutOnBattlefield(factory(), a.PlayerID())
	victim := g.PutOnBattlefield(mage.NewCreature("Payment", "", 1, 1, mage.WithSubTypes("Goblin")), a.PlayerID())
	a.choice = victim.ID()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	g.ResolveStack()
	if g.FindPermanent(first.ID()).CurrentPower(g) != 2 || g.FindPermanent(second.ID()).CurrentPower(g) != 4 {
		t.Fatal("copy or factory source identity")
	}
	if len(a.Graveyard()) != 1 || a.choices != 1 {
		t.Fatal("copy paid the cost again")
	}
}
