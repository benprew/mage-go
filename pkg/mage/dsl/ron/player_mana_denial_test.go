package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONManaDenialManaShort(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, set, "Mana Short")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
			empty := g.PutOnBattlefield(mage.NewLand("No mana"), b.PlayerID())
			other := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
			nonland := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			a.ManaPool().Add(core.Blue, 1)
			a.ManaPool().Add(core.Colorless, 2)
			b.ManaPool().Add(core.Red, 4)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{uuid.New()}, 0); err == nil || a.ManaPool().TotalMana() != 3 {
				t.Fatal("invalid target spent mana")
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
				t.Fatal(err)
			}
			a.ManaPool().Add(core.Green, 2)
			if land.Tapped || b.ManaPool().TotalMana() != 4 {
				t.Fatal("resolved at announcement")
			}
			clone := g.Clone()
			clone.ResolveStack()
			if g.FindPermanent(land.ID()).Tapped || b.ManaPool().TotalMana() != 4 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if !game.FindPermanent(land.ID()).Tapped || !game.FindPermanent(empty.ID()).Tapped || game.FindPermanent(other.ID()).Tapped || game.FindPermanent(nonland.ID()).Tapped {
					t.Fatal("tap collection scope")
				}
				if game.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 0 || game.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 2 {
					t.Fatal("wrong mana pool cleared")
				}
			}
		})
	}
}

func TestRONManaDenialPowerSink(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, scenario := range []string{"paid", "decline", "unaffordable", "uncounterable", "zero-paid", "zero-decline", "departed"} {
			t.Run(set+"/"+scenario, func(t *testing.T) {
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
				b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: scenario != "decline" && scenario != "uncounterable" && scenario != "zero-decline"}
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				victim := repositoryCard(t, "LEB", "Healing Salve")
				victim.SetOwner(b.PlayerID())
				b.AddToHand(victim)
				b.ManaPool().Add(core.White, 1)
				if scenario == "uncounterable" {
					mage.WithUncounterable()(victim.(*mage.BaseCard))
				}
				if err := g.CastSpellByID(b.PlayerID(), victim.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
					t.Fatal(err)
				}
				victimStackID := g.StackPeek().ID
				sink := repositoryCard(t, set, "Power Sink")
				sink.SetOwner(a.PlayerID())
				a.AddToHand(sink)
				x := 2
				if scenario == "zero-paid" || scenario == "zero-decline" {
					x = 0
				}
				a.ManaPool().Add(core.Blue, 1)
				if x > 0 {
					if err := g.CastSpellByID(a.PlayerID(), sink.ID(), []uuid.UUID{victimStackID}, x); err == nil {
						t.Fatal("unpaid X cast accepted")
					}
				}
				a.ManaPool().Add(core.Colorless, x)
				if err := g.CastSpellByID(a.PlayerID(), sink.ID(), []uuid.UUID{victimStackID}, x); err != nil {
					t.Fatal(err)
				}
				b.ManaPool().Add(core.Red, 3)
				if scenario == "unaffordable" {
					b.ManaPool().Clear()

				}
				land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
				empty := g.PutOnBattlefield(mage.NewLand("No mana"), b.PlayerID())
				other := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
				if scenario == "departed" {
					g.GetStack().RemoveByID(victimStackID)
				}
				g.SetXValue(99)
				g.ResolveTopOfStack()
				paid := scenario == "paid" || scenario == "zero-paid"
				if scenario == "departed" {
					if land.Tapped || b.ManaPool().TotalMana() != 3 || b.offers != 0 {
						t.Fatal("all-illegal spell still resolved")
					}
					return
				}
				if (g.GetStack().FindBySourceID(victim.ID()) != nil) != (paid || scenario == "uncounterable") {
					t.Fatal("wrong counter branch")
				}
				if g.FindPermanent(land.ID()).Tapped == paid || g.FindPermanent(empty.ID()).Tapped || g.FindPermanent(other.ID()).Tapped {
					t.Fatal("mana ability or controller filter")
				}
				want := 0
				if paid {
					want = 3 - x
				}
				offers := 1
				if scenario == "unaffordable" {
					offers = 0
				}
				if b.ManaPool().TotalMana() != want || b.offers != offers {
					t.Fatalf("mana=%d want=%d offers=%d", b.ManaPool().TotalMana(), want, b.offers)
				}
			})
		}
	}
}

func compileManaDenialFixture(t *testing.T, rules string) mage.Card {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Scoped",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rules+`])`), "scoped.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	card := factory()
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	return card
}

func TestRONManaDenialScopes(t *testing.T) {
	g, a, b := ronGame()
	fillRONLibrary(t, a, 10)
	fillRONLibrary(t, b, 10)
	card := compileManaDenialFixture(t, `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerActions(Ref_TargetPlayer,[DrawACard,PlayerActions(You,[DrawACard]),PlayerActions(ActionPlayer,[DrawACard]),DrawACard]),DrawACard])))`)
	p := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.ResolveStack()
	g.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		if len(game.GetPlayer(a.PlayerID()).Hand()) != 2 || len(game.GetPlayer(b.PlayerID()).Hand()) != 3 {
			t.Fatal("nested or sibling scope escaped")
		}
	}
}

func TestRONManaDenialMembership(t *testing.T) {
	for _, scenario := range []string{"control", "phase", "cycle", "returned", "removed-ability", "gained-ability", "late-entry"} {
		for _, name := range []string{"Mana Short", "Power Sink"} {
			t.Run(name+"/"+scenario, func(t *testing.T) {
				g, a, b := ronGame()
				victim := repositoryCard(t, "LEB", "Healing Salve")
				victim.SetOwner(b.PlayerID())
				b.AddToHand(victim)
				b.ManaPool().Add(core.White, 1)
				if err := g.CastSpellByID(b.PlayerID(), victim.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
					t.Fatal(err)
				}
				spellID := g.StackPeek().ID
				card := repositoryCard(t, "LEB", name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				targets := []uuid.UUID{b.PlayerID()}
				x := 0
				mana := 2
				if name == "Power Sink" {
					targets = []uuid.UUID{spellID}
					x = 9
					mana = x
				}
				a.ManaPool().Add(core.Blue, 1)
				a.ManaPool().Add(core.Colorless, mana)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, x); err != nil {
					t.Fatal(err)
				}
				landCard := repositoryCard(t, "LEB", "Forest")
				if scenario == "gained-ability" {
					landCard = mage.NewLand("Blank")
				}
				landCard.SetOwner(b.PlayerID())
				p := g.PutOnBattlefield(landCard, b.PlayerID())
				want := true
				switch scenario {
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
					want = false
				case "phase":
					g.PhaseOut(p.ID())
					want = false
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "returned":
					g.BouncePermanentToHand(p)
					b.RemoveFromHand(landCard.ID())
					p = g.PutOnBattlefield(landCard, b.PlayerID())
				case "removed-ability":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					g.ApplyContinuousEffects()
					want = name == "Mana Short"
				case "gained-ability":
					if err := mage.ApplyEffect(g, mage.GrantAbility(mage.NewManaAbility(core.Green)), p.ID(), b.PlayerID(), []uuid.UUID{p.ID()}); err != nil {
						t.Fatal(err)
					}
					g.ApplyContinuousEffects()
				}
				b.ManaPool().AddRestricted(core.Colorless, 1, core.ArtifactSpellsOnly{})
				g.ResolveTopOfStack()
				current := g.FindPermanentIncludingPhased(p.ID())
				if current == nil || current.Tapped != want || b.ManaPool().TotalMana() != 0 || b.Life() != 20 {
					t.Fatal("resolution membership, mana loss or life")
				}
			})
		}
	}
}

func TestRONManaDenialCopiesAndInstances(t *testing.T) {
	g, a, b := ronGame()
	first := repositoryCard(t, "LEB", "Mana Short")
	second := repositoryCard(t, "LEB", "Mana Short")
	for _, card := range []mage.Card{first, second} {
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
	}
	landA := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
	landB := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
	a.ManaPool().Add(core.Blue, 2)
	a.ManaPool().Add(core.Colorless, 4)
	if err := g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByID(a.PlayerID(), second.ID(), []uuid.UUID{a.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	a.ManaPool().Add(core.Red, 2)
	b.ManaPool().Add(core.Green, 3)
	clone := g.Clone()
	g.ResolveTopOfStack()
	if !g.FindPermanent(landA.ID()).Tapped || g.FindPermanent(landB.ID()).Tapped || a.ManaPool().TotalMana() != 0 || b.ManaPool().TotalMana() != 3 {
		t.Fatal("factory instances shared targets")
	}
	copied := g.CopySpellOnStack(first.ID(), b.PlayerID(), true)
	if copied == nil || copied.Targets[0] != a.PlayerID() {
		t.Fatal("copy did not retarget")
	}
	a.ManaPool().Add(core.Blue, 1)
	g.ResolveTopOfStack()
	if a.ManaPool().TotalMana() != 0 || b.ManaPool().TotalMana() != 3 {
		t.Fatal("copy used original target")
	}
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		if !game.FindPermanent(landB.ID()).Tapped || game.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 0 {
			t.Fatal("original or clone binding lost")
		}
	}
}

func TestRONManaDenialInvalidAndInventory(t *testing.T) {
	for _, action := range []string{
		`PlayerActions(You,DrawACard)`, `PlayerActions(You,[],[])`, `PlayerActions(You(),[])`,
		`PlayerActions(Ref_TargetPlayer,[])`, `PlayerActions(Ref_TargetPermanent,[])`,
		`PlayerActions(You,[LoseUnspentMana()])`, `PlayerActions(You,[ChooseACardName(AnyCard)])`,
		`PlayerAction(ActionPlayer,LoseUnspentMana)`,
		`PlayerActions(You,[]),PlayerAction(ActionPlayer,LoseUnspentMana)`,
		`TapEachPermanent(ControlledByAPlayer(SinglePlayer(ActionPlayer)))`,
		`TapEachPermanent(Or([IsCardtype(Land),ControlledByAPlayer(SinglePlayer(You()))]))`,
		`TapEachPermanent(And([ControlledByAPlayer(SinglePlayer(You)),ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer))]))`,
		`TapEachPermanent(ControlledByAPlayer(Opponent()))`,
		`TapEachPermanent(HasAbility(ManaAbility()))`,
	} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + action + `]))])`
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted invalid scope: %s", action)
		}
	}
	source := `Card(Name:"Valid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([PlayerActions(You,[TapEachPermanent(And([HasAbility(ManaAbility),ControlledByAPlayer(SinglePlayer(ActionPlayer))])),LoseUnspentMana])]))])`
	files := fstest.MapFS{"valid.ron": {Data: []byte(source)}, "bad.ron": {Data: []byte(strings.Replace(source, "LoseUnspentMana", "MissingAction", 1))}}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 || len(report.Issues) != 1 {
		t.Fatal("inventory or strict rejection")
	}
	seen := map[string]bool{}
	for _, signature := range report.Signatures {
		if signature.Name == "ActionPlayer" || signature.Name == "ManaAbility" || signature.Name == "TapEachPermanent" {
			seen[signature.Name+":"+signature.Role] = true
		}
	}
	for _, key := range []string{"ActionPlayer:player_reference", "ManaAbility:ability_class", "TapEachPermanent:action"} {
		if !seen[key] {
			t.Fatalf("missing role %s: %v", key, seen)
		}
	}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("partial registry published")
	}
}

func TestRONManaDenialPowerSinkCopy(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: false}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	spells := make([]uuid.UUID, 0, 2)
	for _, player := range []*ronPaymentPlayer{a, b} {
		card := repositoryCard(t, "LEB", "Healing Salve")
		card.SetOwner(player.PlayerID())
		player.AddToHand(card)
		player.ManaPool().Add(core.White, 1)
		if err := g.CastSpellByID(player.PlayerID(), card.ID(), []uuid.UUID{player.PlayerID()}, 0); err != nil {
			t.Fatal(err)
		}
		spells = append(spells, g.StackPeek().ID)
	}
	sink := repositoryCard(t, "LEB", "Power Sink")
	sink.SetOwner(a.PlayerID())
	a.AddToHand(sink)
	a.ManaPool().Add(core.Blue, 1)
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.CastSpellByID(a.PlayerID(), sink.ID(), []uuid.UUID{spells[1]}, 2); err != nil {
		t.Fatal(err)
	}
	copied := g.CopySpellOnStack(sink.ID(), b.PlayerID(), true)
	if copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != spells[0] {
		t.Fatal("copy did not bind different spell target")
	}
	landA := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
	landB := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
	a.ManaPool().Add(core.Red, 3)
	b.ManaPool().Add(core.Green, 3)
	clone := g.Clone()
	g.ResolveTopOfStack()
	if a.ManaPool().TotalMana() != 0 || b.ManaPool().TotalMana() != 3 || !g.FindPermanent(landA.ID()).Tapped || g.FindPermanent(landB.ID()).Tapped {
		t.Fatal("copy payer follows original spell or copy controller")
	}
	if clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 3 {
		t.Fatal("resolution changed clone")
	}
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		if game.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 || game.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 0 || !game.FindPermanent(landB.ID()).Tapped || game.StackSize() != 0 {
			t.Fatal("original or clone target binding changed")
		}
	}
}

func TestRONManaDenialTriggerAndDelayedContexts(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		t.Run(fmt.Sprint(delayed), func(t *testing.T) {
			g, a, b := ronGame()
			rules := `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([PlayerActions(You,[LoseUnspentMana])]))`
			if delayed {
				rules = `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfPlayersNextUpkeep(You),ActionList([PlayerActions(You,[LoseUnspentMana])]))]))`
			}
			card := compileManaDenialFixture(t, rules)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			if delayed {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				g.BouncePermanentToHand(p)
			}
			g.SetTurn(2)
			a.ManaPool().Add(core.Blue, 2)
			b.ManaPool().Add(core.Green, 3)
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || a.ManaPool().TotalMana() != 2 {
				t.Fatal("trigger timing or delayed context")
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.ManaPool().TotalMana() != 2 {
				t.Fatal("clone changed live trigger binding")
			}
			g.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if game.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 || game.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 3 {
					t.Fatal("trigger lost its scoped player")
				}
			}
		})
	}
}
