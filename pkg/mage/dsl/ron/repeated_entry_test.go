package ron

import (
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronRepeatedEntryPlayer struct {
	*ronHandEntryPlayer
	answers  []bool
	order    *[]string
	onChoice func()
}

func (p *ronRepeatedEntryPlayer) ChooseMayAbility(string) bool {
	if p.order != nil {
		*p.order = append(*p.order, p.Name())
	}
	if len(p.answers) == 0 {
		return false
	}
	answer := p.answers[0]
	p.answers = p.answers[1:]
	return answer
}
func (p *ronRepeatedEntryPlayer) ChooseCardFromHand(cards []mage.Card, _ string, _ mage.GameReader) mage.Card {
	p.choices++
	if p.onChoice != nil {
		p.onChoice()
	}
	if len(cards) > 0 {
		return cards[0]
	}
	return nil
}
func ronRepeatedEntryGame() (*mage.Game, *ronRepeatedEntryPlayer, *ronRepeatedEntryPlayer) {
	a := &ronRepeatedEntryPlayer{ronHandEntryPlayer: &ronHandEntryPlayer{BasePlayer: mage.NewBasePlayer("A")}}
	b := &ronRepeatedEntryPlayer{ronHandEntryPlayer: &ronHandEntryPlayer{BasePlayer: mage.NewBasePlayer("B")}}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	return g, a, b
}
func TestRONRepeatedEntryEureka(t *testing.T) {
	g, a, b := ronRepeatedEntryGame()
	var order []string
	a.order, b.order = &order, &order
	a.answers, b.answers = []bool{false, true, false}, []bool{true, false, false}
	for _, p := range []*ronRepeatedEntryPlayer{a, b} {
		for range 2 {
			land := repositoryCard(t, "LEA", "Forest")
			land.SetOwner(p.PlayerID())
			p.AddToHand(land)
		}
	}
	card := repositoryCard(t, "LEG", "Eureka")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast")
	}
	if len(order) != 0 || g.StackSize() != 0 || len(a.Hand()) != 3 {
		t.Fatal("payment rollback")
	}
	a.ManaPool().Add(core.Green, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if len(order) != 0 || len(g.AllBattlefield()) != 0 {
		t.Fatal("announcement made choices")
	}
	g.SetActivePlayerIndex(1)
	g.ResolveStack()
	if !reflect.DeepEqual(order, []string{"A", "B", "A", "B", "A", "B"}) {
		t.Fatal(order)
	}
	if len(a.Hand()) != 1 || len(b.Hand()) != 1 || len(g.AllBattlefield()) != 2 || len(a.Graveyard()) != 1 {
		t.Fatal("entries or termination")
	}
	if g.GetLandsPlayedThisTurn() != 0 {
		t.Fatal("entry counted as land play")
	}
}
func TestRONRepeatedEntryCopyAndClone(t *testing.T) {
	for _, scenario := range []string{"copy", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronRepeatedEntryGame()
			var order []string
			a.order, b.order = &order, &order
			a.answers, b.answers = []bool{true, false}, []bool{true, false}
			for _, p := range []*ronRepeatedEntryPlayer{a, b} {
				land := repositoryCard(t, "LEA", "Forest")
				land.SetOwner(p.PlayerID())
				p.AddToHand(land)
			}
			card := repositoryCard(t, "LEG", "Eureka")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Green, 4)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if scenario == "copy" {
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
				g.ResolveStack()
				if !reflect.DeepEqual(order, []string{"B", "A"}) || len(g.AllBattlefield()) != 2 {
					t.Fatal(order)
				}
			} else {
				clone := g.Clone()
				for i, p := range []*ronRepeatedEntryPlayer{a, b} {
					base := clone.GetPlayer(p.PlayerID()).(*mage.SearchPlayer).BasePlayer
					clone.SetPlayerAt(i, &ronRepeatedEntryPlayer{ronHandEntryPlayer: &ronHandEntryPlayer{BasePlayer: base}, answers: []bool{true}})
				}
				clone.ResolveStack()
				if len(clone.AllBattlefield()) != 2 || len(g.AllBattlefield()) != 0 || len(a.Hand()) != 1 || len(b.Hand()) != 1 {
					t.Fatal("clone isolation")
				}
				g.ResolveStack()
				if len(g.AllBattlefield()) != 2 {
					t.Fatal("original stopped")
				}
			}
		})
	}
}
func TestRONRepeatedEntryLegalAuraAndFreshHand(t *testing.T) {
	g, a, b := ronRepeatedEntryGame()
	a.answers, b.answers = []bool{false, true}, []bool{true}
	aura := repositoryCard(t, "LEA", "Holy Strength")
	aura.SetOwner(a.PlayerID())
	a.AddToHand(aura)
	creature := repositoryCard(t, "LEA", "Grizzly Bears")
	creature.SetOwner(b.PlayerID())
	b.AddToHand(creature)
	card := repositoryCard(t, "LEG", "Eureka")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Green, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	host := g.FindPermanent(creature.ID())
	attached := g.FindPermanent(aura.ID())
	if host == nil || attached == nil || attached.AttachedTo != host.ID() || host.CurrentPower(g) != 3 {
		t.Fatal("later round did not see prior entry")
	}
}
func TestRONRepeatedEntryRejects(t *testing.T) {
	producer := `EachPlayerStartingWithMayAction(AnyPlayer,You,PutACardFromHandOnBattlefield(IsPermanent,[EntersNormally]))`
	continuation := `Unless(NoOneTookAnActionThisWay,[RepeatThisProcess])`
	for _, action := range []string{
		`RepeatThisProcess`, `RepeatableActions([])`, `RepeatableActions([` + continuation + `])`,
		`RepeatableActions([` + continuation + `,` + producer + `])`,
		`RepeatableActions([` + producer + `])`,
		`RepeatableActions([` + producer + `,Unless(AllCoinsCameUpHeads,[RepeatThisProcess])])`,
		`RepeatableActions([` + producer + `,Unless(NoOneTookAnActionThisWay,[RepeatThisProcess()])])`,
		`RepeatableActions([` + producer + `,Unless(NoOneTookAnActionThisWay,[RepeatThisProcess,RepeatThisProcess])])`,
		`RepeatableActions([EachPlayerStartingWithMayAction(AnyPlayer,ActionPlayer,PutACardFromHandOnBattlefield(IsPermanent,[EntersNormally])),` + continuation + `])`,
		`RepeatableActions([EachPlayerStartingWithMayAction(Opponent,You,PutACardFromHandOnBattlefield(IsPermanent,[EntersNormally])),` + continuation + `])`,
		`RepeatableActions([EachPlayerStartingWithMayAction(AnyPlayer,You,PutACardFromHandOnBattlefield(IsPermanent,[EntersTapped])),` + continuation + `])`,
		producer, `Unless(NoOneTookAnActionThisWay,[DrawCard])`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([` + action + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", action)
		}
	}
}
func TestRONRepeatedEntryStrictLoadAndRemainingGap(t *testing.T) {
	node := hostUntapRecord(t, "LEG", "Eureka")
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	if err = mage.ValidateCardDefinition("Eureka", factory()); err != nil {
		t.Fatal(err)
	}
	if err = mage.ValidateCardDefinition("Eureka", factory()); err != nil {
		t.Fatal(err)
	}
	data := `Card(Name:"Eureka",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),ManaCost:[ManaCostGeneric(2),ManaCostG,ManaCostG],Rules:[SpellActions(ActionList([RepeatableActions([EachPlayerStartingWithMayAction(AnyPlayer,You,PutACardFromHandOnBattlefield(IsPermanent,[EntersNormally])),Unless(NoOneTookAnActionThisWay,[RepeatThisProcess])])]))])`
	files := fstest.MapFS{"one.ron": &fstest.MapFile{Data: []byte(data)}}
	if registry, err := LoadFS(files, "*.ron"); err != nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal(err)
	}
	files["gap.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Gap",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[UnknownRule])`)}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("partial collection")
	}
	if _, err := Compile(hostUntapRecord(t, "DRK", "Mana Clash")); err != nil {
		t.Fatal(err)
	}
}

func TestRONRepeatedEntryDeferredTriggersAndProhibition(t *testing.T) {
	for _, prohibited := range []bool{false, true} {
		t.Run(map[bool]string{false: "pending", true: "prohibited"}[prohibited], func(t *testing.T) {
			g, a, b := ronRepeatedEntryGame()
			a.answers, b.answers = []bool{true}, []bool{true}
			for _, p := range []*ronRepeatedEntryPlayer{a, b} {
				land := repositoryCard(t, "LEA", "Forest")
				land.SetOwner(p.PlayerID())
				p.AddToHand(land)
			}
			g.PutOnBattlefield(repositoryCard(t, "LEA", "Ankh of Mishra"), a.PlayerID())
			if prohibited {
				g.PutOnBattlefield(mage.NewEnchantment("Test land entry prohibition", "", mage.WithStaticAbility(mage.WormsOfTheEarthEffect())), a.PlayerID())
			}
			card := repositoryCard(t, "LEG", "Eureka")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Green, 4)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			a.onChoice = func() {
				if a.Life() != 20 || b.Life() != 20 {
					t.Fatal("trigger resolved during a round")
				}
			}
			b.onChoice = a.onChoice
			g.ResolveTopOfStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("trigger resolved inside procedure")
			}
			if prohibited {
				if len(a.Hand()) != 1 || len(b.Hand()) != 1 || g.StackSize() != 0 || a.choices != 1 || b.choices != 1 {
					t.Fatal("failed entry repeated or moved cards")
				}
			} else {
				if g.StackSize() != 2 || len(a.Hand()) != 0 || len(b.Hand()) != 0 {
					t.Fatal("pending entry events")
				}
				g.ResolveStack()
				if a.Life() != 18 || b.Life() != 18 {
					t.Fatal("separate controllers or damage triggers")
				}
			}
		})
	}
}

func TestRONRepeatedEntrySiblingInstancesAndInventory(t *testing.T) {
	process := `RepeatableActions([EachPlayerStartingWithMayAction(AnyPlayer,You,PutACardFromHandOnBattlefield(IsPermanent,[EntersNormally])),Unless(NoOneTookAnActionThisWay,[RepeatThisProcess])])`
	text := `Card(Name:"Two processes",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([` + process + `,` + process + `]))])`
	files := fstest.MapFS{"sibling.ron": &fstest.MapFile{Data: []byte(text)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronRepeatedEntryGame()
	a.answers = []bool{false, true}
	land := repositoryCard(t, "LEA", "Forest")
	land.SetOwner(a.PlayerID())
	a.AddToHand(land)
	card, err := registry.CreateCard("Two processes")
	if err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(land.ID()) == nil || a.choices != 1 {
		t.Fatal("sibling process results shared")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]bool{}
	for _, s := range report.Signatures {
		if s.Name == "PutACardFromHandOnBattlefield" && s.Role == "action" {
			roles["action"] = true
		}
		if s.Name == "IsPermanent" && s.Role == "hand_entry_filter" {
			roles["filter"] = true
		}
		if s.Name == "You" && s.Role == "player_reference" {
			roles["player"] = true
		}
		if s.Name == "NoOneTookAnActionThisWay" && s.Role == "condition" {
			roles["condition"] = true
		}
	}
	if len(roles) != 4 {
		t.Fatal("missing procedure roles", roles)
	}
}

func TestRONRepeatedEntryBoundStart(t *testing.T) {
	src := `Card(Name:"Bound round",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPlayer(Opponent)],ActionList([RepeatableActions([EachPlayerStartingWithMayAction(AnyPlayer,Ref_TargetPlayer,PutACardFromHandOnBattlefield(IsPermanent,[EntersNormally])),Unless(NoOneTookAnActionThisWay,[RepeatThisProcess])])])))] )`
	nodes, err := Parse(strings.NewReader(src), "bound.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronRepeatedEntryGame()
	var order []string
	a.order, b.order = &order, &order
	a.answers, b.answers = []bool{true}, []bool{true}
	for _, p := range []*ronRepeatedEntryPlayer{a, b} {
		land := repositoryCard(t, "LEA", "Forest")
		land.SetOwner(p.PlayerID())
		p.AddToHand(land)
	}
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !reflect.DeepEqual(order, []string{"B", "A"}) || len(g.AllBattlefield()) != 2 {
		t.Fatal("bound starting player", order)
	}
}
