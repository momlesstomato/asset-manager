package models

import (
	"strings"
)

// Report contains the results of a furniture integrity check.
type Report struct {
	TotalExpected       int      `json:"total_expected"`
	TotalFound          int      `json:"total_found"`
	MissingAssets       []string `json:"missing_assets"`
	UnregisteredAssets  []string `json:"unregistered_assets"`
	MalformedAssets     []string `json:"malformed_assets"`
	ParameterMismatches []string `json:"parameter_mismatches,omitempty"`
	GeneratedAt         string   `json:"generated_at"`
	ExecutionTime       string   `json:"execution_time"`
}

// FurnitureDetailReport contains the detailed integrity check for a single item.
type FurnitureDetailReport struct {
	ID              int      `json:"id"`
	ClassName       string   `json:"class_name"`
	Name            string   `json:"name"`
	NitroFile       string   `json:"nitro_file,omitempty"`
	FileExists      bool     `json:"file_exists"`
	InFurniData     bool     `json:"in_furnidata"`
	InDB            bool     `json:"in_db"`
	IntegrityStatus string   `json:"integrity_status"` // "PASS", "FAIL", "WARNING"
	Mismatches      []string `json:"mismatches,omitempty"`
}

// FurnitureData represents the structure of FurniData.json
type FurnitureData struct {
	RoomItemTypes struct {
		FurniType []FurnitureItem `json:"furnitype"`
	} `json:"roomitemtypes"`
	WallItemTypes struct {
		FurniType []FurnitureItem `json:"furnitype"`
	} `json:"wallitemtypes"`
}

// FurnitureItem represents a single furniture definition matching FURNIDATA.md
type FurnitureItem struct {
	// Common Parameters
	ID              int    `json:"id"`
	ClassName       string `json:"classname"`
	Revision        int    `json:"revision"`
	Category        string `json:"category"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	AdURL           string `json:"adurl,omitempty"`
	OfferID         int    `json:"offerid,omitempty"`
	Buyout          bool   `json:"buyout,omitempty"`
	RentOfferID     int    `json:"rentofferid,omitempty"`
	RentBuyout      bool   `json:"rentbuyout,omitempty"`
	BC              bool   `json:"bc,omitempty"`
	ExcludedDynamic bool   `json:"excludeddynamic,omitempty"`
	CustomParams    string `json:"customparams,omitempty"`
	SpecialType     int    `json:"specialtype,omitempty"`
	FurniLine       string `json:"furniline,omitempty"`
	Environment     string `json:"environment,omitempty"`
	Rare            bool   `json:"rare,omitempty"`

	// Floor Item Specifics
	DefaultDir int `json:"defaultdir,omitempty"`
	XDim       int `json:"xdim,omitempty"`
	YDim       int `json:"ydim,omitempty"`
	PartColors struct {
		Color []string `json:"color"`
	} `json:"partcolors,omitempty"`
	CanStandOn bool `json:"canstandon,omitempty"`
	CanSitOn   bool `json:"cansiton,omitempty"`
	CanLayOn   bool `json:"canlayon,omitempty"`
}

// Validate checks if the furniture item has the minimum required fields and valid formats.
func (i FurnitureItem) Validate() string {
	if i.ID == 0 {
		return "missing id"
	}
	if i.ClassName == "" {
		return "missing classname"
	}
	if i.Name == "" {
		return "missing name"
	}
	if i.Revision == 0 {
		// Revision 0 is technically possible but usually it starts at 1?
		// Docs say "Asset version number". If missing, what happens?
		// Let's assume it should be present. But maybe 0 is valid.
		// However, missing in JSON implies 0 in int.
		// Let's warn if 0? Or just "missing revision" if we treat 0 as default/missing.
		// Many ancient assets might not have it tailored?
		// Safest is to not enforce Revision != 0 unless strictly known.
		// But let's enforce provided fields.
	}
	if i.Category == "" {
		// Category seems important for classification.
		// "General classification tag".
		return "missing category"
	}

	// Validate ClassName format
	// Format: base_name or base_name*color_id
	if strings.Contains(i.ClassName, "*") {
		parts := strings.Split(i.ClassName, "*")
		if len(parts) != 2 {
			return "invalid classname format: too many asterisks"
		}
		if parts[0] == "" {
			return "invalid classname format: empty base name"
		}
		if parts[1] == "" {
			return "invalid classname format: empty color index"
		}
		// Check if color index is numeric?
		// "parses the suffix as the colorIndex (variable)"
		// Typically numeric, but effectively a string in the name?
		// Docs say "color_id". Example "chair_wood*1".
		// Let's not be too strict on the ID content unless we know it must be int.
	}

	return ""
}
