package ron

import (
	"maps"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronChosenCostPlayer struct {
	*mage.BasePlayer
	accept          bool
	invalid         bool
	choices, offers int
}

func (p *ronChosenCostPlayer) ChooseMayAbility(string) bool {
	p.offers++
	return p.accept
}

func (p *ronChosenCostPlayer) ChooseCardsFromHand(n int, _ string, _ mage.GameReader) []mage.Card {
	p.choices++
	if p.invalid {
		return []mage.Card{mage.NewLand("Foreign")}
	}
	hand := p.Hand()
	return append([]mage.Card(nil), hand[len(hand)-min(n, len(hand)):]...)
}

func chosenDiscardRecord(t *testing.T, set, name string) Node {
	t.Helper()
	data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), set+".ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range nodes {
		if node.Fields["Name"].String == name {
			return node
		}
	}
	t.Fatal("missing record", name)
	return Node{}
}

func TestRONChosenDiscardCostArmorerClause(t *testing.T) {
	n := chosenDiscardRecord(t, "FEM", "Dwarven Armorer")
	rule := n.Fields["Rules"].Args[0]
	cost := rule.Args[0]
	targeted := rule.Args[1]
	choice := targeted.Args[1].Args[0].Args[0]
	for _, mode := range []string{"pay", "mana", "empty", "invalid", "illegal", "phase", "returned", "copy", "source-left"} {
		t.Run(mode, func(t *testing.T) {
			// Each real choice body can run after its choice adapter is added.
			body := Node{Kind: Tuple, Name: "ActionList", Args: []Node{choice.Args[0].Args[0]}}
			branch := targeted
			branch.Args = []Node{targeted.Args[0], body}
			clause := rule
			clause.Args = []Node{cost, branch}
			node := n
			node.Fields = make(map[string]Node, len(n.Fields))
			maps.Copy(node.Fields, n.Fields)
			node.Fields["Rules"] = Node{Kind: List, Args: []Node{clause}}
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			if err = mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			a := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true, invalid: mode == "invalid"}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
			targetCard := mage.NewCreature("Target", "", 2, 3)
			targetCard.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			first, last := mage.NewLand("First"), mage.NewLand("Last")
			first.SetOwner(a.PlayerID())
			last.SetOwner(a.PlayerID())
			if mode != "empty" {
				a.AddToHand(first)
				a.AddToHand(last)
			}
			mana := 1
			if mode == "mana" {
				mana = 0
			}
			a.ManaPool().Add(core.Red, mana)
			id := target.ID()
			if mode == "illegal" {
				id = b.PlayerID()
			}
			err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{id})
			if mode == "mana" || mode == "empty" || mode == "invalid" || mode == "illegal" {
				wantHand := 2
				if mode == "empty" {
					wantHand = 0
				}
				if err == nil || a.ManaPool().TotalMana() != mana || len(a.Hand()) != wantHand || len(a.Graveyard()) != 0 || g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
					t.Fatal("failed cost changed resources")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(a.Hand()) != 1 || a.Hand()[0].ID() != first.ID() || len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != last.ID() || a.choices != 1 || !g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 0 || target.CurrentToughness(g) != 3 {
				t.Fatal("wrong choice or payment timing")
			}
			switch mode {
			case "phase":
				g.PhaseOut(target.ID())
			case "returned":
				g.RemoveFromBattlefield(target)
				target = g.PutOnBattlefield(targetCard, b.PlayerID())
			case "source-left":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			want := 3
			if mode != "phase" && mode != "returned" {
				want++
				if mode == "copy" {
					want++
				}
			}
			if g.FindPermanentIncludingPhased(target.ID()).CurrentToughness(g) != want || clone.FindPermanentIncludingPhased(target.ID()).CurrentToughness(clone) != 3 || len(a.Graveyard()) != 1 || a.choices != 1 {
				t.Fatal("target identity, clone or repeated payment")
			}
			clone.ResolveStack()
			if clone.FindPermanentIncludingPhased(target.ID()).CurrentToughness(clone) != want {
				t.Fatal("clone lost pending action")
			}
		})
	}
}

func TestRONChosenDiscardCostOptional(t *testing.T) {
	for _, branch := range []string{"If(CostWasPaid", "Unless(CostWasntPaid", "Unless(CostWasPaid", "If(CostWasntPaid"} {
		for _, mode := range []string{"pay", "decline", "empty", "invalid", "control", "returned", "phased", "copy"} {
			t.Run(branch+"/"+mode, func(t *testing.T) {
				a := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: mode != "decline", invalid: mode == "invalid"}
				b := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
				g := mage.NewGame(a, b)
				factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([MayCost(DiscardACard),`+branch+`,[GainLife(Integer(2))])]))`)
				card := factory()
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				if mode != "empty" {
					c := mage.NewLand("A card")
					c.SetOwner(a.PlayerID())
					a.AddToHand(c)
				}
				c := mage.NewLand("B card")
				c.SetOwner(b.PlayerID())
				b.AddToHand(c)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if a.choices != 0 || a.offers != 0 {
					t.Fatal("resolution payment made early")
				}
				switch mode {
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "returned":
					g.RemoveFromBattlefield(p)
					g.PutOnBattlefield(card, b.PlayerID())
				case "phased":
					g.PhaseOut(p.ID())
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				g.ResolveStack()
				paid := mode != "decline" && mode != "empty" && mode != "invalid"
				paidBranch := branch == "If(CostWasPaid" || branch == "Unless(CostWasntPaid"
				want := 20
				if paid == paidBranch {
					want = 22
				}
				grave := 0
				if paid {
					grave = 1
				}
				if a.Life() != want || len(a.Graveyard()) != grave || clone.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("wrong optional outcome or clone mutation")
				}
				if mode == "copy" {
					wantB := 20
					if paidBranch {
						wantB = 22
					}
					if b.Life() != wantB || len(b.Graveyard()) != 1 {
						t.Fatal("copy payer followed original")
					}
				} else if b.offers != 0 || b.choices != 0 {
					t.Fatal("payment followed source controller")
				}
			})
		}
	}
}

func TestRONChosenDiscardCostInvalid(t *testing.T) {
	for _, rule := range []string{
		`Activated(DiscardACard(),ActionList([GainLife(Integer(1))]))`,
		`Activated(DiscardACard(Integer(1)),ActionList([GainLife(Integer(1))]))`,
		`Activated(DiscardNumberCards(Integer(1)),ActionList([GainLife(Integer(1))]))`,
		`Activated(PayMana([]),ActionList([MayCost(DiscardACard)]))`,
		`Activated(PayMana([]),ActionList([MayCost(DiscardACard()),If(CostWasPaid,[GainLife(Integer(1))])]))`,
		`Activated(PayMana([]),ActionList([MayCost(DiscardACard),GainLife(Integer(1)),If(CostWasPaid,[GainLife(Integer(1))])]))`,
		`Activated(PayMana([]),ActionList([PlayerMayCost(Ref_TargetPlayer,DiscardACard),If(CostWasPaid,[GainLife(Integer(1))])]))`,
		`Activated(PayMana([]),ActionList([MustCost(DiscardACard),If(CostWasPaid,[GainLife(Integer(1))])]))`,
		`Activated(PayMana([]),ActionList([If(CostWasPaid,[GainLife(Integer(1))])]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", rule)
		}
	}
}

func TestRONChosenDiscardCostWarMachineClause(t *testing.T) {
	n := chosenDiscardRecord(t, "ATQ", "Mishra's War Machine")
	rule := n.Fields["Rules"].Args[1]
	actions := rule.Args[1].Args[0].Args
	continuation := actions[1]
	continuation.Args = []Node{continuation.Args[0], {Kind: List, Args: continuation.Args[1].Args[:1]}}
	clause := rule
	clause.Args = []Node{rule.Args[0], {Kind: Tuple, Name: "ActionList", Args: []Node{{Kind: List, Args: []Node{actions[0], continuation}}}}}
	for _, mode := range []string{"pay", "decline", "empty", "prevent"} {
		t.Run(mode, func(t *testing.T) {
			node := n
			node.Fields = make(map[string]Node, len(n.Fields))
			maps.Copy(node.Fields, n.Fields)
			node.Fields["Rules"] = Node{Kind: List, Args: []Node{n.Fields["Rules"].Args[0], clause}}
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			a := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: mode != "decline"}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			if p.CurrentPower(g) != 5 || !p.HasAttr(core.Banding) {
				t.Fatal("real metadata lost")
			}
			if mode != "empty" {
				c := mage.NewLand("Chosen")
				c.SetOwner(a.PlayerID())
				a.AddToHand(c)
			}
			if mode == "prevent" {
				if err := mage.ApplyEffect(g, mage.PreventDamageThisTurn(mage.You, mage.AnyPermanent), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				a.accept = false
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep triggered")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || a.choices != 0 {
				t.Fatal("wrong payment timing")
			}
			g.ResolveStack()
			life := 17
			grave := 0
			if mode == "pay" {
				life = 20
				grave = 1
			}
			if mode == "prevent" {
				life = 20
			}
			if a.Life() != life || len(a.Graveyard()) != grave {
				t.Fatal("real unpaid damage clause")
			}
		})
	}
}

func TestRONChosenDiscardCostIndependentPayers(t *testing.T) {
	a := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
	g := mage.NewGame(a, b)
	c := mage.NewLand("A card")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	d := mage.NewLand("B card")
	d.SetOwner(b.PlayerID())
	b.AddToHand(d)
	rule := `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerMayCost(Ref_TargetPlayer,DiscardACard),If(CostWasPaid,[MayCost(DiscardACard),Unless(CostWasPaid,[GainLife(Integer(8))]),GainLife(Integer(2))]),MayCost(DiscardACard),Unless(CostWasPaid,[GainLife(Integer(1))])])))`
	p := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", rule)(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(a.Graveyard()) != 1 || len(b.Graveyard()) != 1 || a.Life() != 23 || b.Life() != 20 || a.choices != 1 || b.choices != 1 {
		t.Fatal("nested payments shared payer or result")
	}
}

func TestRONChosenDiscardCostStrictBoundaries(t *testing.T) {
	for _, tc := range []struct{ set, name, gap string }{
		{"ATQ", "Mishra's War Machine", "activation history requires the resolving activated ability"},
		{"LEG", "Land's Edge", "DiscardedCardPassesFilter"},
	} {
		n := chosenDiscardRecord(t, tc.set, tc.name)
		if _, err := Compile(n); err == nil || !strings.Contains(err.Error(), tc.gap) {
			t.Fatalf("%s boundary: %v", tc.name, err)
		}
	}
	src := `Card(Name:"Supported",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(DiscardACard,ActionList([GainLife(Integer(1))]))])`
	files := fstest.MapFS{"valid.ron": {Data: []byte(src)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Supported")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Supported")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID() == second.ID() || first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("shared factory state")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(src, "DiscardACard,", "DiscardACard(),", 1))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Supported") {
		t.Fatal("invalid collection installed")
	}
	report, err := InspectFS(fstest.MapFS{"valid.ron": files["valid.ron"]}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range report.Signatures {
		if entry.Name == "DiscardACard" && entry.Role == "cost" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing cost role")
	}
}

type ronPreventChosenDiscard struct{ source uuid.UUID }

func (r *ronPreventChosenDiscard) SourceID() uuid.UUID         { return r.source }
func (*ronPreventChosenDiscard) GetDuration() core.Duration    { return core.Indefinite }
func (*ronPreventChosenDiscard) IsActive(mage.GameReader) bool { return true }
func (*ronPreventChosenDiscard) Matches(action mage.Action, _ mage.GameReader) bool {
	discard, ok := action.(*mage.DiscardAction)
	return ok && !discard.CausedByEffect()
}
func (*ronPreventChosenDiscard) Replace(mage.Action, *mage.Game) mage.Action { return nil }
func (r *ronPreventChosenDiscard) Clone() mage.ReplacementEffect {
	return &ronPreventChosenDiscard{source: r.source}
}

func TestRONChosenDiscardCostReplacementsAndInstances(t *testing.T) {
	for _, mode := range []string{"two", "effect-only", "prevented", "optional-prevented"} {
		t.Run(mode, func(t *testing.T) {
			a := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			g := mage.NewGame(a, mage.NewBasePlayer("B"))
			rule := `ActivatedWithModifiers(And([PayMana([ManaCostGeneric(1)]),TapPermanent(ThisPermanent),DiscardACard]),ActionList([GainLife(Integer(2))]),ActivateOnlyOnceEachTurn)`
			if mode == "optional-prevented" {
				rule = `Activated(PayMana([]),ActionList([MayCost(DiscardACard),Unless(CostWasPaid,[GainLife(Integer(2))])]))`
			}
			factory := permanentScopeFactory(t, "Artifact", rule)
			one := g.PutOnBattlefield(factory(), a.PlayerID())
			two := g.PutOnBattlefield(factory(), a.PlayerID())
			for _, label := range []string{"First", "Last"} {
				c := mage.NewLand(label)
				c.SetOwner(a.PlayerID())
				a.AddToHand(c)
			}
			a.ManaPool().Add(core.Colorless, 2)
			if mode == "effect-only" {
				g.PutOnBattlefield(mage.NewArtifact("Replacement", "", mage.WithStaticAbility(mage.DiscardToLibraryReplacement())), a.PlayerID())
			}
			if mode == "prevented" || mode == "optional-prevented" {
				g.AddReplacementEffect(&ronPreventChosenDiscard{source: two.ID()})
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), one.ID(), 0, nil)
			if mode == "prevented" {
				if err == nil || len(a.Hand()) != 2 || len(a.Graveyard()) != 0 || a.ManaPool().TotalMana() != 2 || g.FindPermanent(one.ID()).Tapped || g.StackSize() != 0 {
					t.Fatal("replacement failure spent resources")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if mode == "two" {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), two.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			if mode == "optional-prevented" {
				if len(a.Hand()) != 2 || len(a.Graveyard()) != 0 || a.Life() != 22 {
					t.Fatal("prevented optional cost ran paid branch")
				}
				return
			}
			grave, life := 1, 22
			if mode == "two" {
				grave = 2
				life = 24
			}
			if len(a.Graveyard()) != grave || len(a.Hand()) != 2-grave || len(a.Library()) != 0 || a.Life() != life {
				t.Fatal("instances shared choice or effect-only replacement applied to cost")
			}
			g.UntapPermanent(g.FindPermanent(one.ID()))
			beforeMana := a.ManaPool().TotalMana()
			if err := g.ActivateAbilityByIndex(a.PlayerID(), one.ID(), 0, nil); err == nil || a.ManaPool().TotalMana() != beforeMana || len(a.Graveyard()) != grave {
				t.Fatal("repeat bypassed activation modifier")
			}
		})
	}
}
