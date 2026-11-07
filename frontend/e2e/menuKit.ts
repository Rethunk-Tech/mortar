import type { Page } from '@playwright/test'

export const open = (p: Page) => p.getByRole('menu').locator('[role^="menuitem"]').first().waitFor()

// A title bar menu opens without focus inside it, so Escape never reaches it; its backdrop takes the click instead.
export async function close(p: Page) {
  await clickBackdrop(p)
  await p.getByRole('menu').waitFor({ state: 'hidden' })
}

/** Clicks the page corner, where an open menu or select popover's full-window backdrop catches it. */
export async function clickBackdrop(p: Page) {
  const { width, height } = p.viewportSize() ?? { width: 1400, height: 840 }
  await p.mouse.click(width - 5, height - 5)
}
