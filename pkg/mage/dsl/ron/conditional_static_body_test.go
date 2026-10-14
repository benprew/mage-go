package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const conditionalBodyPrevention = `PreventDamage(DamageWouldBeDealtByAPermanentToRecipient(IsCardtype(Artifact),Permanent(ThisPermanent)),[PreventThatDamage])`
const conditionalBodyLayer = `PermanentLayerEffect(ThisPermanent,[AdjustPT(1,2)])`
const conditionalBodyAttack = `PermanentRuleEffect(ThisPermanent,[CantAttack])`

func conditionalBodyCompile(t *testing.T, rule string) (mage.CardFactory, error) {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Conditional body",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),ManaCost:[ManaCostGeneric(1)],Rules:[`+rule+`],CardPT:(Power:2,Toughness:20))`), "conditional-body.ron")
	if err != nil {
		t.Fatal(err)
	}
	return Compile(nodes[0])
}

func TestRONConditionalStaticBodyGameplay(t *testing.T) {
	for _, form := range []string{"If", "Unless", "IfElse"} {
		t.Run(form, func(t *testing.T) {
			body := `[` + conditionalBodyAttack + `,` + conditionalBodyPrevention + `,` + conditionalBodyLayer + `]`
			rule := form + `(PermanentPassesFilter(ThisPermanent,IsUntapped),` + body
			if form == "IfElse" {
				rule += `,[PermanentLayerEffect(ThisPermanent,[AdjustPT(3,4)])]`
			}
			rule += `)`
			factory, err := conditionalBodyCompile(t, rule)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			source := g.PutOnBattlefield(mage.NewArtifact("Source", ""), b.PlayerID())
			for _, tapped := range []bool{false, true, false} {
				g.MutablePermanent(p.ID()).Tapped = tapped
				g.ApplyContinuousEffects()
				active := !tapped
				if form == "Unless" {
					active = tapped
				}
				power, toughness := 2, 20
				if active {
					power, toughness = 3, 22
				} else if form == "IfElse" {
					power, toughness = 5, 24
				}
				checkConditionalPT(t, g, p.ID(), power, toughness)
				g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
				if !tapped && g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) == active {
					t.Fatal("mixed attack condition")
				}
				before := g.FindPermanent(p.ID()).Damage
				g.DealDamageToPermanent(g.FindPermanent(p.ID()), 1, source.ID())
				want := before + 1
				if active {
					want = before
				}
				if g.FindPermanent(p.ID()).Damage != want {
					t.Fatal("mixed prevention condition")
				}
			}
			clone := g.Clone()
			clone.MutablePermanent(p.ID()).Tapped = true
			clone.ApplyContinuousEffects()
			if form == "Unless" {
				checkConditionalPT(t, g, p.ID(), 2, 20)
			} else {
				checkConditionalPT(t, g, p.ID(), 3, 22)
			}
		})
	}
}

func TestRONConditionalStaticBodyInvalid(t *testing.T) {
	for _, rule := range []string{
		`If(PermanentPassesFilter(ThisPermanent,IsUntapped),[])`,
		`IfElse(PermanentPassesFilter(ThisPermanent,IsUntapped),[` + conditionalBodyLayer + `],[` + conditionalBodyPrevention + `,DrawACard])`,
		`If(PermanentPassesFilter(ThisPermanent,IsUntapped),[` + conditionalBodyPrevention + `,PermanentLayerEffect(ThisPermanent,[AddAbility([Flying])]),` + conditionalBodyLayer + `])`,
		`If(PermanentPassesFilter(ThisPermanent,IsUntapped),[PreventDamage(DamageWouldBeDealtByAPermanentToRecipient(IsCardtype(Artifact),Permanent(Ref_TargetPermanent)),[PreventThatDamage])])`,
		`If(PermanentPassesFilter(ThisPermanent,IsUntapped),[` + conditionalBodyPrevention + `,Banding])`,
	} {
		if _, err := conditionalBodyCompile(t, rule); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONConditionalStaticBodyRepositoryRegression(t *testing.T) {
	t.Run("Kird Ape and Sedge Troll", TestRONConditionalStaticController)
	t.Run("Giant Tortoise", TestRONConditionalStaticUntapped)
	t.Run("permanent source prevention", TestRONStaticPreventionCards)
	t.Run("replacement order", TestRONStaticPreventionReplacementOrder)
}

func TestRONConditionalStaticBodyLoadAndInstances(t *testing.T) {
	source := `Card(Name:"Mixed body",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[If(PermanentPassesFilter(ThisPermanent,IsUntapped),[` + conditionalBodyPrevention + `,` + conditionalBodyLayer + `])],CardPT:(Power:2,Toughness:20))`
	files := fstest.MapFS{"mixed.ron": {Data: []byte(source)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	first, err := registry.CreateCard("Mixed body")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Mixed body")
	if err != nil {
		t.Fatal(err)
	}
	left := g.PutOnBattlefield(first, a.PlayerID())
	right := g.PutOnBattlefield(second, b.PlayerID())
	g.MutablePermanent(left.ID()).Tapped = true
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, left.ID(), 2, 20)
	checkConditionalPT(t, g, right.ID(), 3, 22)
	sourcePermanent := g.PutOnBattlefield(mage.NewArtifact("Damage source", ""), a.PlayerID())
	saved := g.CaptureDamageSource(sourcePermanent.ID())
	g.DealSimultaneousDamage(mage.PermanentDamageAssignment(g.FindPermanent(left.ID()), 1, saved), mage.PermanentDamageAssignment(g.FindPermanent(right.ID()), 1, saved))
	if g.FindPermanent(left.ID()).Damage != 1 || g.FindPermanent(right.ID()).Damage != 0 {
		t.Fatal("factory subjects shared")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(source, conditionalBodyLayer, "DrawACard", 1))}
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("partial collection accepted")
	}
	if len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("rejection changed registry")
	}
	if _, err := registry.CreateCard("Mixed body"); err != nil {
		t.Fatal(err)
	}
}

func TestRONConditionalStaticBodyHostUntap(t *testing.T) {
	source := `Card(Name:"Mixed host",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),If(PermanentPassesFilter(ThisPermanent,HasACounterOfType(NetCounter)),[PermanentRuleEffect(HostPermanent,[DoesntUntapDuringControllersUntap]),PermanentLayerEffect(HostPermanent,[AdjustPT(1,1)])])])`
	nodes, err := Parse(strings.NewReader(source), "mixed-host.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 8), b.PlayerID())
	aura := g.PutOnBattlefield(factory(), a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	g.SetActivePlayerIndex(1)
	for _, count := range []int{0, 1, 0} {
		g.MutablePermanent(aura.ID()).Counters[core.Net] = count
		g.ApplyContinuousEffects()
		checkConditionalPT(t, g, host.ID(), 2+count, 8+count)
		g.TapPermanent(g.FindPermanent(host.ID()))
		g.RunStepWithPriority(core.Untap)
		if g.FindPermanent(host.ID()).Tapped != (count > 0) {
			t.Fatal("mixed host untap condition")
		}
	}
	g.MutablePermanent(aura.ID()).Counters[core.Net] = 1
	g.ApplyContinuousEffects()
	clone := g.Clone()
	g.PhaseOut(aura.ID())
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, host.ID(), 2, 8)
	checkConditionalPT(t, clone, host.ID(), 3, 9)
	clone.RemoveFromBattlefield(clone.FindPermanent(aura.ID()))
	clone.ApplyContinuousEffects()
	checkConditionalPT(t, clone, host.ID(), 2, 8)
}
