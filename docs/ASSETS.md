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
