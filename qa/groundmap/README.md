# Ground Map — per-background standable ground (2026-09-20)

Vision-measured ground mapping so combat sprites stand ON the ground of every
battle background (no floating over lava/sky, no sinking into panel zones).

## What ships

| File | Purpose |
|---|---|
| `pkg/combat/groundmap.json` | Canonical data: per-background measured `groundStartY` (player/enemy zones, canvas space 1024x687) + recomputed depth ladders `playerRows` / `enemyRows` / `summonRows` + `pvpFeetY`. Embedded via `go:embed`. |
| `pkg/combat/groundmap.go` | Loader + `groundAdjustedY(bg, side, legacyY)` (remaps the legacy 440/390/340/290 ladder onto the background's rows; linear extrapolation for wrapped rows) + `groundPvpFeetY(bg)`. Falls back to legacy behaviour on any missing entry. |
| `pkg/combat/groundmap_test.go` | 8 unit tests: cave/beach grounding, summon remap, wrap extrapolation, fallback, full-coverage + shadow-anchored validation of every entry. |
| `annotated/*.png` | Each background in canvas space with green ground-start line + 150/170/210px sprite silhouettes standing at the mapped slots (vision verification overlays). |
| `renders/*.png` | Real `/api/combat` renders through the mapped pipeline (PvE 4v5+2 summons on the problem backgrounds + PvP on spark_15). |

## How it was measured

1. Every background was composited into canvas space exactly as the renderer
   does (`imaging.Fill` cover+center-crop to 1024x687, then the 31% black
   overlay) — measurements are taken on what is actually painted.
2. Vision pass per background: ground/horizon line, standable floor edges at
   the slot columns (player X 80..300, enemy X 715..930), foreground
   obstructions (foliage on night beach, obelisk in env1, colonnades in
   spark_15).
3. Row ladders: front row = PvE cap 440 (panel top 455 minus 12px shadow),
   deeper rows descend by `min(50, (440 - groundStart)/3)` px, floored at 4px
   so every row stays on standable ground / shadow-anchored on shallow-floor
   stages (env caves).

## Deployment (manual, Box 2)

```
cd ~/bot_generation && export PATH=/usr/local/go/bin:$PATH
go build -o bot-generation . && pm2 restart bot-generation-go --update-env
```

Deploy is safe/half-deployable: if `groundmap.json` fails to load the service
logs a warning and behaves exactly like before (legacy slot tables).
