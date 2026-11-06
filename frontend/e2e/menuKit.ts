import type { Page } from '@playwright/test'

export const open = (p: Page) => p.getByRole('menu').locator('[role^="menuitem"]').first().waitFor()

// A title bar menu opens without focus inside it, so Escape never reaches it; its backdrop takes the click instead.
export async function close(p: Page) {
  await p.keyboard.press('Escape')
  const menu = p.getByRole('menu')
  const gone = await menu.waitFor({ state: 'hidden', timeout: 500 }).then(
    () => true,
    () => false,
  )
  if (!gone) {
    const { width, height } = p.viewportSize() ?? { width: 1400, height: 840 }
    await p.mouse.click(width - 5, height - 5)
    await menu.waitFor({ state: 'hidden' })
  }
}
