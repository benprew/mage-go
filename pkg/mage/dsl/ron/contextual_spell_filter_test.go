package ron

import (
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONContextualSpellFilterInvokePrejudice(t *testing.T) {
	for _, scenario := range []string{"no-creatures", "shares", "multi", "colorless", "opponent-creature", "noncreature", "phased", "phase-cycle", "pay", "short", "decline", "source-departed", "later-shares", "later-loses-color", "spell-control", "self-cast", "copy", "failed-cast"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: scenario == "pay"}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			g.SetActivePlayerIndex(1)
			card := repositoryCard(t, "LEG", "Invoke Prejudice")
			card.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			donorCard := mage.NewCreature("Donor", "{G}", 2, 2)
			if scenario == "multi" {
				donorCard = mage.NewCreature("Donor", "{G}{U}", 2, 2)
			}
			if scenario == "colorless" {
				donorCard = mage.NewCreature("Donor", "", 2, 2)
			}
			if scenario == "noncreature" {
				donorCard = mage.NewArtifact("Donor", "{G}")
			}
			controller := a.PlayerID()
			if scenario == "opponent-creature" {
				controller = b.PlayerID()
			}
			var donor *mage.Permanent
			if scenario != "no-creatures" && scenario != "pay" && scenario != "short" && scenario != "decline" && scenario != "source-departed" && scenario != "later-shares" && scenario != "spell-control" && scenario != "copy" && scenario != "failed-cast" {
				donor = g.PutOnBattlefield(donorCard, controller)
			}
			if scenario == "phased" {
				g.PhaseOut(donor.ID())
			}
			if scenario == "phase-cycle" {
				g.PhaseIn(g.PhaseOut(donor.ID()))
			}
			spell := repositoryCard(t, "LEB", "Llanowar Elves")
			if scenario == "colorless" {
				spell = repositoryCard(t, "ATQ", "Ornithopter")
			}
			caster := b.BasePlayer
			if scenario == "self-cast" {
				caster = a.BasePlayer
				g.SetActivePlayerIndex(0)
			}
			spell.SetOwner(caster.PlayerID())
			caster.AddToHand(spell)
			if scenario != "failed-cast" {
				caster.ManaPool().Add(core.Green, 1)
			}
			err := g.CastSpellByID(caster.PlayerID(), spell.ID(), nil, 0)
			if scenario == "failed-cast" {
				if err == nil || g.StackSize() != 0 || a.offers != 0 || b.offers != 0 {
					t.Fatal("failed casting changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			victim := g.StackPeek()
			if scenario == "later-shares" {
				g.PutOnBattlefield(donorCard, a.PlayerID())
			}
			if scenario == "later-loses-color" {
				g.BouncePermanentToHand(donor)
			}
			g.PutTriggersOnStack()
			triggered := scenario != "shares" && scenario != "multi" && scenario != "phase-cycle" && scenario != "later-loses-color" && scenario != "self-cast"
			if g.StackSize() != 1+boolInt(triggered) {
				t.Fatalf("stack = %d, triggered = %v", g.StackSize(), triggered)
			}
			if !triggered {
				return
			}
			b.ManaPool().Clear()
			a.ManaPool().Clear()
			if scenario != "short" {
				b.ManaPool().Add(core.Colorless, 1)
			}
			if scenario == "source-departed" {
				g.DestroyPermanent(g.FindPermanent(source.ID()))
			}
			if scenario == "spell-control" {
				victim.Controller = a.PlayerID()
			}
			if scenario == "copy" {
				g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
			}
			clone := g.Clone()
			clone.SetPlayerAt(1, &ronPaymentPlayer{BasePlayer: clone.GetPlayer(b.PlayerID()).(*mage.SearchPlayer).BasePlayer, accept: b.accept})
			clone.ResolveTopOfStack()
			if a.offers != 0 || b.offers != 0 {
				t.Fatal("clone changed live choices")
			}
			g.ResolveTopOfStack()
			survives := scenario == "pay"
			if (g.GetStack().FindByID(victim.ID) != nil) != survives {
				t.Fatal("wrong resolution payment or counter")
			}
			if (clone.GetStack().FindByID(victim.ID) != nil) != survives {
				t.Fatal("clone lost event spell")
			}
			if a.offers != 0 {
				t.Fatal("source controller paid")
			}
			offers := 1
			if scenario == "short" {
				offers = 0
			}
			if b.offers != offers {
				t.Fatalf("offers = %d", b.offers)
			}
		})
	}
}

func TestRONContextualSpellFilterInvalid(t *testing.T) {
	for _, filter := range []string{
		`ControlledByAPlayer`, `ControlledByAPlayer()`, `ControlledByAPlayer(You)`,
		`ControlledByAPlayer(SinglePlayer())`, `ControlledByAPlayer(SinglePlayer(You()))`,
		`ControlledByAPlayer(SinglePlayer(Opponent))`, `ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer))`,
		`ControlledByAPlayer(SinglePlayer(Trigger_ThatPlayer))`, `ControlledByAPlayer(SinglePlayer(ThisPermanent))`,
		`SharesAColorWithAPermanent`, `SharesAColorWithAPermanent()`, `SharesAColorWithAPermanent(IsCardtype(Creature),AnyPermanent)`,
		`SharesAColorWithAPermanent(SinglePermanent(Ref_TargetPermanent))`,
		`SharesAColorWithAPermanent(ControlledByAPlayer(SinglePlayer(ActionPlayer)))`,
		`SharesAColorWithAPermanent(And([]))`,
		`SharesAColorWithAPermanent(Or([ControlledByAPlayer(SinglePlayer(You)),IsCardtype(Creature)]))`,
		`SharesAColorWithAPermanent(And([ControlledByAPlayer(SinglePlayer(You)),ControlledByAPlayer(SinglePlayer(You))]))`,
		`ExceptFor`, `ExceptFor()`, `ExceptFor(AnySpell,AnySpell)`, `ExceptFor(Or([]))`,
		`Or([AnySpell,ControlledByAPlayer(SinglePlayer(You()))])`,
	} {
		for _, rule := range []string{
			`SpellActions(Targeted([TargetSpell(` + filter + `)],ActionList([CounterSpell(Ref_TargetSpell)])))`,
			`TriggerA(WhenAPlayerCastsASpell(AnyPlayer,` + filter + `),ActionList([CounterSpell(Trigger_ThatSpell)]))`,
		} {
			typ := "Instant"
			if strings.HasPrefix(rule, "Trigger") {
				typ = "Enchantment"
			}
			record := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[`+typ+`],Subtypes:[]),Rules:[`+rule+`])`)
			if _, err := Compile(record); err == nil {
				t.Fatalf("accepted %s", rule)
			}
		}
	}
}

func TestRONContextualSpellFilterTargets(t *testing.T) {
	for _, filter := range []string{
		`ControlledByAPlayer(SinglePlayer(You))`,
		`And([IsCardtype(Artifact),ControlledByAPlayer(SinglePlayer(You))])`,
		`Or([IsCardtype(Creature),And([IsCardtype(Artifact),ControlledByAPlayer(SinglePlayer(You))])])`,
		`ExceptFor(ExceptFor(ControlledByAPlayer(SinglePlayer(You))))`,
	} {
		for _, change := range []string{"none", "spell-control", "source-control", "source-departed", "source-phased", "source-returned", "copy", "recast"} {
			t.Run(filter+"/"+change, func(t *testing.T) {
				factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),Targeted([TargetSpell(`+filter+`)],ActionList([CounterSpell(Ref_TargetSpell)])))`)
				g, a, b := ronGame()
				source := g.PutOnBattlefield(factory(), a.PlayerID())
				victim := castRONSpell(t, g, a, repositoryCard(t, "LEB", "Sol Ring"))
				own := factory().Abilities()[0].(*mage.ActionDefinition).Targets()[0]
				other := factory().Abilities()[0].(*mage.ActionDefinition).Targets()[0]
				if own == other {
					t.Fatal("shared declaration")
				}
				before := a.ManaPool().TotalMana()
				for range 2 {
					if !slices.Equal(own.Possible(a.PlayerID(), source.Card, g), []uuid.UUID{victim.ID}) || len(own.Possible(b.PlayerID(), source.Card, g)) != 0 {
						t.Fatal("wrong controller predicate")
					}
				}
				if before != a.ManaPool().TotalMana() || source.Tapped || g.StackSize() != 1 {
					t.Fatal("query changed game")
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID}); err != nil {
					t.Fatal(err)
				}
				clone := g.Clone()
				switch change {
				case "spell-control":
					victim.Controller = b.PlayerID()
				case "source-control":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), source.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "source-departed", "source-returned":
					g.BouncePermanentToHand(source)
					if change == "source-returned" {
						g.PutOnBattlefield(source.Card, b.PlayerID())
					}
				case "source-phased":
					g.PhaseOut(source.ID())
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				case "recast":
					g.GetStack().RemoveByID(victim.ID)
					trigger := g.StackPeek()
					g.GetStack().RemoveByID(trigger.ID)
					g.PushStack(&mage.StackObject{ID: uuid.New(), SourceID: victim.SourceID, Card: victim.Card, Controller: a.PlayerID()})
					g.PushStack(trigger)
				}
				g.ResolveTopOfStack()
				survives := change == "spell-control" || change == "copy"
				if change != "recast" && (g.GetStack().FindByID(victim.ID) != nil) != survives {
					t.Fatal("wrong current target legality")
				}
				if change == "copy" {
					g.ResolveTopOfStack()
					if g.GetStack().FindByID(victim.ID) != nil {
						t.Fatal("original lost its controller")
					}
				}
				if change == "recast" && g.StackSize() != 1 {
					t.Fatal("saved target followed new stack visit")
				}
				clone.ResolveTopOfStack()
				if clone.GetStack().FindByID(victim.ID) != nil {
					t.Fatal("clone lost target or controller")
				}
			})
		}
	}
}

func TestRONContextualSpellFilterNativeAndInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	records, err := Parse(strings.NewReader(string(data)), "LEG.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record Node
	for _, node := range records {
		if node.Fields["Name"].String == "Invoke Prejudice" {
			record = node
		}
	}
	if record.Kind == 0 {
		t.Fatal("missing real record")
	}
	files := fstest.MapFS{"one.ron": &fstest.MapFile{Data: []byte(string(data))}}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range report.Issues {
		if item.Card == "Invoke Prejudice" {
			t.Fatal(item.Message)
		}
	}
	for name, role := range map[string]string{"ExceptFor": "spell_filter", "SharesAColorWithAPermanent": "spell_filter", "ControlledByAPlayer": "permanent_filter", "SinglePlayer": "player_filter"} {
		if !slices.ContainsFunc(report.Signatures, func(s InventorySignature) bool { return s.Name == name && s.Role == role }) {
			t.Fatalf("missing role %s/%s", name, role)
		}
	}
	line := strings.Split(string(data), "\n")[record.Position.Line-1]
	subset := fstest.MapFS{"one.ron": &fstest.MapFile{Data: []byte(line)}, "reprint.ron": &fstest.MapFile{Data: []byte(line)}}
	registry, err := LoadFS(subset, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Invoke Prejudice")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Invoke Prejudice")
	if err != nil {
		t.Fatal(err)
	}
	if first.Abilities()[0].AbilityID() == second.Abilities()[0].AbilityID() {
		t.Fatal("shared ability identity")
	}
	subset["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Unsupported",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([Unsupported]))])`)}
	if partial, err := LoadFS(subset, "*.ron"); err == nil || partial != nil {
		t.Fatal("partial registry published")
	}
	if LoadInto(registry, subset, "*.ron") == nil {
		t.Fatal("invalid replacement accepted")
	}
	if _, err := registry.CreateCard("Invoke Prejudice"); err != nil {
		t.Fatal("failed load erased registry")
	}
}

func TestRONContextualSpellFilterInvokeLifetimeAndManaValue(t *testing.T) {
	for _, scenario := range []string{"x-pay", "x-short", "removed-spell", "returned-spell", "source-phased", "source-returned", "source-phased-before", "source-removed-before", "noncreature-spell"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			sourceCard := repositoryCard(t, "LEG", "Invoke Prejudice")
			castRONSpell(t, g, a.BasePlayer, sourceCard)
			g.ResolveTopOfStack()
			source := g.FindPermanent(sourceCard.ID())
			if source == nil {
				t.Fatal("source did not enter")
			}
			if scenario == "source-phased-before" {
				g.PhaseOut(source.ID())
			}
			if scenario == "source-removed-before" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
			}
			g.SetActivePlayerIndex(1)
			spell := mage.NewCreature("X creature", "{X}{G}", 2, 2)
			if scenario == "noncreature-spell" {
				spell = mage.NewArtifact("Artifact", "{X}{G}")
			}
			spell.SetOwner(b.PlayerID())
			b.AddToHand(spell)
			b.ManaPool().Add(core.Green, 1)
			b.ManaPool().Add(core.Colorless, 3)
			if err := g.CastSpellByID(b.PlayerID(), spell.ID(), nil, 3); err != nil {
				t.Fatal(err)
			}
			victim := g.StackPeek()
			g.PutTriggersOnStack()
			if scenario == "source-phased-before" || scenario == "source-removed-before" || scenario == "noncreature-spell" {
				if g.StackSize() != 1 {
					t.Fatal("inactive or nonmatching source triggered")
				}
				return
			}
			b.ManaPool().Clear()
			b.ManaPool().Add(core.Colorless, 4)
			if scenario == "x-short" {
				b.ManaPool().Clear()
				b.ManaPool().Add(core.Colorless, 3)
			}
			if scenario == "source-phased" {
				g.PhaseOut(source.ID())
			}
			if scenario == "source-returned" {
				g.BouncePermanentToHand(source)
				a.RemoveFromHand(sourceCard.ID())
				g.PutOnBattlefield(sourceCard, b.PlayerID())
			}
			if scenario == "removed-spell" || scenario == "returned-spell" {
				g.GetStack().RemoveByID(victim.ID)
				if scenario == "returned-spell" {
					trigger := g.StackPeek()
					g.GetStack().RemoveByID(trigger.ID)
					g.PushStack(&mage.StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: b.PlayerID(), XValue: 7})
					g.PushStack(trigger)
				}
			}
			g.ResolveTopOfStack()
			if scenario == "removed-spell" || scenario == "returned-spell" {
				if b.offers != 1 || b.ManaPool().TotalMana() != 0 {
					t.Fatalf("departure mana value: offers=%d mana=%d", b.offers, b.ManaPool().TotalMana())
				}
				if scenario == "returned-spell" && g.StackSize() != 1 {
					t.Fatal("old trigger followed new spell")
				}
				return
			}
			paid := scenario != "x-short"
			if (g.GetStack().FindByID(victim.ID) != nil) != paid {
				t.Fatal("wrong X payment result")
			}
			expected := 0
			if !paid {
				expected = 3
			}
			if b.ManaPool().TotalMana() != expected || a.offers != 0 {
				t.Fatal("partial payment or wrong player")
			}
		})
	}
}

func TestRONContextualSpellFilterIndependentTriggers(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	g.SetActivePlayerIndex(1)
	for range 2 {
		g.PutOnBattlefield(repositoryCard(t, "LEG", "Invoke Prejudice"), a.PlayerID())
	}
	victim := castRONSpell(t, g, b.BasePlayer, repositoryCard(t, "LEB", "Llanowar Elves"))
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatal("missing independent source triggers")
	}
	b.ManaPool().Clear()
	b.ManaPool().Add(core.Colorless, 1)
	g.ResolveTopOfStack()
	if g.GetStack().FindByID(victim.ID) == nil || b.offers != 1 || b.ManaPool().TotalMana() != 0 {
		t.Fatal("first trigger did not use its own payment")
	}
	g.ResolveTopOfStack()
	if g.GetStack().FindByID(victim.ID) != nil || b.offers != 1 {
		t.Fatal("second trigger inherited first payment")
	}
}

func TestRONContextualSpellFilterRemainingGoblinBlocker(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/ATQ.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	records, err := Parse(strings.NewReader(string(data)), "ATQ.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range records {
		if record.Fields["Name"].String != "Goblin Artisans" {
			continue
		}
		if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "IsntTheTargetOfAnAbility") {
			t.Fatalf("expected independent ability exclusion gap: %v", err)
		}
		return
	}
	t.Fatal("missing real record")
}
