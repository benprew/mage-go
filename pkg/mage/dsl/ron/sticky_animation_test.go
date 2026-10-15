package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const ronStickyAnimation = `EachPermanentStickyLayerEffect(And([IsNonCardtype(Creature),IsCardtype(Artifact)]),[LosesAllAbilities,AddCardtype(Artifact),AddCardtype(Creature),SetPowerAndToughnessBoth(ManaValueOfPermanent(EachablePermanent))],UntilEndOfTurn)`

func TestRONStickyAnimationRecords(t *testing.T) {
	for _, set := range []string{"ATQ", "3ED"} {
		t.Run(set, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, set, "Titania's Song")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			artifact := g.PutOnBattlefield(repositoryCard(t, "LEA", "Sol Ring"), b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast succeeded")
			}
			if g.FindPermanent(artifact.ID()).HasType(core.TypeCreature) {
				t.Fatal("failed casting changed artifacts")
			}
			a.ManaPool().Add(core.Green, 4)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			check := func(p *mage.Permanent, want bool) {
				t.Helper()
				p = g.FindPermanent(p.ID())
				if p.HasType(core.TypeCreature) != want {
					t.Fatal("wrong animation membership")
				}
				if want && (p.CurrentPower(g) != 1 || p.CurrentToughness(g) != 1 || len(p.RuntimeAbilities) != 0) {
					t.Fatal("animation lost later layers")
				}
			}
			check(artifact, true)
			if err := g.ActivateAbilityByIndex(b.PlayerID(), artifact.ID(), 0, nil); err == nil {
				t.Fatal("removed mana ability activated")
			}
			source := g.FindPermanent(card.ID())
			g.BouncePermanentToHand(source)
			check(artifact, true)
			late := g.PutOnBattlefield(repositoryCard(t, "LEA", "Sol Ring"), a.PlayerID())
			check(late, true)
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			check(artifact, false)
			check(late, false)
			if len(g.FindPermanent(artifact.ID()).RuntimeAbilities) == 0 {
				t.Fatal("printed ability did not return")
			}
		})
	}
}

func TestRONStickyAnimationInvalid(t *testing.T) {
	for _, rule := range []string{
		strings.ReplaceAll(ronStickyAnimation, "UntilEndOfTurn", "Indefinite"),
		strings.ReplaceAll(ronStickyAnimation, "UntilEndOfTurn", "UntilEndOfTurn()"),
		strings.ReplaceAll(ronStickyAnimation, "IsNonCardtype(Creature)", "IsCardtype(Creature)"),
		strings.ReplaceAll(ronStickyAnimation, "IsCardtype(Artifact)", "IsCardtype(Land)"),
		strings.ReplaceAll(ronStickyAnimation, "LosesAllAbilities,", ""),
		strings.ReplaceAll(ronStickyAnimation, "LosesAllAbilities", "LosesAllAbilities()"),
		strings.ReplaceAll(ronStickyAnimation, "AddCardtype(Artifact),", ""),
		strings.ReplaceAll(ronStickyAnimation, "AddCardtype(Creature)", "AddCardtype(Land)"),
		strings.ReplaceAll(ronStickyAnimation, "EachablePermanent", "ThisPermanent"),
		strings.ReplaceAll(ronStickyAnimation, "ManaValueOfPermanent(EachablePermanent)", "Integer(3)"),
		strings.ReplaceAll(ronStickyAnimation, "LosesAllAbilities,", "LosesAllAbilities,LosesAllAbilities,"),
		strings.ReplaceAll(ronStickyAnimation, "UntilEndOfTurn)", "UntilEndOfTurn,Integer(1))"),
		`Activated(PayMana([]),ActionList([` + ronStickyAnimation + `]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", rule)
		}
	}
}

func TestRONStickyAnimationCollection(t *testing.T) {
	record := staticBaseRecord(t, "ATQ", "Titania's Song")
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	if err = mage.ValidateCardDefinition("Titania's Song", factory()); err != nil {
		t.Fatal(err)
	}
	source := `Card(Name:"Animation",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[` + ronStickyAnimation + `])`
	files := fstest.MapFS{"valid.ron": {Data: []byte(source)}}
	registry := mage.NewRegistry()
	if err = LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(strings.ReplaceAll(source, "Animation", "Invalid"), "UntilEndOfTurn", "Indefinite"))}
	if err = LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Animation") || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection changed the registry")
	}
	report, err := InspectFS(fstest.MapFS{"valid.ron": files["valid.ron"]}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{}
	for _, signature := range report.Signatures {
		roles[signature.Name] = signature.Role
	}
	if roles["LosesAllAbilities"] != "static_modification" || roles["UntilEndOfTurn"] != "duration" || roles["EachablePermanent"] != "permanent_reference" {
		t.Fatal("wrong sticky operand roles", roles)
	}
}
