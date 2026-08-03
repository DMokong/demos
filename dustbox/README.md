# Dustbox — landing page prototypes

Five self-contained prototypes for **Dustbox**, Dustin's personal sandbox site
(projects, prototypes, talks & decks, thesis & papers, blog, reports for teams
and leadership).

Open `index.html` in a browser — no build step, no dependencies. Each prototype
is a single HTML file you can copy out and use as a starting point.

| File | Concept | The gimmick |
|---|---|---|
| `prototype-a-sandbox-hero.html` | **Sandbox Hero** — the red sandbox drawn in SVG, sitting *on* the page (no comic framing) | Click the sand to kick it; **Jump in!** erupts the whole surface with a screen shake and a `SPLOPP!` |
| `prototype-b-dust-on-glass.html` | **Dust on Glass** — a window at night; the site name is a message wiped in the dust | The title is literally erased out of a canvas dust layer; your cursor wipes more away, and the dust slowly resettles |
| `prototype-c-drawn-in-sand.html` | **Scrawled in Sand** — the entire page is the sand surface | Your cursor is a twig: drag to carve grooves (the wind fades them), click to kick a burst of grains |
| `prototype-d-the-story.html` | **The Story** — the four-panel strip dissolved into the page as a scroll narrative (plan → cannonball → tornado → build again) | Panels are cropped out of one image with feathered edges and multiply blending — no frames, no strip. Scrolling to each scene fires its moment: grains trickle while Calvin draws, Hobbes' landing erupts real sand with a SPLOPP, panel three grows a live particle tornado, panel four settles into drift. Calvin's sand diagram continues out of the artwork onto the page |
| `prototype-e-living-sandbox.html` | **The Sandbox** — the panel-1 picture *is* the sandbox, sitting on the page | The box is one object on the paper (white background blended away). The diagram Calvin is drawing runs out of the box across the page as grooves scratched into sand, and each groove ends at a section of the site — so the sitemap is literally his sand drawing. Sand drifts off the box and kicks up when you poke it. Nothing flies in or interrupts |

They share a common content skeleton (hero → six category cards → "latest"
list → footer), so you can graft the sections of one onto the shell of another.

## Dropping in your Calvin & Hobbes graphics

The characters are Bill Watterson's, so the pages don't try to redraw them —
instead each page has a slot that picks up an image from `assets/` and hides
itself if the file is missing:

| File to add | Used by | Notes |
|---|---|---|
| `assets/calvin-sandbox.png` | A (beside the sandbox), C (bottom corner) | The small "Calvin at the red sandbox" image works as-is: `mix-blend-mode: multiply` melts its white background into the page, so it sits *on* the page rather than in a panel |
| `assets/calvin-sill.png` | B (on the windowsill) | B has a dark background, so this one needs a **transparent** PNG (multiply doesn't work on dark) |
| `assets/calvin-hobbes-strip.png` | D (all four scenes) | Save the whole 2×2 four-panel strip as **one** image — D crops each quadrant itself. If a black frame edge peeks into a scene, nudge that scene's `background-position` (`.s1 .crop` … `.s4 .crop`) a percent or two, or raise the `--z` zoom in `:root` |
| `assets/sandbox-scene.png` | E (the sandbox itself) | **The important one.** Crop panel 1 of the strip — Calvin kneeling at his sandbox, drawing. A white background is fine; the page blends it into the paper. Until it exists, E draws a stand-in sandbox |
| `assets/cutout-hobbes.png` | E (optional, still) | A transparent PNG of Hobbes, who just sits beside the box watching. He does not move |

Worth keeping in mind for a public site: sand, sandbox, red box, "I'm Calvin
energy" are all safely yours; the actual character artwork is fine as a private
homage but is the one part you'd want to swap for commissioned/original art if
the site ever becomes more than personal.

## Tuning the sand

Each file has one self-contained `<script>` at the bottom. The knobs are all
near the top of it:

- **A** — `spawn()` defaults (gravity/speed/spread), `MAX` grain cap, the
  `eruption()` count (190), ambient-drift interval (700 ms).
- **B** — dust density (`(w*h)/300` grains), wipe brush size (`34 * DPR`),
  resettle rate (60 grains / 400 ms).
- **C** — groove width (`5 * DPR`), wind fade (`0.03` alpha / 1.2 s ≈ 40 s
  lifetime), kick burst count (60).
- **E** — the branch origin (`originPoint()`, a fraction of the artwork box, so
  it follows whatever picture you drop in), node positions (inline `left`/`top`
  percentages, in the same 1600×900 space as the diagram), reveal timing
  (900 ms, staggered 160 ms), and ambient drift (one grain / 700 ms).

All respect `prefers-reduced-motion` (ambient loops and shakes switch
off; direct interactions still work).
