import { useLingui } from '@lingui/react/macro'
import { Box, Button, MenuItem, Select, Typography } from '@mui/material'
import { PickFolder } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { SetByKey } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { space } from '../../theme/density.ts'
import { persist } from '../persist.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

// Joins the folders the way the settings store them: one path-list string, ';' on Windows and ':' elsewhere.
const separator = navigator.userAgent.includes('Windows') ? ';' : ':'
const split = (value: string) => (value === '' ? [] : value.split(separator))

type Mode = 'offer' | 'auto'
const KEYS: Record<Mode, string> = { offer: 'watchFolders', auto: 'autoInstallFolders' }

// Each folder either offers the archives that land in it, or installs itself as a mod whenever it changes (a mod
// author's build output); the mode picks which setting holds it.
function WatchFolders() {
  const { t } = useLingui()
  const offered = split(useSettings((s) => s.watchFolders ?? ''))
  const auto = split(useSettings((s) => s.autoInstallFolders ?? ''))
  const lists: Record<Mode, string[]> = { offer: offered.filter((d) => !auto.includes(d)), auto }
  const save = (mode: Mode, next: string[]) => SetByKey(KEYS[mode], next.join(separator), '')
  const failed = t`Could not save that setting`
  const remove = (dir: string, mode: Mode) =>
    persist(
      () =>
        save(
          mode,
          lists[mode].filter((d) => d !== dir),
        ),
      failed,
    )
  const move = (dir: string, from: Mode, to: Mode) =>
    persist(async () => {
      await save(to, [...lists[to], dir])
      await save(
        from,
        lists[from].filter((d) => d !== dir),
      )
    }, failed)
  const add = () =>
    persist(async () => {
      const dir = await PickFolder(t`Folder to watch`)
      if (dir && !lists.offer.includes(dir) && !auto.includes(dir)) {
        await save('offer', [...lists.offer, dir])
      }
    }, failed)
  const rows = [
    ...lists.offer.map((dir) => ({ dir, mode: 'offer' as const })),
    ...auto.map((dir) => ({ dir, mode: 'auto' as const })),
  ]
  return (
    <SettingRow
      label={t`Also watch these folders`}
      description={t`Offer installs: new archives (.zip, .7z, .rar, .tar.gz and the like) that land in the folder are offered for install. Auto-install: the folder itself is installed into the open profile as one mod each time its files change, for a mod's build output; a rebuild replaces the last one. Subfolders are not watched.`}
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, alignItems: 'flex-start' }}>
        {rows.map(({ dir, mode }) => (
          <Box key={dir} sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
            <Typography noWrap={true} title={dir} sx={{ maxWidth: 360 }}>
              {dir}
            </Typography>
            <Select
              size="small"
              value={mode}
              inputProps={{ 'aria-label': t`What to do with ${dir}` }}
              onChange={(e) => {
                const to = e.target.value as Mode
                if (to !== mode) {
                  move(dir, mode, to)
                }
              }}
            >
              <MenuItem value="offer">{t`Offer installs`}</MenuItem>
              <MenuItem value="auto">{t`Auto-install`}</MenuItem>
            </Select>
            <Button size="small" onClick={() => remove(dir, mode)}>
              {t`Remove`}
            </Button>
          </Box>
        ))}
        <Button variant="outlined" onClick={add}>
          {t`Add folder…`}
        </Button>
      </Box>
    </SettingRow>
  )
}

export { WatchFolders }
