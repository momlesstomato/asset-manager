# Plus Emulator Documentation

## Overview
This document details the database schema for the **Plus Emulator**, specifically the `furniture` table, and its relationship with the client-side `FurniData.json` (documented in `docs/gamedata/FURNIDATA.md`).

## Table: furniture

This table defines the static properties of furniture items. It serves as the server-side counterpart to `FurniData.json`.

| Column | Type | Relation to `FurniData.json` | Description |
| :--- | :--- | :--- | :--- |
| `id` | `int` | **No Relation** | **Unique Identifier**. The unique ID for the item definition. Referenced by `catalog_items`. |
| `sprite_id` | `int` | `id` | **Sprite ID**. The ID used by the client to find the graphical assets (SWF files). Maps directly to the `id` field in FurniData. |
| `item_name` | `string` | `classname` | **Internal Name**. The identifying name used by the server. Maps to `classname` (determines asset loading). |
| `public_name` | `string` | `name` | **Display Name**. The name shown in the inventory (if not overridden). Maps to `name`. |
| `type` | `enum` | `type` (Structure) | **Item Type**. `s`=Floor, `i`=Wall. Determines packet structure and FurniData section (`roomitemtypes` vs `wallitemtypes`). |
| `width` | `int` | `xdim` | **Width (X)**. Tiles occupied along X-axis. Maps to `xdim`. |
| `length` | `int` | `ydim` | **Length (Y)**. Tiles occupied along Y-axis. Maps to `ydim`. |
| `stack_height` | `double` | **No Direct Relation** | **Stack Height**. The height of the item for stacking logic. |
| `can_stack` | `0/1` | **No Direct Relation** | **Stackable**. `1` if other items can be placed on top of this item. |
| `is_walkable` | `0/1` | `canstandon` | **Walkable**. `1` if avatars can walk onto this item. Maps to `canstandon`. |
| `can_sit` | `0/1` | `cansiton` | **Seatable**. `1` if avatars sit when interacting with this item. Maps to `cansiton`. |
| `allow_recycle` | `0/1` | **No Relation** | **Recyclable**. `1` if this item can be recycled. |
| `allow_trade` | `0/1` | **No Relation** | **Tradable**. `1` if users can trade this item. |
| `allow_marketplace_sell` | `0/1` | `buyout` (loose) | **Marketplace**. `1` if this item can be sold on the marketplace. |
| `allow_gift` | `0/1` | **No Relation** | **Giftable**. `1` if this item can be gifted. |
| `allow_inventory_stack` | `0/1` | **No Relation** | **Inventory Stacking**. `1` if identical items condense into one slot in inventory. |
| `interaction_type` | `string` | `specialtype` (logic) | **Interaction Logic**. Server-side class mapping (e.g., `gate`, `dice`). See **Interaction Types** section. |
| `interaction_modes_count` | `int` | **No Direct Relation** | **State Count**. Max number of interaction states. Cycles `state = (state + 1) % count`. |
| `vending_ids` | `string` | **No Relation** | **Vending Items**. IDs of hand items given by this vending machine (interaction_type: `vendingmachine`). |
| `height_adjustable` | `string` | **No Relation** | **Adjustable Heights**. Comma-separated heights (e.g., `0.5,1.0`). Used for `stacktool`. |
| `effect_id` | `int` | **No Relation** | **Effect ID**. Effect ID applied by this item (e.g., functional pads). |
| `is_rare` | `0/1` | `rare` | **Rare Flag**. Denotes if item is considered rare. Maps to `rare` boolean. |
| `extra_rot` | `0/1` | **No Relation** | **Extra Rotation**. `1` enables special rotation intervals. |

---

## Interaction Types (`interaction_type`)

The `interaction_type` column determines the specific server-side logic. Below are common registered values (based on `InteractionTypes.cs`).

### General
- `default` / (empty): Standard furniture.
- `gate`, `onewaygate`, `vip_gate`: Gate logic.
- `bed`, `tent`: Sleeping/sitting logic.
- `sit`: Generic sitting logic.
- `vendingmachine`: Gives item in `vending_ids`.
- `dice`: Random state 1-6.
- `teleport`: Linked teleporters.
- `musicdisc`, `jukebox`: Music logic.
- `badge`, `badge_display`: Badge logic.
- `effect`: Applies `effect_id`.
- `dimmer`: Room background/lighting.
- `trophy`: Achievement display.
- `stacktool`: Magic stacker logic using `height_adjustable`.

### Wired & Games
- `wired_trigger`, `wired_effect`, `wired_condition`: Wired logic boxes.
- `wf_floor_switch`: Wired switch.
- `banzaiteleport`, `banzaipuck`, `bb_gate`: Battle Banzai logic.
- `freezetimer`, `freezetile`, `freezegate`: Freeze game logic.
- `football`, `ball`: Physics balls.

See source code (`InteractionTypes.cs`) for the complete enum list.
