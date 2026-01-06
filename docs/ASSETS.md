# Asset Folder Structure

This document explains the structure and contents of the assets managed by the application.

## Bundled Assets (`bundled`)
Contains all the `.nitro` assets that will be rendered by the client.

| Path | Description |
|------|-------------|
| `bundled/effect/` | Contains the effect assets. |
| `bundled/figure/` | Contains all the clothing assets. |
| `bundled/furniture/` | Contains all the furniture (furnis). |
| `bundled/generic/` | Contains reusable assets like rooms or holders. |
| `bundled/pet/` | Contains pet bundles. |

## Catalog Images (`c_images`)
Short for "Catalog of Images". This directory is a replica of how images are organized in public production hotels.
These folders contain images currently used by the Nitro renderer.

**Key Contents:**
- Image icons (used when no photo of the image is available)
- Campaign views
- Targeted offers
- Catalog front page images

*Note: The remaining images in this directory are used by the CMS and internal hotel systems.*

## Legacy & Icons (`dcr`)
| Path | Description |
|------|-------------|
| `dcr/hof_furni/icons` | **Mandatory.** Contains the furniture icons. |
| `dcr/hof_furni/mp3` | Contains Sound Machine files. |

*Note: `hof_furni` elements are not used by the Nitro client itself.*

## Client Configuration & Resources

| Directory | Description |
|-----------|-------------|
| `gamedata/` | Nitro client JSON configurations. |
| `logos/` | Nitro client logos. |
| `sounds/` | `.mp3` sounds for the client. |

### FurnitureData Structure (`gamedata/FurnitureData.json`)

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

### EffectMap Structure (`gamedata/EffectMap.json`)

```json
{
  "effects": [
    {
      "id": "string",
      "lib": "string",
      "type": "string",
      "revision": "integer"
    }
  ]
}
```

### FigureData Structure (`gamedata/FigureData.json`)

```json
{
  "palettes": [
    {
      "id": "integer",
      "colors": [
        {
          "id": "integer",
          "index": "integer",
          "club": "integer",
          "selectable": "boolean",
          "hexCode": "string"
        }
      ]
    }
  ],
  "setTypes": [
    {
      "type": "string",
      "paletteId": "integer",
      "mandatory_f_0": "boolean",
      "mandatory_f_1": "boolean",
      "mandatory_m_0": "boolean",
      "mandatory_m_1": "boolean",
      "sets": [
        {
          "id": "integer",
          "gender": "string",
          "club": "integer",
          "colorable": "boolean",
          "selectable": "boolean",
          "preselectable": "boolean",
          "sellable": "boolean",
          "parts": [
            {
              "id": "integer",
              "type": "string",
              "colorable": "boolean",
              "index": "integer",
              "colorindex": "integer"
            }
          ],
          "hiddenLayers?": [
            {
              "partType": "string"
            }
          ]
        }
      ]
    }
  ]
}
```

### ProductData Structure (`gamedata/ProductData.json`)

```json
{
  "productdata": {
    "product": [
      {
        "code": "string",
        "name": "string",
        "description": "string"
      }
    ]
  }
}
```

### Avatar Actions Structure (`gamedata/HabboAvatarActions.json`)

```json
{
  "actions": [
    {
      "id": "string",
      "state": "string",
      "precedence": "integer",
      "geometryType": "string",
      "assetPartDefinition": "string",

      "main?": "boolean|integer",
      "activePartSet?": "string",
      "prevents?": ["string"],
      "animation?": "boolean|integer",
      "startFromFrameZero?": "boolean",
      "preventHeadTurn?": "boolean",
      "lay?": "string",
      "isDefault?": "boolean",

      "types?": [
        {
          "id": "integer|string",
          "animated?": "boolean",
          "prevents?": ["string"],
          "preventHeadTurn?": "boolean"
        }
      ],

      "params?": [
        {
          "id": "string",
          "value": "string"
        }
      ]
    }
  ],

  "actionOffsets": [
    {
      "action": "string",
      "offsets": [
        {
          "size": "string",
          "direction": "integer",
          "x": "integer",
          "y": "integer",
          "z": "number"
        }
      ]
    }
  ]
}
```

### Texts Structure (`gamedata/ExternalTexts.json` & `gamedata/UITexts.json`)

Both files follow a simple key-value structure.

```json
{
  "key": "string"
}
```
