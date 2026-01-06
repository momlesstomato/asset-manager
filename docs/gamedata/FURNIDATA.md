# FurniData Parameter Documentation

## Overview
This document details the structure of `FurniData`, which serves as the definitive dictionary for all furniture items (Floor and Wall). It maps visual assets (class names) to their logical behaviors and catalog definitions.

## FurnitureData Structure (`gamedata/FurnitureData.json`)

```json
{
  "roomitemtypes": {
    "furnitype": [
      {
        "id": "integer",
        "classname": "string",
        "revision": "integer",
        "category": "string",
        "defaultdir": "integer",
        "xdim": "integer",
        "ydim": "integer",
        "partcolors": {
          "color": ["string"]
        },
        "name": "string",
        "description": "string",
        "adurl": "string",
        "offerid": "integer",
        "buyout": "boolean",
        "rentofferid": "integer",
        "rentbuyout": "boolean",
        "bc": "boolean",
        "excludeddynamic": "boolean",
        "customparams": "string",
        "specialtype": "integer",
        "canstandon": "boolean",
        "cansiton": "boolean",
        "canlayon": "boolean",
        "furniline": "string",
        "environment": "string",
        "rare": "boolean"
      }
    ]
  },
  "wallitemtypes": {
    "furnitype": [
      {
        "id": "integer",
        "classname": "string",
        "revision": "integer",
        "category": "string",
        "name": "string",
        "description": "string",
        "adurl": "string",
        "offerid": "integer",
        "buyout": "boolean",
        "rentofferid": "integer",
        "rentbuyout": "boolean",
        "bc": "boolean",
        "excludeddynamic": "boolean",
        "customparams": "string",
        "specialtype": "integer",
        "furniline": "string",
        "environment": "string",
        "rare": "boolean",

        "defaultdir?": "integer",
        "xdim?": "integer",
        "ydim?": "integer",
        "partcolors?": {
          "color": ["string"]
        },
        "canstandon?": "boolean",
        "cansiton?": "boolean",
        "canlayon?": "boolean"
      }
    ]
  }
}
```

## 1. Common Parameters (Floor & Wall)
| Parameter | Type | Description | Usage & Behavior |
| :--- | :--- | :--- | :--- |
| id | `integer` | Unique identifier. | The unique key used for looking up item definitions in memory. |
| `classname` | `string` | The resource identifier. | **CRITICAL:** This string determines which asset file is loaded. <br> • *Format:* `base_name` or `base_name*color_id`. <br> • If a `*` is present (e.g., `chair_wood*1`), the renderer parses the suffix as the `colorIndex` (variable) for that item. |
| name | `string` | Display name. | Used for UI text, such as tooltips or inventory names. |
| description | `string` | Description text. | Used for UI text, providing details about the item. |
| revision | `integer` | Asset version number. | Used for cache busting and verifying if the asset needs to be re-downloaded. |
| category | `string` | Classification. | General classification tag (e.g., `furniture`, `wall`, `poster`). |
| offerid | `integer` | Catalog Offer ID (Purchase). | Links the furniture definition to a specific "Purchase" offer in the Catalog. |
| buyout | `boolean` | Purchase is buyout? | Defines if the purchase offer represents a buyout transaction. |
| rentofferid | `integer` | Catalog Offer ID (Rent). | Links the furniture definition to a specific "Rent" offer in the Catalog. |
| rentbuyout | `boolean` | Rent is buyout? | Defines if the rent offer represents a buyout transaction. |
| bc | `boolean` | Builders Club. | Boolean flag indicating if this item follows standard Builders Club availability rules. |
| customparams| `string` | Custom parameter data. | A generic string field used for specialized logic or configurations (e.g., state counts, multi-state definitions). |
| specialtype | `integer` | Logic identifier. | Numerical ID used to trigger hard-coded logic behavior (e.g., 1 for Wallpapers, 2 for Floors, 3 for Landscapes). |
| excludeddynamic| `boolean`| Exclude dynamic updates?| If `true`, the object is excluded from certain dynamic state update cycles (optimizations). |
| furniline | `string` | Furniture Line. | Identifier for grouping items into sets or campaigns (e.g., "val_14", "kitchen"). |
| environment | `string` | Environment info. | Additional environmental classification data. |
| rare | `boolean` | Is Rare? | Flag indicating if the item is considered a "Rare," often used for special UI badges or sorting priority. |
| adurl | `string` | Advertising URL. | If populated, interacting with the furniture may trigger opening this URL in a browser. |
---
## 2. Floor Item Specifics (`roomitemtypes`)
| Parameter | Type | Description | Usage & Behavior |
| :--- | :--- | :--- | :--- |
| `xdim` | `integer` | Width in tiles. | Determines the physical width of the object on the room grid. |
| `ydim` | `integer` | Depth in tiles. | Determines the physical depth of the object on the room grid. |
| `partcolors` | object | Recolor definitions. | **Parsing Logic:** <br> • Reads a list of hex strings (e.g., `#FFFFFF`). <br> • Converts them to integers for runtime tinting of specific sprite layers. |
| `canstandon` | `boolean` | Walkable? | Determines if an avatar can walk onto the tile occupied by this item. |
| `cansiton` | `boolean` | Sittable? | Determines if an avatar can sit on this item. Triggers the Sit action. |
| `canlayon` | `boolean` | Layable? | Determines if an avatar can lay on this item. Triggers the Lay action. |
| `defaultdir` | `integer` | Default rotation. | *Note:* Often available in the data structure but frequently ignored by renderers. |
---
## 3. Wall Item Specifics (`wallitemtypes`)
| Parameter | Type | Description | Usage & Behavior |
| :--- | :--- | :--- | :--- |
| `xdim` | `integer` | Width. | *Ignored:* Wall items generally do not use grid dimensions in the same way floor items do. |
| `ydim` | `integer` | Height. | *Ignored:* Wall items generally do not use grid dimensions in the same way floor items do. |
| `defaultdir` | `integer` | Default direction. | *Ignored:* Wall item direction is usually determined by wall placement logic. |
## Important Parsing Logic
- **Color Index Extraction**: The system parses the `classname` string.
  - **Input**: `furniture_basic_chair*4`
  - **Result**:
    - `Class Name` = `furniture_basic_chair`
    - `Color Index` = `4`
  - This mechanism allows a single graphical asset (SWF/Nitro file) to be reused for multiple unique entries in `FurniData`, simply by applying different default variable states.

## 4. Field Details: FurniLine

**Vinculation:** Properly linked and mapped to the `furniLine` property on the `IFurnitureData` interface.

**Usage:**
- **Primary Function:** Used extensively in **Catalog Search** features.
- **Search Filtering:** The file `src/components/catalog/views/page/common/CatalogSearchView.tsx` uses it to group or find furniture items when searching.
- **Builders Club:** Acts as a grouping identifier to organize items in Builders Club pages so that items from the same product line (e.g., "kitchen", "scifi") appear together.

## 5. Field Details: SpecialType

**Vinculation:** Linked to the `specialType` property on `IFurnitureData`.

**Usage:**
- Critical for defining functional logic and context menus for specific interactive items.
- Maps to the `FurniCategory` class constants.

**Category Key Mappings:**
| Category | Constant Name | ID | Description |
| :--- | :--- | :--- | :--- |
| **Pet Interactions** | `PET_SHAMPOO` | 13 | Used for washing pets. |
| | `PET_SADDLE` | 16 | Used for riding horses/pets. |
| **Monster Plants** | `MONSTERPLANT_SEED` | 19 | Transformation or planting seed. |
| | `REVIVAL` | 20 | Reviving a monster plant. |
| | `REBREED` | 21 | Rebreeding logic. |
| | `FERTILIZE` | 22 | Fertilizing logic. |
| **Other** | `FIGURE_PURCHASABLE_SET` | 23 | Clothing furniture entities. |