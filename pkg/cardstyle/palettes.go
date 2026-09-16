package cardstyle

// palettes.go - the nine rebuilt design systems' color identities.
// Style 7 (Royal Decree) is intentionally absent: baked art, no kit.

func Stonekeep() Palette {
	return Palette{
		ID: 1, Name: "Stonekeep",
		Bg: Hex(0x4a4c52), Bg2: Hex(0x33353b), Vign: 70,
		Panel: Hex(0xa8aaae), PanelEd: Hex(0x232529),
		Ink: Hex(0x1a1c20), Muted: Hex(0x4a4e56),
		Accent: Hex(0x2c2e34), Accent2: Hex(0xffb340),
		Track: Hex(0x3a3d45), Fill: Hex(0x9aa2b0), Done: Hex(0x6eaa78),
		Seal: Hex(0x2c2e34), SealTx: Hex(0xd4dae4), Caption: Hex(0x4a4e56),
	}
}

func GoldenArcanum() Palette {
	return Palette{
		ID: 2, Name: "Golden Arcanum",
		Bg: Hex(0x171232), Bg2: Hex(0x241c46), Vign: 80,
		Panel: HexA(0x282146, 242), PanelEd: Hex(0xd4af37),
		Ink: Hex(0xf0e6be), Muted: Hex(0xbaac84),
		Accent: Hex(0xd4af37), Accent2: Hex(0x8cc8a0),
		Track: Hex(0x141028), Fill: Hex(0xd4af37), Done: Hex(0x8cc8a0),
		Seal: Hex(0x78581c), SealTx: Hex(0xf4dc8c), Caption: Hex(0xbaac84),
	}
}

func RetroCourt() Palette {
	return Palette{
		ID: 3, Name: "Retro Court",
		Bg: Hex(0xdeceaa), Bg2: Hex(0xe8dcbc), Vign: 34,
		Panel: Hex(0xf0e4c4), PanelEd: Hex(0x7a2c3a),
		Ink: Hex(0x3a2c20), Muted: Hex(0x705840),
		Accent: Hex(0x7a2c3a), Accent2: Hex(0x466e50),
		Track: Hex(0x5a4836), Fill: Hex(0x7a2c3a), Done: Hex(0x466e50),
		Seal: Hex(0x7a2c3a), SealTx: Hex(0xf4e2c4), Caption: Hex(0x705840),
	}
}

func Woodmere() Palette {
	return Palette{
		ID: 4, Name: "Woodmere",
		Bg: Hex(0x583a22), Bg2: Hex(0x422a18), Vign: 76,
		Panel: Hex(0xe2c08e), PanelEd: Hex(0x3c2814),
		Ink: Hex(0x34220f), Muted: Hex(0x74522e),
		Accent: Hex(0xd68a2e), Accent2: Hex(0xf0b45e),
		Track: Hex(0x462e18), Fill: Hex(0xd68a2e), Done: Hex(0x78b452),
		Seal: Hex(0x6e4622), SealTx: Hex(0xf4d694), Caption: Hex(0x74522e),
	}
}

func EmblemNoir() Palette {
	return Palette{
		ID: 5, Name: "Emblem Noir",
		Bg: Hex(0x0e0e10), Bg2: Hex(0x18181c), Vign: 90,
		Panel: Hex(0x1e1e22), PanelEd: Hex(0xc6a664),
		Ink: Hex(0xeeece6), Muted: Hex(0x9e9a92),
		Accent: Hex(0xc6a664), Accent2: Hex(0x94202c),
		Track: Hex(0x101012), Fill: Hex(0xc6a664), Done: Hex(0x6eaa78),
		Seal: Hex(0x94202c), SealTx: Hex(0xf4d68c), Caption: Hex(0x9e9a92),
	}
}

// SoulForge - Style 6. Owner-provided Allocate card defines this identity:
// deep indigo night sky, antique gold hairlines, glowing cyan numerals.
func SoulForge() Palette {
	return Palette{
		ID: 6, Name: "Soul Forge",
		Bg: Hex(0x1b1b30), Bg2: Hex(0x232345), Vign: 60,
		Panel: HexA(0x20203a, 235), PanelEd: Hex(0xc9a961),
		Ink: Hex(0xece4cc), Muted: Hex(0xa89f8a),
		Accent: Hex(0xc9a961), Accent2: Hex(0x7fd7d0),
		Track: Hex(0x17172a), Fill: Hex(0x7fd7d0), Done: Hex(0xc9a961),
		Seal: Hex(0x2a2a4a), SealTx: Hex(0x7fd7d0), Caption: Hex(0xc9a961),
	}
}

func NeonArcade() Palette {
	return Palette{
		ID: 8, Name: "Neon Arcade",
		Bg: Hex(0x0a0c1c), Bg2: Hex(0x121630), Vign: 80,
		Panel: HexA(0x10142c, 235), PanelEd: Hex(0x40e0ff),
		Ink: Hex(0xe0f2ff), Muted: Hex(0x80a8d0),
		Accent: Hex(0xff40a0), Accent2: Hex(0x40e0ff),
		Track: HexA(0x40e0ff, 36), Fill: Hex(0xff40a0), Done: Hex(0x40e0a0),
		Seal: HexA(0xff40a0, 230), SealTx: Hex(0xfff0fa), Caption: Hex(0x80a8d0),
	}
}

func RuneMonolith() Palette {
	return Palette{
		ID: 9, Name: "Rune Monolith",
		Bg: Hex(0x2c322e), Bg2: Hex(0x1e2220), Vign: 80,
		Panel: Hex(0x424a44), PanelEd: Hex(0xe88c40),
		Ink: Hex(0xe0e6d6), Muted: Hex(0x9ca698),
		Accent: Hex(0xe88c40), Accent2: Hex(0xffbe78),
		Track: Hex(0x161815), Fill: Hex(0xe88c40), Done: Hex(0x82be6e),
		Seal: Hex(0xb44618), SealTx: Hex(0xfad696), Caption: Hex(0x9ca698),
	}
}

func CrimsonCourt() Palette {
	return Palette{
		ID: 10, Name: "Crimson Court",
		Bg: Hex(0x420c16), Bg2: Hex(0x2c0810), Vign: 90,
		Panel: HexA(0x601622, 242), PanelEd: Hex(0xd4a856),
		Ink: Hex(0xf4e2ce), Muted: Hex(0xc69c92),
		Accent: Hex(0xd4a856), Accent2: Hex(0xe8dcC8),
		Track: Hex(0x38060c), Fill: Hex(0xd4a856), Done: Hex(0x82be6e),
		Seal: Hex(0x8c1824), SealTx: Hex(0xf4d696), Caption: Hex(0xc69c92),
	}
}

// ForID returns the kit palette for a style id (0 / 7 / unknown = nil).
func ForID(id int) *Palette {
	switch id {
	case 1:
		p := Stonekeep()
		return &p
	case 2:
		p := GoldenArcanum()
		return &p
	case 3:
		p := RetroCourt()
		return &p
	case 4:
		p := Woodmere()
		return &p
	case 5:
		p := EmblemNoir()
		return &p
	case 6:
		p := SoulForge()
		return &p
	case 8:
		p := NeonArcade()
		return &p
	case 9:
		p := RuneMonolith()
		return &p
	case 10:
		p := CrimsonCourt()
		return &p
	}
	return nil
}
