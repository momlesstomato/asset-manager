package models

import (
	"strconv"
)

// DBFurnitureItem represents a normalized furniture item from the database
// used for comparison with FurniData.json.
type DBFurnitureItem struct {
	ID          int
	SpriteID    int
	ItemName    string // classname
	PublicName  string // name
	Width       int    // xdim
	Length      int    // ydim
	StackHeight float64
	CanStack    bool
	CanSit      bool
	CanWalk     bool // canstandon
	CanLay      bool
	Type        string // s or i
	Interaction string
	IsRare      bool
}

// ArcturusItemsBase represents the 'items_base' table in Arcturus.
type ArcturusItemsBase struct {
	ID            int     `gorm:"column:id;primaryKey"`
	SpriteID      int     `gorm:"column:sprite_id"`
	ItemName      string  `gorm:"column:item_name"`
	PublicName    string  `gorm:"column:public_name"`
	Width         int     `gorm:"column:width"`
	Length        int     `gorm:"column:length"`
	StackHeight   float64 `gorm:"column:stack_height"`
	AllowStack    int     `gorm:"column:allow_stack"`      // tinyint(1)
	AllowSit      int     `gorm:"column:allow_sit"`        // tinyint(1)
	AllowLay      int     `gorm:"column:allow_lay"`        // tinyint(1)
	AllowWalk     int     `gorm:"column:allow_walk"`       // tinyint(1)
	Type          string  `gorm:"column:type"`             // s, i, etc
	ExtensionType string  `gorm:"column:interaction_type"` // interaction_type
}

// TableName overrides the table name for Arcturus.
func (ArcturusItemsBase) TableName() string {
	return "items_base"
}

// ToNormalized converts the emulator specific struct to a normalized one.
func (a ArcturusItemsBase) ToNormalized() DBFurnitureItem {
	return DBFurnitureItem{
		ID:          a.ID,
		SpriteID:    a.SpriteID,
		ItemName:    a.ItemName,
		PublicName:  a.PublicName,
		Width:       a.Width,
		Length:      a.Length,
		StackHeight: a.StackHeight,
		CanStack:    a.AllowStack == 1,
		CanSit:      a.AllowSit == 1,
		CanWalk:     a.AllowWalk == 1,
		CanLay:      a.AllowLay == 1,
		Type:        a.Type,
		Interaction: a.ExtensionType,
		IsRare:      false, // Arcturus doesn't use rare column for logic usually, often legacy '0'.
	}
}

// CometFurniture represents the 'furniture' table in Comet.
type CometFurniture struct {
	ID              int    `gorm:"column:id;primaryKey"`
	SpriteID        int    `gorm:"column:sprite_id"`
	ItemName        string `gorm:"column:item_name"`
	PublicName      string `gorm:"column:public_name"`
	Width           int    `gorm:"column:width"`
	Length          int    `gorm:"column:length"`
	StackHeight     string `gorm:"column:stack_height"` // Comet uses varchar for stack_height, sometimes doubles
	CanStack        string `gorm:"column:can_stack"`    // enum('0','1')
	CanSit          string `gorm:"column:can_sit"`      // enum('0','1')
	CanLay          string `gorm:"column:can_lay"`      // enum('0','1')
	IsWalkable      string `gorm:"column:is_walkable"`  // enum('0','1')
	Type            string `gorm:"column:type"`         // s, i
	InteractionType string `gorm:"column:interaction_type"`
}

// TableName overrides the table name for Comet.
func (CometFurniture) TableName() string {
	return "furniture"
}

// ToNormalized converts Comet furniture to normalized.
func (c CometFurniture) ToNormalized() DBFurnitureItem {
	sh, _ := strconv.ParseFloat(c.StackHeight, 64)
	return DBFurnitureItem{
		ID:          c.ID,
		SpriteID:    c.SpriteID,
		ItemName:    c.ItemName,
		PublicName:  c.PublicName,
		Width:       c.Width,
		Length:      c.Length,
		StackHeight: sh,
		CanStack:    c.CanStack == "1",
		CanSit:      c.CanSit == "1",
		CanWalk:     c.IsWalkable == "1",
		CanLay:      c.CanLay == "1",
		Type:        c.Type,
		Interaction: c.InteractionType,
		IsRare:      false, // Comet doesn't seemingly have 'rare' column in basic schema provided?
	}
}

// PlusFurniture represents the 'furniture' table in Plus.
type PlusFurniture struct {
	ID              int     `gorm:"column:id;primaryKey"`
	SpriteID        int     `gorm:"column:sprite_id"`
	ItemName        string  `gorm:"column:item_name"`
	PublicName      string  `gorm:"column:public_name"`
	Width           int     `gorm:"column:width"`
	Length          int     `gorm:"column:length"`
	StackHeight     float64 `gorm:"column:stack_height"`
	CanStack        int     `gorm:"column:can_stack"`
	CanSit          int     `gorm:"column:can_sit"`
	IsWalkable      int     `gorm:"column:is_walkable"`
	Type            string  `gorm:"column:type"` // s, i
	InteractionType string  `gorm:"column:interaction_type"`
	IsRare          int     `gorm:"column:is_rare"`
}

// TableName overrides the table name for Plus.
func (PlusFurniture) TableName() string {
	return "furniture"
}

// ToNormalized converts Plus furniture to normalized.
func (p PlusFurniture) ToNormalized() DBFurnitureItem {
	return DBFurnitureItem{
		ID:          p.ID,
		SpriteID:    p.SpriteID,
		ItemName:    p.ItemName,
		PublicName:  p.PublicName,
		Width:       p.Width,
		Length:      p.Length,
		StackHeight: p.StackHeight,
		CanStack:    p.CanStack == 1,
		CanSit:      p.CanSit == 1,
		CanWalk:     p.IsWalkable == 1,
		CanLay:      false, // Plus doesn't strictly have can_lay in the table provided in docs (or it wasn't listed)
		Type:        p.Type,
		Interaction: p.InteractionType,
		IsRare:      p.IsRare == 1,
	}
}
