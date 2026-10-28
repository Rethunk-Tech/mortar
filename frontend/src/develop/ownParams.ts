import { i18n, type MessageDescriptor } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import type { Param } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'

// Text of the questions Mortar adds itself (a Param with `own` set), keyed by id. Template questions keep the
// template author's words. internal/devsvc/asset_test.go fails when an own id is missing here.
export const OWN_PARAMS: Record<string, { label: MessageDescriptor; hint: MessageDescriptor }> = {
  description: { label: msg`Description`, hint: msg`What the mod does, in one line.` },
  modAuthor: { label: msg`Author`, hint: msg`Who holds the copyright.` },
  team: { label: msg`Thunderstore team`, hint: msg`The team the mod is published under.` },
}

export function localizedParam(p: Param): Param {
  const own = p.own ? OWN_PARAMS[p.id] : undefined
  return own ? { ...p, label: i18n._(own.label), description: i18n._(own.hint) } : p
}
