# Dustbox assets

Drop your Calvin & Hobbes graphics here — the prototypes pick them up by
filename and hide the slot if a file is missing:

- `calvin-sandbox.png` — the small "Calvin at the red sandbox" image
  (white background is fine; the pages blend it away). Used by prototypes A and C.
- `calvin-sill.png` — any transparent-background PNG to sit on prototype B's
  windowsill (B is dark, so it needs real transparency, not a white box).
- `calvin-hobbes-strip.png` — the full 2×2 four-panel strip (Calvin drawing
  the diagram / Hobbes' SPLOPP / the sand tornado / the aftermath) as one
  image. Prototype D crops the quadrants out of it and blends them into the
  page; until the file exists, D shows dashed drop-slots where each scene goes.

Prototype E ("The Living Sandbox") animates individual character cutouts.
These need **transparent** backgrounds — crop the pose from the strip, then
run it through any background-remover:

- `cutout-calvin.png` — Calvin kneeling/drawing (panel 1). He kneels
  bottom-left, bobbing, "drawing" the looping doodles beside him.
- `cutout-hobbes.png` — Hobbes mid-leap (panel 2). He flies in over the
  rim every ~16 seconds and lands in an eruption.
- `cutout-duo.png` — the two of them sitting (panel 4), shown near the
  footer. Optional.

Nothing in this folder is required — every page renders complete without them
(E shows a small note pointing at this folder until its cutouts exist).
