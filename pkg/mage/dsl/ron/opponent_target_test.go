package ron

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronTargetChooser struct {
	*mage.BasePlayer
	requested  []uuid.UUID
	candidates [][]uuid.UUID
	queue      [][]uuid.UUID
}

func (p *ronTargetChooser) ChooseTargets(possible []uuid.UUID, _, _ int, _ *mage.Game) []uuid.UUID {
	p.candidates = append(p.candidates, slices.Clone(possible))
	if len(p.queue) > 0 {
		choice := p.queue[0]
		p.queue = p.queue[1:]
		return slices.Clone(choice)
	}
	return slices.Clone(p.requested)
}

func opponentTargetGame() (*mage.Game, *ronTargetChooser, *ronTargetChooser) {
	a := &ronTargetChooser{BasePlayer: mage.NewBasePlayer("A")}
	b := &ronTargetChooser{BasePlayer: mage.NewBasePlayer("B")}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	return g, a, b
}

func opponentTargetBody(controlled, grouped bool) string {
	target := `TargetPermanentOfAPlayersChoice(IsCardtype(Creature),Opponent)`
	if controlled {
		target = `TargetPermanentOfAPlayersChoiceTheyControl(IsCardtype(Creature),Opponent)`
	}
	if grouped {
		return `Targeted([TargetPermanent(And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))])),` + target + `],ActionList([TapPermanent(Ref_TargetPermanent1),UntapPermanent(Ref_TargetPermanent2),GainLife(Integer(1))]))`
	}
	return `Targeted([` + target + `],ActionList([UntapPermanent(Ref_TargetPermanent),GainLife(Integer(1))]))`
}

func TestRONOpponentTargetsGameplay(t *testing.T) {
	for _, controlled := range []bool{false, true} {
		for _, grouped := range []bool{false, true} {
			for _, activation := range []bool{false, true} {
				for _, change := range []string{"none", "control", "return", "phase", "phase-cycle", "shroud", "all-illegal", "source-leaves"} {
					t.Run(fmt.Sprintf("%v/%v/%v/%s", controlled, grouped, activation, change), func(t *testing.T) {
						g, a, b := opponentTargetGame()
						mine := g.PutOnBattlefield(mage.NewCreature("Mine", "", 2, 3), a.PlayerID())
						victimCard := mage.NewCreature("Chosen", "", 2, 3)
						victim := g.PutOnBattlefield(victimCard, b.PlayerID())
						other := g.PutOnBattlefield(mage.NewCreature("Supplied", "", 2, 3), b.PlayerID())
						g.TapPermanent(victim)
						hidden := g.PutOnBattlefield(mage.NewCreature("Phased", "", 2, 3), b.PlayerID())
						g.PhaseOut(hidden.ID())
						protected := g.PutOnBattlefield(mage.NewCreature("Shroud", "", 2, 3), b.PlayerID())
						protected.GrantBaseAttr(core.Shroud)
						land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
						hexproof := g.PutOnBattlefield(mage.NewCreature("Hexproof", "", 2, 3), b.PlayerID())
						hexproof.GrantBaseAttr(core.Hexproof)
						ownHexproof := g.PutOnBattlefield(mage.NewCreature("Own hexproof", "", 2, 3), a.PlayerID())
						ownHexproof.GrantBaseAttr(core.Hexproof)
						b.requested = []uuid.UUID{victim.ID()}
						factory := singularGroupsFactory(t, opponentTargetBody(controlled, grouped), activation)
						card := factory()
						card.SetOwner(a.PlayerID())
						var source *mage.Permanent
						if activation {
							source = g.PutOnBattlefield(card, a.PlayerID())
						} else {
							a.AddToHand(card)
						}
						ids := []uuid.UUID{other.ID()}
						if grouped {
							ids = []uuid.UUID{mine.ID(), other.ID()}
						}
						announce := func() error {
							if activation {
								return g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, ids)
							}
							return g.CastSpellByID(a.PlayerID(), card.ID(), ids, 0)
						}
						if announce() == nil {
							t.Fatal("unpaid action accepted")
						}
						if g.StackSize() != 0 || activation && g.FindPermanent(source.ID()).Tapped {
							t.Fatal("failed payment changed state")
						}
						a.ManaPool().Add(core.Blue, 1)
						b.candidates = nil
						if err := announce(); err != nil {
							t.Fatal(err)
						}
						if len(a.candidates) != 0 || len(b.candidates) != 1 {
							t.Fatal("wrong target chooser")
						}
						possible := b.candidates[0]
						if !slices.Contains(possible, victim.ID()) || !slices.Contains(possible, other.ID()) || slices.Contains(possible, hidden.ID()) || slices.Contains(possible, protected.ID()) || slices.Contains(possible, land.ID()) || slices.Contains(possible, mine.ID()) == controlled {
							t.Fatal("wrong choice candidates", possible)
						}
						if slices.Contains(possible, hexproof.ID()) || slices.Contains(possible, ownHexproof.ID()) == controlled {
							t.Fatal("chooser changed hexproof legality")
						}
						want := []uuid.UUID{victim.ID()}
						if grouped {
							want = []uuid.UUID{mine.ID(), victim.ID()}
						}
						if !slices.Equal(g.StackPeek().Targets, want) {
							t.Fatal("supplied target replaced the opponent choice")
						}
						if !g.FindPermanent(victim.ID()).Tapped || g.FindPermanent(mine.ID()).Tapped || a.Life() != 20 {
							t.Fatal("early action")
						}
						switch change {
						case "control":
							g.AddControlEffect(mage.ControlEffectSpec{SourceID: mine.ID(), TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
						case "return":
							g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
							g.PutOnBattlefield(victimCard, b.PlayerID())
							g.TapPermanent(g.FindPermanent(victim.ID()))
						case "phase":
							g.PhaseOut(victim.ID())
						case "phase-cycle":
							g.PhaseIn(g.PhaseOut(victim.ID()))
						case "shroud":
							g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
						case "all-illegal":
							g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
							g.MutablePermanent(mine.ID()).GrantBaseAttr(core.Shroud)
						case "source-leaves":
							if activation {
								g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
							}
						}
						clone := g.Clone()
						g.ResolveStack()
						if !clone.FindPermanentIncludingPhased(victim.ID()).Tapped || clone.GetPlayer(a.PlayerID()).Life() != 20 {
							t.Fatal("clone changed before resolution")
						}
						clone.ResolveStack()
						legal := change != "return" && change != "phase" && change != "shroud" && change != "all-illegal" && (!controlled || change != "control")
						for _, game := range []*mage.Game{g, clone} {
							if game.FindPermanentIncludingPhased(victim.ID()).Tapped == legal {
								t.Fatal("chosen target legality or identity")
							}
							firstLegal := grouped && change != "all-illegal"
							if game.FindPermanent(mine.ID()).Tapped != firstLegal {
								t.Fatal("ordinary target group changed")
							}
							life := 20
							if legal || firstLegal {
								life++
							}
							if game.GetPlayer(a.PlayerID()).Life() != life {
								t.Fatal("partial or all-illegal resolution")
							}
						}
						if len(b.candidates) != 1 {
							t.Fatal("choice repeated at resolution")
						}
					})
				}
			}
		}
	}
}

func TestRONOpponentTargetsInvalid(t *testing.T) {
	for _, target := range []string{
		`TargetPermanentOfAPlayersChoice(AnyPermanent)`, `TargetPermanentOfAPlayersChoice(AnyPermanent,You)`,
		`TargetPermanentOfAPlayersChoice(AnyPermanent,Opponent())`, `TargetPermanentOfAPlayersChoice(AnyPermanent,Ref_TargetPlayer)`,
		`TargetPermanentOfAPlayersChoiceTheyControl(AnyPermanent)`, `TargetPermanentOfAPlayersChoiceTheyControl(AnyPermanent,SinglePlayer(Opponent))`,
		`TargetPermanentOfAPlayersChoiceTheyControl(AnyPermanent,Opponent,Opponent)`,
		`TargetPermanentOfAPlayersChoiceTheyControl(ControlledByPlayer(You),Opponent)`,
		`TargetPermanentOfAPlayersChoiceTheyControl(ControlledByAPlayer(SinglePlayer(You)),Opponent)`,
		`TargetPermanentOfAPlayersChoiceTheyControl(And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))]),Opponent)`,
		`TargetPermanentOfPlayersChoice(AnyPermanent,Opponent)`,
	} {
		for _, grouped := range []bool{false, true} {
			targets := target
			ref := "Ref_TargetPermanent"
			if grouped {
				targets = `TargetPermanent(AnyPermanent),` + target
				ref = "Ref_TargetPermanent2"
			}
			src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([` + targets + `],ActionList([TapPermanent(` + ref + `)])))])`
			nodes, err := Parse(strings.NewReader(src), "bad-choice.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Compile(nodes[0]); err == nil {
				t.Fatal("accepted invalid chooser", target)
			}
		}
	}
}

func TestRONOpponentTargetsInventoryAndStrictLoad(t *testing.T) {
	src := `Card(Name:"Choice",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(` + opponentTargetBody(true, true) + `)])`
	files := fstest.MapFS{"choice.ron": {Data: []byte(src)}}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(src, "Opponent)", "You)"))}
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("invalid collection installed")
	}
	if len(registry.RegisteredCardNames()) != 1 || !registry.CardRegistered("Choice") {
		t.Fatal("failed load changed registry")
	}
	nested := strings.ReplaceAll(src, "UntapPermanent(Ref_TargetPermanent2)", "UnknownAction")
	report, err := InspectFS(fstest.MapFS{"nested.ron": {Data: []byte(nested)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]bool{}
	for _, sig := range report.Signatures {
		roles[sig.Role+"/"+sig.Name] = true
	}
	for _, key := range []string{"target/TargetPermanentOfAPlayersChoiceTheyControl", "permanent_filter/IsCardtype", "player_reference/Opponent"} {
		if !roles[key] {
			t.Fatal("missing nested role", key)
		}
	}
	if report.Compiled != 0 {
		t.Fatal("inventory accepted unsupported action")
	}
}

func TestRONOpponentTargetsCopyAndInstances(t *testing.T) {
	for _, activation := range []bool{false, true} {
		t.Run(fmt.Sprint(activation), func(t *testing.T) {
			g, a, b := opponentTargetGame()
			first := g.PutOnBattlefield(mage.NewCreature("Mine", "", 2, 3), a.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Theirs", "", 2, 3), b.PlayerID())
			replacement1 := g.PutOnBattlefield(mage.NewCreature("Copy mine", "", 2, 3), b.PlayerID())
			replacement2 := g.PutOnBattlefield(mage.NewCreature("Copy theirs", "", 2, 3), a.PlayerID())
			g.TapPermanent(second)
			g.TapPermanent(replacement2)
			factory := singularGroupsFactory(t, opponentTargetBody(true, true), activation)
			for range 2 {
				card := factory()
				card.SetOwner(a.PlayerID())
				a.ManaPool().Add(core.Blue, 1)
				b.requested = []uuid.UUID{second.ID()}
				if activation {
					source := g.PutOnBattlefield(card, a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{first.ID(), second.ID()}); err != nil {
						t.Fatal(err)
					}
				} else {
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID(), second.ID()}, 0); err != nil {
						t.Fatal(err)
					}
				}
			}
			original := g.StackPeek()
			b.queue = [][]uuid.UUID{{replacement1.ID()}, {replacement2.ID()}}
			a.requested = []uuid.UUID{replacement2.ID()}
			b.candidates = nil
			a.candidates = nil
			copied := g.CopyStackObjectDirect(original, b.PlayerID(), true)
			if copied == nil || !slices.Equal(copied.Targets, []uuid.UUID{replacement1.ID(), replacement2.ID()}) || !slices.Equal(original.Targets, []uuid.UUID{first.ID(), second.ID()}) {
				t.Fatal("copy bindings or controller restrictions")
			}
			if len(a.candidates) != 0 || len(b.candidates) != 2 {
				t.Fatal("copy chooser routing")
			}
			if !slices.Contains(b.candidates[1], replacement2.ID()) || slices.Contains(b.candidates[1], replacement1.ID()) {
				t.Fatal("copy target uses original controller")
			}
			clone := g.Clone()
			g.ResolveTopOfStack()
			if !g.FindPermanent(replacement1.ID()).Tapped || g.FindPermanent(replacement2.ID()).Tapped || g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped {
				t.Fatal("copy resolved wrong groups")
			}
			if clone.FindPermanent(replacement1.ID()).Tapped || !clone.FindPermanent(replacement2.ID()).Tapped {
				t.Fatal("copy resolution changed clone")
			}
			g.ResolveStack()
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if !game.FindPermanent(first.ID()).Tapped || game.FindPermanent(second.ID()).Tapped || game.GetPlayer(a.PlayerID()).Life() != 22 || game.GetPlayer(b.PlayerID()).Life() != 21 {
					t.Fatal("factory or copy state shared")
				}
			}
		})
	}
}

func TestRONOpponentTargetsTriggerAndModes(t *testing.T) {
	body := opponentTargetBody(true, false)
	for _, context := range []string{"trigger", "spell-mode", "activation-mode"} {
		t.Run(context, func(t *testing.T) {
			g, a, b := opponentTargetGame()
			victim := g.PutOnBattlefield(mage.NewCreature("Chosen", "", 2, 3), b.PlayerID())
			g.TapPermanent(victim)
			b.requested = []uuid.UUID{victim.ID()}
			kind, rule := "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),`+body+`)`
			if context == "spell-mode" {
				kind = "Instant"
				rule = `SpellActions(Modal_ChooseOne([` + body + `,ActionList([GainLife(Integer(4))])]))`
			}
			if context == "activation-mode" {
				rule = `Activated(PayMana([]),Modal_ChooseOne([` + body + `,ActionList([GainLife(Integer(4))])]))`
			}
			factory := permanentScopeFactory(t, kind, rule)
			card := factory()
			card.SetOwner(a.PlayerID())
			if context == "spell-mode" {
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
			} else {
				source := g.PutOnBattlefield(card, a.PlayerID())
				if context == "activation-mode" {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
				} else {
					g.PutTriggersOnStack()
				}
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			}
			if len(b.candidates) != 1 || g.StackSize() != 1 || !g.FindPermanent(victim.ID()).Tapped {
				t.Fatalf("target placement or chooser routing: prompts=%v stack=%d tapped=%v", b.candidates, g.StackSize(), g.FindPermanent(victim.ID()).Tapped)
			}
			g.ResolveStack()
			if g.FindPermanent(victim.ID()).Tapped || a.Life() != 21 {
				t.Fatal("mode or trigger target binding")
			}
		})
	}
}

func TestRONOpponentTargetsRecordFragments(t *testing.T) {
	for _, tc := range []struct{ set, name, blocker string }{
		{"DRK", "Preacher", ""},
		{"PROMOS", "Arena", ""},
		{"LEG", "Nova Pentacle", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := chosenDiscardRecord(t, tc.set, tc.name)
			if _, err := Compile(record); (tc.blocker == "" && err != nil) || (tc.blocker != "" && (err == nil || !strings.Contains(err.Error(), tc.blocker))) {
				t.Fatalf("next full-record blocker: %v", err)
			}
			rules := record.Fields["Rules"]
			selected := -1
			for i, rule := range rules.Args {
				if rule.Name == "Activated" {
					selected = i
					break
				}
			}
			if selected < 0 {
				t.Fatal("no activation")
			}
			rule := rules.Args[selected]
			targetList := rule.Args[1].Args[0]
			action := `ActionList([TapPermanent(Ref_TargetPermanent)])`
			if tc.name == "Arena" {
				action = `ActionList([TapPermanent(Ref_TargetPermanent1),TapPermanent(Ref_TargetPermanent2)])`
			}
			nodes, err := Parse(strings.NewReader(action), "fragment-action.ron")
			if err != nil {
				t.Fatal(err)
			}
			targeted := rule.Args[1]
			targeted.Args = slices.Clone(targeted.Args)
			targeted.Args[1] = nodes[0]
			rule.Args = slices.Clone(rule.Args)
			rule.Args[1] = targeted
			rules.Args = []Node{rule}
			record.Fields["Rules"] = rules
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			if len(targetList.Args) == 0 {
				t.Fatal("missing source target declarations")
			}
			g, a, b := opponentTargetGame()
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			mine := g.PutOnBattlefield(mage.NewCreature("Mine", "", 2, 3), a.PlayerID())
			victim := g.PutOnBattlefield(mage.NewCreature("Chosen", "", 2, 3), b.PlayerID())
			b.requested = []uuid.UUID{victim.ID()}
			a.ManaPool().Add(core.Colorless, 3)
			ids := []uuid.UUID{mine.ID()}
			if tc.name == "Arena" {
				ids = append(ids, mine.ID())
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, ids); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(source.ID()).Tapped || g.FindPermanent(victim.ID()).Tapped || len(b.candidates) != 1 {
				t.Fatal("source cost or target timing")
			}
			if slices.Contains(b.candidates[0], mine.ID()) != (tc.name == "Nova Pentacle") {
				t.Fatal("source choice restriction")
			}
			g.ResolveStack()
			if !g.FindPermanent(victim.ID()).Tapped || g.FindPermanent(mine.ID()).Tapped != (tc.name == "Arena") {
				t.Fatal("source target group identity")
			}
		})
	}
}

func TestRONOpponentTargetsNoCandidatesAndMalformedChoice(t *testing.T) {
	for _, response := range []string{"missing", "illegal", "duplicate"} {
		g, a, b := opponentTargetGame()
		mine := g.PutOnBattlefield(mage.NewCreature("Mine", "", 2, 3), a.PlayerID())
		card := singularGroupsFactory(t, opponentTargetBody(true, false), true)()
		source := g.PutOnBattlefield(card, a.PlayerID())
		a.ManaPool().Add(core.Blue, 1)
		var victim *mage.Permanent
		if response != "missing" {
			victim = g.PutOnBattlefield(mage.NewCreature("Only legal", "", 2, 3), b.PlayerID())
			g.TapPermanent(victim)
		}
		b.requested = []uuid.UUID{mine.ID()}
		if response == "duplicate" {
			b.requested = []uuid.UUID{victim.ID(), victim.ID()}
		}
		err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{mine.ID()})
		if response == "missing" {
			if err == nil || g.StackSize() != 0 || g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 1 || len(b.candidates) != 0 {
				t.Fatal("empty candidate rollback")
			}
		} else {
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(g.StackPeek().Targets, []uuid.UUID{victim.ID()}) {
				t.Fatal("malformed choice escaped legality")
			}
			g.ResolveStack()
			if g.FindPermanent(victim.ID()).Tapped {
				t.Fatal("fallback target did not resolve")
			}
		}
	}
}

func TestRONOpponentTargetsIndependentGroups(t *testing.T) {
	for _, same := range []bool{false, true} {
		g, a, b := opponentTargetGame()
		first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 3), b.PlayerID())
		second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 3), b.PlayerID())
		if same {
			second = first
		}
		g.TapPermanent(second)
		b.queue = [][]uuid.UUID{{first.ID()}, {second.ID()}}
		body := `Targeted([TargetPermanentOfAPlayersChoiceTheyControl(And([IsCardtype(Creature),ControlledByAPlayer(Opponent)]),Opponent),TargetPermanentOfAPlayersChoice(IsCardtype(Creature),Opponent)],ActionList([TapPermanent(Ref_TargetPermanent1),UntapPermanent(Ref_TargetPermanent2)]))`
		card := singularGroupsFactory(t, body, false)()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.Blue, 1)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{second.ID(), first.ID()}, 0); err != nil {
			t.Fatal(err)
		}
		if len(b.candidates) != 2 || len(a.candidates) != 0 || !slices.Equal(g.StackPeek().Targets, []uuid.UUID{first.ID(), second.ID()}) {
			t.Fatal("choice groups shared state")
		}
		g.ResolveStack()
		if g.FindPermanent(second.ID()).Tapped || g.FindPermanent(first.ID()).Tapped == same {
			t.Fatal("independent word targets or action order")
		}
	}
}
