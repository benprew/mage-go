package ron

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronAlternativePlayer struct {
	*ronPaymentPlayer
	mode, choices int
}

func (p *ronAlternativePlayer) ChooseMode([]string, string) int { p.choices++; return p.mode }

func alternativePlayers() (*ronAlternativePlayer, *ronAlternativePlayer) {
	return &ronAlternativePlayer{ronPaymentPlayer: &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}},
		&ronAlternativePlayer{ronPaymentPlayer: &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}}
}

func TestRONAlternativePaymentErosion(t *testing.T) {
	for _, choice := range []string{"mana", "life", "decline", "unaffordable", "mana-source", "control", "move", "source-left", "source-return", "host-return", "phase-before", "phase-after", "phase-cycle", "copy", "removed", "regeneration"} {
		t.Run(choice, func(t *testing.T) {
			a, b := alternativePlayers()
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			hostCard := repositoryCard(t, "LEB", "Island")
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			other := g.PutOnBattlefield(mage.NewLand("Other"), a.PlayerID())
			card := repositoryCard(t, "DRK", "Erosion")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			a.ManaPool().Add(core.Blue, 3)
			invalid := g.PutOnBattlefield(mage.NewCreature("Invalid", "", 1, 1), b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{invalid.ID()}, 0); err == nil {
				t.Fatal("nonland target accepted")
			}
			if a.ManaPool().TotalMana() != 3 || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed casting changed resources")
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(card.ID()).AttachedTo != host.ID() {
				t.Fatal("Aura did not attach")
			}
			g.TapPermanent(g.FindPermanent(host.ID()))
			if choice == "phase-before" {
				g.PhaseOut(host.ID())
			}
			if choice == "removed" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(card.ID(), core.Indefinite))
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("Aura controller upkeep triggered")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			wantTriggers := 1
			if choice == "phase-before" || choice == "removed" {
				wantTriggers = 0
			}
			if g.StackSize() != wantTriggers || a.offers+b.offers != 0 {
				t.Fatal("trigger or payment timing")
			}
			switch choice {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "move":
				g.Attach(card.ID(), other.ID())
			case "source-left", "source-return":
				g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
				if choice == "source-return" {
					g.PutOnBattlefield(card, a.PlayerID())
					g.Attach(card.ID(), other.ID())
				}
			case "host-return":
				g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
				g.PutOnBattlefield(hostCard, a.PlayerID())
			case "phase-after":
				g.PhaseOut(host.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "regeneration":
				g.AddRegenerationShield(host.ID())
			}
			payer := b
			if choice == "control" || choice == "move" {
				payer = a
			}
			payer.accept = choice != "decline" && choice != "host-return" && choice != "phase-after" && choice != "regeneration"
			if choice == "unaffordable" {
				payer.SetLife(0)
			}
			if choice == "life" {
				payer.mode = 1
				payer.ManaPool().Add(core.Colorless, 1)
			}
			if choice == "mana" {
				payer.ManaPool().Add(core.Colorless, 1)
			}
			if choice == "mana-source" {
				payer.mode = 0
			}
			if choice != "mana-source" {
				g.TapPermanent(g.FindPermanentIncludingPhased(host.ID()))
			}
			if choice == "mana-source" {
				g.UntapPermanent(g.FindPermanent(host.ID()))
			}
			beforeLife, beforeMana := payer.Life(), payer.ManaPool().TotalMana()
			clone := g.Clone()
			clone.ResolveStack()
			if payer.Life() != beforeLife || payer.ManaPool().TotalMana() != beforeMana {
				t.Fatal("clone changed original resources")
			}
			g.ResolveStack()
			paid := payer.accept && choice != "unaffordable" && wantTriggers > 0
			amount := wantTriggers
			if choice == "copy" {
				amount = 2
			}
			lifePayment := paid && choice != "mana" && choice != "mana-source"
			wantLife := beforeLife
			if lifePayment {
				wantLife -= amount
			}
			if payer.Life() != wantLife {
				t.Fatalf("life=%d want=%d", payer.Life(), wantLife)
			}
			wantMana := beforeMana
			if paid && choice == "mana" {
				wantMana--
			}
			if payer.ManaPool().TotalMana() != wantMana {
				t.Fatal("wrong mana payment")
			}
			victim := host.ID()
			if choice == "move" {
				victim = other.ID()
			}
			destroyed := !paid && wantTriggers > 0 && choice != "host-return" && choice != "phase-after" && choice != "regeneration"
			if (g.FindPermanentIncludingPhased(victim) == nil) != destroyed {
				t.Fatal("destruction did not keep identity or regeneration")
			}
			if choice == "source-return" && g.FindPermanent(other.ID()) == nil {
				t.Fatal("old trigger used returned Aura attachment")
			}
			if choice == "mana-source" && !g.FindPermanent(host.ID()).Tapped {
				t.Fatal("payment did not activate land mana")
			}
		})
	}
}

func TestRONAlternativePaymentThrullWizard(t *testing.T) {
	for _, choice := range []string{"black", "generic", "decline", "unaffordable", "mana-source", "source-left", "source-phase", "controller", "illegal", "uncounterable", "copy"} {
		t.Run(choice, func(t *testing.T) {
			a, b := alternativePlayers()
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			spell := castRONSpell(t, g, b.BasePlayer, repositoryCard(t, "LEB", "Dark Ritual"))
			b.ManaPool().Clear()
			source := g.PutOnBattlefield(repositoryCard(t, "FEM", "Thrull Wizard"), a.PlayerID())
			if err := mage.ValidateCardDefinition(source.Card.Name(), source.Card); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{spell.ID}); err == nil {
				t.Fatal("unpaid activation accepted")
			}
			a.ManaPool().Add(core.Black, 1)
			a.ManaPool().Add(core.Colorless, 1)
			wrong := castRONSpell(t, g, b.BasePlayer, repositoryCard(t, "LEB", "Lightning Bolt"), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{wrong.ID}); err == nil {
				t.Fatal("nonblack spell target accepted")
			}
			g.GetStack().RemoveByID(wrong.ID)
			if a.ManaPool().TotalMana() != 2 || g.StackSize() != 1 {
				t.Fatal("invalid activation spent resources")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{spell.ID}); err != nil {
				t.Fatal(err)
			}
			b.ManaPool().Clear()
			payer := b
			switch choice {
			case "source-left":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "source-phase":
				g.PhaseOut(source.ID())
			case "controller":
				spell.Controller = a.PlayerID()
				payer = a
			case "illegal":
				g.GetStack().RemoveByID(spell.ID)
			case "uncounterable":
				mage.WithUncounterable()(spell.Card.(*mage.BaseCard))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			payer.accept = choice != "decline" && choice != "uncounterable"
			if choice != "unaffordable" && choice != "mana-source" {
				payer.ManaPool().Add(core.Colorless, 3)
			}
			if choice == "black" {
				payer.ManaPool().Add(core.Black, 1)
			}
			var landID uuid.UUID
			if choice == "mana-source" {
				landID = g.PutOnBattlefield(repositoryCard(t, "LEB", "Swamp"), payer.PlayerID()).ID()
			}
			if a.offers+b.offers != 0 {
				t.Fatal("payment before resolution")
			}
			clone := g.Clone()
			clone.ResolveTopOfStack()
			if a.offers+b.offers != 0 || g.StackSize() < 1 {
				t.Fatal("clone changed live choices or stack")
			}
			g.ResolveTopOfStack()
			paid := choice != "decline" && choice != "unaffordable" && choice != "uncounterable" && choice != "illegal"
			if (g.GetStack().FindByID(spell.ID) != nil) != (paid || choice == "uncounterable") {
				t.Fatal("counter or payment result")
			}
			wantOffers := 1
			if choice == "unaffordable" || choice == "illegal" {
				wantOffers = 0
			}
			if payer.offers != wantOffers {
				t.Fatalf("offers=%d want=%d", payer.offers, wantOffers)
			}
			if choice == "black" && payer.ManaPool().TotalMana() != 3 {
				t.Fatal("black alternative did not pay exactly black")
			}
			if choice == "generic" && payer.ManaPool().TotalMana() != 0 {
				t.Fatal("generic alternative did not pay three")
			}
			if choice == "mana-source" && !g.FindPermanent(landID).Tapped {
				t.Fatal("black mana ability was not used")
			}
			if !paid && choice != "unaffordable" && payer.ManaPool().TotalMana() != 3 {
				t.Fatal("unpaid branch spent resources")
			}
		})
	}
}

func TestRONAlternativePaymentDeclarations(t *testing.T) {
	for _, cost := range []string{"Or([])", "Or([PayMana([])])", "Or(PayMana([]))", "Or([PayMana([ManaCostX]),PayLife(Integer(1))])", "Or([PayLife(Integer(-1)),PayMana([])])", "Or([PayLife(ValueX),PayMana([])])", "Or([TapPermanent(ThisPermanent),PayMana([])])", "Or([SacrificeAPermanent(IsCardtype(Creature)),PayMana([])])", "Or([PayLife(Integer(1)),PayMana([])],PayMana([]))", "PayLife(Integer(-1))", "PayLife(ValueX)"} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([MayCost(%s),Unless(CostWasPaid,[GainLife(Integer(2))])]))])`, cost)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", cost)
		}
	}
}

func TestRONAlternativePaymentNestedAndInventory(t *testing.T) {
	source := `Card(Name:"Nested",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([MayCost(Or([PayMana([ManaCostU]),Or([PayMana([ManaCostGeneric(1)]),PayLife(Integer(1))])])),If(CostWasPaid,[MayCost(PayLife(Integer(2))),If(CostWasPaid,[GainLife(Integer(5))])]),MayCost(PayLife(Integer(0))),Unless(CostWasPaid,[GainLife(Integer(20))])]))])`
	nodes, err := Parse(strings.NewReader(source), "nested.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	a, b := alternativePlayers()
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	castRONSpell(t, g, a.BasePlayer, factory())
	a.ManaPool().Clear()
	g.ResolveStack()
	if a.Life() != 22 || a.offers != 3 || a.choices != 0 {
		t.Fatalf("independent or nested cost outcomes: life/offers/choices=%d/%d/%d", a.Life(), a.offers, a.choices)
	}
	report, err := InspectFS(fstest.MapFS{"nested.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range report.Signatures {
		if entry.Name == "Or" || entry.Name == "PayLife" || entry.Name == "PayMana" {
			if entry.Role != "resolution_cost" || entry.Support != "unverified" {
				t.Fatalf("cost inventory role: %+v", entry)
			}
		}
	}
	if !slices.ContainsFunc(report.Signatures, func(s InventorySignature) bool { return s.Name == "PayLife" && s.Role == "resolution_cost" }) {
		t.Fatal("nested cost was not inventoried")
	}
}

func TestRONAlternativePaymentPolarityAndZero(t *testing.T) {
	for _, branch := range []string{"If", "Unless"} {
		for _, condition := range []string{"CostWasPaid", "CostWasntPaid"} {
			for _, accept := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/%t", branch, condition, accept), func(t *testing.T) {
					factory := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([MayCost(Or([PayMana([]),PayLife(Integer(0))])),`+branch+`(`+condition+`,[GainLife(Integer(2))])]))`)
					a, b := alternativePlayers()
					a.accept = accept
					g := mage.NewGame(a, b)
					g.SetStep(core.PrecombatMain)
					castRONSpell(t, g, a.BasePlayer, factory())
					a.ManaPool().Clear()
					g.ResolveStack()
					wantLife, wantChoices := 20, 0
					if ((branch == "If") == (condition == "CostWasPaid")) == accept {
						wantLife += 2
					}
					if accept {
						wantChoices = 1
					}
					if a.Life() != wantLife || a.offers != 1 || a.choices != wantChoices {
						t.Fatal("zero payment was automatic or branch polarity changed")
					}
				})
			}
		}
	}
}

func TestRONAlternativePaymentInstancesAndRetarget(t *testing.T) {
	a, b := alternativePlayers()
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	first := castRONSpell(t, g, b.BasePlayer, repositoryCard(t, "LEB", "Dark Ritual"))
	second := castRONSpell(t, g, b.BasePlayer, repositoryCard(t, "LEB", "Dark Ritual"))
	one, two := repositoryCard(t, "FEM", "Thrull Wizard"), repositoryCard(t, "FEM", "Thrull Wizard")
	if one.Abilities()[0] == two.Abilities()[0] {
		t.Fatal("factory instances share ability declarations")
	}
	source := g.PutOnBattlefield(one, a.PlayerID())
	a.ManaPool().Add(core.Black, 1)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{first.ID}); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	copyObj := g.CopyStackObjectDirect(original, b.PlayerID(), false)
	if copyObj == nil {
		t.Fatal("copy failed")
	}
	copyObj.Targets[0] = second.ID
	copyObj.TargetZones[second.ID] = core.ZoneStack
	b.ManaPool().Clear()
	b.ManaPool().Add(core.Colorless, 3)
	clone := g.Clone()
	g.ResolveTopOfStack()
	if b.ManaPool().TotalMana() != 0 || b.offers != 1 || g.GetStack().FindByID(second.ID) == nil || original.Targets[0] != first.ID {
		t.Fatal("retargeted copy changed original or did not pay")
	}
	g.ResolveTopOfStack()
	if g.GetStack().FindByID(first.ID) != nil || g.GetStack().FindByID(second.ID) == nil {
		t.Fatal("independent payment result or spell identity changed")
	}
	if clone.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 3 || clone.GetStack().FindByID(first.ID) == nil {
		t.Fatal("live payment changed clone")
	}
}

func TestRONAlternativePaymentNativeUpkeepAndLifeClause(t *testing.T) {
	a, b := alternativePlayers()
	g := mage.NewGame(a, b)
	g.SetTurn(1)
	host := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
	source := g.PutOnBattlefield(repositoryCard(t, "DRK", "Erosion"), a.PlayerID())
	g.Attach(source.ID(), host.ID())
	b.mode = 1
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.Untap)
	g.RunStepWithPriority(core.Upkeep)
	if b.Life() != 19 || b.offers != 1 || a.offers != 0 || g.FindPermanent(host.ID()) == nil || g.StackSize() != 0 {
		t.Fatal("native upkeep did not offer the host controller's life payment")
	}
	n := hostUntapRecord(t, "DRK", "Season of the Witch")
	if _, err := Compile(n); err == nil || strings.Contains(err.Error(), "optional payment cost (PayLife)") {
		t.Fatalf("full card must retain its independent end-step blocker: %v", err)
	}
	rules := n.Fields["Rules"]
	rules.Args = append([]Node(nil), rules.Args[:1]...)
	n.Fields["Rules"] = rules
	factory, err := Compile(n)
	if err != nil {
		t.Fatal(err)
	}
	for _, accept := range []bool{false, true} {
		a, b := alternativePlayers()
		a.accept = accept
		game := mage.NewGame(a, b)
		permanent := game.PutOnBattlefield(factory(), a.PlayerID())
		game.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		wantLife := 20
		if accept {
			wantLife -= 2
		}
		if a.Life() != wantLife || a.offers != 1 || (game.FindPermanent(permanent.ID()) != nil) != accept {
			t.Fatal("exact life upkeep clause failed")
		}
	}
}
