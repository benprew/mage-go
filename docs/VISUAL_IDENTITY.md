# mage-go Visual Identity

A reference for anyone building UI, web pages, or marketing materials for the project.

## Logo

The logo is an SVG in `docs/logo.svg`. It depicts a **WUBRG pentagon** — the five mana colors arranged at the vertices of a regular pentagon, connected by the pentagon's edges, with the inner pentagram star drawn in faint gold. Two concentric arcane circles (outer and inner) frame the geometry. Spokes radiate from the center to each vertex. A small gold dot marks the center.

Each mana pip uses the same radial gradients as the inline mana pips (see Mana Color Identity below). The overall feel is an arcane diagram — something you'd find in a planeswalker's grimoire.

**Usage:** The logo works on dark backgrounds only. It has no background fill of its own. Do not place it on light surfaces. Do not add drop shadows or outer glows beyond what's built into the SVG.

## Philosophy

The visual identity draws from what made early Magic distinctive: the feeling of cracking open a Revised starter deck in 1994, reading The Duelist by lamplight, browsing fan sites on Geocities. It should feel **arcane**, **textural**, and **a little mysterious** — not clean and corporate.

Think: aged parchment, gold leaf, candlelit libraries, the ornate card frames before 8th Edition. The 90s internet aesthetic — dark backgrounds, visible borders, content-dense pages that reward reading — but executed with modern CSS and taste.

**Do not** reach for Bootstrap, Tailwind utility soup, or any design system that flattens everything into sanitized cards and rounded corners. The whole point is that it *doesn't* look like every other developer landing page.

## Color Palette

| Name            | Hex       | Usage                                    |
|-----------------|-----------|------------------------------------------|
| Void Black      | `#0a0a0c` | Page backgrounds, deepest darks          |
| Parchment       | `#d4c5a0` | Body text, primary readable color        |
| Aged Gold       | `#c9a84c` | Headings, accents, borders, emphasis     |
| Mana Blue       | `#4a82b0` | Links, interactive elements, highlights  |
| Blood Red       | `#8b2020` | Danger states, power/aggression accents  |
| Forest Dark     | `#2d4a2d` | Success states, nature/growth accents    |
| Artifact Grey   | `#6b6b7b` | Secondary text, muted labels             |
| Enchant Purple  | `#5c3a6e` | Special accents, rare/mythic indicators  |
| Dim Gold        | `#8a7a5a` | Tertiary text, flavor text, subtitles    |
| Deep Border     | `#2a2520` | Card-frame borders, divider lines        |
| Code Tint       | `#a0c0d8` | Inline code text                         |
| Terminal Green   | `#6a8a6a` | Terminal output, CLI references          |

### Usage Rules

- **Backgrounds** are always near-black (`#0a0a0c` to `#12121a`). Never white. Never light grey. The darkness is the canvas.
- **Gold is the accent**, not the background. Use it for headings, borders, and small highlights. Overuse kills the effect.
- **Parchment text on void black** is the default reading experience. High contrast but warm — not the harsh white-on-black of a code editor.
- **Blue is for interaction only.** Links, buttons, focused states. Not decoration.
- **Red and purple are rare.** Use them like mythic rares — sparingly, for impact.

## Typography

### Fonts

| Role     | Font Stack                                           | Notes                                       |
|----------|------------------------------------------------------|---------------------------------------------|
| Display  | `'Cinzel', Georgia, serif`                           | Headings, navigation, stat numbers          |
| Flavor   | `'MedievalSharp', Georgia, serif`                    | Subtitles, flavor quotes, decorative text   |
| Body     | `Georgia, 'Times New Roman', serif`                  | All paragraph text                          |
| Code     | `'SF Mono', 'Fira Code', 'Consolas', monospace`     | Code blocks, terminal output, CLI refs      |

Google Fonts imports: `Cinzel:wght@400;700` and `MedievalSharp`.

### Rules

- Headings are **uppercase**, **letter-spaced** (`0.08em`–`0.15em`), gold.
- Body text is serif. Always. Serif fonts evoke the era and the game's typographic roots (Goudy Medieval on the early cards).
- Never use sans-serif for content. If you need a clean UI font for form elements or app chrome, prefer the system serif stack anyway.
- Font sizes are modest. The 90s web was readable because it wasn't trying to be a billboard.

## Texture & Depth

### Noise Overlay

A subtle SVG fractalNoise filter is applied as a `::before` pseudo-element on `body`, at 3% opacity. This gives the void-black background the grain of old paper or a dark matte surface. Don't remove it — it's the difference between "digital black" and "physical black."

### Card Frames

Content sections are styled as "card frames" — bordered containers that evoke the MTG card border:

- Background: subtle gradient (`#12121a` → `#0e0e14`)
- Outer border: `1px solid #2a2520`
- Layered box-shadow for depth (inset shadow + multiple outlines)
- Inner border: `1px solid rgba(201,168,76,0.08)` — a faint gold inner line, like the card frame's inner border

### Ornamental Dividers

Between major sections, use the gold gradient divider:

```css
background: linear-gradient(90deg,
  transparent 0%, #3a3020 15%, #c9a84c 40%,
  #e8d48b 50%, #c9a84c 60%, #3a3020 85%, transparent 100%
);
```

Or the text ornament: `— ◊ —` in `#2a2520`.

### Mana Pips

The five colors are represented as small radial-gradient circles with the classic WUBRG order. Each pip uses a two-stop radial gradient (lighter center, darker edge) and an inset shadow for the embossed look of physical mana symbols.

## Layout Principles

- **Max content width: ~780px.** Long lines are hard to read. The narrow column forces focus.
- **Generous vertical spacing.** Let sections breathe. The 90s web was dense but not cramped.
- **Center-aligned navigation** with diamond separators (`◊`). Evokes the sparse nav bars of old fan sites.
- **Two-column grids** for feature lists. Falls to single column on mobile.
- **Terminal previews** use a fake window chrome (three dots) for personality.

## Mana Color Identity

When representing the five colors of Magic, use these specific values:

| Color | Pip Background                              | Text Color |
|-------|---------------------------------------------|------------|
| White | `radial-gradient(#f8f4e0, #d4c8a0)`        | `#3a3020`  |
| Blue  | `radial-gradient(#6aa8d4, #2a5a8a)`        | `#e0eaf4`  |
| Black | `radial-gradient(#4a4a52, #1a1a20)`        | `#a0a0a8`  |
| Red   | `radial-gradient(#c84a30, #7a2010)`        | `#f4d0c0`  |
| Green | `radial-gradient(#4a8a3a, #1a4a10)`        | `#d0e8c0`  |

Always display in WUBRG order (White, Blue, Black, Red, Green).

## Set Status Indicators

- **Complete:** `#4a8a3a` (Forest Dark), uppercase, small
- **In Progress:** `#b89a30` (warm gold-amber), uppercase, small

## Tone of Voice

The site copy should be:

- **Direct and confident.** State what the project does. Don't hedge.
- **Technical but not academic.** Developers are the audience. Use precise terms (LIFO, SBA, continuous effect) without over-explaining.
- **Flavored but not cosplay.** The occasional italic aside or Magic reference is good. Writing entire paragraphs in-character is not.
- **Respect the source material.** Magic is a 30-year-old game with deep mechanical history. The tone should convey that the authors take the rules seriously.

## Don'ts

- Don't use gradients on text (except `text-shadow` glow on the title).
- Don't animate anything. Static pages. The 90s web didn't bounce.
- Don't use rounded corners on content containers. Card frames have sharp edges.
- Don't use icons from icon libraries. If you need an icon, use a text symbol or a mana pip.
- Don't put anything in a "hero section" with a massive background image. The title, subtitle, and mana pips are the hero.
- Don't use light mode. There is no light mode. The void is the only mode.
