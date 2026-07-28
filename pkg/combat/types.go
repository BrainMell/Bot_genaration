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
        Name        string `json:"name"`
        Species     string `json:"species"`     // e.g. "skeleton", "flame_elemental"
        CurrentHP   int    `json:"currentHP"`
        MaxHP       int    `json:"maxHp"`
        JustDied    bool   `json:"justDied"`
        OwnerIndex  int    `json:"ownerIndex"`  // which player summoned it (for positioning)
        IsStationary bool  `json:"isStationary"` // turrets don't move
}

// CombatRequest is the payload sent from Node.js
type CombatRequest struct {
        Players    []Player `json:"players"`
        Enemies    []Enemy  `json:"enemies"`
        Summons    []Summon `json:"summons"`     // 💡 Phase 7: summoned allies
        CombatType string   `json:"combatType"`  // "PVE" or "PVP"
        Rank       string   `json:"rank"`
        Background string   `json:"background"`  // Filename only
}
