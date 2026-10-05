import { compact } from '../game/compact.ts'

// Count pills beside a profile's name; in the narrow sidebar they sit in the button's corner.
const pill = {
  flexShrink: 0,
  ml: 0.5,
  px: '7px',
  py: '1px',
  borderRadius: '10px',
  color: '#1b1a17',
  fontSize: 12,
  fontWeight: 700,
}

const sidebarPill = {
  ...pill,
  [compact]: {
    position: 'absolute' as const,
    top: 1,
    right: 1,
    ml: 0,
    px: '4px',
    fontSize: 10,
  },
  '[data-collapsed="true"] &': {
    position: 'absolute' as const,
    top: 1,
    right: 1,
    ml: 0,
    px: '4px',
    fontSize: 10,
  },
}

export { pill, sidebarPill }
