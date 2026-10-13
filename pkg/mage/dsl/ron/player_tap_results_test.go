package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const ronTapProducer = `EachPlayerAction(AnyPlayer,TapAnyNumberOfPermanents(And([IsUntapped,IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(ActionPlayer))])))`
const ronTapRead = `PlayerAction(You,GainLife(NumberOfPermanentsTappedThisWayByPlayer(You)))`

func TestRONPlayerTapResultsRaidingPartyInstruction(t *testing.T) {
	record := staticBaseRecord(t, "FEM", "Raiding Party")
	activated := record.Fields["Rules"].Args[1]
	producer := activated.Args[1].Args[0].Args[0]
	activated.Args[1] = Node{Kind: Tuple, Name: "ActionList", Args: []Node{{Kind: List, Args: []Node{producer}}}}
	factory := staticBaseSubrule(t, record, activated)
	a := &ronOptionalDiscardPlayer{BasePlayer: mage.NewBasePlayer("A"), count: 1}
	b := &ronOptionalDiscardPlayer{BasePlayer: mage.NewBasePlayer("B"), count: 2}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	sourceCard := factory()
	sourceCard.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || a.requests != 0 || b.requests != 0 || g.StackSize() != 0 {
		t.Fatal("failed sacrifice payment ran choices or queued a body")
	}
	orc := mage.NewCreature("Orc", "", 1, 1)
	orc.AddSubType("Orc")
	g.PutOnBattlefield(orc, a.PlayerID())
	members := make([]*mage.Permanent, 0, 4)
	for _, player := range []*mage.BasePlayer{a.BasePlayer, b.BasePlayer} {
		for range 2 {
			members = append(members, g.PutOnBattlefield(mage.NewCreature("White", "{W}", 1, 1), player.PlayerID()))
		}
	}
	red := g.PutOnBattlefield(mage.NewCreature("Red", "{R}", 1, 1), a.PlayerID())
	tapped := g.PutOnBattlefield(mage.NewCreature("Tapped", "{W}", 1, 1), a.PlayerID())
	g.TapPermanent(tapped)
	phased := g.PutOnBattlefield(mage.NewCreature("Phased", "{W}", 1, 1), b.PlayerID())
	g.PhaseOut(phased.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.requests != 0 || b.requests != 0 || len(a.Graveyard()) != 1 {
		t.Fatal("tap choices ran during payment")
	}
	g.DestroyPermanent(source)
	g.ResolveStack()
	count := 0
	for _, p := range members {
		if g.FindPermanent(p.ID()).Tapped {
			count++
		}
	}
	if count != 3 || red.Tapped || a.requests != 1 || b.requests != 1 {
		t.Fatal("real tap instruction", count, a.requests, b.requests)
	}
	if _, err := Compile(staticBaseRecord(t, "FEM", "Raiding Party")); err != nil {
		t.Fatal("complete procedure rejected", err)
	}
}
func TestRONPlayerTapResultsScopesCopiesClonesAndDelay(t *testing.T) {
	scopedRead := `PlayerActions(You,[GainLife(NumberOfPermanentsTappedThisWayByPlayer(ActionPlayer))])`
	factory := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([`+ronTapProducer+`,`+scopedRead+`,CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+ronTapRead+`]))]))`)
	a := &ronOptionalDiscardPlayer{BasePlayer: mage.NewBasePlayer("A"), count: 1}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	g.PutOnBattlefield(mage.NewCreature("One", "", 1, 1), a.PlayerID())
	g.PutOnBattlefield(mage.NewCreature("Two", "", 1, 1), a.PlayerID())
	card := factory()
	castRONSpell(t, g, a.BasePlayer, card)
	clone := g.Clone()
	if g.CopySpellOnStack(card.ID(), a.PlayerID(), false) == nil {
		t.Fatal("copy absent")
	}
	g.ResolveStack()
	if a.Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("copy or clone results")
	}
	g.SetStep(core.EndStep)
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatal("delayed frames lost counts", a.Life())
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 24 {
		t.Fatal("clone did not make its own maximum choice")
	}
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 24 || a.Life() != 24 {
		t.Fatal("clone lost its saved count or changed the original")
	}
	reset := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([`+ronTapProducer+`,`+ronTapRead+`,`+ronTapProducer+`,`+ronTapRead+`]))`)()
	castRONSpell(t, g, a.BasePlayer, reset)
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatal("empty producer reused old result")
	}
}
func TestRONPlayerTapResultsInvalid(t *testing.T) {
	for _, body := range []string{
		ronTapRead, ronTapRead + `,` + ronTapProducer,
		strings.Replace(ronTapProducer, "AnyPlayer", "Opponent", 1),
		strings.Replace(ronTapProducer, "SinglePlayer(ActionPlayer)", "SinglePlayer(You)", 1),
		ronTapProducer + `,PlayerAction(You,GainLife(NumberOfPermanentsTappedThisWayByPlayer(ActionPlayer)))`,
		`FlipACoin_OnLose([` + ronTapProducer + `]),` + ronTapRead,
		`EachPlayerAction(AnyPlayer,TapAnyNumberOfPermanents)`,
		`EachPlayerAction(AnyPlayer,TapAnyNumberOfPermanents(AnyPermanent))`,
		ronTapProducer + `,PlayerAction(You,GainLife(NumberOfPermanentsTappedThisWayByPlayer(You,Opponent)))`,
		strings.Replace(ronTapProducer, "And([IsUntapped", "Or([IsUntapped", 1),
		strings.Replace(ronTapProducer, "IsUntapped,", "ControlledByAPlayer(SinglePlayer(ActionPlayer)),", 1),
		strings.Replace(ronTapProducer, "ActionPlayer", "ActionPlayer()", 1),
		ronTapProducer + `,PlayerAction(You,GainLife(NumberOfPermanentsTappedThisWayByPlayer(EachablePlayer)))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + body + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-taps.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", body)
		}
	}
}
func TestRONPlayerTapResultsInventoryAndAtomicLoad(t *testing.T) {
	src := `Card(Name:"Taps",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + ronTapProducer + `,` + ronTapRead + `]))])`
	fs := fstest.MapFS{"taps.ron": {Data: []byte(src)}}
	report, err := InspectFS(fs, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatal(report.Issues)
	}
	found := false
	for _, entry := range report.Signatures {
		if entry.Name == "You" && strings.Contains(entry.Context, "NumberOfPermanentsTappedThisWayByPlayer") {
			found = true
			if entry.Role != "player_reference" {
				t.Fatal(entry.Role)
			}
		}
	}
	if !found {
		t.Fatal("missing typed result role")
	}
	fs["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(src, ronTapProducer+`,`, "", 1))}
	registry, err := LoadFS(fs, "*.ron")
	if err == nil || registry != nil {
		t.Fatal("partial registry published")
	}
}

func TestRONPlayerTapResultsActivationSources(t *testing.T) {
	for _, change := range []string{"phase", "return", "control"} {
		t.Run(change, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([`+ronTapProducer+`,`+ronTapRead+`]))`)
			a := &ronOptionalDiscardPlayer{BasePlayer: mage.NewBasePlayer("A"), count: 1}
			b := &ronOptionalDiscardPlayer{BasePlayer: mage.NewBasePlayer("B"), count: 2}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			for _, player := range []*mage.BasePlayer{a.BasePlayer, b.BasePlayer} {
				for range 2 {
					g.PutOnBattlefield(mage.NewCreature("Member", "", 1, 1), player.PlayerID())
				}
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if a.requests != 0 || b.requests != 0 {
				t.Fatal("choices ran at activation")
			}
			switch change {
			case "phase":
				g.PhaseOut(source.ID())
			case "return":
				g.BouncePermanentToHand(source)
				g.PutOnBattlefield(card, b.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.ResolveStack()
			if a.Life() != 21 || b.Life() != 20 || a.requests != 1 || b.requests != 1 {
				t.Fatal("source changed the saved controller or binding")
			}
		})
	}
	for _, rules := range []string{`Activated(` + ronTapProducer + `,ActionList([]))`, `EachPlayerEffect(AnyPlayer,[TapAnyNumberOfPermanents(AnyPermanent)])`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rules+`])`), "invalid-tap-context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("tap operation accepted in a nonresolving context")
		}
	}
}
