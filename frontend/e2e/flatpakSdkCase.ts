import type { Page } from '@playwright/test'
import type { Tools } from '../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import type { Case } from './render-budget.ts'
import { patchCalls } from './renderCoverage.ts'
import type { dialogKit } from './renderDialogs.ts'

const TOOLS = 542_138_650
const INSTALL = 'flatpak install flathub org.freedesktop.Sdk.Extension.dotnet10//26.08'

/**
 * New mod's Prerequisites in a Flatpak without Flathub's .NET SDK extension. The server is not a Flatpak (setting
 * FLATPAK_ID would turn every Flatpak path on for the whole run), so the Tools answer is edited to what the Flatpak
 * gives: no SDK and the install command.
 */
export function flatpakSdkCases({ developTab, button }: ReturnType<typeof dialogKit>): Case[] {
  const stub = patchCalls<Tools>([TOOLS], (tools) => ({
    ...tools,
    dotnetSdks: [],
    flatpakDotnetInstall: INSTALL,
  }))
  return [
    {
      name: 'develop-new-mod-flatpak-sdk',
      setup: async (p: Page) => {
        await p.route('**/wails/runtime', stub)
        await developTab(p)
      },
      run: async (p: Page) => {
        await button(p, /^New mod…/)
        await p.getByRole('dialog').last().getByText(INSTALL).waitFor()
      },
      teardown: async (p: Page) => {
        await p.keyboard.press('Escape')
        await p.unroute('**/wails/runtime', stub)
      },
    },
  ]
}
