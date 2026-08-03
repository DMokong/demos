# Dustbox — landing page prototypes

Three self-contained prototypes for **Dustbox**, Dustin's personal sandbox site
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
| `prototype-e-living-sandbox.html` | **The Living Sandbox** — one continuous animated scene: the page IS the inside of the box | A red wooden rim frames the whole viewport, the sand surface is the page (Prototype C's draw/kick interactions included). Calvin's cutout kneels bottom-left gently bobbing while looping stick-doodles draw themselves beside him (grains trickling off the stroke); Hobbes' cutout leaps in over the rim every ~16 s and lands in a full eruption — SPLOPP, screen shake, and a crater that the wind slowly fades |

All three share the same content skeleton (hero → six category cards → "latest"
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
| `assets/cutout-calvin.png` | E (kneeling bottom-left, drawing) | **Transparent** PNG — crop Calvin kneeling from panel 1 of the strip and remove the background (any background-remover tool) |
| `assets/cutout-hobbes.png` | E (the recurring cannonball) | **Transparent** PNG — crop mid-leap Hobbes from panel 2 |
| `assets/cutout-duo.png` | E (sitting together near the footer) | **Transparent** PNG — crop the two of them from panel 4. Optional |

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

All three respect `prefers-reduced-motion` (ambient loops and shakes switch
off; direct interactions still work).
