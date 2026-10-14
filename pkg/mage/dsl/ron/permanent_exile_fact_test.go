package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const exileThrullCondition = `APermanentWasExiledThisWay(And([IsCardtype(Creature),IsCreatureType(Thrull)]))`

func TestRONPermanentExileFactGameplay(t *testing.T) {
	for _, state := range []string{"thrull", "other", "gained", "lost", "copy", "return", "delay"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			body := `If(` + exileThrullCondition + `,[GainLife(Integer(2))]),If(APermanentWasExiledThisWay(IsCardtype(Creature)),[GainLife(Integer(1))])`
			if state == "delay" {
				body = `CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + body + `]))`
			}
			factory := permanentScopeFactory(t, "Instant", `CastEffect(AdditionalCastingCost(`+matchingExileRON+`)),SpellActions(ActionList([`+body+`]))`)
			card := mage.NewCreature("Guest", "", 2, 2, mage.WithSubTypes("Thrull"))
			if state == "other" || state == "gained" {
				card = mage.NewCreature("Guest", "", 2, 2)
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			if state == "gained" {
				p.SubTypeOverride = []string{"Thrull"}
			}
			if state == "lost" {
				p.SubTypeOverride = []string{"Goblin"}
			}
			castRONSpell(t, g, a, factory())
			if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			if state == "return" {
				if _, ok := g.RemoveFromExile(card.ID()); !ok {
					t.Fatal("paid card missing from exile")
				}
				g.PutOnBattlefield(card, b.PlayerID())
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			if state == "delay" {
				g.RunStepWithPriority(core.EndStep)
				clone.RunStepWithPriority(core.EndStep)
			}
			want := 23
			if state == "other" || state == "lost" {
				want = 21
			}
			if a.Life() != want || clone.GetPlayer(a.PlayerID()).Life() != want {
				t.Fatal("wrong saved payment facts", a.Life(), want)
			}
			if state == "copy" && b.Life() != 23 {
				t.Fatal("copy lost facts or payer/controller binding", b.Life())
			}
		})
	}
}

func TestRONPermanentExileFactSourceClause(t *testing.T) {
	for _, payment := range []string{"Basal Thrull", "Grizzly Bears"} {
		t.Run(payment, func(t *testing.T) {
			record := hostUntapRecord(t, "FEM", "Soul Exchange")
			if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "permanent counter type") {
				t.Fatal("complete record must keep its counter adapter gap", err)
			}
			rules := record.Fields["Rules"]
			spell := rules.Args[1]
			targeted := spell.Args[0]
			actions := targeted.Args[1]
			branch := actions.Args[0].Args[1]
			branch.Args = []Node{branch.Args[0], {Kind: List, Args: []Node{nameScopeNode(t, `TapPermanent(ThePermanentPutOnTheBattlefieldThisWay)`)}}}
			actions.Args = []Node{{Kind: List, Args: []Node{actions.Args[0].Args[0], branch}}}
			targeted.Args = []Node{targeted.Args[0], actions}
			spell.Args = []Node{targeted}
			rules.Args = []Node{rules.Args[0], spell}
			record.Fields["Rules"] = rules
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			g, a, _ := ronGame()
			set := "LEB"
			if payment == "Basal Thrull" {
				set = "FEM"
			}
			paid := repositoryCard(t, set, payment)
			g.PutOnBattlefield(paid, a.PlayerID())
			guest := repositoryCard(t, "LEB", "Grizzly Bears")
			guest.SetOwner(a.PlayerID())
			a.AddToGraveyard(guest)
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Black, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{guest.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(paid.ID()) != nil || g.FindPermanent(guest.ID()) != nil {
				t.Fatal("incorrect payment or entry timing")
			}
			g.ResolveStack()
			p := g.FindPermanent(guest.ID())
			if p == nil || p.Tapped != (payment == "Basal Thrull") {
				t.Fatal("source payment fact did not govern the returned entrant")
			}
		})
	}
}

func TestRONPermanentExileFactInstances(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `CastEffect(AdditionalCastingCost(`+matchingExileRON+`)),SpellActions(ActionList([If(`+exileThrullCondition+`,[GainLife(Integer(2))]),If(`+exileThrullCondition+`,[GainLife(Integer(1))])]))`)
	g, a, b := ronGame()
	g.PutOnBattlefield(repositoryCard(t, "FEM", "Basal Thrull"), a.PlayerID())
	castRONSpell(t, g, a, factory())
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), a.PlayerID())
	castRONSpell(t, g, a, factory())
	if g.StackSize() != 2 {
		t.Fatal("missing independent stack instances")
	}
	clone := g.Clone()
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 20 || g.StackSize() != 0 {
		t.Fatal("later false payment overwrote earlier facts", a.Life())
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.StackSize() != 2 {
		t.Fatal("live resolution changed clone")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 23 {
		t.Fatal("clone lost independent facts")
	}
}

func TestRONPermanentExileFactInvalid(t *testing.T) {
	for _, rules := range []string{
		`SpellActions(ActionList([If(` + exileThrullCondition + `,[GainLife(Integer(1))])]))`,
		`Activated(` + matchingExileRON + `,ActionList([If(` + exileThrullCondition + `,[GainLife(Integer(1))])]))`,
		`CastEffect(AdditionalCastingCost(` + matchingExileRON + `)),SpellActions(ActionList([If(APermanentWasExiledThisWay(IsTapped),[GainLife(Integer(1))])]))`,
		`CastEffect(AdditionalCastingCost(` + matchingExileRON + `)),SpellActions(ActionList([If(APermanentWasExiledThisWay(And([])),[GainLife(Integer(1))])]))`,
		`CastEffect(AdditionalCastingCost(` + matchingExileRON + `)),SpellActions(ActionList([If(APermanentWasExiledThisWay(),[GainLife(Integer(1))])]))`,
		`CastEffect(AdditionalCastingCost(` + matchingExileRON + `)),SpellActions(ActionList([If(` + exileThrullCondition + `,[])]))`,
		`CastEffect(AdditionalCastingCost(` + matchingExileRON + `)),SpellActions(ActionList([If(APermanentWasExiledThisWay(IsCreatureType(Forest)),[GainLife(Integer(1))])]))`,
		`CastEffect(AdditionalCastingCost(` + matchingExileRON + `)),SpellActions(ActionList([If(APermanentWasExiledThisWay(IsNonToken()),[GainLife(Integer(1))])]))`,
	} {
		kind := "Instant"
		if strings.HasPrefix(rules, "Activated") {
			kind = "Artifact"
		}
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[`+kind+`],Subtypes:[]),Rules:[`+rules+`])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid exile fact accepted", rules)
		}
	}
}

func TestRONPermanentExileFactFilters(t *testing.T) {
	for _, tc := range []struct {
		filter string
		match  bool
	}{
		{`IsCardtype(Creature)`, true}, {`IsNonCardtype(Creature)`, false},
		{`IsCreatureType(Thrull)`, true}, {`IsNonCreatureType(Thrull)`, false},
		{`IsLandType(Forest)`, false}, {`IsNonLandType(Forest)`, true},
		{`IsEnchantmentType(Aura)`, false}, {`IsNonToken`, true},
		{`IsColor(Black)`, true}, {`IsNonColor(Black)`, false},
		{`And([IsCardtype(Creature),Or([IsColor(Green),IsCreatureType(Thrull)])])`, true},
		{`Or([IsCardtype(Artifact),IsColor(Green)])`, false},
	} {
		t.Run(tc.filter, func(t *testing.T) {
			g, a, _ := ronGame()
			g.PutOnBattlefield(repositoryCard(t, "FEM", "Basal Thrull"), a.PlayerID())
			factory := permanentScopeFactory(t, "Instant", `CastEffect(AdditionalCastingCost(`+matchingExileRON+`)),SpellActions(ActionList([If(APermanentWasExiledThisWay(`+tc.filter+`),[GainLife(Integer(1))])]))`)
			castRONSpell(t, g, a, factory())
			g.ResolveStack()
			want := 20
			if tc.match {
				want++
			}
			if a.Life() != want {
				t.Fatal("wrong historical characteristic filter", a.Life(), want)
			}
		})
	}
}
