package models

type CometFurniture struct {
	ID                    int     `gorm:"primaryKey;column:id"`
	ItemName              string  `gorm:"column:item_name;type:varchar(255)"`
	PublicName            string  `gorm:"column:public_name;type:varchar(255)"`
	Type                  string  `gorm:"column:type;type:enum('s','i','e');default:s"`
	Width                 int     `gorm:"column:width;default:1"`
	Length                int     `gorm:"column:length;default:1"`
	StackHeight           string  `gorm:"column:stack_height;type:varchar(255);default:1"` // Comet uses varchar
	CanStack              string  `gorm:"column:can_stack;type:enum('0','1');default:1"`   // Comet uses enum for bools often
	CanSit                string  `gorm:"column:can_sit;type:enum('0','1');default:0"`
	CanLay                string  `gorm:"column:can_lay;type:enum('0','1');default:0"`
	IsWalkable            string  `gorm:"column:is_walkable;type:enum('0','1');default:0"`
	SpriteID              int     `gorm:"column:sprite_id;default:0"`
	AllowRecycle          string  `gorm:"column:allow_recycle;type:enum('0','1');default:1"`
	AllowTrade            string  `gorm:"column:allow_trade;type:enum('0','1');default:1"`
	AllowMarketplaceSell  string  `gorm:"column:allow_marketplace_sell;type:enum('0','1');default:0"`
	AllowGift             string  `gorm:"column:allow_gift;type:enum('0','1');default:1"`
	AllowInventoryStack   string  `gorm:"column:allow_inventory_stack;type:enum('0','1');default:1"`
	InteractionType       string  `gorm:"column:interaction_type;type:varchar(255);default:default"`
	InteractionModesCount int     `gorm:"column:interaction_modes_count;default:1"`
	VendingIDs            string  `gorm:"column:vending_ids;type:varchar(255);default:0"`
	EffectID              int     `gorm:"column:effect_id;default:0"`
	IsArrow               string  `gorm:"column:is_arrow;type:enum('0','1');default:0"`
	FootFigure            string  `gorm:"column:foot_figure;type:enum('0','1');default:0"`
	StackMultiplier       string  `gorm:"column:stack_multiplier;type:enum('0','1');default:0"`
	Subscriber            string  `gorm:"column:subscriber;type:enum('0','1');default:0"`
	VariableHeights       string  `gorm:"column:variable_heights;type:varchar(255);default:0"`
	FlatID                int     `gorm:"column:flat_id;default:-1"`
	Revision              int     `gorm:"column:revision;default:45554"`
	Description           string  `gorm:"column:description;type:varchar(255)"`
	SpecialType           int     `gorm:"column:specialtype;default:1"`
	CanLayOn              string  `gorm:"column:canlayon;type:enum('0','1');default:0"` // Legacy duplicate
	RequiresRights        string  `gorm:"column:requires_rights;type:enum('0','1');default:1"`
	SongID                int     `gorm:"column:song_id;default:0"`
	Colors                *string `gorm:"column:colors;type:longtext;default:NULL"`
	Deleteable            bool    `gorm:"column:deleteable;type:tinyint(1);default:1"` // Tinyint here acc to doc
}

func (CometFurniture) TableName() string {
	return "furniture"
}
