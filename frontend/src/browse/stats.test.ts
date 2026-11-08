import { expect, test } from 'bun:test'
import { i18n } from '@lingui/core'
import { I18nProvider } from '@lingui/react'
import { renderHook } from '@testing-library/react'
import { createElement, type ReactNode } from 'react'
import type { ListRow } from '../mods/listColumns.ts'
import { popularitySortLabel, useModCardStats } from './stats.ts'

i18n.load('en', {})
i18n.activate('en')

function wrapper({ children }: { children: ReactNode }) {
  return createElement(I18nProvider, { i18n }, children)
}

function row(
  source: string,
  endorsements: number,
): Pick<ListRow, 'mod' | 'size' | 'details'> & { source: string } {
  return {
    mod: { endorsements } as ListRow['mod'],
    source,
  }
}

test('installed-card popularity uses the source metric, not endorsement copy', () => {
  const { result } = renderHook(() => useModCardStats(row('thunderstore', 4), ['popularity']), {
    wrapper,
  })
  expect(result.current[0]?.icon).toBe('like')
  expect(result.current[0]?.full).toContain('like')
  expect(result.current[0]?.full).not.toContain('endorsement')

  const modio = renderHook(() => useModCardStats(row('modio', 4), ['popularity']), { wrapper })
  expect(modio.result.current[0]?.full).toContain('rating')
  expect(modio.result.current[0]?.full).not.toContain('endorsement')
})

test('endorsements sort label follows the same source metric', () => {
  expect(popularitySortLabel('thunderstore')).toContain('top rated')
  expect(popularitySortLabel('paradox')).toContain('top rated')
  expect(popularitySortLabel('modio')).toContain('top rated')
  expect(popularitySortLabel('modrinth')).toContain('followed')
  expect(popularitySortLabel('vsmoddb')).toContain('followed')
  expect(popularitySortLabel('curseforge')).toContain('popular')
  expect(popularitySortLabel('nexus')).toContain('endorsed')
})
