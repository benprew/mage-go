package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const ronKeptProducer = `EachPlayerAction(AnyPlayer,ChooseUptoNumberPermanentsForEach(Integer(2),IsLandType(Plains),NumberOfPermanentsTappedThisWayByPlayer(EachablePlayer)))`
const ronKeptDestroy = `DestroyEachPermanent(And([IsLandType(Plains),NotChosenThisWay]))`

func TestRONPermanentSetChoiceRaidingParty(t *testing.T) {
	for _, count := range []int{0, 1, 2} {
		a := &ronOptionalDiscardPlayer{BasePlayer: mage.NewBasePlayer("A"), count: count}
		b := &ronOptionalDiscardPlayer{BasePlayer: mage.NewBasePlayer("B"), count: 0}
		g := mage.NewGame(a, b)
		g.SetTurn(1)
		g.SetStep(core.PrecombatMain)
		source := g.PutOnBattlefield(repositoryCard(t, "FEM", "Raiding Party"), a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), len(source.Card.Abilities())-1, nil); err == nil || a.requests != 0 {
			t.Fatal("failed cost ran choices")
		}
		orc := mage.NewCreature("Orc", "", 1, 1)
		orc.AddSubType("Orc")
		orc.SetOwner(a.PlayerID())
		g.PutOnBattlefield(orc, a.PlayerID())
		for range 2 {
			g.PutOnBattlefield(mage.NewCreature("White", "{W}", 1, 1), a.PlayerID())
		}
		lands := make([]*mage.Permanent, 0, 4)
		for range 4 {
			land := mage.NewLand("Plains")
			land.AddSubType("Plains")
			land.SetOwner(b.PlayerID())
			lands = append(lands, g.PutOnBattlefield(land, b.PlayerID()))
		}
		phasedCard := mage.NewLand("Phased")
		phasedCard.AddSubType("Plains")
		phased := g.PutOnBattlefield(phasedCard, b.PlayerID())
		g.PhaseOut(phased.ID())
		nonPlains := g.PutOnBattlefield(mage.NewLand("Other"), b.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), len(source.Card.Abilities())-1, nil); err != nil {
			t.Fatal(err)
		}
		if a.requests != 0 || b.requests != 0 {
			t.Fatal("early effect choices")
		}
		g.RemoveFromBattlefield(source)
		g.ResolveStack()
		remaining := 0
		for _, p := range lands {
			if g.FindPermanent(p.ID()) != nil {
				remaining++
			}
		}
		if remaining != count || g.FindPermanent(nonPlains.ID()) == nil || g.FindPermanentIncludingPhased(phased.ID()) == nil {
			t.Fatal("saved union or live membership", count, remaining)
		}
		if a.requests != 1+count {
			t.Fatal("one bounded choice per tapped creature", a.requests)
		}
	}
}

func TestRONPermanentSetChoiceInvalid(t *testing.T) {
	cases := []string{
		ronTapProducer + `,` + strings.Replace(ronKeptProducer, `Integer(2)`, `Integer(2,3)`, 1),
		ronTapProducer + `,` + strings.Replace(ronKeptProducer, `Integer(2)`, `ValueX`, 1),
		ronTapProducer + `,` + strings.Replace(ronKeptProducer, `NumberOfPermanentsTappedThisWayByPlayer(EachablePlayer)`, `NumberOfPermanentsTappedThisWayByPlayer(EachablePlayer,You)`, 1),
		ronTapProducer + `,` + ronKeptProducer + `,` + strings.Replace(ronKeptDestroy, `NotChosenThisWay`, `NotChosenThisWay(You)`, 1),
		ronKeptProducer + `,` + ronKeptDestroy,
		ronTapProducer + `,` + strings.Replace(ronKeptProducer, `Integer(2)`, `Integer(-1)`, 1),
		ronTapProducer + `,` + strings.Replace(ronKeptProducer, `EachablePlayer`, `ActionPlayer`, 1),
		ronTapProducer + `,` + strings.Replace(ronKeptProducer, `AnyPlayer`, `Opponent`, 1),
		ronTapProducer + `,` + strings.Replace(ronKeptProducer, `IsLandType(Plains)`, `ControlledByPlayer(EachablePlayer)`, 1),
		ronKeptDestroy + `,` + ronTapProducer + `,` + ronKeptProducer,
		ronTapProducer + `,If(PermanentPassesFilter(ThisPermanent,IsUntapped),[` + ronKeptProducer + `]),` + ronKeptDestroy,
	}
	for _, body := range cases {
		records, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[Activated(PayMana([ManaCostGeneric(1)]),ActionList([`+body+`]))])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(records[0]); err == nil {
			t.Fatal("invalid scope accepted", body)
		}
	}
}

func TestRONPermanentSetChoiceCopiesClonesDelayAndScope(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([`+ronTapProducer+`,`+ronKeptProducer+`,CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+ronKeptDestroy+`]))]))`)
	a := &ronOptionalDiscardPlayer{BasePlayer: mage.NewBasePlayer("A"), count: 1}
	b := &ronOptionalDiscardPlayer{BasePlayer: mage.NewBasePlayer("B"), count: 0}
	g := mage.NewGame(a, b)
	g.SetTurn(1)
	g.SetStep(core.PrecombatMain)
	g.PutOnBattlefield(mage.NewCreature("Creature", "", 1, 1), a.PlayerID())
	for range 3 {
		c := mage.NewLand("Plains")
		c.AddSubType("Plains")
		c.SetOwner(b.PlayerID())
		g.PutOnBattlefield(c, b.PlayerID())
	}
	card := factory()
	castRONSpell(t, g, a.BasePlayer, card)
	clone := g.Clone()
	if g.CopySpellOnStack(card.ID(), a.PlayerID(), false) == nil {
		t.Fatal("missing copy")
	}
	g.ResolveStack()
	if len(b.Graveyard()) != 0 {
		t.Fatal("delay ran early")
	}
	clone.ResolveStack()
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	clone.ResolveStack()
	if len(clone.GetPlayer(b.PlayerID()).Graveyard()) != 1 || len(b.Graveyard()) != 0 {
		t.Fatal("clone choices or saved results shared")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.ResolveStack()
	if len(b.Graveyard()) != 3 {
		t.Fatal("independent copied producer or delayed frame lost", len(b.Graveyard()))
	}
}

func TestRONPermanentSetChoiceInventoryAndAtomicLoad(t *testing.T) {
	src := `Card(Name:"Kept",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + ronTapProducer + `,` + ronKeptProducer + `,` + ronKeptDestroy + `]))])`
	files := fstest.MapFS{"kept.ron": {Data: []byte(src)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal(err, report.Issues)
	}
	found := false
	for _, entry := range report.Signatures {
		if entry.Name == "EachablePlayer" {
			found = true
			if entry.Role != "player_reference" {
				t.Fatal("count role", entry.Role)
			}
		}
	}
	if !found {
		t.Fatal("missing local player role")
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(src, ronKeptProducer+`,`, "", 1))}
	registry, err := LoadFS(files, "*.ron")
	if err == nil || registry != nil {
		t.Fatal("partial collection published")
	}
}

func TestRONPermanentSetChoiceBranchDominance(t *testing.T) {
	prefix := `Card(Name:"Branch",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + ronTapProducer + `,If(PermanentPassesFilter(ThisPermanent,IsUntapped),[`
	for _, escape := range []bool{false, true} {
		end := ronKeptProducer + `,` + ronKeptDestroy + `])]))])`
		if escape {
			end = ronKeptProducer + `]),` + ronKeptDestroy + `]))])`
		}
		nodes, err := Parse(strings.NewReader(prefix+end), "branch.ron")
		if err != nil {
			t.Fatal(err)
		}
		_, err = Compile(nodes[0])
		if escape {
			if err == nil || !strings.Contains(err.Error(), "unbound plural permanent choice") {
				t.Fatal("wrong branch rejection", err)
			}
		} else if err != nil {
			t.Fatal("local branch rejected", err)
		}
	}
}
