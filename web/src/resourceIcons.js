import { Coins, Cuboid, Star, Leaf, Zap, Flame } from '@lucide/vue'

// Maps each ResourceType to a Lucide icon component for intuitive at-a-glance
// display. Lucide keeps things trademark-free (plain SVG glyphs, no brand
// marks) while matching the rest of the UI's existing icon language.
// Titanium has no direct Lucide equivalent — real-world Terraforming Mars
// boards commonly use a star for it, so Star is the closest intuitive match.
export const RESOURCE_ICONS = {
  MC: Coins,
  Steel: Cuboid,
  Titanium: Star,
  Plant: Leaf,
  Energy: Zap,
  Heat: Flame,
}
