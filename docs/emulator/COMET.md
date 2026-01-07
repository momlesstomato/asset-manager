# Comet Emulator Documentation

## Overview
This document details the database schema for the **Comet Emulator**, specifically the `furniture` table, and its relationship with the client-side `FurniData.json` (documented in `docs/gamedata/FURNIDATA.md`).

## Table: furniture

This table defines the properties, behaviors, and interaction logic for every furniture item available in the hotel. The server-side mapping relies on `ItemDefinition.java`.

| Column | Type | Default | Relation to `FurniData.json` | Description |
| :--- | :--- | :--- | :--- | :--- |
| `id` | `INT` | Auto Inc | **No Relation** | **Unique Identifier**. Primary key. Referenced by `catalog_items`. |
| `sprite_id` | `INT` | `0` | `id` | **Sprite ID**. The identifier used by the client to load graphical assets (SWF). Maps directly to `id` field in FurniData. |
| `item_name` | `VARCHAR` | - | `classname` | **Internal Name**. The identifying name used by the server. Maps to `classname` (determines asset loading). Special prefixes like `wallpaper`, `landscape`, `floor` trigger decor logic. `ads_` triggers ad logic. |
| `public_name` | `VARCHAR` | - | `name` | **Display Name**. The default name shown to users. Maps to `name`. |
| `width` | `INT` | `1` | `xdim` | **Width (X)**. Size along X-axis (tiles). Maps to `xdim`. |
| `length` | `INT` | `1` | `ydim` | **Length (Y)**. Size along Y-axis (tiles). Maps to `ydim`. |
| `stack_height` | `VARCHAR` | `1` | **No Direct Relation** | **Stack Height**. The height of the item. Parsed as `double`. If `0`, defaults to `0.001` in server to prevent Z-fighting. |
| `can_stack` | `ENUM(0,1)`| `1` | **No Direct Relation** | **Stackable**. `1` if other items can be placed on top of this item. |
| `can_sit` | `ENUM(0,1)`| `0` | `cansiton` | **Sittable**. `1` if avatars can sit on this item. Maps to `cansiton`. |
| `can_lay` | `ENUM(0,1)`| `0` | `canlayon` | **Layable**. `1` if avatars can lay on this item. Maps to `canlayon`. |
| `is_walkable` | `ENUM(0,1)`| `0` | `canstandon` | **Walkable**. `1` if avatars can walk through/over this item. Maps to `canstandon`. |
| `allow_recycle` | `ENUM(0,1)`| `1` | **No Relation** | **Recyclable**. `1` if the item can be recycled. |
| `allow_trade` | `ENUM(0,1)`| `1` | **No Relation** | **Tradable**. `1` if the item can be traded between users. |
| `allow_marketplace_sell` | `ENUM(0,1)`| `0` | `buyout` (loose) | **Marketplace**. `1` if the item can be sold on the Marketplace. |
| `allow_gift` | `ENUM(0,1)`| `1` | **No Relation** | **Giftable**. `1` if the item can be sent as a gift. |
| `allow_inventory_stack` | `ENUM(0,1)`| `1` | **No Relation** | **Inventory Stackable**. `1` if multiple identical items stack in the user's inventory view. |
| `type` | `ENUM` | `s` | `type` (Structure) | **Item Type**. `s`=Floor, `i`=Wall, `e`=Effect. Determines packet structure and FurniData section (`roomitemtypes` vs `wallitemtypes`). |
| `interaction_type` | `VARCHAR` | `default` | `specialtype` (logic) | **Interaction Logic**. Maps specific server-side Java classes to the item. See **Interaction Types** section. |
| `interaction_modes_count` | `INT` | `1` | **No Direct Relation** | **State Count**. Number of states (modes). Used for double-click interactions (cycles state). |
| `vending_ids` | `VARCHAR` | `0` | **No Relation** | **Vending Logic**. Comma-separated list of Item IDs that this machine makes available (e.g., `1,2,3`). |
| `variable_heights` | `VARCHAR` | `0` | **No Relation** | **Dynamic Heights**. Double array separated by semicolons (e.g., `1.0;1.5;2.0`). Used for items with changeable heights. |
| `effect_id` | `INT` | `0` | **No Relation** | **Effect Giver**. Effect ID applied to user when interacting or walking on item. |
| `song_id` | `INT` | `0` | **No Relation** | **Music Disc**. If non-zero, this item contains data for the song with this ID. |
| `revision` | `INT` | `45554` | `revision` | **Revision**. Asset version number. Maps to `revision`. |
| `description` | `VARCHAR` | - | `description` | **Description**. Text description of the item. Maps to `description`. |
| `specialtype` | `INT` | `1` | **No Direct Relation** | **Special Type**. Use case varies, often generic flags. |
| `canlayon` | `ENUM(0,1)`| `0` | `canlayon` | **Layable (Legacy)**. Duplicate of `can_lay` but exists in schema. |
| `requires_rights` | `ENUM(0,1)`| `1` | **No Relation** | **Rights Required**. If `1`, usuall users need room rights to interact. |
| `colors` | `LONGTEXT` | `NULL` | `partcolors` | **Colors**. JSON data for item color customization. Maps to `partcolors`. |
| `deleteable` | `TINYINT` | `1` | **No Relation** | **Deletable**. If `1`, the item can be deleted (trash bin). |
| `is_arrow` | `ENUM(0,1)`| `0` | **No Relation** | **Arrow Logic**. Likely used for directional tiles. |
| `foot_figure` | `ENUM(0,1)`| `0` | **No Relation** | Legacy/Unused. |
| `stack_multiplier` | `ENUM(0,1)`| `0` | **No Relation** | Legacy/Unused. |
| `subscriber` | `ENUM(0,1)`| `0` | `bc` (loose) | **HC/VIP**. If `1`, usually restricted to Club members. |
| `flat_id` | `INT` | `-1` | `offerid` | **Offer ID**. Mapped to `offerId` in Java. Often used to link back to a catalog offer. |

---

## Interaction Types (`interaction_type`)

The `interaction_type` column acts as a key to map specific server-side Java classes to the item.

### General
- `default`: Standard furniture.
- `teleport`, `teleport_door`, `teleport_pad`: Teleport linking.
- `gate`, `dice`, `bottle`: Specific game logic.
- `bed`: Siting/Sleeping logic (beyond basic flags).
- `ads_background`: Ad Logic.

### Wired & Games
- `wf_act_*`: Wired Actions.
- `wf_trg_*`: Wired Triggers.
- `wf_cnd_*`: Wired Conditions.
- `wf_xtra_*`: Wired Extras.
- `bb_patch`: Battle Banzai patch.
- `football_goal`: Football gate.
- `horse_jump`: Horse jumping obstacle.
- `gym_equipment`: Treadmills, etc.
- `freeze_tile`: Freeze game tile.

## Special Logic Notes
- **Ads**: Items are treated as Ad Furni if `item_name` is `ads_mpu_720`, `ads_background`, etc., or `interaction_type` is `ads_background`.
- **Room Decor**: Items starting with `wallpaper`, `landscape`, or `floor` are treated as room decoration.
- **Music**: Items with `song_id > 0` are treated as playable music discs.
