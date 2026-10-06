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

func TestRONSourceControlCards(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		tapped    bool
	}{{"ARN", "Aladdin", false}, {"FEM", "Thrull Champion", false}, {"LEG", "Rubinia Soulsinger", true}, {"LEG", "Willow Satyr", true}} {
		for _, change := range []string{"none", "source-return", "source-phase", "source-control", "untap", "target-return", "target-phase", "target-cycle", "shroud", "copy", "pending-clone"} {
			t.Run(tc.name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(b.PlayerID())
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				source := g.PutOnBattlefield(card, a.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				victimCard := mage.NewCreature("Victim", "", 2, 4, mage.WithSubTypes("Thrull"), mage.WithSuperTypes(core.SuperLegendary), mage.WithCardType(core.TypeArtifact))
				victimCard.SetOwner(b.PlayerID())
				victim := g.PutOnBattlefield(victimCard, b.PlayerID())
				if tc.name == "Aladdin" {
					a.ManaPool().Add(core.Red, 2)
					a.ManaPool().Add(core.Colorless, 1)
				}
				index := 1
				if tc.name == "Aladdin" {
					index = 0
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, []uuid.UUID{victim.ID()}); err != nil {
					t.Fatal(err)
				}
				if victim.ControllerID() != b.PlayerID() || !source.Tapped {
					t.Fatal("announcement executed control")
				}
				switch change {
				case "source-return":
					g.RemoveFromBattlefield(source)
					g.PutOnBattlefield(card, a.PlayerID())
				case "source-phase":
					g.PhaseOut(source.ID())
				case "source-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "untap":
					g.UntapPermanent(source)
				case "target-return":
					g.RemoveFromBattlefield(victim)
					victim = g.PutOnBattlefield(victimCard, b.PlayerID())
				case "target-phase":
					g.PhaseOut(victim.ID())
				case "target-cycle":
					p := g.PhaseOut(victim.ID())
					g.PhaseIn(p)
				case "shroud":
					g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
				case "copy":
					g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
				}
				clone := g.Clone()
				clone.ResolveStack()
				if change == "pending-clone" && victim.ControllerID() != b.PlayerID() {
					t.Fatal("clone resolved original")
				}
				g.ResolveStack()
				want := a.PlayerID()
				if change == "source-return" || change == "source-phase" || change == "source-control" || change == "target-return" || change == "target-phase" || change == "shroud" || (change == "untap" && tc.tapped) {
					want = b.PlayerID()
				}
				current := g.FindPermanentIncludingPhased(victim.ID())
				if current.ControllerID() != want || clone.FindPermanentIncludingPhased(victim.ID()).ControllerID() != want {
					t.Fatal("wrong controller")
				}
				if change == "none" {
					if tc.name == "Thrull Champion" && (source.CurrentPower(g) != 3 || current.CurrentPower(g) != 3) {
						t.Fatal("global Thrull bonus")
					}
					if tc.tapped && !source.HasAttr(core.AttrMayNotUntap) {
						t.Fatal("missing optional untap")
					}
					g.UntapPermanent(source)
					g.ApplyContinuousEffects()
					if tc.tapped && current.ControllerID() != b.PlayerID() {
						t.Fatal("untap did not end control")
					}
				}
			})
		}
	}
}

func TestRONSourceControlInvalid(t *testing.T) {
	for _, duration := range []string{`UntilPermanentChangesControl(Opponent,ThisPermanent)`, `UntilPermanentChangesControl(You(),ThisPermanent)`, `UntilPermanentChangesControl(You,HostPermanent)`, `UntilPermanentChangesControl(You,ThisPermanent())`, `UntilPermanentChangesControl(You)`, `UntilPermanentChangesControl`, `Or([])`, `Or([UntilPermanentChangesControl(You,ThisPermanent)])`, `Or([UntilPermanentChangesControl(You,ThisPermanent),ForAsLongAsPermanentRemainsTapped(Ref_TargetPermanent)])`, `Or([UntilPermanentChangesControl(You,ThisPermanent),ForAsLongAsPermanentRemainsTapped(ThisPermanent),UntilEndOfTurn])`, `ForAsLongAsPermanentRemainsTapped(ThisPermanent)`, `UntilEndOfTurn`} {
		src := fmt.Sprintf(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),Targeted([TargetPermanent(AnyPermanent)],ActionList([GainControlOfPermanentUntil(Ref_TargetPermanent,%s)])))])`, duration)
		nodes, err := Parse(strings.NewReader(src), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid duration accepted", duration)
		}
	}
}

func TestRONSourceControlRollbackAndCasting(t *testing.T) {
	for _, tc := range []struct {
		set, name               string
		index, power, toughness int
		mana                    string
	}{{"ARN", "Aladdin", 0, 1, 1, "{2}{R}{R}"}, {"FEM", "Thrull Champion", 1, 3, 3, "{4}{B}"}, {"LEG", "Rubinia Soulsinger", 1, 2, 3, "{2}{G}{W}{U}"}, {"LEG", "Willow Satyr", 1, 1, 1, "{2}{G}{G}"}} {
		t.Run(tc.name+"/casting", func(t *testing.T) {
			g, a, _ := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid creature cast")
			}
			cost := core.ParseManaCost(tc.mana)
			for color, count := range map[core.Color]int{core.White: cost.White, core.Blue: cost.Blue, core.Black: cost.Black, core.Red: cost.Red, core.Green: cost.Green} {
				a.ManaPool().Add(color, count)
			}
			a.ManaPool().Add(core.Colorless, cost.Generic)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil || a.ManaPool().TotalMana() != 0 {
				t.Fatal("casting timing or payment")
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			if source == nil || source.ControllerID() != a.PlayerID() || source.Card.Owner() != a.PlayerID() || source.CurrentPower(g) != tc.power || source.CurrentToughness(g) != tc.toughness || !source.HasAttr(core.AttrSummonSick) {
				t.Fatal("full card metadata or entry")
			}
		})
		for _, reason := range []string{"sick", "tapped", "wrong-controller", "source-phase", "target-phase", "wrong-type", "extra-target", "unpaid"} {
			if reason == "unpaid" && tc.name != "Aladdin" {
				continue
			}
			t.Run(tc.name+"/"+reason, func(t *testing.T) {
				g, a, b := ronGame()
				source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 4, mage.WithSubTypes("Thrull"), mage.WithSuperTypes(core.SuperLegendary), mage.WithCardType(core.TypeArtifact)), b.PlayerID())
				a.ManaPool().Add(core.Red, 2)
				a.ManaPool().Add(core.Colorless, 1)
				controller := a.PlayerID()
				ids := make([]uuid.UUID, 1, 2)
				ids[0] = victim.ID()
				wantTapped := false
				wantMana := 3
				switch reason {
				case "sick":
					source.GrantBaseAttr(core.AttrSummonSick)
				case "tapped":
					source.Tapped = true
					wantTapped = true
				case "wrong-controller":
					controller = b.PlayerID()
				case "source-phase":
					g.PhaseOut(source.ID())
				case "target-phase":
					g.PhaseOut(victim.ID())
				case "wrong-type":
					wrong := mage.NewLand("Wrong type")
					if tc.name != "Rubinia Soulsinger" {
						wrong = mage.NewCreature("Wrong type", "", 1, 1)
					}
					ids[0] = g.PutOnBattlefield(wrong, b.PlayerID()).ID()
				case "extra-target":
					ids = append(ids, source.ID())
				case "unpaid":
					a.ManaPool().Clear()
					wantMana = 0
				}
				if err := g.ActivateAbilityByIndex(controller, source.ID(), tc.index, ids); err == nil {
					t.Fatal("invalid activation succeeded")
				}
				if g.FindPermanentIncludingPhased(source.ID()).Tapped != wantTapped || a.ManaPool().TotalMana() != wantMana || g.StackSize() != 0 || g.FindPermanentIncludingPhased(victim.ID()).ControllerID() != b.PlayerID() {
					t.Fatal("rollback changed resources")
				}
			})
		}
	}
}

func TestRONSourceControlUntapAndInstances(t *testing.T) {
	for _, name := range []string{"Rubinia Soulsinger", "Willow Satyr"} {
		for _, change := range []string{"decline", "accept", "stun", "gone-target", "phased-target"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				a := &ronStoragePlayer{ronCounterPayer: &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A")}}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				first := g.PutOnBattlefield(repositoryCard(t, "LEG", name), a.PlayerID())
				second := g.PutOnBattlefield(repositoryCard(t, "LEG", "Willow Satyr"), a.PlayerID())
				victims := make([]uuid.UUID, 0, 2)
				for index, source := range []*mage.Permanent{first, second} {
					source.RevokeBaseAttr(core.AttrSummonSick)
					victim := g.PutOnBattlefield(mage.NewCreature(fmt.Sprintf("Victim %d", index), "", 2, 2, mage.WithSuperTypes(core.SuperLegendary)), b.PlayerID())
					victims = append(victims, victim.ID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{victim.ID()}); err != nil {
						t.Fatal(err)
					}
				}
				g.ResolveStack()
				clone := g.Clone()
				g.UntapPermanent(g.FindPermanent(second.ID()))
				if g.FindPermanent(victims[0]).ControllerID() != a.PlayerID() || g.FindPermanent(victims[1]).ControllerID() != b.PlayerID() {
					t.Fatal("instances share duration")
				}
				switch change {
				case "accept":
					a.untap = true
				case "stun":
					a.untap = true
					g.MutablePermanent(first.ID()).AddCounter(core.Stun, 1)
				case "gone-target":
					g.RemoveFromBattlefield(g.FindPermanent(victims[0]))
				case "phased-target":
					g.PhaseOut(victims[0])
				}
				g.RunStepWithPriority(core.Untap)
				g.ApplyContinuousEffects()
				wantTapped := change != "accept"
				if g.FindPermanent(first.ID()).Tapped != wantTapped {
					t.Fatal("native untap choice or stun replacement")
				}
				if clone.FindPermanent(victims[1]).ControllerID() != a.PlayerID() || !clone.FindPermanent(second.ID()).Tapped {
					t.Fatal("installed clone changed")
				}
				if change == "phased-target" {
					g.PhaseIn([]uuid.UUID{victims[0]})
					g.ApplyContinuousEffects()
				}
				if victim := g.FindPermanent(victims[0]); victim != nil {
					want := a.PlayerID()
					if !wantTapped {
						want = b.PlayerID()
					}
					if victim.ControllerID() != want {
						t.Fatal("untap or target phasing ended wrong duration")
					}
				}
			})
		}
	}
}

func TestRONSourceControlCopiesAndContexts(t *testing.T) {
	a := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Rubinia Soulsinger"), a.PlayerID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 2), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 2), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{first.ID()}); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	a.targets = []uuid.UUID{second.ID()}
	copyObject := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if copyObject == nil || copyObject.Targets[0] != second.ID() || original.Targets[0] != first.ID() {
		t.Fatal("copy did not retarget independently")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(second.ID()).ControllerID() != a.PlayerID() || g.FindPermanent(first.ID()).ControllerID() != b.PlayerID() {
		t.Fatal("copy used original target")
	}
	g.ResolveStack()
	if g.FindPermanent(first.ID()).ControllerID() != a.PlayerID() {
		t.Fatal("original lost binding")
	}
	g.UntapPermanent(g.FindPermanent(source.ID()))
	if g.FindPermanent(first.ID()).ControllerID() != b.PlayerID() || g.FindPermanent(second.ID()).ControllerID() != b.PlayerID() {
		t.Fatal("one source did not end both durations")
	}

	action := `GainControlOfPermanentUntil(ThisPermanent,UntilPermanentChangesControl(You,ThisPermanent))`
	for _, rules := range []string{
		`Activated(PayMana([]),ActionList([` + action + `]))`,
		`Activated(PayMana([]),Modal_ChooseOne([ActionList([` + action + `]),ActionList([GainLife(Integer(1))])]))`,
		`TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([` + action + `]))`,
	} {
		factory := permanentScopeFactory(t, "Artifact", rules)
		card := factory()
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		if card.Abilities()[0] == factory().Abilities()[0] {
			t.Fatal("factory shares abilities")
		}
	}
	for _, body := range []string{
		`ActionList([GainControlOfPermanentUntil(Ref_TargetPermanent,UntilPermanentChangesControl(You,ThisPermanent))])`,
		`Targeted([TargetPlayer(AnyPlayer)],ActionList([GainControlOfPermanentUntil(Ref_TargetPermanent,UntilPermanentChangesControl(You,ThisPermanent))]))`,
		`ActionList([GainControlOfPermanentUntil(ThisPermanent,UntilPermanentChangesControl(You,ThisPermanent))])`,
	} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(` + body + `)])`
		nodes, err := Parse(strings.NewReader(src), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("unbound or wrong source context accepted")
		}
	}
}

func TestRONSourceControlInventory(t *testing.T) {
	src := `Card(Name:"Nested",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([Unknown,GainControlOfPermanentUntil(ThisPermanent,Or([UntilPermanentChangesControl(You,ThisPermanent),ForAsLongAsPermanentRemainsTapped(ThisPermanent)]))]))])`
	report, err := InspectFS(fstest.MapFS{"nested.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 0 || len(report.Issues) != 1 {
		t.Fatal("inventory admitted unsupported parent")
	}
	roles := map[string]bool{}
	for _, sig := range report.Signatures {
		roles[sig.Role+"/"+sig.Name] = true
	}
	for _, key := range []string{"action/GainControlOfPermanentUntil", "source_control_duration/Or", "source_control_duration/UntilPermanentChangesControl", "source_control_duration/ForAsLongAsPermanentRemainsTapped", "player_reference/You", "permanent_reference/ThisPermanent"} {
		if !roles[key] {
			t.Fatal("missing nested role", key)
		}
	}
}
