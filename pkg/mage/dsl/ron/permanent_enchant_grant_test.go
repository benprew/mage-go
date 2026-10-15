package ron

import (
	"maps"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestRONPermanentEnchantGrantAnimateDead(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := chosenDiscardRecord(t, set, "Animate Dead")
			if _, err := Compile(record); err != nil {
				t.Fatal("complete full record rejected", err)
			}
			rules := record.Fields["Rules"]
			body := rules.Args[1].Args[2]
			body.Args = append([]Node(nil), body.Args...)
			body.Args[0].Args = append([]Node(nil), body.Args[0].Args[:2]...)
			record.Fields = maps.Clone(record.Fields)
			record.Fields["Rules"] = Node{Kind: List, Args: []Node{rules.Args[0], {Kind: Tuple, Name: "TriggerA", Args: []Node{rules.Args[1].Args[0], body}}}}
			if _, err := Compile(record); err != nil {
				t.Fatal(err)
			}
			plan, err := compileActions(body, scope{permanent: true, aura: true, graveyardEnchant: true, enteringSource: true})
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			guest := mage.NewCreature("Guest", "", 3, 4)
			guest.SetOwner(b.PlayerID())
			b.AddToGraveyard(guest)
			card := mage.NewAura("Probe", "{B}", mage.WithCastTarget(mage.GraveyardCreatureEnchantTarget()))
			card.SetOwner(a.PlayerID())
			aura := g.PutOnBattlefield(card, a.PlayerID())
			effects := plan.build().effects
			if err := mage.ApplyEffect(g, mage.Sequence(effects...), aura.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 2), b.PlayerID())
			g.Attach(aura.ID(), other.ID())
			if aura.AttachedTo == other.ID() {
				t.Fatal("linked enchant accepted unrelated creature")
			}
			g.Attach(aura.ID(), guest.ID())
			g.CheckStateBasedActions()
			if g.FindPermanent(aura.ID()) == nil || aura.AttachedTo != guest.ID() {
				t.Fatal("real grant and return did not permit the exact host")
			}
		})
	}
}

func TestRONPermanentEnchantGrantInvalid(t *testing.T) {
	const good = `CreatePermanentLayerEffect(ThisPermanent,[LosesAbility(Enchant(IsCardtype(Creature))),AddAbility([EnchantPermanent(And([IsCardtype(Creature),PutOntoBattlefieldByPermanent(ThisPermanent)]))])])`
	for _, pair := range [][2]string{
		{"ThisPermanent,[LosesAbility", "HostPermanent,[LosesAbility"},
		{"IsCardtype(Creature),Put", "IsCardtype(Land),Put"},
		{"ByPermanent(ThisPermanent)", "ByPermanent(HostPermanent)"},
		{"ByPermanent(ThisPermanent)", "ByPermanent(ThisPermanent())"},
		{"IsCardtype(Creature),Put", "IsCardtype(Creature()),Put"},
		{"LosesAbility(Enchant(IsCardtype(Creature)))", "LosesAbility(Enchant(IsCardtype(Land)))"},
		{"And([IsCardtype(Creature),PutOntoBattlefieldByPermanent(ThisPermanent)])", "IsCardtype(Creature)"},
		{"And([IsCardtype(Creature),PutOntoBattlefieldByPermanent(ThisPermanent)])", "And([IsCardtype(Creature),PutOntoBattlefieldByPermanent(ThisPermanent),IsCardtype(Artifact)])"},
		{"EnchantPermanent(And", "EnchantPermanent(You,And"},
		{"AddAbility([EnchantPermanent", "AddAbility([Flying,EnchantPermanent"},
		{"LosesAbility(Enchant(IsCardtype(Creature))),", ""},
		{"ThisPermanent,[", "ThisPermanent,[AdjustPT(1,1),"},
	} {
		bad := strings.Replace(good, pair[0], pair[1], 1)
		source := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantGraveyardCard(IsCardtype(Creature),AnyPlayer),Activated(PayMana([]),ActionList([` + bad + `]))])`
		if _, err := Compile(nameScopeNode(t, source)); err == nil {
			t.Fatalf("invalid declaration accepted: %s", bad)
		}
	}
	node := nameScopeNode(t, good)
	for _, ctx := range []scope{
		{permanent: true, aura: true, graveyardEnchant: true},
		{permanent: true, aura: true, graveyardEnchant: true, enteringSource: true},
		{permanent: true, aura: true, graveyardEnchant: true, resolving: true, drawReplacementBody: true},
		{permanent: true, aura: true, resolving: true},
	} {
		if _, err := compileIndefiniteType(node, ctx); err == nil {
			t.Fatalf("invalid context accepted: %+v", ctx)
		}
	}
}

func TestRONPermanentEnchantGrantStack(t *testing.T) {
	record := chosenDiscardRecord(t, "LEA", "Animate Dead")
	rules := record.Fields["Rules"]
	body := rules.Args[1].Args[2]
	grant := body.Args[0].Args[0]
	filter := grant.Args[1].Args[1].Args[0].Args[0].Args[0]
	body.Args = append([]Node(nil), body.Args...)
	body.Args[0].Args = append([]Node(nil), body.Args[0].Args[:2]...)
	body.Args[0].Args = append(body.Args[0].Args, Node{Kind: Tuple, Name: "TapEachPermanent", Args: []Node{filter}})
	record.Fields = maps.Clone(record.Fields)
	record.Fields["Rules"] = Node{Kind: List, Args: []Node{rules.Args[0], {Kind: Tuple, Name: "TriggerA", Args: []Node{rules.Args[1].Args[0], body}}}}
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	guest := mage.NewCreature("Guest", "", 3, 4)
	guest.SetOwner(b.PlayerID())
	b.AddToGraveyard(guest)
	card := factory()
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	g.PutTriggersOnStack()
	clone := g.Clone()
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	g.ResolveStack()
	if p := g.FindPermanent(guest.ID()); p == nil || !p.Tapped {
		t.Fatal("resolving grant and return lost their live relation")
	}
	if g.FindPermanent(card.ID()) != nil {
		t.Fatal("unattached projection skipped its state action")
	}
	if clone.FindPermanent(guest.ID()) != nil {
		t.Fatal("original changed pending clone")
	}
	clone.ResolveStack()
	if p := clone.FindPermanent(guest.ID()); p == nil || !p.Tapped {
		t.Fatal("pending clone lost source grant and return")
	}
}
