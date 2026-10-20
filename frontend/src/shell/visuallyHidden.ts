// Text or a control for assistive technology that takes no room on screen. Sizes are strings: sx reads a number up to
// 1 as a fraction, so width: 1 would be 100%. Padding, margin and border are zeroed because a control's own padding
// (MUI's Radio has 9px) would otherwise leave its icon showing around the 1px box.
export const visuallyHidden = {
  position: 'absolute',
  width: '1px',
  height: '1px',
  p: 0,
  m: '-1px',
  border: 0,
  overflow: 'hidden',
  clip: 'rect(0 0 0 0)',
  clipPath: 'inset(50%)',
  whiteSpace: 'nowrap',
} as const
