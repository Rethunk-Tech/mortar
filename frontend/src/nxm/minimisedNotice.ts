export const NXM_SHOW_CATEGORY = 'nxm-show'

export function nxmShowCategory(showLabel: string): {
  id: string
  actions: { id: string; title: string }[]
} {
  return { id: NXM_SHOW_CATEGORY, actions: [{ id: 'show', title: showLabel }] }
}

export function nxmShowOptions(
  arrivalId: number,
  title: string,
  body: string,
): { id: string; title: string; body: string; categoryId: string } {
  return { id: `nxm-${arrivalId}`, title, body, categoryId: NXM_SHOW_CATEGORY }
}

export function nxmShowWithIcon(
  arrivalId: number,
  title: string,
  body: string,
  icon: string,
): ReturnType<typeof nxmShowOptions> & {
  attachments?: { id: string; path: string; type: 'appLogoOverride' }[]
} {
  const options = nxmShowOptions(arrivalId, title, body)
  if (!icon) {
    return options
  }
  return {
    ...options,
    attachments: [{ id: 'icon', path: icon, type: 'appLogoOverride' }],
  }
}
