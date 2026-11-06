import { close } from './menuKit.ts'
import type { Case } from './render-budget.ts'
import type { dialogKit } from './renderDialogs.ts'
import {
  DECOMPILE,
  DECOMPILE_MAPPED,
  HAS_DECOMPILER,
  type Stub,
  stubCalls,
} from './renderStubbed.ts'

/** The game code view's right-click menu, on a decompiled type that ILSpy is not here to produce. */
export function codeSymbolCase({ developTab }: ReturnType<typeof dialogKit>): Case {
  let stub: Stub | undefined
  let called = ''
  return {
    name: 'menu-code-symbol',
    setup: async (p) => {
      // ILSpy is not installed here, so the dialog is told it is and the decompiled type is stubbed; it calls the
      // member the search found, so the real lookup resolves its owner.
      let decompiled = ''
      let il = ''
      const hasDecompiler = stubCalls({ [HAS_DECOMPILER]: true })
      stub = async (route) => {
        if ((route.request().postData() ?? '').includes(`"methodID":${DECOMPILE}`)) {
          await route.fulfill({
            contentType: 'application/json',
            body: JSON.stringify(decompiled),
          })
          return
        }
        if ((route.request().postData() ?? '').includes(`"methodID":${DECOMPILE_MAPPED}`)) {
          const links = il.split('\n').map((_, line) => line)
          await route.fulfill({
            contentType: 'application/json',
            body: JSON.stringify({ csharp: decompiled, il, links }),
          })
          return
        }
        await hasDecompiler(route)
      }
      await p.route('**/wails/runtime', stub)
      await developTab(p)
      await p.getByRole('button', { name: /^Game code…/ }).click()
      const dialog = p.getByRole('dialog').last()
      await dialog.getByRole('combobox', { name: /^Search the game's code/ }).fill('player update')
      const first = p.getByRole('option').first()
      await first.locator('p').nth(1).waitFor()
      const [signature = '', type = ''] = await first.locator('p').allTextContents()
      called = signature.split('(')[0] ?? ''
      decompiled = [
        `public class ${type.split('.').at(-1)}`,
        '{',
        '    private void RenderCheck()',
        '    {',
        `        ${called}();`,
        '    }',
        '}',
      ].join('\n')
      // The IL line for the call sits at the C# line's index, so the line map resolves the call's owner.
      il = decompiled
        .split('\n')
        .map((line) =>
          line.includes(`${called}();`) ? `call instance void ${type}::${called}()` : '',
        )
        .join('\n')
      await first.click()
      await dialog
        .getByRole('region', { name: 'Code' })
        .getByText(/\w+\(\)/)
        .first()
        .waitFor()
    },
    run: async (p) => {
      // The stubbed call's name, so the menu looks the member up.
      const at = await p.getByRole('region', { name: 'Code' }).evaluate((region, name) => {
        const walker = document.createTreeWalker(region, NodeFilter.SHOW_TEXT)
        for (let node = walker.nextNode(); node; node = walker.nextNode()) {
          const index = (node.textContent ?? '').indexOf(name)
          if (index >= 0) {
            const range = document.createRange()
            range.setStart(node, index)
            range.setEnd(node, index + name.length)
            const box = range.getBoundingClientRect()
            return { x: box.x + box.width / 2, y: box.y + box.height / 2 }
          }
        }
        return null
      }, called)
      if (!at) {
        throw new Error('the decompiled code shows no call to right-click')
      }
      await p.mouse.click(at.x, at.y, { button: 'right' })
      await p
        .getByRole('menuitem', { name: /^Find usages/ })
        .first()
        .waitFor()
    },
    teardown: async (p) => {
      await close(p)
      await p.keyboard.press('Escape')
      if (stub) {
        await p.unroute('**/wails/runtime', stub)
      }
    },
  }
}
