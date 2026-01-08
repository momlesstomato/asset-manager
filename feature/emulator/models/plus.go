package models

type PlusFurniture struct {
	ID                    int     `gorm:"primaryKey;column:id"`
	SpriteID              int     `gorm:"column:sprite_id"`                   // No default specified in doc, implied
	ItemName              string  `gorm:"column:item_name;type:varchar(100)"` // Length guessed, typical
	PublicName            string  `gorm:"column:public_name;type:varchar(100)"`
	Type                  string  `gorm:"column:type;type:enum('s','i')"`
	Width                 int     `gorm:"column:width"`
	Length                int     `gorm:"column:length"`
	StackHeight           float64 `gorm:"column:stack_height;type:double"`
	CanStack              bool    `gorm:"column:can_stack;type:tinyint(1)"`
	IsWalkable            bool    `gorm:"column:is_walkable;type:tinyint(1)"`
	CanSit                bool    `gorm:"column:can_sit;type:tinyint(1)"`
	AllowRecycle          bool    `gorm:"column:allow_recycle;type:tinyint(1)"`
	AllowTrade            bool    `gorm:"column:allow_trade;type:tinyint(1)"`
	AllowMarketplaceSell  bool    `gorm:"column:allow_marketplace_sell;type:tinyint(1)"`
	AllowGift             bool    `gorm:"column:allow_gift;type:tinyint(1)"`
	AllowInventoryStack   bool    `gorm:"column:allow_inventory_stack;type:tinyint(1)"`
	InteractionType       string  `gorm:"column:interaction_type;type:varchar(100)"`
	InteractionModesCount int     `gorm:"column:interaction_modes_count"`
	VendingIDs            string  `gorm:"column:vending_ids;type:varchar(255)"`
	HeightAdjustable      string  `gorm:"column:height_adjustable;type:varchar(100)"`
	EffectID              int     `gorm:"column:effect_id"`
	IsRare                bool    `gorm:"column:is_rare;type:tinyint(1)"`
	ExtraRot              bool    `gorm:"column:extra_rot;type:tinyint(1)"`
}

func (PlusFurniture) TableName() string {
	return "furniture"
}
