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
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

var landDependentCards = []struct{ set, name string }{
	{"ARN", "Dandân"}, {"LEB", "Sea Serpent"}, {"LEB", "Pirate Ship"},
	{"3ED", "Island Fish Jasconius"}, {"FEM", "Vodalian Knights"},
}

func TestRONLandDependenceCards(t *testing.T) {
	for _, tc := range landDependentCards {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			own := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(b.PlayerID())
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			check := func(game *mage.Game, want bool) {
				t.Helper()
				game.ApplyContinuousEffects()
				if game.FindPermanent(p.ID()).CanDeclareAsAttacker(game) != want {
					t.Fatalf("attack legality, want %v", want)
				}
			}
			check(g, false)
			defender := g.PutOnBattlefield(repositoryCard(t, "LEB", "Tropical Island"), b.PlayerID())
			check(g, true)
			clone := g.Clone()
			clone.PhaseOut(defender.ID())
			check(clone, false)
			check(g, true)
			g.PhaseOut(defender.ID())
			check(g, false)
			g.PhaseIn([]uuid.UUID{defender.ID()})
			check(g, true)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: own.ID(), TargetID: defender.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			check(g, false)
			remover := g.PutOnBattlefield(mage.NewArtifact("Remover", ""), a.PlayerID())
			removal := mage.RemoveAllAbilities(p.ID(), core.WhileOnBattlefield)
			removal.SetSourceID(remover.ID())
			g.AddContinuousEffect(removal)
			check(g, true)
			g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
			check(g, false)
			g.PhaseOut(own.ID())
			g.RemoveFromBattlefield(g.FindPermanent(defender.ID()))
			g.CheckStateTriggers()
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatalf("state trigger count = %d", g.StackSize())
			}
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
			g.ResolveStack()
			if g.FindPermanent(p.ID()) != nil || !slices.ContainsFunc(b.Graveyard(), func(c mage.Card) bool { return c.ID() == card.ID() }) {
				t.Fatal("state trigger did not sacrifice to owner after condition became false")
			}
		})
	}
}

func TestRONLandDependenceStateInstances(t *testing.T) {
	for _, scenario := range []string{"counter", "control", "returned", "phase", "copy", "clone", "two"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			island := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
			card := repositoryCard(t, "ARN", "Dandân")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.BouncePermanentToHand(island)
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("missing trigger")
			}
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("active state trigger repeated")
			}
			switch scenario {
			case "counter":
				g.CounterStackObject(g.StackPeek().ID)
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatal("countered state trigger did not rearm")
				}
			case "control":
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: island.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "returned":
				g.BouncePermanentToHand(p)
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
				a.RemoveFromHand(card.ID())
				returned := g.PutOnBattlefield(card, a.PlayerID())
				g.ResolveStack()
				if g.FindPermanent(returned.ID()) == nil {
					t.Fatal("old trigger sacrificed returned incarnation")
				}
				return
			case "phase":
				g.PhaseOut(p.ID())
				g.ResolveStack()
				if g.FindPermanentIncludingPhased(p.ID()) == nil {
					t.Fatal("trigger sacrificed phased source")
				}
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
				g.PhaseIn([]uuid.UUID{p.ID()})
				return
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveTopOfStack()
				if g.FindPermanent(p.ID()) == nil {
					t.Fatal("copy controller sacrificed another player's permanent")
				}
			case "clone":
				clone := g.Clone()
				clone.ResolveStack()
				if clone.FindPermanent(p.ID()) != nil || g.FindPermanent(p.ID()) == nil {
					t.Fatal("clone resolution isolation")
				}
			case "two":
				other := g.PutOnBattlefield(repositoryCard(t, "ARN", "Dandân"), a.PlayerID())
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 2 {
					t.Fatal("factory state trigger identities collided")
				}
				g.ResolveTopOfStack()
				if g.FindPermanent(other.ID()) != nil || g.FindPermanent(p.ID()) == nil {
					t.Fatal("second trigger used first source")
				}
			}
			g.ResolveStack()
			if (g.FindPermanent(p.ID()) != nil) != (scenario == "control") {
				t.Fatal("source/controller identity")
			}
		})
	}
}

func TestRONLandDependenceNativeAttack(t *testing.T) {
	a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
	g := mage.NewGame(a, b)
	g.SetTurn(1)
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
	p := g.PutOnBattlefield(repositoryCard(t, "ARN", "Dandân"), a.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	a.SetAttackers(1, []string{"Dandân"})
	g.RunStepWithPriority(core.DeclareAttackers)
	if g.GetCombat().IsAttacking(p.ID()) || g.FindPermanent(p.ID()).Tapped {
		t.Fatal("illegal attack committed")
	}
	land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
	g.RunStepWithPriority(core.DeclareAttackers)
	if !g.GetCombat().IsAttacking(p.ID()) {
		t.Fatal("legal attack rejected")
	}
	g.RemoveFromBattlefield(land)
	g.ApplyContinuousEffects()
	if !g.GetCombat().IsAttacking(p.ID()) {
		t.Fatal("restriction removed declared attacker")
	}
}

func TestRONLandDependenceOtherAbilities(t *testing.T) {
	for _, name := range []string{"Pirate Ship", "Vodalian Knights", "Island Fish Jasconius"} {
		t.Run(name, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			g.SetStep(core.PrecombatMain)
			ownIsland := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
			set := "LEB"
			if name == "Vodalian Knights" {
				set = "FEM"
			}
			if name == "Island Fish Jasconius" {
				set = "3ED"
			}
			p := g.PutOnBattlefield(repositoryCard(t, set, name), a.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			if name == "Island Fish Jasconius" {
				g.TapPermanent(p)
				g.RunStepWithPriority(core.Untap)
				if !g.FindPermanent(p.ID()).Tapped {
					t.Fatal("fish untapped during untap step")
				}
				a.ManaPool().Add(core.Blue, 3)
				g.RunStepWithPriority(core.Upkeep)
				g.ResolveStack()
				if g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 0 || a.offers != 1 {
					t.Fatal("fish upkeep payment")
				}
				return
			}
			index := 2
			var targets []uuid.UUID
			if name == "Pirate Ship" {
				index = 1
				targets = []uuid.UUID{b.PlayerID()}
			}
			if name == "Vodalian Knights" {
				g.TapPermanent(ownIsland)
				if !p.HasKeyword(core.FirstStrike) {
					t.Fatal("first strike missing")
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), index, targets); err == nil {
					t.Fatal("unpaid activation")
				}
				a.ManaPool().Add(core.Blue, 1)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), index, targets); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if name == "Pirate Ship" {
				if b.Life() != 19 || !g.FindPermanent(p.ID()).Tapped {
					t.Fatal("ship damage or tap cost")
				}
			} else {
				if !g.FindPermanent(p.ID()).HasKeyword(core.Flying) {
					t.Fatal("flying not granted")
				}
				g.RunStepWithPriority(core.Cleanup)
				if g.FindPermanent(p.ID()).HasKeyword(core.Flying) {
					t.Fatal("flying did not expire")
				}
			}
		})
	}
}

func TestRONLandDependenceInvalid(t *testing.T) {
	attack := `PermanentRuleEffect(ThisPermanent,[CantAttackUnlessDefendingPlayer(PlayerPassesFilter(DefendingPlayer,ControlsA(IsLandType(Island))))])`
	state := `TriggerA(WhenAPlayerControlsNoPermanents(SinglePlayer(You),IsLandType(Island)),ActionList([SacrificePermanent(ThisPermanent)]))`
	for _, rule := range []string{
		`PermanentRuleEffect(ThisPermanent,[CantAttackUnlessDefendingPlayer])`,
		`PermanentRuleEffect(ThisPermanent,[CantAttackUnlessDefendingPlayer()])`,
		strings.Replace(attack, "[CantAttackUnless", "[DoesntUntapDuringControllersUntap,CantAttackUnless", 1),
		strings.Replace(attack, "PlayerPassesFilter(", "PlayerPassesFilter(You,", 1),
		strings.Replace(attack, "ControlsA(IsLandType(Island))", "ControlsA()", 1),
		strings.Replace(state, "SinglePlayer(You)", "SinglePlayer(You())", 1),
		strings.Replace(state, "IsLandType(Island)", "IsLandType()", 1),
		strings.Replace(state, "IsLandType(Island)", "IsLandType(Island,Swamp)", 1),
		strings.Replace(state, "IsLandType(Island)", "IsCardtype(Land())", 1),
		strings.Replace(attack, "ThisPermanent", "HostPermanent", 1),
		strings.Replace(attack, "ThisPermanent", "ThisPermanent()", 1),
		strings.Replace(attack, "DefendingPlayer,", "You,", 1),
		strings.Replace(attack, "ControlsA(", "ControlsNo(", 1),
		strings.Replace(attack, "IsLandType(Island)", "IsCardtype(Creature)", 1),
		strings.Replace(attack, "IsLandType(Island)", "IsLandType(Island())", 1),
		strings.Replace(state, "SinglePlayer(You)", "AnyPlayer", 1),
		strings.Replace(state, "SinglePlayer(You)", "SinglePlayer(Opponent)", 1),
		strings.Replace(state, "IsLandType(Island)", "IsLandType(Unknown)", 1),
		strings.Replace(state, "ThisPermanent", "Trigger_ThatPermanent", 1),
		strings.Replace(state, "WhenAPlayerControlsNoPermanents(", "WhenAPlayerControlsNoPermanents(You,", 1),
	} {
		assertConditionalRejected(t, rule)
	}
}

func TestRONLandDependenceCasting(t *testing.T) {
	for _, tc := range landDependentCards {
		t.Run(tc.name, func(t *testing.T) {
			g, a, _ := ronGame()
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed cast changed state")
			}
			for _, color := range []core.Color{core.Blue, core.Colorless} {
				a.ManaPool().Add(color, 10)
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(card.ID()) == nil || len(a.Hand()) != 0 || a.ManaPool().TotalMana() == 20 {
				t.Fatal("paid creature cast")
			}
		})
	}
}

func TestRONLandDependenceSubtypes(t *testing.T) {
	for _, subtype := range []string{"Plains", "Island", "Swamp", "Mountain", "Forest", "Desert"} {
		t.Run(subtype, func(t *testing.T) {
			rule := fmt.Sprintf(`PermanentRuleEffect(ThisPermanent,[CantAttackUnlessDefendingPlayer(PlayerPassesFilter(DefendingPlayer,ControlsA(IsLandType(%s))))]),TriggerA(WhenAPlayerControlsNoPermanents(SinglePlayer(You),IsLandType(%s)),ActionList([SacrificePermanent(ThisPermanent)]))`, subtype, subtype)
			nodes, err := Parse(strings.NewReader(`Card(Name:"Subject",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:2),Rules:[`+rule+`])`), "land-dependence.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			own := g.PutOnBattlefield(mage.NewLand("Own", mage.WithSubTypes(subtype)), a.PlayerID())
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			g.ApplyContinuousEffects()
			if p.CanDeclareAsAttacker(g) {
				t.Fatal("own land satisfied defending player")
			}
			g.PutOnBattlefield(mage.NewLand("Other", mage.WithSubTypes(subtype)), b.PlayerID())
			g.ApplyContinuousEffects()
			if !g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("matching defending land rejected")
			}
			g.BouncePermanentToHand(own)
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("opponent land satisfied source controller")
			}
			g.ResolveStack()
			if g.FindPermanent(p.ID()) != nil {
				t.Fatal("no-land state action")
			}
		})
	}
}

func TestRONLandDependenceInactiveSources(t *testing.T) {
	for _, scenario := range []string{"phase", "remove-ability", "control"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
			p := g.PutOnBattlefield(repositoryCard(t, "ARN", "Dandân"), a.PlayerID())
			switch scenario {
			case "phase":
				g.PhaseOut(p.ID())
			case "remove-ability":
				remover := g.PutOnBattlefield(mage.NewArtifact("Remover", ""), a.PlayerID())
				removal := mage.RemoveAllAbilities(p.ID(), core.WhileOnBattlefield)
				removal.SetSourceID(remover.ID())
				g.AddContinuousEffect(removal)
				g.ApplyContinuousEffects()
			case "control":
				other := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.ApplyContinuousEffects()
			}
			g.BouncePermanentToHand(land)
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("inactive source or old controller triggered")
			}
		})
	}
}

func TestRONLandDependenceReprints(t *testing.T) {
	paths, err := os.ReadDir("../../../../cards/mtgish")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
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
		for _, n := range nodes {
			if !slices.ContainsFunc(landDependentCards, func(c struct{ set, name string }) bool { return c.name == n.Fields["Name"].String }) {
				continue
			}
			files[fmt.Sprintf("%s-%d.ron", path.Name(), n.Position.Line)] = &fstest.MapFile{Data: []byte(strings.Split(string(data), "\n")[n.Position.Line-1])}
		}
	}
	r, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.RegisteredCardNames()) != 5 {
		t.Fatal("reprint names")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 16 {
		t.Fatalf("compiled reprints = %d", report.Compiled)
	}
	for name, role := range map[string]string{"CantAttackUnlessDefendingPlayer": "permanent_rule_operation", "PlayerPassesFilter": "defending_player_condition", "ControlsA": "static_player_filter", "DefendingPlayer": "player_reference", "WhenAPlayerControlsNoPermanents": "event"} {
		if !slices.ContainsFunc(report.Signatures, func(s InventorySignature) bool { return s.Name == name && s.Role == role }) {
			t.Errorf("missing inventory role %s: %s", name, role)
		}
	}
	for _, tc := range landDependentCards {
		first, _ := r.CreateCard(tc.name)
		second, _ := r.CreateCard(tc.name)
		if first.ID() == second.ID() || first.Abilities()[0] == second.Abilities()[0] {
			t.Fatal("factory instance sharing")
		}
	}
}
