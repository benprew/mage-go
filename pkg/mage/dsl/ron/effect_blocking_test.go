package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronEffectBlockingPlayer struct {
	mage.Player
	name   string
	accept bool
}

func TestRONEffectBlockingLegalityAndLifetime(t *testing.T) {
	for _, change := range []string{"decline", "not-blocking", "tapped", "flying", "shroud", "target-phase", "target-return", "target-control", "phase-cycle", "attacker-leaves", "copy", "clone"} {
		t.Run(change, func(t *testing.T) {
			a := &ronEffectBlockingPlayer{Player: mage.NewBasePlayer("A"), name: "Second", accept: change != "decline"}
			b := &ronEffectBlockingPlayer{Player: mage.NewBasePlayer("B"), accept: false}
			g := mage.NewGame(a, b)
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 10), a.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 3, 10), a.PlayerID())
			blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 1, 10), b.PlayerID())
			blocker.Card.SetOwner(b.PlayerID())
			first.RevokeBaseAttr(core.AttrSummonSick)
			second.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{first.ID(), second.ID()})
			g.SetStep(core.DeclareBlockers)
			if change != "not-blocking" {
				g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: first.ID()}})
			}
			card := repositoryCard(t, "LEA", "False Orders")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Red, 1)
			g.SetStep(core.DeclareAttackers)
			if g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{blocker.ID()}, 0) == nil || a.ManaPool().TotalMana() != 1 {
				t.Fatal("wrong timing or failed-cast rollback")
			}
			g.SetStep(core.DeclareBlockers)
			if g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID()}, 0) == nil || a.ManaPool().TotalMana() != 1 {
				t.Fatal("attacking player's creature was a legal target")
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{blocker.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "tapped":
				g.TapPermanent(blocker)
			case "flying":
				second.GrantBaseAttr(core.Flying)
			case "shroud":
				second.GrantBaseAttr(core.Shroud)
			case "target-phase":
				g.PhaseOut(blocker.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(blocker.ID()))
			case "target-return":
				g.RemoveFromBattlefield(blocker)
				g.PutOnBattlefield(blocker.Card, b.PlayerID())
			case "target-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: blocker.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "attacker-leaves":
				g.RemoveFromBattlefield(second)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "clone":
				original := g
				g = g.Clone()
				g.SetPlayerAt(0, &ronEffectBlockingPlayer{Player: g.GetPlayer(a.PlayerID()), name: "Second", accept: true})
				g.ResolveStack()
				if !original.GetCombat().IsBlocking(blocker.ID()) || original.GetCombat().GroupFor(second.ID()).Blocked {
					t.Fatal("clone changed original combat")
				}
			}
			g.ResolveStack()
			want := change != "decline" && change != "target-phase" && change != "target-return" && change != "target-control" && change != "attacker-leaves"
			if g.GetCombat().GroupFor(second.ID()) != nil && g.GetCombat().GroupFor(second.ID()).Blocked != want {
				t.Fatal("wrong pending block or target recheck")
			}
			if change == "decline" && (!g.GetCombat().GroupFor(first.ID()).Unblocked || g.GetCombat().IsBlocking(blocker.ID())) {
				t.Fatal("decline canceled the preceding removal")
			}
			if change == "phase-cycle" && !g.GetCombat().GroupFor(first.ID()).Blocked {
				t.Fatal("phase cycle incorrectly unblocked the former attacker")
			}
		})
	}
}

func TestRONEffectBlockingInvalid(t *testing.T) {
	good := `ChooseAnAttackingCreatureForBlockerToBlock(And([IsAttacking,IsCardtype(Creature)]),ThisPermanent)`
	for _, action := range []string{
		`ChooseAnAttackingCreatureForBlockerToBlock`, `ChooseAnAttackingCreatureForBlockerToBlock()`,
		`ChooseAnAttackingCreatureForBlockerToBlock(IsAttacking)`, `ChooseAnAttackingCreatureForBlockerToBlock(IsAttacking,ThisPermanent,ThisPermanent)`,
		`ChooseAnAttackingCreatureForBlockerToBlock(UnknownFilter,ThisPermanent)`, `ChooseAnAttackingCreatureForBlockerToBlock(IsAttacking,You)`,
		`ChooseAnAttackingCreatureForBlockerToBlock(IsAttacking,ThisPermanent())`, `ChooseAnAttackingCreatureForBlockerToBlock(IsAttacking,Ref_TargetPermanent)`,
		`ChooseAnAttackingCreatureForBlockerToBlock(IsAttacking,Trigger_ThatPermanent)`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[Activated(PayMana([]),ActionList([MayAction(`+action+`)]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid choice", action)
		}
	}
	nodes, err := Parse(strings.NewReader(good), "context.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{permanent: true}, {permanent: true, resolving: true, enteringSource: true}, {permanent: true, resolving: true, drawReplacementBody: true}} {
		if _, err := compileOptionalEffectBlocking(nodes[0], ctx); err == nil {
			t.Fatal("accepted invalid body context")
		}
	}
	if _, err := compileEffect(nodes[0], scope{permanent: true, resolving: true}); err == nil {
		t.Fatal("accepted required choice as optional")
	}
}

func TestRONEffectBlockingLoadAndInventory(t *testing.T) {
	files := fstest.MapFS{}
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.HasPrefix(line, `Card(Name: "False Orders",`) {
				files[set+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	if len(files) != 5 {
		t.Fatal("missing records")
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("False Orders")
	if err != nil {
		t.Fatal("missing registered card")
	}
	second, err := registry.CreateCard("False Orders")
	if err != nil || first.ID() == second.ID() {
		t.Fatal("shared factory state")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 5 {
		t.Fatal("inventory", err)
	}
	for _, name := range []string{"IsAttacking", "Ref_TargetPermanent"} {
		want := "permanent_filter"
		if name == "Ref_TargetPermanent" {
			want = "permanent_reference"
		}
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == want {
				found = true
			}
		}
		if !found {
			t.Fatal("missing semantic role", name)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([UnsupportedAction]))])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("published rejected collection")
	}
	if _, err := registry.CreateCard("False Orders"); err != nil {
		t.Fatal("failed load erased prior registry")
	}
}

func TestRONEffectBlockingIndependentInstances(t *testing.T) {
	a := &ronEffectBlockingPlayer{Player: mage.NewBasePlayer("A"), name: "Attacker", accept: true}
	b := &ronEffectBlockingPlayer{Player: mage.NewBasePlayer("B"), accept: false}
	g := mage.NewGame(a, b)
	attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 10), a.PlayerID())
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 10), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 10), b.PlayerID())
	attacker.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
	g.SetStep(core.DeclareBlockers)
	g.ExecuteBlockers(nil)
	for _, id := range []uuid.UUID{first.ID(), second.ID()} {
		card := repositoryCard(t, "LEA", "False Orders")
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.Red, 1)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{id}, 0); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	if len(g.GetCombat().GroupFor(attacker.ID()).BlockerIDs) != 2 {
		t.Fatal("factories shared target bindings")
	}
	if err := mage.ApplyEffect(g, mage.RemovePermanentFromCombatAndUnblock(mage.This), first.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !g.GetCombat().GroupFor(attacker.ID()).Blocked {
		t.Fatal("lost the second block's history")
	}
	card := repositoryCard(t, "LEA", "False Orders")
	card.SetOwner(b.PlayerID())
	b.AddToHand(card)
	b.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(b.PlayerID(), card.ID(), []uuid.UUID{second.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.GetCombat().IsBlocking(second.ID()) || !g.GetCombat().GroupFor(attacker.ID()).Blocked {
		t.Fatal("wrong chooser or historical unblocking")
	}
}

func TestRONEffectBlockingResolvingScopes(t *testing.T) {
	for _, body := range []string{
		`Activated(PayMana([]),ActionList([MayAction(ChooseAnAttackingCreatureForBlockerToBlock(IsAttacking,ThisPermanent))]))`,
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([MayAction(ChooseAnAttackingCreatureForBlockerToBlock(IsAttacking,ThisPermanent))]))`,
		`Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([MayAction(ChooseAnAttackingCreatureForBlockerToBlock(IsAttacking,Ref_TargetPermanent))])))`,
	} {
		factory := permanentScopeFactory(t, "Artifact", body)
		card := factory()
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

func (p *ronEffectBlockingPlayer) ChooseMayAbility(string) bool { return p.accept }

func (p *ronEffectBlockingPlayer) ChoosePermanent(candidates []*mage.Permanent, _ string, _ mage.GameReader) *mage.Permanent {
	for _, candidate := range candidates {
		if candidate.Name() == p.name {
			return candidate
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	return nil
}

func TestRONEffectBlockingFalseOrders(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			a, b := &ronEffectBlockingPlayer{Player: mage.NewBasePlayer("A"), name: "Second", accept: true}, mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 10), a.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 3, 10), a.PlayerID())
			blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 1, 10), b.PlayerID())
			first.RevokeBaseAttr(core.AttrSummonSick)
			second.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{first.ID(), second.ID()})
			g.SetStep(core.DeclareBlockers)
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: first.ID()}})
			card := repositoryCard(t, set, "False Orders")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{blocker.ID()}, 0); err == nil {
				t.Fatal("unpaid cast succeeded")
			}
			a.ManaPool().Add(core.Red, 1)
			a.name = "Second"
			a.accept = true
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{blocker.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if !g.GetCombat().GroupFor(first.ID()).Unblocked || !g.GetCombat().GroupFor(second.ID()).Blocked || !g.GetCombat().IsBlocking(blocker.ID()) {
				t.Fatal("wrong block reassignment")
			}
			g.ExecuteCombatDamage()
			if b.Life() != 18 || g.FindPermanent(blocker.ID()).Damage != 3 || g.FindPermanent(second.ID()).Damage != 1 {
				t.Fatal("wrong actual combat damage")
			}
		})
	}
}
