package ron

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronChosenManaPlayer struct {
	mage.Player
	amount int
	calls  int
}

func (p *ronChosenManaPlayer) ChooseNonnegativeNumber(_ string) int {
	p.calls++
	return p.amount
}
func (p *ronChosenManaPlayer) ChooseMayAbility(string) bool { panic("extra offer") }

func TestRONChosenManaPaymentPowerLeak(t *testing.T) {
	for _, amount := range []int{0, 1, 2, 3, 5} {
		t.Run(strconv.Itoa(amount), func(t *testing.T) {
			a := &ronChosenManaPlayer{Player: mage.NewBasePlayer("A"), amount: 0}
			b := &ronChosenManaPlayer{Player: mage.NewBasePlayer("B"), amount: amount}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			host := g.PutOnBattlefield(mage.NewEnchantment("host", ""), b.PlayerID())
			c := repositoryCard(t, "LEA", "Power Leak")
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
				t.Fatal("unpaid cast")
			}
			a.ManaPool().Add(core.Blue, 2)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			b.ManaPool().Add(core.Green, 3)
			queueRONUpkeep(g, a)
			if g.StackSize() != 0 {
				t.Fatal("wrong upkeep")
			}
			queueRONUpkeep(g, b)
			if b.calls != 0 {
				t.Fatal("early payment")
			}
			g.ResolveStack()
			paid := amount
			if amount > 3 {
				paid = 0
			}
			if b.Life() != 20-max(0, 2-paid) || b.ManaPool().TotalMana() != 3-paid || a.calls != 0 || b.calls != 1 {
				t.Fatalf("life=%d mana=%d choices=%d/%d", b.Life(), b.ManaPool().TotalMana(), a.calls, b.calls)
			}
		})
	}
}

func TestRONChosenManaPaymentInvalid(t *testing.T) {
	producer := `PlayerAction(You,PayAnyAmountOfMana)`
	consumer := `PermanentDealsDamageAndPreventSomeOfIt(ThisPermanent,Integer(2),Player(You),AmountOfManaPaidThisWay)`
	for _, body := range []string{
		consumer,
		consumer + "," + producer,
		producer + "," + strings.Replace(consumer, "Player(You)", "EachPlayer(AnyPlayer)", 1),
		producer + "," + strings.Replace(consumer, "ThisPermanent", "Ref_TargetPermanent", 1),
		producer + "," + strings.Replace(consumer, "Integer(2)", "Integer()", 1),
		producer + "," + strings.Replace(consumer, "AmountOfManaPaidThisWay", "AmountOfManaPaidThisWay()", 1),
		strings.Replace(producer, "PayAnyAmountOfMana", "PayAnyAmountOfMana()", 1) + "," + consumer,
		`PlayerAction(ActionPlayer,PayAnyAmountOfMana),` + consumer,
		`If(PlayerPassesFilter(You,ControlsA(IsCardtype(Land))),[` + producer + `]),` + consumer,
	} {
		source := `Card(Name:"invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + body + `]))])`
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONChosenManaPaymentPowerLeakLifecycle(t *testing.T) {
	for _, state := range []string{"host-control-before", "host-control-after", "reattach", "source-control", "departed", "returned", "phased", "cycle", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			a := &ronChosenManaPlayer{Player: mage.NewBasePlayer("A"), amount: 0}
			b := &ronChosenManaPlayer{Player: mage.NewBasePlayer("B"), amount: 1}
			g := mage.NewGame(a, b)
			host := g.PutOnBattlefield(mage.NewEnchantment("host", ""), b.PlayerID())
			other := g.PutOnBattlefield(mage.NewEnchantment("other", ""), a.PlayerID())
			c := repositoryCard(t, "LEA", "Power Leak")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.Attach(p.ID(), host.ID())
			if state == "host-control-before" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				queueRONUpkeep(g, b)
				if g.StackSize() != 0 {
					t.Fatal("used former host controller")
				}
				queueRONUpkeep(g, a)
			} else {
				queueRONUpkeep(g, b)
			}
			if g.StackSize() != 1 {
				t.Fatal("missing host upkeep trigger")
			}
			switch state {
			case "host-control-after":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "reattach":
				g.Attach(p.ID(), other.ID())
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: host.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "departed", "returned":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if state == "returned" {
					a.RemoveFromHand(c.ID())
					q := g.PutOnBattlefield(c, b.PlayerID())
					g.Attach(q.ID(), other.ID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			b.ManaPool().Add(core.Green, 1)
			clone := g.Clone()
			clone.SetPlayerAt(0, &ronChosenManaPlayer{Player: clone.GetPlayer(a.PlayerID()), amount: 0})
			clone.SetPlayerAt(1, &ronChosenManaPlayer{Player: clone.GetPlayer(b.PlayerID()), amount: 0})
			observed := 0
			g.SetOnDamageDealt(func(source, target string, amount int, combat bool) {
				if source != "Power Leak" || combat || amount < 1 {
					t.Error("wrong damage source or amount")
				}
				observed++
			})
			g.ResolveStack()
			if clone.GetPlayer(b.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 1 {
				t.Fatal("clone shared payment")
			}
			clone.ResolveStack()
			if state == "host-control-before" {
				if a.Life() != 18 || b.Life() != 20 || a.calls != 1 || b.calls != 0 {
					t.Fatal("host controller not captured")
				}
			} else {
				wantLife, wantCalls, wantClone := 19, 1, 18
				if state == "copy" {
					wantLife, wantCalls, wantClone = 17, 2, 16
				}
				if b.Life() != wantLife || b.calls != wantCalls || a.calls != 0 || b.ManaPool().TotalMana() != 0 || observed != wantCalls || clone.GetPlayer(b.PlayerID()).Life() != wantClone {
					t.Fatalf("lost event payer or source: life=%d calls=%d/%d clone=%d", b.Life(), a.calls, b.calls, clone.GetPlayer(b.PlayerID()).Life())
				}
			}
		})
	}
}

func TestRONChosenManaPaymentBindingsAndDelay(t *testing.T) {
	a := &ronChosenManaPlayer{Player: mage.NewBasePlayer("A"), amount: 1}
	g := mage.NewGame(a, mage.NewBasePlayer("B"))
	body := `PlayerAction(You,PayAnyAmountOfMana),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([GainLife(AmountOfManaPaidThisWay)])),PlayerAction(You,PayAnyAmountOfMana),GainLife(AmountOfManaPaidThisWay)`
	c := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([`+body+`]))`)()
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	a.ManaPool().Add(core.Green, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 20 || a.calls != 2 {
		t.Fatal("adjacent payments shared a result")
	}
	a.amount = 3
	g.RunStepWithPriority(core.EndStep)
	if a.Life() != 21 || a.calls != 2 {
		t.Fatal("delay did not retain first payment")
	}

	c = permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PlayerAction(ControllerOfPermanent(ThisPermanent),PayAnyAmountOfMana),PermanentDealsDamageAndPreventSomeOfIt(ThisPermanent,Integer(2),Player(You),AmountOfManaPaidThisWay)]))]))`)()
	c.SetOwner(a.PlayerID())
	p = g.PutOnBattlefield(c, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	for range 2 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || a.calls != 2 {
			t.Fatal("waiting inspection executed payment or lost source")
		}
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	a.amount = 1
	a.ManaPool().Add(core.Green, 1)
	g.RunStepWithPriority(core.EndStep)
	if a.Life() != 20 || a.calls != 3 {
		t.Fatal("delayed payment lost LKI")
	}
}

func TestRONChosenManaPaymentStrictLoad(t *testing.T) {
	files := fstest.MapFS{}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		node := repositoryRecord(t, set, "Power Leak", set+".ron")
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		files[set+".ron"] = &fstest.MapFile{Data: []byte(strings.Split(string(data), "\n")[node.Position.Line-1])}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("reprints failed", err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 6 || len(report.Issues) != 0 {
		t.Fatal("inventory failed", err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "AmountOfManaPaidThisWay" {
			found = true
			if signature.Role != "value" || signature.Occurrences != 6 {
				t.Fatal("wrong payment result role")
			}
		}
	}
	if !found {
		t.Fatal("missing result inventory")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([GainLife(AmountOfManaPaidThisWay)]))])`)}
	if loaded, err := LoadFS(files, "*.ron"); err == nil || loaded != nil {
		t.Fatal("partial registry published")
	}
}

func TestRONChosenManaPaymentPowerLeakManaSources(t *testing.T) {
	a := mage.NewBasePlayer("A")
	b := &ronChosenManaPlayer{Player: mage.NewBasePlayer("B"), amount: 3}
	g := mage.NewGame(a, b)
	host := g.PutOnBattlefield(mage.NewEnchantment("host", ""), b.PlayerID())
	aura := g.PutOnBattlefield(repositoryCard(t, "LEA", "Power Leak"), a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	lands := make([]*mage.Permanent, 0, 3)
	for range 3 {
		lands = append(lands, g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID()))
	}
	queueRONUpkeep(g, b)
	g.ResolveStack()
	if b.Life() != 20 || b.calls != 1 || b.ManaPool().TotalMana() != 0 {
		t.Fatal("resolution mana abilities failed")
	}
	for _, p := range lands {
		if !g.FindPermanent(p.ID()).Tapped {
			t.Fatal("mana source not tapped")
		}
	}
	g.DealDamageToPlayer(b, 2, aura.ID())
	if b.Life() != 18 {
		t.Fatal("extra payment left a prevention shield")
	}
}

func TestRONChosenManaPaymentPowerLeakDetectionAndLegality(t *testing.T) {
	for _, state := range []string{"live", "phased", "removed-abilities", "reattached"} {
		a := &ronChosenManaPlayer{Player: mage.NewBasePlayer("A"), amount: 0}
		b := &ronChosenManaPlayer{Player: mage.NewBasePlayer("B"), amount: 0}
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		host := g.PutOnBattlefield(mage.NewEnchantment("host", ""), b.PlayerID())
		creature := g.PutOnBattlefield(mage.NewCreature("wrong host", "", 1, 3), b.PlayerID())
		c := repositoryCard(t, "LEA", "Power Leak")
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
		a.ManaPool().Add(core.Blue, 2)
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{creature.ID()}, 0); err == nil {
			t.Fatal("illegal enchant target")
		}
		if a.ManaPool().TotalMana() != 2 || g.StackSize() != 0 {
			t.Fatal("illegal cast spent mana")
		}
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		p := g.FindPermanent(c.ID())
		switch state {
		case "phased":
			g.PhaseOut(p.ID())
		case "removed-abilities":
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
		case "reattached":
			other := g.PutOnBattlefield(mage.NewEnchantment("other", ""), a.PlayerID())
			g.Attach(p.ID(), other.ID())
		}
		queueRONUpkeep(g, b)
		want := 0
		if state == "live" {
			want = 1
		}
		if g.StackSize() != want {
			t.Fatal("inactive or reattached source supplied old upkeep")
		}
		g.ResolveStack()
		if state == "live" && b.Life() != 18 {
			t.Fatal("zero payment lost damage")
		}
	}
}
