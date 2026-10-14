package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSourcePluralSacrificeCostGameplay(t *testing.T) {
	record := repositoryRecord(t, "LEG", "Sword of the Ages", "LEG.lines.ron")
	body, err := Parse(strings.NewReader("ActionList([DrawACard])"), "payment-body.ron")
	if err != nil {
		t.Fatal(err)
	}
	rules := record.Fields["Rules"]
	rules.Args[1].Args[1] = body[0]
	record.Fields["Rules"] = rules
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	a := &sourcePresentSacrificeChooser{ronNumericChooser: &ronNumericChooser{BasePlayer: mage.NewBasePlayer("A"), answer: 1}}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	if !source.Tapped {
		t.Fatal("entry rule lost")
	}
	g.UntapPermanent(source)
	victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 2), a.PlayerID())
	a.source = source.ID()
	a.SetLibrary([]mage.Card{mage.NewLand("Drawn")})
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if !a.sawSource || g.FindPermanent(source.ID()) != nil || g.FindPermanent(victim.ID()) != nil || len(a.Hand()) != 0 {
		t.Fatal("source left before optional choices or body resolved early")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if len(a.Hand()) != 0 || len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 {
		t.Fatal("clone changed live frame")
	}
	g.ResolveStack()
	if len(a.Hand()) != 1 || a.calls != 1 {
		t.Fatal("body or choices repeated")
	}
}

type sourcePresentSacrificeChooser struct {
	*ronNumericChooser
	source    uuid.UUID
	sawSource bool
}

func (p *sourcePresentSacrificeChooser) ChoosePermanent(candidates []*mage.Permanent, text string, g mage.GameReader) *mage.Permanent {
	p.sawSource = g.FindPermanent(p.source) != nil
	for _, subject := range candidates {
		if subject.ID() == p.source {
			return nil
		}
	}
	return p.BasePlayer.ChoosePermanent(candidates, text, g)
}

func TestRONSourcePluralSacrificeCostInvalid(t *testing.T) {
	optional := `SacrificeAnyNumberOfPermanents(And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))]))`
	for _, cost := range []string{
		`And([SacrificePermanent(ThisPermanent),SacrificePermanent(ThisPermanent),` + optional + `])`,
		`And([SacrificePermanent(HostPermanent),` + optional + `])`,
		`And([SacrificePermanent(ThisPermanent()),` + optional + `])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(`+cost+`,ActionList([DrawACard]))])`), "invalid-combined.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid combined cost", cost)
		}
	}
}

func TestRONSourcePluralSacrificeCostFactoryAndNestedCosts(t *testing.T) {
	optional := `SacrificeAnyNumberOfPermanents(And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))]))`
	for _, cost := range []string{
		`And([TapPermanent(ThisPermanent),SacrificePermanent(ThisPermanent),` + optional + `])`,
		`And([TapPermanent(ThisPermanent),And([` + optional + `,SacrificePermanent(ThisPermanent)])])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Combined",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(`+cost+`,ActionList([DrawACard]))])`), "combined.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		first, second := factory(), factory()
		if first.ID() == second.ID() || first.Abilities()[0].AbilityID() == second.Abilities()[0].AbilityID() {
			t.Fatal("factory shares identities")
		}
		for _, card := range []mage.Card{first, second} {
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
		}
		a := &ronNumericChooser{BasePlayer: mage.NewBasePlayer("A"), answer: 0}
		g := mage.NewGame(a, mage.NewBasePlayer("B"))
		g.SetStep(core.PrecombatMain)
		g.PutOnBattlefield(first, a.PlayerID())
		g.PutOnBattlefield(second, a.PlayerID())
		a.SetLibrary([]mage.Card{mage.NewLand("Drawn")})
		if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) == nil || a.calls != 1 {
			t.Fatal("zero payment or factory isolation failed")
		}
		g.ResolveStack()
		if len(a.Hand()) != 1 {
			t.Fatal("body failed")
		}
	}
}
