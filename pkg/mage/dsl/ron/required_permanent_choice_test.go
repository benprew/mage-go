package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronAuraHostChooser struct {
	*mage.BasePlayer
	choose func([]*mage.Permanent) *mage.Permanent
	calls  int
}

func (p *ronAuraHostChooser) ChoosePermanent(candidates []*mage.Permanent, _ string, _ mage.GameReader) *mage.Permanent {
	p.calls++
	return p.choose(candidates)
}

func TestRONRequiredPermanentChoice(t *testing.T) {
	for _, state := range []string{"legal", "shroud", "protection", "empty", "phased", "stale-aura", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			record := repositoryRecord(t, "LEG", "Takklemaggot", "choice.ron")
			if _, err := Compile(record); err != nil {
				t.Fatal(err)
			}
			rules := record.Fields["Rules"]
			death := rules.Args[2]
			actions := death.Args[1].Args[0]
			branch := actions.Args[1]
			branch.Args[2] = nameScopeNode(t, `Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([GainLife(Integer(3))]))])`).Fields["Rules"].Args[0].Args[1].Args[0]
			actions.Args[1] = branch
			death.Args[1].Args[0] = actions
			rules.Args = []Node{rules.Args[0], death}
			record.Fields["Rules"] = rules
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			g, a, base := ronGame()
			b := &ronAuraHostChooser{BasePlayer: base}
			b.choose = func(candidates []*mage.Permanent) *mage.Permanent { return candidates[len(candidates)-1] }
			g.SetPlayerAt(1, b)
			firstCard := mage.NewCreature("First", "", 2, 2)
			firstCard.SetOwner(b.PlayerID())
			first := g.PutOnBattlefield(firstCard, b.PlayerID())
			var next *mage.Permanent
			if state != "empty" {
				c := mage.NewCreature("Next", "", 2, 2)
				if state == "protection" {
					c = mage.NewCreature("Protected", "", 2, 2, mage.WithAbility(mage.ProtectionFromColor(core.Black)))
				}
				next = g.PutOnBattlefield(c, a.PlayerID())
				if state == "shroud" {
					next.GrantBaseAttr(core.Shroud)
				}
				if state == "phased" {
					g.PhaseOut(next.ID())
				}
				if state == "legal" {
					g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 2), a.PlayerID())
				}
			}
			card := factory()
			card.SetOwner(b.PlayerID())
			aura := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(aura.ID(), first.ID())
			g.DestroyPermanent(first)
			g.CheckStateBasedActions()
			g.PutTriggersOnStack()
			if b.calls != 0 {
				t.Fatal("choice before resolution")
			}
			if state == "stale-aura" {
				c, _ := b.RemoveFromGraveyard(card.ID())
				b.AddToGraveyard(c)
			}
			if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			run := g
			if state == "clone" {
				run = g.Clone()
			}
			run.ResolveStack()
			failed := state == "empty" || state == "protection" || state == "phased" || state == "stale-aura"
			got := run.FindPermanent(card.ID())
			if failed {
				if got != nil || run.GetPlayer(a.PlayerID()).Life() != 23 {
					t.Fatal("failure branch lost")
				}
				return
			}
			if got == nil || got.ControllerID() != a.PlayerID() || got.Card.Owner() != b.PlayerID() {
				t.Fatal("wrong returned Aura")
			}
			if state == "legal" {
				if b.calls != 1 || got.AttachedTo == next.ID() {
					t.Fatal("wrong chooser")
				}
			} else if got.AttachedTo != next.ID() {
				t.Fatal("wrong host")
			}
			if state == "clone" && g.FindPermanent(card.ID()) != nil {
				t.Fatal("clone changed original")
			}
			if state == "copy" && a.Life() != 23 {
				t.Fatal("copy reused moved card")
			}
		})
	}
}

func TestRONRequiredPermanentChoiceBranches(t *testing.T) {
	for _, condition := range []string{"If(CostWasPaid,[TapPermanent(TheChosenPermanent)])", "Unless(CostWasntPaid,[TapPermanent(TheChosenPermanent)])", "IfElse(CostWasntPaid,[GainLife(Integer(3))],[TapPermanent(TheChosenPermanent)])"} {
		factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([PlayerMustCost(You,ChooseAPermanent(IsCardtype(Creature))),`+condition+`]))`)
		g, a, b := ronGame()
		host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 2), b.PlayerID())
		p := g.PutOnBattlefield(factory(), a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if !host.Tapped {
			t.Fatal("lost success binding")
		}
	}
}

func TestRONRequiredPermanentChoiceInvalid(t *testing.T) {
	for _, body := range []string{
		`PlayerMustCost(You,ChooseAPermanent())`,
		`PlayerMustCost(You,ChooseAPermanent(IsCardtype(Creature))),If(CostWasntPaid,[TapPermanent(TheChosenPermanent)])`,
		`PlayerMustCost(You,ChooseAPermanent(IsCardtype(Creature))),If(CostWasPaid,[GainLife(Integer(1))]),TapPermanent(TheChosenPermanent)`,
		`PlayerMustCost(You,ChooseAPermanent(IsCardtype(Creature))),GainLife(Integer(1)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`PlayerMustCost(You,ChooseAPermanent(IsCardtype(Creature))),If(CostWasPaid(),[GainLife(Integer(1))])`,
		`PlayerMustCost(You,ChooseAPermanent(IsCardtype(Creature))),IfElse(CostWasPaid,[],[GainLife(Integer(1))])`,
		`PlayerMustCost(You,ChooseAPermanent(And([IsCardtype(Creature),CanBeEnchantedByDeadGuest]))),If(CostWasPaid,[GainLife(Integer(1))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONRequiredPermanentChoiceNested(t *testing.T) {
	body := `PlayerMustCost(You,ChooseAPermanent(IsCardtype(Creature))),IfElse(CostWasPaid,[PlayerMustCost(You,ChooseAPermanent(IsCardtype(Land))),IfElse(CostWasPaid,[GainLife(Integer(100))],[GainLife(Integer(2))]),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([TapPermanent(TheChosenPermanent)]))],[GainLife(Integer(100))]),PlayerMustCost(You,ChooseAPermanent(IsCardtype(Land))),IfElse(CostWasPaid,[GainLife(Integer(100))],[GainLife(Integer(3))])`
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([`+body+`]))`)
	g, a, b := ronGame()
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 2), b.PlayerID())
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 25 || host.Tapped {
		t.Fatal("nested result leaked or delayed body ran early")
	}
	g.RemoveFromBattlefield(host)
	returned := g.PutOnBattlefield(host.Card, b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if returned.Tapped {
		t.Fatal("saved choice followed a new visit")
	}
}

func TestRONRequiredPermanentChoiceInvalidAuraFilter(t *testing.T) {
	for _, filter := range []string{
		`CanBeEnchantedByDeadGuest`,
		`And([IsCardtype(Creature),CanBeEnchantedByDeadGuest()])`,
		`And([CanBeEnchantedByDeadGuest,IsCardtype(Creature)])`,
		`And([IsCardtype(Creature),CanBeEnchantedByDeadGuest,AnyPermanent])`,
		`Or([IsCardtype(Creature),CanBeEnchantedByDeadGuest])`,
	} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),TriggerA(WhenAPermanentDies(SinglePermanent(HostPermanent)),ActionList([PlayerMustCost(ControllerOfDeadPermanent,ChooseAPermanent(`+filter+`)),If(CostWasPaid,[GainLife(Integer(1))])]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", filter)
		}
	}
	for _, rule := range []string{
		`AsPermanentEnters(ThisPermanent,[PlayerMustCost(You,ChooseAPermanent(IsCardtype(Creature))),If(CostWasPaid,[GainLife(Integer(1))])])`,
		`ReplaceWouldDraw(PlayerWouldDrawDuringTheirDrawStep(You),[PlayerMustCost(You,ChooseAPermanent(IsCardtype(Creature))),If(CostWasPaid,[GainLife(Integer(1))])])`,
	} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted context %s", rule)
		}
	}
}
