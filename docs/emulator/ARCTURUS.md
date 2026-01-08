# Arcturus Morningstar Emulator Documentation

## Overview
This document details the database schema for the **Arcturus Morningstar** emulator, specifically the `items_base` table, and its relationship with the client-side `FurniData.json` (documented in `docs/gamedata/FURNIDATA.md`).

## Table: items_base

This table defines the base properties of furniture items. It serves as the server-side counterpart to `FurniData.json`.

| Column | Type | Default | Relation to `FurniData.json` | Description |
| :--- | :--- | :--- | :--- | :--- |
| `id` | `int` | `AUTO_INCREMENT` | **No Relation** | Primary Key (Database ID). Unique identifier used internally by the database. |
| `sprite_id` | `int` | `0` | `id` | The unique identifier used by the client to identify the items definition. Maps directly to the `id` field in FurniData. |
| `item_name` | `varchar(70)` | `0` | `classname` | The technical name of the item. Determines which asset (SWF/Nitro) is loaded. Maps to `classname`. |
| `public_name` | `varchar(56)` | `0` | `name` | The display name of the item shown in the catalog/inventory (if not overridden by external texts). Maps to `name`. |
| `width` | `int` | `1` | `xdim` | The width of the item in tiles. Maps to `xdim`. |
| `length` | `int` | `1` | `ydim` | The depth of the item in tiles. Maps to `ydim`. |
| `stack_height` | `double(4,2)` | `0.00` | **No Direct Relation** | The height of the item in z-axis units. Used by the server to calculate stacking height for items placed on top. |
| `allow_stack` | `tinyint(1)` | `1` | **No Direct Relation** | Server-side flag. Determines if *other furniture* can be stacked on top of this item. |
| `allow_sit` | `tinyint(1)` | `0` | `cansiton` | Server-side flag. Determines if an avatar can sit on this item. Maps to `cansiton`. |
| `allow_lay` | `tinyint(1)` | `0` | `canlayon` | Server-side flag. Determines if an avatar can lay on this item. Maps to `canlayon`. |
| `allow_walk` | `tinyint(1)` | `0` | `canstandon` | Server-side flag. Determines if an avatar can walk onto this item. Maps to `canstandon`. |
| `allow_gift` | `tinyint(1)` | `1` | **No Relation** | Server-side flag. Determines if the item can be gifted. |
| `allow_trade` | `tinyint(1)` | `1` | **No Relation** | Server-side flag. Determines if the item can be traded between users. |
| `allow_recycle` | `tinyint(1)` | `0` | **No Relation** | Server-side flag. Determines if the item can be recycled in the Ecotron/Recycler. |
| `allow_marketplace_sell` | `tinyint(1)` | `0` | `buyout` (loose relation) | Server-side flag. Determines if the item can be sold on the Marketplace. |
| `allow_inventory_stack` | `tinyint(1)` | `1` | **No Relation** | Server-side flag. Determines if multiple items of this type stack into a single slot in the user's inventory. |
| `type` | `varchar(3)` | `s` | `type` (Structure) | Loaded into `Item.type` as `FurnitureType` enum. Categorizes item for serialization/logic. `s`=Floor, `i`=Wall, `e`=Effect, `b`=Badge, `r`=Robot, `h`=Habbo Club, `p`=Pet. Defaults to `s` if unknown. |
| `interaction_type` | `varchar(500)` | `default` | `specialtype` (loose relation) | Defines the server-side interaction logic (e.g., `gate`, `bed`, `dice`). While `specialtype` in FurniData triggers client logic, this triggers server logic. |
| `interaction_modes_count` | `int` | `1` | **No Direct Relation** | Loaded into `Item.stateCount`. Interaction cycles: `(currentState + 1) % interaction_modes_count`. If 0 or 1, item typically does not change state. |
| `vending_ids` | `varchar(255)` | `0` | **No Relation** | Comma-separated list of Item IDs (hand items) that this vending machine gives out. |
| `multiheight` | `varchar(50)` | `0` | **No Relation** | Loaded into `Item.multiHeights` as double array. Semicolon-separated doubles (e.g. `1.0;1.1`). Logic: `Height = multiHeights[current_state % multiHeights.length]`. |
| `customparams` | `varchar(25600)` | `NULL` | `customparams` | Loaded into `Item.customParams`. Sent to client for: Badges (`type`='b'), Robots (`type`='r'), Posters (`item_name`='poster'), Music Discs (`item_name` starts with 'SONG '). |
| `effect_id_male` | `int` | `0` | **No Relation** | The effect ID applied to a male avatar when using/wearing this item (if applicable). |
| `effect_id_female` | `int` | `0` | **No Relation** | The effect ID applied to a female avatar when using/wearing this item (if applicable). |
| `clothing_on_walk` | `varchar(255)` | `NULL` | **No Relation** | Specifies clothing UUID/IDs to apply when walking into this item (e.g., changing booths). |
| `page_id` | `varchar(250)` | `NULL` | **No Relation** | Not read by emulator/Item object. Legacy field for catalog linking, but Arcturus uses `catalog_items`. |
| `rare` | `enum` | `0` | **No Relation** | **Legacy Field**. Not read by emulator/Item object. Can have values '0'-'4' (speculated frontend relation), but emulator logic only uses 0/1. No effect on server-side logic (trading, rarities, etc). |

## Notes

- **Primary Key vs Sprite ID**: It is crucial to distinguish between `id` (database row ID) and `sprite_id`. The client uses `sprite_id` to link graphical assets.
- **Server vs Client Authority**: While `FurniData.json` dictates what the client *sees* (visuals, tooltips), `items_base` dictates what the server *allows* (walking, sitting, trading). Discrepancies can lead to "ghost" behaviors.
