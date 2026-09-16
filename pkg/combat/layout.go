package combat

// Slot is a single fixed position on the battlefield, in canvas coordinates.
// X is the horizontal center of the sprite (feet anchor).
// Y is the ground line (feet Y). The sprite's top-left draw position is
// derived as (X - spriteW/2, Y - spriteH).
type Slot struct {
	X, Y float64
}

// ═══════════════════════════════════════════════════════════════════════════
// PvE SLOT TABLES - ZIGZAG FORMATION
// ═══════════════════════════════════════════════════════════════════════════
//
// Canvas: 1024 x 687. Canvas center X = 512.
//
// LAYOUT RULE:
//   Players stay LEFT of center, pushed toward the left edge.
//   Enemies stay RIGHT of center, pushed toward the right edge.
//   A wide empty "no man's land" gap sits at the canvas middle (X≈512)
//   between the two formations so they never visually merge.
//
// Current gap: player max X = 300, enemy min X = 730.
//   Gap = X 300..730 = 430px, centered at X=515 ≈ canvas center 512.
//
// ZIGZAG PATTERN:
//   Each side uses 4 horizontal columns. Odd rows (1,3,5...) place sprites at
//   columns 1+3; even rows (2,4,6...) place sprites at columns 2+4 (the gaps).
//   This ensures NO sprite sits directly behind another - every back-row
//   sprite peeks out through a gap in the row ahead.
//
//   Row 1 (Y=440):  col1        col3           ← front
//   Row 2 (Y=390):     col2        col4         ← offset into gaps
//   Row 3 (Y=340):  col1        col3           ← aligned with row 1, deeper
//   Row 4 (Y=290):     col2        col4         ← offset, deepest
//
//   Depth spacing = 50px between rows. With 150px-tall player sprites, the
//   back-row sprite's top 50px always clears the front-row sprite's head,
//   AND the horizontal offset means the back sprite also peeks out sideways.
//
// UI panels (PvE, bottom only - Y ≥ 455):
//   - Left  (player_state): X=-22..431,  Y=469..713
//   - Right (Options_menu): X=597..1040, Y=455..713
// Sprite feet at Y ≤ 440 (above panels). Shadows extend ~12px below feet
// (to Y≤452), still above Options_menu top (Y=455).

// PlayerSlots - 8 slots in a 4-column zigzag, all LEFT of center (X ≤ 300).
// Front row = slots 0-3 (up to 4 players). Back rows = slots 4-7.
// Slot 0 is the main player (rendered with the left UI panel).
//
// Columns: X1=80, X2=153, X3=227, X4=300
//
//	Row 1 (Y=440): X1, X3    → slots 0, 1
//	Row 2 (Y=390): X2, X4    → slots 2, 3
//	Row 3 (Y=340): X1, X3    → slots 4, 5
//	Row 4 (Y=290): X2, X4    → slots 6, 7
var PlayerSlots = []Slot{
	{80, 440},  // slot 0 - front-left
	{227, 440}, // slot 1 - front-right
	{153, 390}, // slot 2 - mid-left (in gap, slightly back)
	{300, 390}, // slot 3 - mid-right (in gap, slightly back)
	{80, 340},  // slot 4 - back-left (aligned with slot 0, deeper)
	{227, 340}, // slot 5 - back-right (aligned with slot 1, deeper)
	{153, 290}, // slot 6 - deep-left (in gap, deepest)
	{300, 290}, // slot 7 - deep-right (in gap, deepest)
}

// SummonSlots - 8 slots in two neat rows BEHIND the player formation.
// Smaller sprites (75px) so columns pack tightly.
//
// 💡 FIX 2026-09-11 (visual audit round 2): the old cluster (X=50..160, rows
// Y=418..313) sat ON TOP of the main player (slot X=80, Y=440) - one summon
// half-hid behind its master and two summons painted straight over the master's
// body (E10), while 4+ summons stacked into a pyramid (D1). New rows sit fully
// BEHIND the front-row players (Y=345/305, heads poke above the party line),
// span the party width, and stay left of the no-man's land and above the panels.
var SummonSlots = []Slot{
	{60, 345},  // slot 0 - companion upper-left of the main player
	{150, 345}, // slot 1 - behind, between P1/P2 columns
	{240, 345}, // slot 2 - behind, between P2/P3 columns
	{330, 345}, // slot 3 - behind the rightmost player
	{105, 305}, // slot 4 - deep-left
	{195, 305}, // slot 5 - deep-center-left
	{285, 305}, // slot 6 - deep-center-right
	{375, 305}, // slot 7 - deep-right
}

// EnemySlots - 8 slots in a staggered formation, all RIGHT of center (X ≥ 715).
// Slot 0 is the main enemy (front-left, closest to players).
//
// 💡 FIX 2026-09-11 (visual audit round 2): the old columns (730/787/843/900)
// packed 170px-wide mobs at a 57px pitch - every encounter of 3+ enemies merged
// into one overlapping blob with labels crossing bodies (E5/E7/E8/E11). New
// geometry spreads the formation across the full right half (pitch 115-140px),
// keeps every slot on-canvas for 210px bosses (max center X = 930 → half-width
// ≤ 95 fits), and staggers rows so back-row sprites peek through the gaps.
var EnemySlots = []Slot{
	{715, 440}, // slot 0 - front-left (closest to players)
	{835, 440}, // slot 1 - front-center
	{930, 440}, // slot 2 - front-right
	{775, 390}, // slot 3 - mid-left (in gap)
	{895, 390}, // slot 4 - mid-right (in gap)
	{715, 340}, // slot 5 - back-left
	{835, 340}, // slot 6 - back-center
	{930, 340}, // slot 7 - back-right
}

// slotFor returns the slot at index i, wrapping with a Y-depth shift for
// indices beyond the table length. The shift moves each wrap row 30px higher
// (further back) so >8 entities of one type don't collapse onto the same
// position. This is a safety net - if real matches exceed 8 of one type,
// extend the slot table instead of relying on the wrap.
func slotFor(table []Slot, i int) Slot {
	n := len(table)
	if n == 0 {
		return Slot{0, 0}
	}
	slot := table[i%n]
	wrap := i / n
	if wrap > 0 {
		slot.Y -= float64(wrap) * 30
	}
	return slot
}

// ═══════════════════════════════════════════════════════════════════════════
// PvE SPRITE HEIGHT CONSTANTS (post-crop)
// ═══════════════════════════════════════════════════════════════════════════
//
// All sprites are normalized to a FIXED HEIGHT (not width): crop transparent
// padding first, then resize so the visible content's height equals the
// target. Width floats by source aspect ratio.
//
// Per-type heights:
//   - PvE player:  150px
//   - PvE summon:   75px  (~half player height, clearly smaller)
//   - PvE enemy:   170px  (larger than players for visual weight)
//   - PvE boss:    210px
//   - PvP 1v1 human:   150px
//   - PvP 1v1 summon:  130px
//   - PvP summon duel: 180px  (unchanged - already correct)
const (
	pvePlayerSpriteH = 150
	pveSummonSpriteH = 75
	pveEnemySpriteH  = 170
	pveBossSpriteH   = 210
	pvpHumanSpriteH  = 150
	pvpSummonSpriteH = 130 // PvP 1v1 when a player is a summon (NOT the duel path)
)
