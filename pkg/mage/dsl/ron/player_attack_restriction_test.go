package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPlayerAttackRestrictionSanctuary(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, mode := range []int{0, 1} {
			t.Run(set+string(rune('0'+mode)), func(t *testing.T) {
				a := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("A"), choice: mode}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetTurn(3)
				g.SetStep(core.PrecombatMain)
				card := repositoryCard(t, set, "Island Sanctuary")
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
					t.Fatal("unpaid cast")
				}
				a.ManaPool().Add(core.White, 1)
				a.ManaPool().Add(core.Colorless, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				seedRONDrawLibrary(a.BasePlayer)
				seedRONDrawLibrary(b)
				g.PlayerDrawCard(a)
				if len(a.Hand()) != 1 || a.prompts != 0 {
					t.Fatal("outside draw-step replacement")
				}
				g.SetStep(core.Draw)
				g.PlayerDrawCard(b)
				if len(b.Hand()) != 1 || a.prompts != 0 {
					t.Fatal("opponent draw replaced")
				}
				g.PlayerDrawCard(a)
				want := 1
				if mode == 1 {
					want = 2
				}
				if len(a.Hand()) != want || a.prompts != 1 || g.StackSize() != 0 {
					t.Fatal("draw choice or immediate resolution")
				}
				source := g.FindPermanent(card.ID())
				g.RemoveFromBattlefield(source)
				for _, kw := range []core.Attr{0, core.Flying, core.Islandwalk, core.Swampwalk} {
					c := mage.NewCreature("Attacker", "", 2, 3)
					if kw != 0 {
						c = mage.NewCreature("Attacker", "", 2, 3, mage.WithKeyword(kw))
					}
					p := g.PutOnBattlefield(c, b.PlayerID())
					p.RevokeBaseAttr(core.AttrSummonSick)
					allowed := mode == 1 || kw == core.Flying || kw == core.Islandwalk
					if p.CanDeclareAsAttacker(g) != allowed {
						t.Fatalf("keyword %v attack=%v", kw, !allowed)
					}
				}
				clone := g.Clone()
				g.AdvanceToNextTurn()
				g.RunStepWithPriority(core.DeclareAttackers)
				var ids []uuid.UUID
				for _, p := range g.AllBattlefield() {
					if p.ControllerID() == b.PlayerID() {
						ids = append(ids, p.ID())
					}
				}
				g.ExecuteAttackers(b.PlayerID(), ids)
				count := 2
				if mode == 1 {
					count = 4
				}
				actual := 0
				for _, id := range ids {
					if g.GetCombat().IsAttacking(id) {
						actual++
					}
				}
				if actual != count {
					t.Fatalf("declared %d want %d", actual, count)
				}
				g.AdvanceToNextTurn()
				g.BeginActiveTurn()
				for _, id := range ids {
					p := g.MutablePermanent(id)
					p.Tapped = false
					if !p.CanDeclareAsAttacker(g) {
						t.Fatal("protection did not expire")
					}
					if mode == 0 && !p.HasKeyword(core.Flying) && !p.HasKeyword(core.Islandwalk) && clone.FindPermanent(id).CanDeclareAsAttacker(clone) {
						t.Fatal("clone shared expiry")
					}
				}
			})
		}
	}
}

func TestRONPlayerAttackRestrictionInvalid(t *testing.T) {
	valid := `CreatePlayerEffectUntil(You,[CantBeAttackedExceptBy(HasAbility(Flying))],UntilPlayersNextTurn(You))`
	for _, body := range []string{
		`CreatePlayerEffectUntil`, `CreatePlayerEffectUntil()`,
		strings.Replace(valid, "You,", "ThisPermanent,", 1),
		strings.Replace(valid, "You,", "Ref_TargetPlayer,", 1),
		strings.Replace(valid, "UntilPlayersNextTurn(You)", "UntilPlayersNextTurn(Trigger_ThatPlayer)", 1),
		strings.Replace(valid, "UntilPlayersNextTurn(You)", "UntilPlayersNextTurn()", 1),
		strings.Replace(valid, "UntilPlayersNextTurn(You)", "UntilEndOfTurn", 1),
		strings.Replace(valid, "CantBeAttackedExceptBy(HasAbility(Flying))", "CantBeAttackedExceptBy", 1),
		strings.Replace(valid, "CantBeAttackedExceptBy(HasAbility(Flying))", "CantBeAttackedExceptBy()", 1),
		strings.Replace(valid, "HasAbility(Flying)", "Ref_TargetPermanent", 1),
		strings.Replace(valid, "HasAbility(Flying)", "HasAbility(Flying())", 1),
		strings.Replace(valid, "[CantBeAttackedExceptBy(HasAbility(Flying))]", "[]", 1),
		strings.Replace(valid, "[CantBeAttackedExceptBy(HasAbility(Flying))]", "[CantBeAttackedExceptBy(HasAbility(Flying)),CantPlayCardInHand(TheCardReturnedToHandThisWay)]", 1),
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid declaration", body)
		}
	}
}

func TestRONPlayerAttackRestrictionInventory(t *testing.T) {
	source := `Card(Name:"Protection",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([CreatePlayerEffectUntil(You,[CantBeAttackedExceptBy(HasAbility(Flying))],UntilPlayersNextTurn(You))]))])`
	report, err := InspectFS(fstest.MapFS{"test.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, sig := range report.Signatures {
		if sig.Name == "HasAbility" {
			found = true
			if sig.Role != "permanent_filter" {
				t.Fatal("attacker filter role", sig.Role)
			}
		}
	}
	if !found {
		t.Fatal("missing attacker filter signature")
	}
}

func TestRONPlayerAttackRestrictionBindings(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),Targeted([TargetPlayer(AnyPlayer)],ActionList([CreatePlayerEffectUntil(Ref_TargetPlayer,[CantBeAttackedExceptBy(HasAbility(Flying)),CantBeAttackedExceptBy(HasAbility(Landwalk(IsLandType(Island))))],UntilPlayersNextTurn(You))])))`)
	g, a, b := ronGame()
	g.SetTurn(3)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	second := g.PutOnBattlefield(factory(), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{a.PlayerID()}); err == nil {
		t.Fatal("unpaid activation")
	}
	a.ManaPool().Add(core.Colorless, 1)
	b.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), second.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.PhaseOut(source.ID())
	g.RemoveFromBattlefield(g.FindPermanent(second.ID()))
	for _, controller := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
		for _, keywords := range [][]core.Attr{nil, {core.Flying}, {core.Islandwalk}, {core.Flying, core.Islandwalk}} {
			options := make([]mage.CardOption, len(keywords))
			for i, keyword := range keywords {
				options[i] = mage.WithKeyword(keyword)
			}
			p := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 3, options...), controller)
			p.RevokeBaseAttr(core.AttrSummonSick)
			if p.CanDeclareAsAttacker(g) != (len(keywords) == 2) {
				t.Fatal("independent restrictions did not intersect")
			}
		}
	}
	g.AdvanceToNextTurn()
	g.BeginActiveTurn()
	for _, p := range g.AllBattlefield() {
		if !p.HasType(core.TypeCreature) {
			continue
		}
		allowed := p.ControllerID() == a.PlayerID() || p.HasKeyword(core.Flying) && p.HasKeyword(core.Islandwalk)
		if p.CanDeclareAsAttacker(g) != allowed {
			t.Fatal("factory player bindings shared")
		}
	}
	g.AdvanceToNextTurn()
	g.BeginActiveTurn()
	for _, p := range g.AllBattlefield() {
		if p.HasType(core.TypeCreature) && !p.CanDeclareAsAttacker(g) {
			t.Fatal("second binding did not expire")
		}
	}
}

func TestRONPlayerAttackRestrictionLiveFilter(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreatePlayerEffectUntil(ControllerOfPermanent(ThisPermanent),[CantBeAttackedExceptBy(HasAbility(Flying))],UntilPlayersNextTurn(ControllerOfPermanent(ThisPermanent)))]))`)
	g, a, b := ronGame()
	g.SetTurn(3)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.ResolveStack()
	card := mage.NewCreature("Attacker", "", 2, 3)
	p := g.PutOnBattlefield(card, a.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	if p.CanDeclareAsAttacker(g) {
		t.Fatal("resolution controller not saved")
	}
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Flying)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("new flying ignored")
	}
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("removed flying ignored")
	}
	g.PhaseIn(g.PhaseOut(p.ID()))
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	returned := g.PutOnBattlefield(card, a.PlayerID())
	returned.RevokeBaseAttr(core.AttrSummonSick)
	if returned.CanDeclareAsAttacker(g) {
		t.Fatal("returned creature bypassed live rule")
	}
	g.AdvanceToNextTurn()
	g.BeginActiveTurn()
	if !g.FindPermanent(returned.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("saved resolution duration player ignored")
	}
}
