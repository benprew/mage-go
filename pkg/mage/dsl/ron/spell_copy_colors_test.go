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

type ronColorCopyPlayer struct {
	*mage.BasePlayer
	target   uuid.UUID
	calls    int
	possible []uuid.UUID
}

func (p *ronColorCopyPlayer) ChooseTargets(possible []uuid.UUID, _, _ int, _ *mage.Game) []uuid.UUID {
	p.calls++
	p.possible = slices.Clone(possible)
	if p.target == uuid.Nil {
		return nil
	}
	return []uuid.UUID{p.target}
}

func TestRONSpellCopyColorsFork(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, change := range []string{"keep", "retarget", "depart", "recast", "copy", "clone", "recolor", "counter"} {
			t.Run(set+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				chooser := &ronColorCopyPlayer{BasePlayer: b}
				g.SetPlayerAt(1, chooser)
				for range 10 {
					a.AddToLibrary(mage.NewLand("A"))
					b.AddToLibrary(mage.NewLand("B"))
				}
				original := castRONSpell(t, g, a, repositoryCard(t, "LEA", "Ancestral Recall"), a.PlayerID())
				fork := repositoryCard(t, set, "Fork")
				fork.SetOwner(b.PlayerID())
				b.AddToHand(fork)
				if err := g.CastSpellByID(b.PlayerID(), fork.ID(), []uuid.UUID{original.ID}, 0); err == nil || g.StackSize() != 1 || len(b.Hand()) != 1 {
					t.Fatal("unpaid cast changed state")
				}
				b.ManaPool().Add(core.Red, 2)
				if err := g.CastSpellByID(b.PlayerID(), fork.ID(), []uuid.UUID{original.ID}, 0); err != nil {
					t.Fatal(err)
				}
				if chooser.calls != 0 || g.StackSize() != 2 {
					t.Fatal("copy before resolution")
				}
				wantA, wantB := 6, 0
				switch change {
				case "retarget":
					chooser.target = b.PlayerID()
					wantA, wantB = 3, 3
				case "depart", "recast":
					g.CounterStackObject(original.ID)
					wantA = 0
					if change == "recast" {
						a.RemoveFromGraveyard(original.Card.ID())
						forkObject := g.GetStack().RemoveByID(g.StackPeek().ID)
						castRONSpell(t, g, a, original.Card, a.PlayerID())
						g.GetStack().Push(forkObject)
						wantA = 3
					}
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy Fork")
					}
					wantA = 9
				case "recolor":
					g.ChangeObjectColor(original.ID, core.Green)
				case "counter":
					g.CounterStackObject(g.StackPeek().ID)
					wantA = 3
				}
				clone := g.Clone()
				g.ResolveTopOfStack()
				if change != "depart" && change != "recast" && change != "counter" {
					cp := g.StackPeek()
					if !cp.IsCopy || cp.Card.Name() != "Ancestral Recall" || cp.Controller != b.PlayerID() || cp.SourceID == original.SourceID || !slices.Equal(g.EffectiveColors(cp.ID), []core.Color{core.Red}) {
						t.Fatal("wrong copy colors or identity")
					}
					if change == "clone" {
						g.ChangeObjectColor(cp.ID, core.Black)
						next := g.CopyStackObjectDirect(cp, b.PlayerID(), false)
						if !slices.Equal(g.EffectiveColors(next.ID), []core.Color{core.Red}) {
							t.Fatal("copy of copy lost red exception")
						}
						wantA = 9
					}
				}
				g.ResolveStack()
				if len(a.Hand()) != wantA || len(b.Hand()) != wantB || b.ManaPool().TotalMana() != 0 {
					t.Fatalf("draws A/B=%d/%d want %d/%d", len(a.Hand()), len(b.Hand()), wantA, wantB)
				}
				if len(clone.GetPlayer(a.PlayerID()).Hand()) != 0 {
					t.Fatal("shared clone state")
				}
				clone.ResolveStack()
				cloneA := wantA
				cloneB := wantB
				if change == "clone" {
					cloneA = 6
				}
				if change == "retarget" {
					cloneA, cloneB = 6, 0
				}
				if len(clone.GetPlayer(a.PlayerID()).Hand()) != cloneA || len(clone.GetPlayer(b.PlayerID()).Hand()) != cloneB {
					t.Fatal("clone lost copied body")
				}
			})
		}
	}
}

const ronColorCopyAction = `CopySpellAndMayChooseNewTargetsWithEffects(Ref_TargetSpell,[SetColor(SimpleColorList([Red]))])`

func TestRONSpellCopyColorsInvalid(t *testing.T) {
	for _, action := range []string{
		`CopySpellAndMayChooseNewTargetsWithEffects`,
		`CopySpellAndMayChooseNewTargetsWithEffects(Ref_TargetSpell)`,
		strings.Replace(ronColorCopyAction, "Ref_TargetSpell", "ThisSpell", 1),
		strings.Replace(ronColorCopyAction, "Ref_TargetSpell", "Ref_TargetSpell()", 1),
		strings.Replace(ronColorCopyAction, "Ref_TargetSpell", "Ref_TargetPermanent", 1),
		strings.Replace(ronColorCopyAction, "[SetColor(SimpleColorList([Red]))]", "[]", 1),
		strings.Replace(ronColorCopyAction, "[Red]", "[Purple]", 1),
		strings.Replace(ronColorCopyAction, "[Red]", "[Colorless]", 1),
		strings.Replace(ronColorCopyAction, "[Red]", "[Red()]", 1),
		strings.Replace(ronColorCopyAction, "SimpleColorList([Red])", "TheChosenColor", 1),
		strings.Replace(ronColorCopyAction, "SetColor(SimpleColorList([Red]))", "SetColor(SimpleColorList([Red])),SetColor(SimpleColorList([Blue]))", 1),
		strings.Replace(ronColorCopyAction, "SetColor", "AddColor", 1),
	} {
		source := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetSpell(Or([IsCardtype(Instant),IsCardtype(Sorcery)]))],ActionList([%s])))])`, action)
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
	for _, rule := range []string{`SpellActions(ActionList([` + ronColorCopyAction + `]))`, `Activated(` + ronColorCopyAction + `,ActionList([]))`, `AsPermanentEnters(ThisPermanent,[` + ronColorCopyAction + `])`} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rule + `])`
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid context")
		}
	}
}

func TestRONSpellCopyColorsAtomicLoad(t *testing.T) {
	files := fstest.MapFS{}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "Fork"`) {
				files[set+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + ronColorCopyAction + `]))])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("invalid collection published")
	}
	if _, err := registry.CreateCard("Fork"); err != nil {
		t.Fatal("atomic rejection lost registry", err)
	}
}

func TestRONSpellCopyColorsHurricane(t *testing.T) {
	g, a, b := ronGame()
	flyer := g.PutOnBattlefield(mage.NewCreature("Protected flyer", "", 1, 8, mage.WithKeyword(core.Flying), mage.WithAbility(mage.ProtectionFromColors(core.Red))), b.PlayerID())
	hurricane := repositoryCard(t, "LEA", "Hurricane")
	hurricane.SetOwner(a.PlayerID())
	a.AddToHand(hurricane)
	a.ManaPool().Add(core.Green, 1)
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.CastSpellByID(a.PlayerID(), hurricane.ID(), nil, 3); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	castRONSpell(t, g, b, repositoryCard(t, "LEA", "Fork"), original.ID)
	clone := g.Clone()
	g.ResolveTopOfStack()
	cp := g.StackPeek()
	if !cp.IsCopy || cp.XValue != 3 || len(cp.Targets) != 0 || cp.Controller != b.PlayerID() || !slices.Equal(g.EffectiveColors(cp.ID), []core.Color{core.Red}) {
		t.Fatal("sorcery copy lost X or colors")
	}
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		if game.GetPlayer(a.PlayerID()).Life() != 14 || game.GetPlayer(b.PlayerID()).Life() != 14 || game.FindPermanent(flyer.ID()).Damage != 3 {
			t.Fatal("red copied damage source or prevention", game.FindPermanent(flyer.ID()).Damage)
		}
	}
	if g.GetSorceriesCastThisTurn(a.PlayerID()) != 1 || g.GetSorceriesCastThisTurn(b.PlayerID()) != 0 || g.GetInstantsCastThisTurn(b.PlayerID()) != 1 {
		t.Fatal("copy generated a cast event")
	}
	if len(a.Graveyard()) != 1 || len(b.Graveyard()) != 1 {
		t.Fatal("spell copies entered graveyards")
	}
}

func TestRONSpellCopyColorsShapes(t *testing.T) {
	for _, filter := range []string{`IsCardtype(Instant)`, `IsCardtype(Sorcery)`, `Or([IsCardtype(Instant),IsCardtype(Sorcery)])`, `And([IsColor(Blue),IsCardtype(Instant)])`} {
		for _, colors := range []string{`[]`, `[White,Blue,Black,Red,Green]`} {
			rule := `SpellActions(Targeted([TargetSpell(` + filter + `)],ActionList([CopySpellAndMayChooseNewTargetsWithEffects(Ref_TargetSpell,[SetColor(SimpleColorList(` + colors + `))])])))`
			factory := permanentScopeFactory(t, "Instant", rule)
			first, second := factory(), factory()
			if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
				t.Fatal("factory shared target declarations")
			}
			g, a, b := ronGame()
			victim := mage.Card(mage.NewInstant("Blue", "{U}", mage.WithAbility(mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(2))))))
			if strings.Contains(filter, `IsCardtype(Sorcery)`) && !strings.Contains(filter, `Instant`) {
				victim = mage.NewSorcery("Blue", "{U}", mage.WithAbility(mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(2)))))
			}
			original := castRONSpell(t, g, a, victim)
			castRONSpell(t, g, b, first, original.ID)
			g.ResolveTopOfStack()
			want := []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green}
			if colors == `[]` {
				want = nil
			}
			if !slices.Equal(g.EffectiveColors(g.StackPeek().ID), want) {
				t.Fatal("fixed color list")
			}
			g.ResolveStack()
			if a.Life() != 22 || b.Life() != 22 {
				t.Fatal("copy controller")
			}
		}
	}
	for _, filter := range []string{`AnySpell`, `IsCardtype(Creature)`, `Or([IsCardtype(Instant),IsCardtype(Creature)])`, `And([IsColor(Blue)])`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad scope",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetSpell(`+filter+`)],ActionList([`+ronColorCopyAction+`])))])`), "bad-scope.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted unsupported spell context", filter)
		}
	}
}

func TestRONSpellCopyColorsText(t *testing.T) {
	g, a, b := ronGame()
	for range 6 {
		a.AddToLibrary(mage.NewLand("Draw"))
	}
	original := castRONSpell(t, g, a, repositoryCard(t, "LEA", "Ancestral Recall"), a.PlayerID())
	fork := castRONSpell(t, g, b, repositoryCard(t, "LEA", "Fork"), original.ID)
	clone := g.Clone()
	change, err := mage.ReplaceColorWord(core.Red, core.White)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.RevisePrintedSpellText(fork.ID, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	clone.ResolveTopOfStack()
	if !slices.Equal(g.EffectiveColors(g.StackPeek().ID), []core.Color{core.White}) || !slices.Equal(clone.EffectiveColors(clone.StackPeek().ID), []core.Color{core.Red}) {
		t.Fatal("text revision changed another game or lost copy color")
	}
	copyOfCopy := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	if !slices.Equal(g.EffectiveColors(copyOfCopy.ID), []core.Color{core.White}) {
		t.Fatal("modified exception was not copiable")
	}
}

func TestRONSpellCopyColorsRetarget(t *testing.T) {
	for _, pick := range []string{"legal", "protected", "cancel"} {
		t.Run(pick, func(t *testing.T) {
			g, a, b := ronGame()
			chooser := &ronColorCopyPlayer{BasePlayer: b}
			g.SetPlayerAt(1, chooser)
			protected := g.PutOnBattlefield(mage.NewCreature("Protected", "", 1, 8, mage.WithAbility(mage.ProtectionFromColors(core.Red))), a.PlayerID())
			legal := g.PutOnBattlefield(mage.NewCreature("Legal", "", 1, 8), a.PlayerID())
			original := castRONSpell(t, g, a, repositoryCard(t, "LEA", "Psionic Blast"), protected.ID())
			castRONSpell(t, g, b, repositoryCard(t, "LEA", "Fork"), original.ID)
			if pick == "legal" {
				chooser.target = legal.ID()
			}
			if pick == "protected" {
				chooser.target = protected.ID()
			}
			g.ResolveTopOfStack()
			if chooser.calls != 1 || slices.Contains(chooser.possible, protected.ID()) || !slices.Contains(chooser.possible, legal.ID()) {
				t.Fatal("copy color was not used during target choice")
			}
			g.ResolveStack()
			wantLegal, wantB := 0, 20
			if pick == "legal" {
				wantLegal, wantB = 4, 18
			}
			if g.FindPermanent(protected.ID()).Damage != 4 || g.FindPermanent(legal.ID()).Damage != wantLegal || a.Life() != 18 || b.Life() != wantB {
				t.Fatal("target legality or copied source/controller")
			}
		})
	}
}

func TestRONSpellCopyColorsModal(t *testing.T) {
	for mode := range 2 {
		g, base, b := ronGame()
		a := &ronModePlayer{BasePlayer: base, mode: mode}
		g.SetPlayerAt(0, a)
		g.SetPlayerAt(1, &ronColorCopyPlayer{BasePlayer: b})
		original := castRONSpell(t, g, base, repositoryCard(t, "LEA", "Healing Salve"), b.PlayerID())
		castRONSpell(t, g, b, repositoryCard(t, "LEA", "Fork"), original.ID)
		g.ResolveTopOfStack()
		cp := g.StackPeek()
		if cp.ModeChoice != mode || len(cp.ModalTargets) != 2 || len(cp.ModalTargets[1-mode]) != 0 || a.prompts != 1 {
			t.Fatal("mode or unchosen target groups changed")
		}
		g.ResolveStack()
		want := 26
		if mode == 1 {
			want = 20
		}
		if b.Life() != want {
			t.Fatal("copied mode did not resolve")
		}
		if mode == 1 {
			castRONSpell(t, g, base, repositoryCard(t, "LEA", "Lightning Bolt"), b.PlayerID())
			g.ResolveStack()
			if b.Life() != 20 {
				t.Fatal("prevention mode lost its target")
			}
		}
	}
}

func TestRONSpellCopyColorsFireball(t *testing.T) {
	g, a, b := ronGame()
	g.SetPlayerAt(1, &ronColorCopyPlayer{BasePlayer: b})
	card := repositoryCard(t, "LEA", "Fireball")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	a.ManaPool().Add(core.Colorless, 6)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{a.PlayerID(), b.PlayerID()}, 5); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	fork := repositoryCard(t, "LEA", "Fork")
	fork.SetOwner(b.PlayerID())
	b.AddToHand(fork)
	b.ManaPool().Add(core.Red, 2)
	if err := g.CastSpellByID(b.PlayerID(), fork.ID(), []uuid.UUID{original.ID}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	cp := g.StackPeek()
	if cp.XValue != 5 || len(cp.Targets) != 2 || !slices.Equal(cp.Targets, original.Targets) || b.ManaPool().TotalMana() != 0 {
		t.Fatal("X, target count or repeated casting surcharge")
	}
	g.ResolveStack()
	if a.Life() != 16 || b.Life() != 16 {
		t.Fatal("copied division or rounding")
	}
}

func TestRONSpellCopyColorsContexts(t *testing.T) {
	nodes, err := Parse(strings.NewReader(ronColorCopyAction), "copy-context.ron")
	if err != nil {
		t.Fatal(err)
	}
	target := &spellTargetPlan{name: "Ref_TargetSpell", instantOrSorcery: true}
	for _, ctx := range []scope{
		{spellTargets: true, activation: true, spellTarget: target},
		{spellTargets: true, permanent: true, spellTarget: target},
		{spellTargets: true, delayedStep: true, spellTarget: target},
		{spellTargets: true, drawReplacementBody: true, spellTarget: target},
		{spellTarget: target},
	} {
		if _, err := compileSpellColorCopy(nodes[0], ctx); err == nil || !strings.Contains(err.Error(), "spell color copy context") {
			t.Fatal("unsupported context reached lowering", err)
		}
	}
}
