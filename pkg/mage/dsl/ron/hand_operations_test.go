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

func addRONHandCard(p *mage.BasePlayer, c mage.Card) { c.SetOwner(p.PlayerID()); p.AddToHand(c) }

func TestRONHandOperationsCards(t *testing.T) {
	for _, name := range []string{"Amnesia", "Inquisition"} {
		for _, self := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/self=%t", name, self), func(t *testing.T) {
				g, a, b := ronGame()
				target := b
				if self {
					target = a
				}
				spell := repositoryCard(t, "DRK", name)
				if err := mage.ValidateCardDefinition(name, spell); err != nil {
					t.Fatal(err)
				}
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				white := mage.NewCreature("White", "{W}", 1, 1)
				blue := mage.NewCreature("Blue", "{U}", 1, 1)
				land := mage.NewLand("Land")
				for _, c := range []mage.Card{white, blue, land} {
					addRONHandCard(target, c)
				}
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.PlayerID()}, 0); err == nil {
					t.Fatal("unpaid cast")
				}
				if len(g.PublicReveals()) != 0 || g.StackSize() != 0 || len(target.Hand()) != 3+boolInt(self) {
					t.Fatal("rollback")
				}
				a.ManaPool().Add(core.Colorless, 3)
				if name == "Amnesia" {
					a.ManaPool().Add(core.Blue, 3)
				} else {
					a.ManaPool().Add(core.Black, 1)
				}
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{uuid.New()}, 0); err == nil {
					t.Fatal("illegal target")
				}
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.PlayerID()}, 0); err != nil {
					t.Fatal(err)
				}
				late := mage.NewCreature("Late white", "{W}{U}", 1, 1)
				addRONHandCard(target, late)
				if len(g.PublicReveals()) != 0 || len(target.Hand()) != 4 {
					t.Fatal("acted before resolution")
				}

				clone := g.Clone()
				clone.ResolveStack()
				if len(target.Hand()) != 4 || len(g.PublicReveals()) != 0 {
					t.Fatal("clone changed original")
				}
				events := 0
				watcher := mage.NewTriggered(core.EvtDiscard, false).SetCondition(func(_ *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
					events++
					if len(g.PublicReveals()) != 4 || len(target.Hand()) != 1 {
						t.Error("reveal order or partial discard")
					}
					return false
				})
				g.PutOnBattlefield(mage.NewArtifact("Observer", "", mage.WithAbility(watcher)), a.PlayerID())
				g.ResolveStack()
				if len(g.PublicReveals()) != 4 || g.PublicReveals()[0].CardID != white.ID() {
					t.Fatal("wrong revealed hand")
				}
				for _, game := range []*mage.Game{g, clone} {
					player := game.GetPlayer(target.PlayerID())
					if name == "Amnesia" {
						if len(player.Hand()) != 1 || player.Hand()[0].ID() != land.ID() || player.Life() != 20 {
							t.Fatal("nonland discard")
						}
					} else if len(player.Hand()) != 4 || player.Life() != 18 {
						t.Fatal("current white count or damage")
					}
					if len(game.GetPlayer(a.PlayerID()).Graveyard()) != 1+boolInt(name == "Amnesia" && self)*3 {
						t.Fatal("spell disposal")
					}
				}
				if name == "Amnesia" && events != 3 {
					t.Fatal("discard events")
				}
			})
		}
	}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func TestRONHandOperationsCopiesAndPrevention(t *testing.T) {
	for _, name := range []string{"Amnesia", "Inquisition"} {
		g, a, b := ronGame()
		for _, p := range []*mage.BasePlayer{a, b} {
			addRONHandCard(p, mage.NewCreature("White", "{W}", 1, 1))
			addRONHandCard(p, mage.NewLand("Land"))
		}
		spell := repositoryCard(t, "DRK", name)
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		a.ManaPool().Add(core.Blue, 6)
		a.ManaPool().Add(core.Black, 6)
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
			t.Fatal(err)
		}
		cp := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
		if cp == nil {
			t.Fatal("copy")
		}
		g.AddPreventionShield(b.PlayerID(), 1)
		g.ResolveTopOfStack()
		if len(g.PublicReveals()) != 2 || len(a.Hand()) != 2 || len(a.Graveyard()) != 0 {
			t.Fatal("copy binding or spell disposal")
		}
		if b.Life() != 20 {
			t.Fatal("damage ignored prevention")
		}
		g.ResolveStack()
		if name == "Amnesia" && (len(b.Hand()) != 1 || len(g.PublicReveals()) != 3) {
			t.Fatal("original hand at resolution")
		}
		if name == "Inquisition" && (b.Life() != 19 || len(g.PublicReveals()) != 4) {
			t.Fatal("repeat damage or reveal")
		}
	}
}

func TestRONHandOperationsInvalid(t *testing.T) {
	for _, action := range []string{
		`PlayerAction(Ref_TargetPlayer,RevealHand())`, `PlayerAction(Ref_TargetPlayer,RevealHand(Integer(1)))`,
		`PlayerAction(Ref_TargetPlayer,DiscardCards)`, `PlayerAction(Ref_TargetPlayer,DiscardCards(IsNonCardtype))`,
		`PlayerAction(Ref_TargetPlayer,DiscardCards(IsNonCardtype(Unknown)))`, `PlayerAction(Ref_TargetPlayer,DiscardCards(IsColor(White())))`,
		`PlayerAction(ActionPlayer,RevealHand)`, `PlayerAction(Ref_TargetPermanent,RevealHand)`,
		`SpellDealsDamage(ThisSpell,TheNumberOfCardsOfTypeInPlayersHand(IsColor(White),You,You),Player(You))`,
		`SpellDealsDamage(ThisSpell,TheNumberOfCardsOfTypeInPlayersHand(IsColor(White),ActionPlayer),Player(You))`,
		`PlayerAction(Ref_TargetPlayer,DiscardCards(And([])))`, `PlayerAction(Ref_TargetPlayer,DiscardCards(IsTapped))`,
	} {
		source := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([` + action + `])))])`
		nodes, err := Parse(strings.NewReader(source), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", action)
		}
	}
}

func TestRONHandOperationsInventory(t *testing.T) {
	text := ""
	for line := range strings.SplitSeq(mustRONSet(t, "DRK"), "\n") {
		if strings.Contains(line, `Name: "Amnesia"`) || strings.Contains(line, `Name: "Inquisition"`) {
			text += line + "\n"
		}
	}
	report, err := InspectFS(fstest.MapFS{"hand.ron": {Data: []byte(text)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 2 {
		t.Fatal("real records not compiled")
	}
	for _, sig := range report.Signatures {
		if sig.Role == "unknown" {
			t.Fatal("unknown role", sig.Name)
		}
	}
}

func TestRONHandOperationsEmptyAndReplacements(t *testing.T) {
	for _, name := range []string{"Amnesia", "Inquisition"} {
		g, a, b := ronGame()
		spell := repositoryCard(t, "DRK", name)
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		a.ManaPool().Add(core.Blue, 6)
		a.ManaPool().Add(core.Black, 6)
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if len(g.PublicReveals()) != 0 || b.Life() != 20 || len(b.Hand()) != 0 || len(a.Graveyard()) != 1 {
			t.Fatal("empty hand interrupted resolution")
		}
	}
	a := &ronDiscardPlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	g.SetActivePlayerIndex(1)
	g.PutOnBattlefield(mage.NewArtifact("Replacement", "", mage.WithStaticAbility(mage.DiscardToLibraryReplacement())), a.PlayerID())
	for _, c := range []mage.Card{mage.NewLand("Land"), mage.NewCreature("White", "{W}", 1, 1), mage.NewCreature("Blue", "{U}", 1, 1)} {
		addRONHandCard(a.BasePlayer, c)
	}
	spell := repositoryCard(t, "DRK", "Amnesia")
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	b.ManaPool().Add(core.Blue, 6)
	if err := g.CastSpellByID(b.PlayerID(), spell.ID(), []uuid.UUID{a.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(g.PublicReveals()) != 3 || len(a.Hand()) != 1 || len(a.Library()) != 2 || len(a.Graveyard()) != 0 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 2 {
		t.Fatal("RON discard replacement")
	}
}

func TestRONHandOperationsResolvingContexts(t *testing.T) {
	body := `ActionList([PlayerAction(You,RevealHand),PlayerAction(You,DiscardCards(And([IsCardtype(Creature),Or([IsColor(White),IsColor(Blue)]),IsNonColor(Red)]))),GainLife(TheNumberOfCardsOfTypeInPlayersHand(IsNonCardtype(Creature),You))])`
	for _, change := range []string{"normal", "leave", "return", "phase", "control"} {
		t.Run(change, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `Activated(TapPermanent(ThisPermanent),`+body+`)`)
			first, second := factory(), factory()
			if first.Abilities()[0] == second.Abilities()[0] {
				t.Fatal("shared ability")
			}
			g, a, b := ronGame()
			source := g.PutOnBattlefield(first, a.PlayerID())
			other := g.PutOnBattlefield(second, b.PlayerID())
			for _, p := range []*mage.BasePlayer{a, b} {
				addRONHandCard(p, mage.NewCreature("Blue", "{U}", 1, 1))
				addRONHandCard(p, mage.NewCreature("Red", "{R}", 1, 1))
				addRONHandCard(p, mage.NewLand("Land"))
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(b.PlayerID(), other.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "leave", "return":
				g.RemoveFromBattlefield(source)
				if change == "return" {
					g.PutOnBattlefield(first, b.PlayerID())
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if len(a.Hand()) != 3 || len(b.Hand()) != 3 || len(g.PublicReveals()) != 0 {
				t.Fatal("early hand actions")
			}
			g.ResolveStack()
			if len(g.PublicReveals()) != 6 || len(a.Hand()) != 2 || len(b.Hand()) != 2 || a.Life() != 21 || b.Life() != 21 {
				t.Fatal("source changed controller or crossed instances")
			}
		})
	}
	for _, rule := range []string{
		`TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),` + body + `)`,
		`Activated(TapPermanent(ThisPermanent),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,` + body + `)]))`,
	} {
		g, a, _ := ronGame()
		addRONHandCard(a, mage.NewCreature("White", "{W}", 1, 1))
		addRONHandCard(a, mage.NewLand("Land"))
		source := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", rule)(), a.PlayerID())
		if strings.HasPrefix(rule, "Activated") {
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if len(g.PublicReveals()) != 0 {
				t.Fatal("delayed effect early")
			}
			g.RunStepWithPriority(core.EndStep)
		}
		g.PutTriggersOnStack()
		g.ResolveStack()
		if len(a.Hand()) != 1 || len(g.PublicReveals()) != 2 || a.Life() != 21 {
			t.Fatal("trigger context")
		}
	}
}

func TestRONHandOperationsCopyTargets(t *testing.T) {
	for _, name := range []string{"Amnesia", "Inquisition"} {
		g, a, b := ronGame()
		for _, p := range []*mage.BasePlayer{a, b} {
			addRONHandCard(p, mage.NewCreature("White", "{W}", 1, 1))
			addRONHandCard(p, mage.NewLand("Land"))
		}
		spell := repositoryCard(t, "DRK", name)
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		a.ManaPool().Add(core.Blue, 6)
		a.ManaPool().Add(core.Black, 6)
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
			t.Fatal(err)
		}
		cp := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
		if cp == nil {
			t.Fatal("copy")
		}
		cp.Targets = []uuid.UUID{a.PlayerID()}
		g.ResolveTopOfStack()
		if len(g.PublicReveals()) != 2 || g.PublicReveals()[0].Owner != a.PlayerID() || len(b.Hand()) != 2 {
			t.Fatal("copy used original target")
		}
		if name == "Amnesia" && len(a.Hand()) != 1 {
			t.Fatal("copy discard")
		}
		if name == "Inquisition" && a.Life() != 19 {
			t.Fatal("copy damage")
		}
		g.ResolveStack()
		if len(g.PublicReveals()) != 4 || g.PublicReveals()[2].Owner != b.PlayerID() {
			t.Fatal("copy changed original target")
		}
	}
}
