---
version: "alpha"
name: "Aegis Relay — Orbital Defense Network"
description: "Aegis Relay Background Effect is designed for delivering a visual treatment or immersive background effect. Key features include atmospheric visuals, motion depth, and flexible presentation layering. It is suitable for visual-first pages, motion studies, and atmospheric hero treatments."
colors:
  primary: "#D3E488"
  secondary: "#CDD6F0"
  tertiary: "#8FB8F0"
  neutral: "#EEF2FF"
  background: "#000000"
  surface: "#D3E488"
  text-primary: "#6A6288"
  text-secondary: "#D3E488"
  border: "#CDD6F0"
  accent: "#D3E488"
typography:
  headline-lg:
    fontFamily: "System Font"
  body-md:
    fontFamily: "Azeret Mono"
    fontSize: "10px"
    fontWeight: 300
    lineHeight: "16.25px"
    letterSpacing: "0.14em"
    textTransform: "uppercase"
spacing:
  base: "4px"
  sm: "2px"
  md: "4px"
  lg: "6px"
  xl: "10px"
  gap: "2px"
  card-padding: "12px"
---

## Overview

- **Composition cues:**
  - Layout: Grid
  - Content Width: Full Bleed
  - Framing: Glassy
  - Grid: Strong

## Colors

The color system uses light mode with #D3E488 as the main accent and #EEF2FF as the neutral foundation.

- **Primary (#D3E488):** Main accent and emphasis color.
- **Secondary (#CDD6F0):** Supporting accent for secondary emphasis.
- **Tertiary (#8FB8F0):** Reserved accent for supporting contrast moments.
- **Neutral (#EEF2FF):** Neutral foundation for backgrounds, surfaces, and supporting chrome.

- **Usage:** Background: #000000; Surface: #D3E488; Text Primary: #6A6288; Text Secondary: #D3E488; Border: #CDD6F0; Accent: #D3E488

## Typography

Typography pairs System Font for display hierarchy with Azeret Mono for supporting content and interface copy.

- **Headlines (`headline-lg`):** System Font.
- **Body (`body-md`):** Azeret Mono, 10px, weight 300, line-height 16.25px, letter-spacing 0.14em, uppercase.

## Layout

Layout follows a grid composition with reusable spacing tokens. Preserve the grid, full bleed structural frame before changing ornament or component styling. Use 4px as the base rhythm and let larger gaps step up from that cadence instead of introducing unrelated spacing values.

Treat the page as a grid / full bleed composition, and keep that framing stable when adding or remixing sections.

- **Layout type:** Grid
- **Content width:** Full Bleed
- **Base unit:** 4px
- **Scale:** 2px, 4px, 6px, 10px, 12px, 14px, 18px
- **Card padding:** 12px, 18px
- **Gaps:** 2px, 12px, 14px, 16px

## Elevation & Depth

Depth is communicated through glass, border contrast, and reusable shadow or blur treatments. Keep those recipes consistent across hero panels, cards, and controls so the page reads as one material system.

Surfaces should read as glass first, with borders, shadows, and blur only reinforcing that material choice.

- **Surface style:** Glass
- **Borders:** 1px #CDD6F0; 1px #2A2440
- **Shadows:** rgba(0, 0, 0, 0) 0px 0px 0px 0px, rgba(0, 0, 0, 0) 0px 0px 0px 0px, rgba(0, 0, 0, 0.6) 0px 0px 0px 1px inset, rgba(205, 214, 240, 0.05) 0px 1px 1px 0px; rgba(0, 0, 0, 0) 0px 0px 0px 0px, rgba(0, 0, 0, 0) 0px 0px 0px 0px, rgba(205, 214, 240, 0.06) 0px 1px 0px 0px; rgba(211, 228, 136, 0.75) 0px 0px 10px 0px, rgba(0, 0, 0, 0.6) 0px 0px 0px 2px inset
- **Blur:** 4px

### Techniques
- **Gradient border shell:** Use a thin gradient border shell around the main card. Wrap the surface in an outer shell with 18px padding and a 16px radius. Drive the shell with linear-gradient(rgba(28, 24, 44, 0.9), rgba(14, 12, 22, 0.94)) so the edge reads like premium depth instead of a flat stroke. Keep the actual stroke understated so the gradient shell remains the hero edge treatment. Inset the real content surface inside the wrapper with a slightly smaller radius so the gradient only appears as a hairline frame.

## Shapes

Shapes rely on a tight radius system anchored by 6px and scaled across cards, buttons, and supporting surfaces. Icon geometry should stay compatible with that soft-to-controlled silhouette.

Use the radius family intentionally: larger surfaces can open up, but controls and badges should stay within the same rounded DNA instead of inventing sharper or pill-only exceptions.

- **Corner radii:** 6px, 8px, 12px, 16px, 9999px

## Do's and Don'ts

Use these constraints to keep future generations aligned with the current system instead of drifting into adjacent styles.

### Do
- Do use the primary palette as the main accent for emphasis and action states.
- Do keep spacing aligned to the detected 4px rhythm.
- Do reuse the Glass surface treatment consistently across cards and controls.
- Do keep corner radii within the detected 6px, 8px, 12px, 16px, 9999px family.

### Don't
- Don't introduce extra accent colors outside the core palette roles unless the page needs a new semantic state.
- Don't mix unrelated shadow or blur recipes that break the current depth system.
- Don't exceed the detected minimal motion intensity without a deliberate reason.

## Motion

Motion stays restrained and interface-led across text, layout, and scroll transitions. Timing clusters around 300ms. Easing favors ease and cubic-bezier(0.4. Hover behavior focuses on color changes.

**Motion Level:** minimal

**Durations:** 300ms

**Easings:** ease, cubic-bezier(0.4, 0, 0.2, 1)

**Hover Patterns:** color
