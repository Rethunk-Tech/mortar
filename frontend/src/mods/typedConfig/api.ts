import {
  Files,
  Reset,
  ResetAll,
  Schema,
  Set as SetEntry,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/configsvc/service.ts'
import { formatValue, toSections, type WireSection } from './convert.ts'
import type { EntryAt, Target } from './store.ts'
import type { ConfigFile, ConfigSection, ConfigValue } from './types.ts'

const configApi = {
  files: async (t: Target): Promise<ConfigFile[]> =>
    ((await Files(t.game, t.profile, t.id)) ?? []).map((f) => ({
      name: f.name,
      label: f.label || f.name,
      format: f.format,
      sections: [],
    })),
  schema: async (t: Target, file: string): Promise<ConfigSection[]> =>
    toSections(((await Schema(t.game, t.profile, t.id, file)).sections ?? []) as WireSection[]),
  set: (t: Target, at: EntryAt, value: ConfigValue) =>
    SetEntry(t.game, t.profile, t.id, at.file, at.section, at.entry, formatValue(value)),
  reset: (t: Target, at: EntryAt) => Reset(t.game, t.profile, t.id, at.file, at.section, at.entry),
  resetAll: (t: Target, file: string) => ResetAll(t.game, t.profile, t.id, file),
}

export { configApi }
