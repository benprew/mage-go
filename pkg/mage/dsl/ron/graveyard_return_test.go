package ron

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type graveyardReturnCase struct {
	set, name, victim      string
	activated, battlefield bool
	cost                   int
}

var graveyardReturnCards = []graveyardReturnCase{
	{"LEB", "Raise Dead", "Savannah Lions", false, false, 1},
	{"ATQ", "Reconstruction", "Sol Ring", false, false, 1},
	{"LEB", "Regrowth", "Forest", false, false, 2},
	{"LEB", "Resurrection", "Savannah Lions", false, true, 4},
	{"LEG", "Adun Oakenshield", "Savannah Lions", true, false, 3},
	{"ATQ", "Argivian Archaeologist", "Sol Ring", true, false, 2},
	{"DRK", "Skull of Orm", "Crusade", true, false, 5},
}

func TestRONGraveyardReturnCards(t *testing.T) {
	for _, tc := range graveyardReturnCards {
		for _, scenario := range []string{"normal", "absent", "returned", "source-departed", "source-control", "source-phased"} {
			t.Run(tc.name+"/"+scenario, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				victim := repositoryCard(t, "LEB", tc.victim)
				victim.SetOwner(a.PlayerID())
				a.AddToGraveyard(victim)
				for _, c := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green, core.Colorless} {
					a.ManaPool().Add(c, 10)
				}
				var source *mage.Permanent
				if tc.activated {
					source = g.PutOnBattlefield(card, a.PlayerID())
					source.RevokeBaseAttr(core.AttrSummonSick)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
						t.Fatal(err)
					}
					if !g.FindPermanent(source.ID()).Tapped {
						t.Fatal("tap cost not paid")
					}
				} else {
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim.ID()}, 0); err != nil {
						t.Fatal(err)
					}
				}
				if g.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 60-tc.cost {
					t.Fatal("wrong payment amount")
				}
				if len(g.GetPlayer(a.PlayerID()).Hand()) != 0 || g.FindPermanent(victim.ID()) != nil {
					t.Fatal("return before resolution")
				}
				if scenario == "absent" || scenario == "returned" {
					moved, ok := g.MoveFromGraveyard(a.PlayerID(), victim.ID(), core.ZoneHand)
					if !ok {
						t.Fatal("target departure failed")
					}
					a.AddToHand(moved)
					if scenario == "returned" {
						removed, ok := a.RemoveFromHand(victim.ID())
						if !ok {
							t.Fatal("target removal failed")
						}
						a.AddToGraveyard(removed)
					}
				}
				if tc.activated {
					switch scenario {
					case "source-departed":
						g.DestroyPermanent(g.FindPermanent(source.ID()))
					case "source-control":
						g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					case "source-phased":
						g.PhaseOut(source.ID())
					}
				}
				clone := g.Clone()
				clone.ResolveTopOfStack()
				if scenario != "absent" && len(g.GetPlayer(a.PlayerID()).Hand()) != 0 {
					t.Fatal("clone changed original hand")
				}
				if g.FindPermanent(victim.ID()) != nil {
					t.Fatal("clone changed original battlefield")
				}
				g.ResolveTopOfStack()
				hand := g.GetPlayer(a.PlayerID()).Hand()
				success := scenario != "absent" && scenario != "returned"
				inHand := slices.ContainsFunc(hand, func(c mage.Card) bool { return c.ID() == victim.ID() })
				if inHand != (scenario == "absent" || success && !tc.battlefield) {
					t.Fatal("wrong hand destination or stale target followed new visit")
				}
				p := g.FindPermanent(victim.ID())
				if (p != nil) != (success && tc.battlefield) {
					t.Fatal("wrong battlefield destination")
				}
				if p != nil && (p.ControllerID() != a.PlayerID() || p.Card.Owner() != a.PlayerID() || p.Tapped || !p.HasAttr(core.AttrSummonSick)) {
					t.Fatal("wrong entry contract")
				}
				if len(g.GetPlayer(b.PlayerID()).Hand()) != 0 {
					t.Fatal("return used new source controller")
				}
				if clone.StackSize() != 0 || g.StackSize() != 0 {
					t.Fatal("return did not finish resolution")
				}
			})
		}
	}
}

func TestRONGraveyardReturnAnnouncementRollback(t *testing.T) {
	for _, tc := range graveyardReturnCards {
		for _, reason := range []string{"unpaid", "opponent", "wrong-type", "no-target", "two-targets", "source-phased", "sick", "tapped"} {
			if reason == "wrong-type" && tc.name == "Regrowth" || !tc.activated && (reason == "source-phased" || reason == "sick" || reason == "tapped") || reason == "sick" && tc.name == "Skull of Orm" {
				continue
			}
			t.Run(tc.name+"/"+reason, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				name := tc.victim
				if reason == "wrong-type" {
					name = "Forest"
				}
				victim := repositoryCard(t, "LEB", name)
				victim.SetOwner(a.PlayerID())
				holder := a
				if reason == "opponent" {
					holder = b
					victim.SetOwner(b.PlayerID())
				}
				holder.AddToGraveyard(victim)
				if reason != "unpaid" {
					for _, c := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green, core.Colorless} {
						a.ManaPool().Add(c, 10)
					}
				}
				targets := []uuid.UUID{victim.ID()}
				if reason == "no-target" {
					targets = nil
				}
				if reason == "two-targets" {
					targets = append(targets, victim.ID())
				}
				var source *mage.Permanent
				if tc.activated {
					source = g.PutOnBattlefield(card, a.PlayerID())
					if reason != "sick" {
						source.RevokeBaseAttr(core.AttrSummonSick)
					}
					if reason == "source-phased" {
						g.PhaseOut(source.ID())
					}
					if reason == "tapped" {
						g.TapPermanent(source)
					}
				} else {
					a.AddToHand(card)
				}
				if tc.activated && reason == "no-target" {
					holder.RemoveFromGraveyard(victim.ID())
				}
				wantGraveyard := len(holder.Graveyard())
				before := a.ManaPool().TotalMana()
				var err error
				if tc.activated {
					err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets)
				} else {
					err = g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0)
				}
				if err == nil {
					t.Fatal("illegal announcement succeeded")
				}
				if g.StackSize() != 0 || g.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != before || len(holder.Graveyard()) != wantGraveyard {
					t.Fatal("failed announcement changed resources")
				}
				if tc.activated && reason != "source-phased" && g.FindPermanent(source.ID()).Tapped != (reason == "tapped") {
					t.Fatal("failed activation consumed tap cost")
				}
				if !tc.activated && len(g.GetPlayer(a.PlayerID()).Hand()) != 1 {
					t.Fatal("failed cast moved spell")
				}
			})
		}
	}
}

func TestRONGraveyardReturnCopiesAndInstances(t *testing.T) {
	g, a, b := ronModalGame(0)
	g.SetActivePlayerIndex(0)
	first, second := repositoryCard(t, "LEB", "Savannah Lions"), repositoryCard(t, "LEB", "Savannah Lions")
	first.SetOwner(a.PlayerID())
	second.SetOwner(a.PlayerID())
	a.AddToGraveyard(first)
	a.AddToGraveyard(second)
	card := repositoryCard(t, "LEB", "Raise Dead")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	a.targets = []uuid.UUID{second.ID()}
	copied := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if copied == nil || copied.Targets[0] != second.ID() || original.Targets[0] != first.ID() {
		t.Fatal("copy shared target")
	}
	clone := g.Clone()
	g.ResolveTopOfStack()
	if len(a.Hand()) != 1 || a.Hand()[0].ID() != second.ID() {
		t.Fatal("copy used original target")
	}
	g.ResolveTopOfStack()
	if len(a.Hand()) != 2 || len(b.Hand()) != 0 {
		t.Fatal("original lost target")
	}
	clone.ResolveStack()
	if len(clone.GetPlayer(a.PlayerID()).Hand()) != 2 {
		t.Fatal("clone lost target bindings")
	}
	for _, tc := range graveyardReturnCards {
		x, y := repositoryCard(t, tc.set, tc.name), repositoryCard(t, tc.set, tc.name)
		if x.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == y.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
			t.Fatal("factories shared target declarations")
		}
	}
}

func TestRONGraveyardReturnInvalid(t *testing.T) {
	for _, body := range []string{
		`PutGraveyardCardIntoHand`, `PutGraveyardCardIntoHand()`, `PutGraveyardCardIntoHand(Ref_TargetGraveyardCard,You)`,
		`PutGraveyardCardIntoHand(Ref_TargetGraveyardCard())`, `PutGraveyardCardIntoHand(Ref_TargetPermanent)`, `PutGraveyardCardIntoHand(You)`,
		`TapPermanent(Ref_TargetGraveyardCard)`, `PlayerAction(Ref_TargetGraveyardCard,DrawACard)`,
		`PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard)`, `PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard,[])`,
		`PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard,[EntersTapped])`, `PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard,[EntersNormally()])`,
		`PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard,[EntersNormally,EntersNormally])`,
	} {
		assertGraveyardInvalid(t, `SpellActions(Targeted([TargetGraveyardCard(InAPlayersGraveyard(SinglePlayer(You)))],ActionList([`+body+`])))`)
	}
	for _, filter := range []string{`And([])`, `Or([IsCardtype(Creature),InAPlayersGraveyard(SinglePlayer(You))])`, `IsTapped`, `IsCardtype(Bogus)`, `IsCardtype(Creature,Artifact)`, `InAPlayersGraveyard(You)`, `InAPlayersGraveyard(SinglePlayer(Opponent))`, `InAPlayersGraveyard(SinglePlayer(You()))`, `InAPlayersGraveyard(AnyPlayer())`} {
		assertGraveyardInvalid(t, `SpellActions(Targeted([TargetGraveyardCard(`+filter+`)],ActionList([PutGraveyardCardIntoHand(Ref_TargetGraveyardCard)])))`)
	}
	for _, rule := range []string{
		`SpellActions(ActionList([PutGraveyardCardIntoHand(Ref_TargetGraveyardCard)]))`,
		`SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([PutGraveyardCardIntoHand(Ref_TargetGraveyardCard)])))`,
		`SpellActions(Modal_ChooseOne([Targeted([TargetGraveyardCard(IsCardtype(Creature))],ActionList([])),ActionList([PutGraveyardCardIntoHand(Ref_TargetGraveyardCard)])]))`,
		`Activated(PayMana([]),Targeted([TargetGraveyardCard(IsCardtype(Creature))],ActionList([]))),Activated(PayMana([]),ActionList([PutGraveyardCardIntoHand(Ref_TargetGraveyardCard)]))`,
	} {
		assertGraveyardInvalid(t, rule)
	}
}

func assertGraveyardInvalid(t *testing.T, rule string) {
	t.Helper()
	src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rule + `])`
	if strings.HasPrefix(rule, "SpellActions") {
		src = strings.Replace(src, "Cardtypes:[Artifact]", "Cardtypes:[Sorcery]", 1)
	}
	nodes, err := Parse(strings.NewReader(src), "invalid.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Compile(nodes[0]); err == nil {
		t.Fatalf("accepted %s", rule)
	}
}

func TestRONGraveyardReturnReprintsAndInventory(t *testing.T) {
	files := fstest.MapFS{}
	records := 0
	paths, err := os.ReadDir("../../../../cards/mtgish")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		if !strings.HasSuffix(path.Name(), ".ron") {
			continue
		}
		data, err := os.ReadFile("../../../../cards/mtgish/" + path.Name())
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), path.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			if !slices.ContainsFunc(graveyardReturnCards, func(tc graveyardReturnCase) bool {
				return node.Fields["Name"].String == tc.name || node.Fields["Name"].String == "Hell's Caretaker"
			}) {
				continue
			}
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			key := fmt.Sprintf("%s-%d.ron", path.Name(), records)
			files[key] = &fstest.MapFile{Data: []byte(strings.Split(string(data), "\n")[node.Position.Line-1])}
			records++
		}
	}
	if records != 24 {
		t.Fatalf("records=%d", records)
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 8 {
		t.Fatal("wrong canonical count")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != records {
		t.Fatal("inspection disagrees")
	}
	for name, role := range map[string]string{"TargetGraveyardCard": "target", "InAPlayersGraveyard": "card_filter", "Ref_TargetGraveyardCard": "card_reference", "EntersNormally": "entry_operation"} {
		if !slices.ContainsFunc(report.Signatures, func(s InventorySignature) bool { return s.Name == name && s.Role == role }) {
			t.Fatalf("missing role %s for %s", role, name)
		}
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Unsupported",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Unknown])`)}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("published partial registry")
	}
}

func TestRONGraveyardReturnSharedContexts(t *testing.T) {
	for _, context := range []string{"spell", "trigger", "modal"} {
		for _, filter := range []string{`IsCardtype(Creature)`, `InAPlayersGraveyard(AnyPlayer)`, `And([InAPlayersGraveyard(AnyPlayer),IsCardtype(Creature)])`} {
			t.Run(context+"/"+filter, func(t *testing.T) {
				g, a, b := ronModalGame(1)
				g.SetActivePlayerIndex(0)
				victim := repositoryCard(t, "LEB", "Savannah Lions")
				victim.SetOwner(b.PlayerID())
				b.AddToGraveyard(victim)
				actions := `Targeted([TargetGraveyardCard(` + filter + `)],ActionList([PutGraveyardCardIntoHand(Ref_TargetGraveyardCard)]))`
				kind, rule := "Instant", `SpellActions(`+actions+`)`
				if context == "trigger" {
					kind = "Artifact"
					rule = `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),` + actions + `)`
				}
				if context == "modal" {
					rule = `SpellActions(Modal_ChooseOne([Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,DrawACard)])),` + actions + `]))`
				}
				factory := permanentScopeFactory(t, kind, rule)
				card := factory()
				card.SetOwner(a.PlayerID())
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				if context == "trigger" {
					a.targets = []uuid.UUID{victim.ID()}
					g.PutOnBattlefield(card, a.PlayerID())
					g.PutTriggersOnStack()
				} else {
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim.ID()}, 0); err != nil {
						t.Fatal(err)
					}
				}
				g.ResolveStack()
				if len(b.Hand()) != 1 || b.Hand()[0].ID() != victim.ID() || len(a.Hand()) != 0 {
					t.Fatal("explicit any-graveyard target returned to wrong owner")
				}
				if context == "modal" && a.prompts != 1 {
					t.Fatal("mode selection timing changed")
				}
			})
		}
	}
}

func TestRONGraveyardReturnNormalEntry(t *testing.T) {
	for _, scenario := range []string{"entry", "copied-controller", "two-actions"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			nodes, err := Parse(strings.NewReader(`Card(Name:"Entry creature",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:2),Rules:[AsPermanentEnters(ThisPermanent,[EntersTapped]),TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([GainLife(Integer(2))]))])`), "entry.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			victim := factory()
			victim.SetOwner(a.PlayerID())
			a.AddToGraveyard(victim)
			card := repositoryCard(t, "LEB", "Resurrection")
			if scenario == "copied-controller" {
				card = permanentScopeFactory(t, "Sorcery", `SpellActions(Targeted([TargetGraveyardCard(IsCardtype(Creature))],ActionList([PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard,[EntersNormally])])))`)()
			}
			if scenario == "two-actions" {
				card = permanentScopeFactory(t, "Sorcery", `SpellActions(Targeted([TargetGraveyardCard(IsCardtype(Creature))],ActionList([PutGraveyardCardIntoHand(Ref_TargetGraveyardCard),PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard,[EntersNormally])])))`)()
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.White, 4)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			controller := a.PlayerID()
			if scenario == "copied-controller" {
				copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
				if copied == nil {
					t.Fatal("copy failed")
				}
				controller = b.PlayerID()
			}
			g.ResolveTopOfStack()
			if scenario == "two-actions" {
				if len(a.Hand()) != 1 || a.Hand()[0].ID() != victim.ID() || g.FindPermanent(victim.ID()) != nil {
					t.Fatal("second instruction followed moved card")
				}
				return
			}
			p := g.FindPermanent(victim.ID())
			if p == nil || !p.Tapped || p.Card.Owner() != a.PlayerID() || p.ControllerID() != controller {
				t.Fatal("normal entry lost replacement or controller")
			}
			if g.GetPlayer(controller).Life() != 20 {
				t.Fatal("entry trigger resolved inside return")
			}
			g.PutTriggersOnStack()
			g.ResolveTopOfStack()
			if g.GetPlayer(controller).Life() != 22 {
				t.Fatal("entry trigger used wrong controller")
			}
			if scenario == "copied-controller" {
				g.ResolveStack()
				if a.Life() != 20 || b.Life() != 22 {
					t.Fatal("original spell followed returned target")
				}
			}
		})
	}
}

func TestRONGraveyardReturnBattlefieldRequiresCreatureTarget(t *testing.T) {
	for _, filter := range []string{`InAPlayersGraveyard(SinglePlayer(You))`, `IsCardtype(Instant)`, `IsCardtype(Enchantment)`} {
		assertGraveyardInvalid(t, `SpellActions(Targeted([TargetGraveyardCard(`+filter+`)],ActionList([PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard,[EntersNormally])])))`)
	}
}

func TestRONGraveyardReturnCaretaker(t *testing.T) {
	for _, scenario := range []string{"self-sacrifice", "main-step", "opponent-upkeep", "sick", "no-target", "same-sacrifice-target", "opponent-target"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			g.SetStep(core.Upkeep)
			card := repositoryCard(t, "LEG", "Hell's Caretaker")
			card.SetOwner(a.PlayerID())
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			if scenario != "sick" {
				source.RevokeBaseAttr(core.AttrSummonSick)
			}
			victim := repositoryCard(t, "LEB", "Savannah Lions")
			victim.SetOwner(a.PlayerID())
			holder := a
			if scenario == "opponent-target" {
				holder = b
				victim.SetOwner(b.PlayerID())
			}
			holder.AddToGraveyard(victim)
			targets := []uuid.UUID{victim.ID()}
			switch scenario {
			case "main-step":
				g.SetStep(core.PrecombatMain)
			case "opponent-upkeep":
				g.SetActivePlayerIndex(1)
			case "no-target":
				holder.RemoveFromGraveyard(victim.ID())
				targets = nil
			case "same-sacrifice-target":
				targets = []uuid.UUID{source.ID()}
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets)
			if scenario != "self-sacrifice" {
				if err == nil || g.StackSize() != 0 || g.FindPermanent(source.ID()) == nil || g.FindPermanent(source.ID()).Tapped {
					t.Fatal("illegal activation paid a cost")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(source.ID()) != nil || !slices.ContainsFunc(a.Graveyard(), func(c mage.Card) bool { return c.ID() == card.ID() }) || g.FindPermanent(victim.ID()) != nil {
				t.Fatal("self-sacrifice or return timing incorrect")
			}
			clone := g.Clone()
			g.ResolveStack()
			p := g.FindPermanent(victim.ID())
			if p == nil || p.ControllerID() != a.PlayerID() || clone.FindPermanent(victim.ID()) != nil {
				t.Fatal("sacrificed source stopped return or changed clone")
			}
			clone.ResolveStack()
			if clone.FindPermanent(victim.ID()) == nil {
				t.Fatal("clone lost activation")
			}
		})
	}
}

func TestRONGraveyardReturnConjunctionAndCopyLegality(t *testing.T) {
	for _, opponent := range []bool{false, true} {
		g, a, b := ronGame()
		creature := repositoryCard(t, "LEB", "Savannah Lions")
		creature.SetOwner(a.PlayerID())
		a.AddToGraveyard(creature)
		artifact := repositoryCard(t, "LEB", "Sol Ring")
		artifact.SetOwner(a.PlayerID())
		a.AddToGraveyard(artifact)
		both := repositoryCard(t, "ARN", "Brass Man")
		both.SetOwner(a.PlayerID())
		holder := a
		if opponent {
			holder = b
			both.SetOwner(b.PlayerID())
		}
		holder.AddToGraveyard(both)
		card := permanentScopeFactory(t, "Sorcery", `SpellActions(Targeted([TargetGraveyardCard(And([InAPlayersGraveyard(AnyPlayer),And([IsCardtype(Creature),IsCardtype(Artifact)]),InAPlayersGraveyard(SinglePlayer(You))]))],ActionList([PutGraveyardCardIntoHand(Ref_TargetGraveyardCard)])))`)()
		target := card.Abilities()[0].(*mage.ActionDefinition).Targets()[0]
		possible := target.Possible(a.PlayerID(), card, g)
		if slices.Contains(possible, creature.ID()) || slices.Contains(possible, artifact.ID()) || slices.Contains(possible, both.ID()) == opponent {
			t.Fatal("nested conjunction changed card or player restrictions")
		}
	}
	g, a, b := ronGame()
	victim := repositoryCard(t, "LEB", "Savannah Lions")
	victim.SetOwner(a.PlayerID())
	a.AddToGraveyard(victim)
	card := repositoryCard(t, "LEB", "Resurrection")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.White, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(victim.ID()) != nil || len(a.Graveyard()) != 1 {
		t.Fatal("copy kept original controller's graveyard permission")
	}
	g.ResolveTopOfStack()
	p := g.FindPermanent(victim.ID())
	if p == nil || p.ControllerID() != a.PlayerID() {
		t.Fatal("failed copy changed original target")
	}
}
