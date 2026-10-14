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

func TestRONLiveStaticCollectionCamel(t *testing.T) {
	a := &bandedCollectionPlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	card := repositoryCard(t, "ARN", "Camel")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0) == nil {
		t.Fatal("unpaid cast accepted")
	}
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	q := g.PutOnBattlefield(mage.NewCreature("Member", "", 2, 10), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 10), a.PlayerID())
	desert := g.PutOnBattlefield(mage.NewLand("Desert", mage.WithSubTypes("Desert")), b.PlayerID())
	g.DealDamageToPermanent(q, 1, desert.ID())
	if g.FindPermanent(q.ID()).Damage != 1 {
		t.Fatal("inactive rule prevented damage")
	}
	for _, id := range []uuid.UUID{p.ID(), q.ID(), other.ID()} {
		g.MutablePermanent(id).RevokeBaseAttr(core.AttrSummonSick)
	}
	a.attackers = []uuid.UUID{p.ID(), q.ID(), other.ID()}
	a.band = []uuid.UUID{p.ID(), q.ID()}
	g.RunStepWithPriority(core.DeclareAttackers)
	g.DealSimultaneousDamage(mage.PermanentDamageAssignment(g.FindPermanent(p.ID()), 1, g.CaptureDamageSource(desert.ID())), mage.PermanentDamageAssignment(g.FindPermanent(q.ID()), 1, g.CaptureDamageSource(desert.ID())), mage.PermanentDamageAssignment(g.FindPermanent(other.ID()), 1, g.CaptureDamageSource(desert.ID())))
	if g.FindPermanent(p.ID()) == nil || g.FindPermanent(p.ID()).Damage != 0 || g.FindPermanent(q.ID()).Damage != 1 || g.FindPermanent(other.ID()).Damage != 1 {
		t.Fatal("Camel or band prevention")
	}
	clone := g.Clone()
	clone.RemoveFromCombat(q.ID())
	clone.DealDamageToPermanent(clone.FindPermanent(q.ID()), 1, desert.ID())
	if clone.FindPermanent(q.ID()).Damage != 2 || g.FindPermanent(q.ID()).Damage != 1 {
		t.Fatal("live clone membership")
	}
	g.RunStepWithPriority(core.DeclareBlockers)
	life := b.Life()
	g.RunStepWithPriority(core.CombatDamage)
	if b.Life() != life-4 {
		t.Fatal("native band combat")
	}
	g.RemoveFromCombat(q.ID())
	g.DealDamageToPermanent(g.FindPermanent(q.ID()), 1, desert.ID())
	if g.FindPermanent(q.ID()).Damage != 2 {
		t.Fatal("removed band member remained protected")
	}
}

func TestRONLiveStaticCollectionCondition(t *testing.T) {
	f := permanentScopeFactory(t, "Artifact", `If(PlayerPassesFilter(You,ControlsA(And([Other(ThisPermanent),IsCardtype(Creature)]))),[PermanentLayerEffect(ThisPermanent,[AddAbility([Flying])])])`)
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(f(), a.PlayerID())
	if p.HasKeyword(core.Flying) {
		t.Fatal("empty collection matched")
	}
	q := g.PutOnBattlefield(mage.NewCreature("Member", "", 1, 10), a.PlayerID())
	g.ApplyContinuousEffects()
	if !g.FindPermanent(p.ID()).HasKeyword(core.Flying) {
		t.Fatal("collection condition missing")
	}
	g.PhaseOut(q.ID())
	g.ApplyContinuousEffects()
	if g.FindPermanent(p.ID()).HasKeyword(core.Flying) {
		t.Fatal("phased member matched")
	}
}

func TestRONLiveStaticCollectionInvalid(t *testing.T) {
	node := repositoryRecord(t, "ARN", "Camel", "camel.ron")
	for _, bad := range []string{"BandedWith()", "BandedWith(HostPermanent)", "BandedWith(ThisPermanent())", "Or([])", "Other(Trigger_ThatCreature)", "ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer))"} {
		text := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:10),Rules:[If(PermanentPassesFilter(ThisPermanent,IsAttacking),[PreventDamage(DamageWouldBeDealtByAPermanentToARecipient(IsLandType(Desert),APermanent(` + bad + `)),[PreventThatDamage])])])`
		nodes, err := Parse(strings.NewReader(text), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	if _, err := Compile(node); err != nil {
		t.Fatal(err)
	}
}

func TestRONLiveStaticCollectionCamelChanges(t *testing.T) {
	for _, change := range []string{"ability-loss", "member-return", "source-return", "member-phase", "source-phase-cycle", "source-lki", "non-desert", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "ARN", "Camel")
			p := g.PutOnBattlefield(card, a.PlayerID())
			member := mage.NewCreature("Member", "", 2, 10)
			q := g.PutOnBattlefield(member, a.PlayerID())
			desert := g.PutOnBattlefield(mage.NewLand("Desert", mage.WithSubTypes("Desert")), b.PlayerID())
			saved := g.CaptureDamageSource(desert.ID())
			for _, id := range []uuid.UUID{p.ID(), q.ID()} {
				g.MutablePermanent(id).RevokeBaseAttr(core.AttrSummonSick)
			}
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID(), q.ID()})
			if err := g.FormAttackingBand([]uuid.UUID{p.ID(), q.ID()}); err != nil {
				t.Fatal(err)
			}
			want := 0
			switch change {
			case "ability-loss":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				want = 1
			case "member-return":
				g.RemoveFromBattlefield(q)
				q = g.PutOnBattlefield(member, a.PlayerID())
				want = 1
			case "source-return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, a.PlayerID())
				want = 1
			case "member-phase":
				g.PhaseOut(q.ID())
				g.PhaseIn([]uuid.UUID{q.ID()})
				want = 1
			case "source-phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
				want = 1
			case "source-lki":
				g.RemoveFromBattlefield(desert)
				g.PutOnBattlefield(desert.Card, b.PlayerID())
			case "non-desert":
				source := g.PutOnBattlefield(mage.NewLand("Other land"), b.PlayerID())
				saved = g.CaptureDamageSource(source.ID())
				want = 1
			case "clone":
				clone := g.Clone()
				clone.RemoveFromCombat(p.ID())
				clone.DealDamageToPermanentFromSource(clone.FindPermanent(q.ID()), 1, saved)
				if clone.FindPermanent(q.ID()).Damage != 1 || g.FindPermanent(q.ID()).Damage != 0 {
					t.Fatal("clone protection shared")
				}
			}
			g.DealDamageToPermanentFromSource(g.FindPermanent(q.ID()), 1, saved)
			if g.FindPermanent(q.ID()).Damage != want {
				t.Fatalf("damage=%d want=%d", g.FindPermanent(q.ID()).Damage, want)
			}
		})
	}
}

func TestRONLiveStaticCollectionLoadAndScope(t *testing.T) {
	node := repositoryRecord(t, "ARN", "Camel", "camel.ron")
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	if factory().Abilities()[0] == factory().Abilities()[0] {
		t.Fatal("shared factory abilities")
	}
	data, err := os.ReadFile("../../../../cards/mtgish/ARN.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Camel"`) {
			record = line
		}
	}
	if record == "" {
		t.Fatal("missing Camel record")
	}
	files := fstest.MapFS{"camel.ron": {Data: []byte(record)}}
	reg, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	count := len(reg.RegisteredCardNames())
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[PreventDamage(DamageWouldBeDealtByAPermanentToARecipient(IsCardtype(Land),APermanent(SinglePermanent(ThisPermanent))),[PreventThatDamage])])`)}
	if err := LoadInto(reg, files, "*.ron"); err == nil || len(reg.RegisteredCardNames()) != count {
		t.Fatal("invalid collection publication")
	}
	for _, bad := range []string{`PreventDamage(DamageWouldBeDealtByAPermanentToARecipient(IsCardtype(Land),APermanent(Or([]))),[PreventThatDamage])`, `PreventDamage(DamageWouldBeDealtByAPermanentToARecipient(IsCardtype(Land),APermanent(SinglePermanent(ThisPermanent))),[PreventThatDamage,Integer(1)])`, `Activated(PayMana([]),ActionList([TapEachPermanent(Other(ThisPermanent))]))`} {
		if _, err := conditionalBodyCompile(t, bad); err == nil {
			t.Fatal("invalid context accepted")
		}
	}
	bronze := repositoryRecord(t, "LEG", "Bronze Horse", "bronze.ron")
	if _, err := Compile(bronze); err != nil {
		t.Fatalf("supported Bronze Horse rejected: %v", err)
	}
	silhouette := repositoryRecord(t, "LEG", "Silhouette", "silhouette.ron")
	if _, err := Compile(silhouette); err == nil || !strings.Contains(err.Error(), "ASpellOrAbilityWouldCauseASourceToDealDamageToRecipient") {
		t.Fatalf("wrong remaining Silhouette blocker: %v", err)
	}
}

func TestRONLiveStaticCollectionCamelSpellCopy(t *testing.T) {
	g, a, b := ronGame()
	g.SetStep(core.PrecombatMain)
	card := repositoryCard(t, "ARN", "Camel")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	camels := g.FilterBattlefield(mage.Named("Camel"))
	if len(camels) != 2 {
		t.Fatal("missing copied Camel")
	}
	p, q := camels[0], camels[1]
	for _, id := range []uuid.UUID{p.ID(), q.ID()} {
		g.MutablePermanent(id).RevokeBaseAttr(core.AttrSummonSick)
	}
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
	desert := g.PutOnBattlefield(mage.NewLand("Desert", mage.WithSubTypes("Desert")), b.PlayerID())
	g.DealSimultaneousDamage(mage.PermanentDamageAssignment(g.FindPermanent(p.ID()), 1, g.CaptureDamageSource(desert.ID())), mage.PermanentDamageAssignment(g.FindPermanent(q.ID()), 1, g.CaptureDamageSource(desert.ID())))
	g.CheckStateBasedActions()
	if g.FindPermanent(p.ID()) == nil || g.FindPermanent(p.ID()).Damage != 0 || g.FindPermanent(q.ID()) != nil {
		t.Fatal("copied instance used another source condition")
	}
}

func TestRONLiveStaticCollectionNestedControl(t *testing.T) {
	for _, selector := range []string{`ControlledByAPlayer(SinglePlayer(You))`, `Or([And([Other(ThisPermanent),IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))]),SinglePermanent(ThisPermanent)])`} {
		f := permanentScopeFactory(t, "Artifact", `If(PlayerPassesFilter(You,ControlsA(`+selector+`)),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,1)])])`)
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(f(), a.PlayerID())
		if p.CurrentPower(g) != 1 || p.CurrentToughness(g) != 1 {
			t.Fatal("live source/control collection did not match")
		}
	}
	f := permanentScopeFactory(t, "Artifact", `If(PlayerPassesFilter(You,ControlsNo(And([Other(ThisPermanent),IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))]))),[PermanentLayerEffect(ThisPermanent,[AddAbility([Flying])])])`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(f(), a.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Member", "", 1, 10), b.PlayerID())
	if !g.FindPermanent(p.ID()).HasKeyword(core.Flying) {
		t.Fatal("opponent supplied membership")
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: q.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	if g.FindPermanent(p.ID()).HasKeyword(core.Flying) {
		t.Fatal("live control acquisition absent")
	}
}

func TestRONLiveStaticCollectionUnconditional(t *testing.T) {
	f := permanentScopeFactory(t, "Artifact", `PreventDamage(DamageWouldBeDealtByAPermanentToARecipient(IsCardtype(Land),APermanent(And([IsCardtype(Creature),Other(ThisPermanent),ControlledByAPlayer(SinglePlayer(You))]))),[PreventThatDamage])`)
	g, a, b := ronGame()
	g.PutOnBattlefield(f(), a.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Member", "", 1, 10), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 10), b.PlayerID())
	source := g.PutOnBattlefield(mage.NewLand("Source"), b.PlayerID())
	g.DealSimultaneousDamage(mage.PermanentDamageAssignment(q, 1, g.CaptureDamageSource(source.ID())), mage.PermanentDamageAssignment(other, 1, g.CaptureDamageSource(source.ID())))
	if g.FindPermanent(q.ID()).Damage != 0 || g.FindPermanent(other.ID()).Damage != 1 {
		t.Fatal("unconditional live recipients")
	}
}

func TestRONLiveStaticCollectionInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/ARN.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Camel"`) {
			record = line
		}
	}
	report, err := InspectFS(fstest.MapFS{"camel.ron": {Data: []byte(record)}}, "*.ron")
	if err != nil || len(report.Issues) != 0 {
		t.Fatalf("inventory: %v %v", err, report)
	}
	for name, role := range map[string]string{"DamageWouldBeDealtByAPermanentToARecipient": "prevention_event", "APermanent": "prevention_recipient", "BandedWith": "permanent_collection", "ThisPermanent": "permanent_reference"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s role %s", name, role)
		}
	}
}
