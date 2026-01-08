package models

type ArcturusItemsBase struct {
	ID                    int     `gorm:"primaryKey;column:id"`
	SpriteID              int     `gorm:"column:sprite_id;default:0"`
	ItemName              string  `gorm:"column:item_name;type:varchar(70);default:0"`
	PublicName            string  `gorm:"column:public_name;type:varchar(56);default:0"`
	Width                 int     `gorm:"column:width;default:1"`
	Length                int     `gorm:"column:length;default:1"`
	StackHeight           float64 `gorm:"column:stack_height;type:double(4,2);default:0.00"`
	AllowStack            bool    `gorm:"column:allow_stack;type:tinyint(1);default:1"`
	AllowSit              bool    `gorm:"column:allow_sit;type:tinyint(1);default:0"`
	AllowLay              bool    `gorm:"column:allow_lay;type:tinyint(1);default:0"`
	AllowWalk             bool    `gorm:"column:allow_walk;type:tinyint(1);default:0"`
	AllowGift             bool    `gorm:"column:allow_gift;type:tinyint(1);default:1"`
	AllowTrade            bool    `gorm:"column:allow_trade;type:tinyint(1);default:1"`
	AllowRecycle          bool    `gorm:"column:allow_recycle;type:tinyint(1);default:0"`
	AllowMarketplaceSell  bool    `gorm:"column:allow_marketplace_sell;type:tinyint(1);default:0"`
	AllowInventoryStack   bool    `gorm:"column:allow_inventory_stack;type:tinyint(1);default:1"`
	Type                  string  `gorm:"column:type;type:varchar(3);default:s"`
	InteractionType       string  `gorm:"column:interaction_type;type:varchar(500);default:default"`
	InteractionModesCount int     `gorm:"column:interaction_modes_count;default:1"`
	VendingIDs            string  `gorm:"column:vending_ids;type:varchar(255);default:0"`
	MultiHeight           string  `gorm:"column:multiheight;type:varchar(50);default:0"`
	CustomParams          *string `gorm:"column:customparams;type:varchar(25600);default:NULL"` // Nullable
	EffectIDMale          int     `gorm:"column:effect_id_male;default:0"`
	EffectIDFemale        int     `gorm:"column:effect_id_female;default:0"`
	ClothingOnWalk        *string `gorm:"column:clothing_on_walk;type:varchar(255);default:NULL"`
	PageID                *string `gorm:"column:page_id;type:varchar(250);default:NULL"`        // Legacy
	Rare                  string  `gorm:"column:rare;type:enum('0','1','2','3','4');default:0"` // Enum '0'-'4' stored as string often but lets use string
}

func (ArcturusItemsBase) TableName() string {
	return "items_base"
}
