package reconcile

// ServerProfile defines emulator-specific database schema mappings.
type ServerProfile struct {
	// TableName is the name of the furniture table in the database.
	TableName string

	// Columns maps logical field names to actual database column names.
	Columns map[string]string
}

// Column name constants for logical field references.
const (
	ColID          = "id"
	ColSpriteID    = "sprite_id"
	ColItemName    = "item_name"
	ColPublicName  = "public_name"
	ColWidth       = "width"
	ColLength      = "length"
	ColStackHeight = "stack_height"
	ColCanStack    = "can_stack"
	ColCanSit      = "can_sit"
	ColCanWalk     = "can_walk"
	ColCanLay      = "can_lay"
	ColType        = "type"
	ColInteraction = "interaction_type"
	ColIsRare      = "is_rare"
)

// ArcturusProfile returns the server profile for Arcturus Morningstar emulator.
func ArcturusProfile() ServerProfile {
	return ServerProfile{
		TableName: "items_base",
		Columns: map[string]string{
			ColID:          "id",
			ColSpriteID:    "sprite_id",
			ColItemName:    "item_name",
			ColPublicName:  "public_name",
			ColWidth:       "width",
			ColLength:      "length",
			ColStackHeight: "stack_height",
			ColCanStack:    "allow_stack",
			ColCanSit:      "allow_sit",
			ColCanWalk:     "allow_walk",
			ColCanLay:      "allow_lay",
			ColType:        "type",
			ColInteraction: "interaction_type",
		},
	}
}

// CometProfile returns the server profile for Comet emulator.
func CometProfile() ServerProfile {
	return ServerProfile{
		TableName: "furniture",
		Columns: map[string]string{
			ColID:          "id",
			ColSpriteID:    "sprite_id",
			ColItemName:    "item_name",
			ColPublicName:  "public_name",
			ColWidth:       "width",
			ColLength:      "length",
			ColStackHeight: "stack_height",
			ColCanStack:    "can_stack",
			ColCanSit:      "can_sit",
			ColCanWalk:     "is_walkable",
			ColCanLay:      "can_lay",
			ColType:        "type",
			ColInteraction: "interaction_type",
		},
	}
}

// PlusProfile returns the server profile for Plus emulator.
func PlusProfile() ServerProfile {
	return ServerProfile{
		TableName: "furniture",
		Columns: map[string]string{
			ColID:          "id",
			ColSpriteID:    "sprite_id",
			ColItemName:    "item_name",
			ColPublicName:  "public_name",
			ColWidth:       "width",
			ColLength:      "length",
			ColStackHeight: "stack_height",
			ColCanStack:    "can_stack",
			ColCanSit:      "can_sit",
			ColCanWalk:     "is_walkable",
			ColType:        "type",
			ColInteraction: "interaction_type",
			ColIsRare:      "is_rare",
		},
	}
}

// GetProfileByName returns the appropriate server profile for a given emulator name.
func GetProfileByName(emulator string) ServerProfile {
	switch emulator {
	case "arcturus":
		return ArcturusProfile()
	case "comet":
		return CometProfile()
	case "plus":
		return PlusProfile()
	default:
		// Default to Arcturus
		return ArcturusProfile()
	}
}
