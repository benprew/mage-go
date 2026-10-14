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

type bandedCollectionPlayer struct {
	*mage.BasePlayer
	attackers []uuid.UUID
	band      []uuid.UUID
}

func (p *bandedCollectionPlayer) DeclareAttackers(*mage.Game) []uuid.UUID { return p.attackers }
func (p *bandedCollectionPlayer) GetBandFormations(int, *mage.Game) [][]uuid.UUID {
	return [][]uuid.UUID{p.band}
}

func TestRONBandedCollectionNative(t *testing.T) {
	a := &bandedCollectionPlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	card := repositoryCard(t, "FEM", "Icatian Skirmishers")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast accepted")
	}
	a.ManaPool().Add(core.White, 1)
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	q := g.PutOnBattlefield(mage.NewCreature("Member", "", 2, 8), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 8), a.PlayerID())
	for _, id := range []uuid.UUID{p.ID(), q.ID(), other.ID()} {
		g.MutablePermanent(id).RevokeBaseAttr(core.AttrSummonSick)
	}
	a.attackers = []uuid.UUID{p.ID(), q.ID(), other.ID()}
	a.band = []uuid.UUID{p.ID(), q.ID()}
	g.RunStepWithPriority(core.DeclareAttackers)
	if !g.IsBandedWith(p.ID(), q.ID()) || !g.FindPermanent(q.ID()).HasKeyword(core.FirstStrike) || g.FindPermanent(other.ID()).HasKeyword(core.FirstStrike) {
		t.Fatal("native attack did not resolve the band grant")
	}
	g.RunStepWithPriority(core.DeclareBlockers)
	life := b.Life()
	g.RunStepWithPriority(core.FirstStrikeDamage)
	if b.Life() != life-3 {
		t.Fatalf("first strike damage=%d want=3", life-b.Life())
	}
	g.RunStepWithPriority(core.CombatDamage)
	if b.Life() != life-5 {
		t.Fatal("normal damage repeated first strike or omitted other attacker")
	}
	g.RemoveFromCombat(q.ID())
	if !g.FindPermanent(q.ID()).HasKeyword(core.FirstStrike) {
		t.Fatal("resolved grant followed live band membership")
	}
	g.RunStepWithPriority(core.Cleanup)
	if g.FindPermanent(q.ID()).HasKeyword(core.FirstStrike) || !g.FindPermanent(p.ID()).HasKeyword(core.FirstStrike) {
		t.Fatal("cleanup lost printed first strike or retained temporary grant")
	}
}

func TestRONBandedCollectionPending(t *testing.T) {
	for _, change := range []string{"none", "ability-loss", "source-return", "member-return", "source-phase-cycle", "member-phase", "remove", "copy", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "FEM", "Icatian Skirmishers")
			p := g.PutOnBattlefield(card, a.PlayerID())
			memberCard := mage.NewCreature("Member", "", 2, 8)
			q := g.PutOnBattlefield(memberCard, a.PlayerID())
			for _, id := range []uuid.UUID{p.ID(), q.ID()} {
				g.MutablePermanent(id).RevokeBaseAttr(core.AttrSummonSick)
			}
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID(), q.ID()})
			if g.FindPermanent(q.ID()).HasKeyword(core.FirstStrike) {
				t.Fatal("grant ran at detection")
			}
			if err := g.FormAttackingBand([]uuid.UUID{p.ID(), q.ID()}); err != nil {
				t.Fatal(err)
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatalf("triggers=%d want=1", g.StackSize())
			}
			clone := g.Clone()
			switch change {
			case "ability-loss":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			case "source-return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, a.PlayerID())
			case "member-return":
				g.RemoveFromBattlefield(q)
				g.PutOnBattlefield(memberCard, a.PlayerID())
			case "source-phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "member-phase":
				g.PhaseOut(q.ID())
			case "remove":
				g.RemoveFromCombat(q.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("trigger copy failed")
				}
			case "clone":
				clone.RemoveFromCombat(q.ID())
			}
			g.ResolveStack()
			clone.ResolveStack()
			want := change == "none" || change == "ability-loss" || change == "copy" || change == "clone"
			member := g.FindPermanent(q.ID())
			if (member != nil && member.HasKeyword(core.FirstStrike)) != want {
				t.Fatal("grant did not use current exact band membership")
			}
			if clone.FindPermanent(q.ID()).HasKeyword(core.FirstStrike) != (change != "clone") {
				t.Fatal("clone grant used another game's membership")
			}
		})
	}
}

func TestRONBandedCollectionInvalid(t *testing.T) {
	for _, ref := range []string{"BandedWith", "BandedWith()", "BandedWith(ThisPermanent,ThisPermanent)", "BandedWith(ThisPermanent())", "BandedWith(You)", "BandedWith(Trigger_ThatCreature)", "BandedWith(Ref_TargetPermanent)", "BandedWith(HostPermanent)"} {
		t.Run(ref, func(t *testing.T) {
			_, err := conditionalBodyCompile(t, `Activated(PayMana([]),ActionList([TapEachPermanent(`+ref+`)]))`)
			if err == nil {
				t.Fatal("invalid collection accepted")
			}
		})
	}
}

func TestRONBandedCollectionLoad(t *testing.T) {
	node := repositoryRecord(t, "FEM", "Icatian Skirmishers", "FEM.ron")
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	if factory().Abilities()[0] == factory().Abilities()[0] {
		t.Fatal("factories share abilities")
	}
	data := `Card(Name:"Band",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[Activated(PayMana([]),ActionList([TapEachPermanent(BandedWith(ThisPermanent))]))])`
	files := fstest.MapFS{"band.ron": &fstest.MapFile{Data: []byte(data)}}
	records, err := os.ReadFile("../../../../cards/mtgish/FEM.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	for line := range strings.SplitSeq(string(records), "\n") {
		if strings.Contains(line, `Name: "Icatian Skirmishers"`) {
			files["skirmishers.ron"] = &fstest.MapFile{Data: []byte(line)}
		}
	}
	if files["skirmishers.ron"] == nil {
		t.Fatal("complete repository record missing")
	}
	reg, err := LoadFS(files, "*.ron")
	if err != nil || reg == nil {
		t.Fatalf("strict load: %v", err)
	}
	if !reg.CardRegistered("Icatian Skirmishers") {
		t.Fatal("complete repository record did not load")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(data, `Name:"Band"`, `Name:"Bad"`, 1) + `Unknown`)}
	if err := LoadInto(reg, files, "*.ron"); err == nil {
		t.Fatal("invalid collection published")
	}
	if !reg.CardRegistered("Band") || reg.CardRegistered("Bad") {
		t.Fatal("failed load changed the registry")
	}
}

func TestRONBandedCollectionTargetsAndInstances(t *testing.T) {
	for _, change := range []string{"none", "returned-target", "shroud"} {
		t.Run(change, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([UntapEachPermanent(Or([BandedWith(Ref_TargetPermanent),BandedWith(Ref_TargetPermanent)]))])))`)
			g, a, b := ronGame()
			first := g.PutOnBattlefield(factory(), a.PlayerID())
			second := g.PutOnBattlefield(factory(), b.PlayerID())
			leaders := make([]*mage.Permanent, 0, 2)
			members := make([]*mage.Permanent, 0, 2)
			for range 2 {
				p := g.PutOnBattlefield(mage.NewCreature("Leader", "", 1, 10, mage.WithKeyword(core.Banding)), a.PlayerID())
				q := g.PutOnBattlefield(mage.NewCreature("Member", "", 1, 10), a.PlayerID())
				for _, id := range []uuid.UUID{p.ID(), q.ID()} {
					g.MutablePermanent(id).RevokeBaseAttr(core.AttrSummonSick)
				}
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID(), q.ID()})
				if err := g.FormAttackingBand([]uuid.UUID{p.ID(), q.ID()}); err != nil {
					t.Fatal(err)
				}
				leaders = append(leaders, p)
				members = append(members, q)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, []uuid.UUID{leaders[0].ID()}); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(b.PlayerID(), second.ID(), 0, []uuid.UUID{leaders[1].ID()}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "returned-target":
				g.RemoveFromBattlefield(leaders[0])
				g.PutOnBattlefield(leaders[0].Card, a.PlayerID())
			case "shroud":
				g.MutablePermanent(leaders[0].ID()).GrantBaseAttr(core.Shroud)
			}
			g.ResolveStack()
			if g.FindPermanent(members[0].ID()).Tapped != (change != "none") || g.FindPermanent(members[1].ID()).Tapped {
				t.Fatal("target legality or factory bindings mixed independent bands")
			}
		})
	}
}

func TestRONBandedCollectionCapturedLifetime(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "FEM", "Icatian Skirmishers"), a.PlayerID())
	memberCard := mage.NewCreature("Member", "", 1, 10)
	q := g.PutOnBattlefield(memberCard, a.PlayerID())
	for _, id := range []uuid.UUID{p.ID(), q.ID()} {
		g.MutablePermanent(id).RevokeBaseAttr(core.AttrSummonSick)
	}
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID(), q.ID()})
	if err := g.FormAttackingBand([]uuid.UUID{p.ID(), q.ID()}); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	g.ResolveStack()
	g.RemoveFromBattlefield(p)
	g.PhaseIn(g.PhaseOut(q.ID()))
	if !g.FindPermanent(q.ID()).HasKeyword(core.FirstStrike) {
		t.Fatal("captured grant lost its subject after phasing or source departure")
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: q.ID(), TargetID: q.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.ApplyContinuousEffects()
	if !g.FindPermanent(q.ID()).HasKeyword(core.FirstStrike) {
		t.Fatal("control change ended captured grant")
	}
	g.RemoveFromBattlefield(g.FindPermanent(q.ID()))
	returned := g.PutOnBattlefield(memberCard, a.PlayerID())
	if returned.HasKeyword(core.FirstStrike) {
		t.Fatal("returned subject inherited grant")
	}
}
