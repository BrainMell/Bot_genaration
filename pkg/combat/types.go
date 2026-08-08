package combat

// Player represents a player in the combat scene
type Player struct {
        Name           string `json:"name"`
        Class          string `json:"class"` // e.g. "FIGHTER"
        Level          int    `json:"level"`
        HP             int    `json:"hp"`        // Current HP
        MaxHP          int    `json:"maxHp"`
        CurrentHP      int    `json:"currentHP"` // Redundant but often sent
        Energy         int    `json:"energy"`
        MaxEnergy      int    `json:"maxEnergy"`
        AdventurerRank string `json:"adventurerRank"`
        SpriteIndex    int    `json:"spriteIndex"`
        Mode           string `json:"mode,omitempty"`    // 💡 NEW: "summon" when player is a summon
        Species        string `json:"species,omitempty"` // 💡 NEW: summon species (e.g. "wolf", "bat")
}

// Enemy represents an enemy in the combat scene
type Enemy struct {
        Name        string `json:"name"`
        CurrentHP   int    `json:"currentHP"`
        MaxHP       int    `json:"maxHp"`
        IsBoss      bool   `json:"isBoss"`
        JustDied    bool   `json:"justDied"`
        SpriteIndex int    `json:"spriteIndex"`
}

// Summon represents a summoned ally in the combat scene.
// 💡 Summoner System (Phase 7): summons are rendered between enemies
// and the UI overlay, positioned on the player's side of the battlefield.
// Species maps to a sprite via SummonSprites in sprites.go.
type Summon struct {
        Name         string `json:"name"`
        Species      string `json:"species"`      // e.g. "skeleton", "flame_elemental"
        CurrentHP    int    `json:"currentHP"`
        MaxHP        int    `json:"maxHp"`
        JustDied     bool   `json:"justDied"`
        OwnerIndex   int    `json:"ownerIndex"`   // which player summoned it (for positioning)
        IsStationary bool   `json:"isStationary"` // turrets don't move
}

// Action describes a single combat action to animate.
// Sent in CombatRequest.Action so the renderer can produce a multi-frame
// animation showing the attacker winding up, the VFX playing, the target
// reacting (shake + red flash), and HP bars draining.
type Action struct {
        AttackerSide  string `json:"attackerSide"`  // "player" | "enemy" | "summon"
        AttackerIndex int    `json:"attackerIndex"` // index into the respective array
        TargetSide    string `json:"targetSide"`    // "player" | "enemy" | "summon"
        TargetIndex   int    `json:"targetIndex"`
        SkillName     string `json:"skillName"`     // e.g. "Fireball", "Heavy Strike"
        Element       string `json:"element"`       // "fire" | "ice" | "lightning" | "dark" | "holy" | "physical" | "none"
        Vfx           string `json:"vfx"`           // explicit VFX folder name override (optional)
        Damage        int    `json:"damage"`        // damage dealt (0 if healing/buff)
        IsCrit        bool   `json:"isCrit"`
        Missed        bool   `json:"missed"`
        Heal          int    `json:"heal"`          // > 0 if healing instead of damage
}

// CombatRequest is the payload sent from Node.js.
// Action is optional — when present, the renderer can produce an animated
// MP4 instead of a static PNG. When absent, behavior is unchanged.
type CombatRequest struct {
        Players    []Player `json:"players"`
        Enemies    []Enemy  `json:"enemies"`
        Summons    []Summon `json:"summons"`
        CombatType string   `json:"combatType"`
        Rank       string   `json:"rank"`
        Background string   `json:"background"`
        Action     *Action  `json:"action,omitempty"` // optional — drives the animation
        Floor      int      `json:"floor,omitempty"`  // 💡 Abyss floor number — when >0, banner shows "FLOOR N" instead of rank
}
