import { defineTokens } from '@pandacss/dev'

export const easings = defineTokens.easings({
  pulse: { value: 'cubic-bezier(0.4, 0.0, 0.6, 1.0)' },
  default: { value: 'cubic-bezier(0.2, 0.0, 0, 1.0)' },
  'emphasized-in': { value: 'cubic-bezier(0.05, 0.7, 0.1, 1.0)' },
  'emphasized-out': { value: 'cubic-bezier(0.3, 0.0, 0.8, 0.15)' },
  // iOS-style spring overshoot, damping ~0.8 approximated as a CSS bezier.
  bounce: { value: 'cubic-bezier(0.34, 1.56, 0.64, 1)' },
})
