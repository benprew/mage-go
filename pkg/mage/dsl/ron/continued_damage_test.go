package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONContinuedDamageEye(t *testing.T) {
	for _, set := range []string{"ARN", "3ED"} {
		t.Run(set, func(t *testing.T) {
			factory, err := Compile(staticBaseRecord(t, set, "Eye for an Eye"))
			if err != nil {
				t.Fatal(err)
			}
			_, a, b := ronGame()
			chooser := &ronSourceChoicePlayer{BasePlayer: a}
			g := mage.NewGame(chooser, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			source := g.PutOnBattlefield(mage.NewCreature("Source", "{R}", 2, 10), b.PlayerID())
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			a.ManaPool().Add(core.White, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if chooser.calls != 0 {
				t.Fatal("choice before resolution")
			}
			g.ResolveStack()
			if chooser.calls != 1 {
				t.Fatal("missing source choice")
			}
			for _, game := range []*mage.Game{g, g.Clone()} {
				game.DealDamageToPlayerFromSource(game.GetPlayer(a.PlayerID()), 3, game.CaptureDamageSource(source.ID()))
				if game.GetPlayer(a.PlayerID()).Life() != 17 || game.GetPlayer(b.PlayerID()).Life() != 17 {
					t.Fatal("continued damage failed")
				}
			}
		})
	}
}

func TestRONContinuedDamageLifecycle(t *testing.T) {
	for _, state := range []string{"departed", "returned", "phase", "phase-cycle", "control", "cleanup", "copy", "foreign-copy", "pending-clone", "simultaneous", "prevent-original", "prevent-additional"} {
		t.Run(state, func(t *testing.T) {
			a := &generalRONSourcePlayer{BasePlayer: mage.NewBasePlayer("A")}
			b := &generalRONSourcePlayer{BasePlayer: mage.NewBasePlayer("B")}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			source := g.PutOnBattlefield(mage.NewCreature("Selected", "{R}", 2, 10), b.PlayerID())
			a.wanted = source.ID()
			b.wanted = source.ID()
			card := repositoryCard(t, "ARN", "Eye for an Eye")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.White, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if state == "copy" || state == "foreign-copy" {
				controller := a.PlayerID()
				if state == "foreign-copy" {
					controller = b.PlayerID()
				}
				if g.CopyStackObjectDirect(g.StackPeek(), controller, false) == nil {
					t.Fatal("copy failed")
				}
			}
			games := []*mage.Game{g}
			if state == "pending-clone" {
				games = append(games, g.Clone())
			}
			for _, game := range games {
				game.ResolveStack()
				saved := game.CaptureDamageSource(source.ID())
				switch state {
				case "departed", "returned":
					game.BouncePermanentToHand(game.FindPermanent(source.ID()))
					if state == "returned" {
						game.GetPlayer(b.PlayerID()).RemoveFromHand(source.ID())
						game.PutOnBattlefield(source.Card, b.PlayerID())
						saved = game.CaptureDamageSource(source.ID())
					}
				case "phase":
					game.PhaseOut(source.ID())
				case "phase-cycle":
					game.PhaseIn(game.PhaseOut(source.ID()))
				case "control":
					game.AddControlEffect(mage.ControlEffectSpec{SourceID: uuid.New(), TargetID: source.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
					saved = game.CaptureDamageSource(source.ID())
				case "cleanup":
					game.GetEffects().ClearReplacementsEndOfTurn()
				case "prevent-original":
					game.AddPreventionShield(a.PlayerID(), 2)
				case "prevent-additional":
					game.AddPreventionShield(b.PlayerID(), 2)
				}
				if state == "simultaneous" {
					game.DealSimultaneousDamage(mage.PlayerDamageAssignment(game.GetPlayer(a.PlayerID()), 1, saved), mage.PlayerDamageAssignment(game.GetPlayer(a.PlayerID()), 2, saved))
				} else {
					game.DealDamageToPlayerFromSource(game.GetPlayer(a.PlayerID()), 3, saved)
				}
				wantA, wantB := 17, 17
				if state == "returned" || state == "cleanup" {
					wantB = 20
				}
				if state == "control" {
					wantA, wantB = 14, 20
				}
				if state == "copy" {
					wantB = 14
				}
				if state == "prevent-original" {
					wantA = 19
				}
				if state == "prevent-additional" {
					wantB = 19
				}
				if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB {
					t.Fatalf("life %d %d want %d %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life(), wantA, wantB)
				}
				if state == "foreign-copy" {
					game.DealDamageToPlayerFromSource(game.GetPlayer(b.PlayerID()), 3, saved)
					if game.GetPlayer(b.PlayerID()).Life() != 11 {
						t.Fatal("foreign copy lost its own recipient or source controller")
					}
				}
			}
		})
	}
}

const continuedDamageRON = `CreateFutureReplaceWouldDealDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Player(You)),[ContinueDealingDamage,SpellDealsDamage(ThisSpell,WouldDealDamage_ThatMuchDamage,Player(WouldDealDamage_ControllerOfDamageSource))])`

func TestRONContinuedDamageInvalid(t *testing.T) {
	for _, change := range [][2]string{
		{"TheChosenDamageSource", "TheChosenDamageSource()"}, {"Player(You)", "Player(Trigger_ThatPlayer)"}, {"Player(You)", "Player()"},
		{"ContinueDealingDamage,", ""}, {"ContinueDealingDamage,", "ContinueDealingDamage(),"}, {"ContinueDealingDamage,", "ContinueDealingDamage,ContinueDealingDamage,"},
		{"ThisSpell", "ThisPermanent"}, {"ThisSpell", "ThisSpell()"}, {"WouldDealDamage_ThatMuchDamage", "Integer(3)"}, {"WouldDealDamage_ThatMuchDamage", "WouldDealDamage_ThatMuchDamage()"},
		{"Player(WouldDealDamage_ControllerOfDamageSource)", "Player(You)"}, {"Player(WouldDealDamage_ControllerOfDamageSource)", "Permanent(ThisPermanent)"}, {"WouldDealDamage_ControllerOfDamageSource", "WouldDealDamage_ControllerOfDamageSource()"},
	} {
		body := strings.Replace(continuedDamageRON, change[0], change[1], 1)
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([ChooseADamageSource(AnyDamageSource),` + body + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid shape accepted", body)
		}
	}
	for _, body := range []string{
		continuedDamageRON,
		`ChooseADamageSource(AnyDamageSource),` + continuedDamageRON + `,SpellDealsDamage(ThisSpell,WouldDealDamage_ThatMuchDamage,Player(You))`,
		`ChooseADamageSource(AnyDamageSource),` + continuedDamageRON + `,SpellDealsDamage(ThisSpell,Integer(1),Player(WouldDealDamage_ControllerOfDamageSource))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + body + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("unbound event value accepted", body)
		}
	}
	nodes, err := Parse(strings.NewReader(continuedDamageRON), "context.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{permanent: true, resolving: true}, {resolving: false}, {resolving: true, enteringSource: true}, {resolving: true, drawReplacementBody: true}, {resolving: true, delayedStep: true}} {
		ctx.chosenDamageSource = &chosenDamageSourcePlan{}
		if _, err := compileEffect(nodes[0], ctx); err == nil {
			t.Fatal("invalid context accepted")
		}
	}
}

func TestRONContinuedDamageInventory(t *testing.T) {
	report, err := InspectFS(fstest.MapFS{"eye.ron": {Data: []byte(mustRONSet(t, "ARN"))}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"ContinueDealingDamage": "redirection_operation", "WouldDealDamage_ThatMuchDamage": "proposed_damage_value", "WouldDealDamage_ControllerOfDamageSource": "proposed_damage_player_reference"} {
		found := false
		for _, entry := range report.Signatures {
			if entry.Name == name && entry.Role == role {
				found = true
				if entry.CompiledOccurrences != 1 {
					t.Fatal("signature not compiled", entry)
				}
			}
		}
		if !found {
			t.Fatal("missing role", name, role)
		}
	}
}
