import { useLingui } from '@lingui/react/macro'
import { Box, Button, MenuItem, Select, Typography } from '@mui/material'
import { PickFolder } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import {
  SetByKey,
  SetExtraModsProfile,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { installExtraFolder } from '../../downloads/autoInstall.ts'
import { useCurrentGame } from '../../nav/currentGame.ts'
import { useProfiles } from '../../profiles/store.ts'
import { space } from '../../theme/density.ts'
import { persist } from '../persist.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { separator, split } from './pathList.ts'

const OPEN = 'open'

// Each folder installs its mods into the profile picked beside it, or the open one, now and whenever its files change.
function ExtraModsFolders() {
  const { t } = useLingui()
  const game = useCurrentGame()
  const folders = split(useSettings((s) => s.games?.[game]?.extraModsFolder ?? ''))
  const targets = useSettings((s) => s.games?.[game]?.extraModsProfiles)
  const profiles = useProfiles((s) => (s.game?.id === game ? s.profiles : null))
  const save = (next: string[]) => SetByKey('extraModsFolder', next.join(separator), game)
  const failed = t`Could not save that setting`
  const add = () =>
    persist(async () => {
      const dir = await PickFolder(t`Extra mods folder`)
      if (dir && !folders.includes(dir)) {
        await save([...folders, dir])
        installExtraFolder(game, dir)
      }
    }, failed)
  const target = (dir: string, profile: string) =>
    persist(async () => {
      await SetExtraModsProfile(game, dir, profile === OPEN ? '' : profile)
      installExtraFolder(game, dir)
    }, failed)
  return (
    <SettingRow
      label={t`Extra mods folders`}
      description={t`Folders of unpacked mods, or your own build's output. Their mods install into the profile beside each folder, and again whenever their files change.`}
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, alignItems: 'flex-start' }}>
        {folders.map((dir) => (
          <Box key={dir} sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
            <Typography noWrap={true} title={dir} sx={{ maxWidth: 360 }}>
              {dir}
            </Typography>
            <Select
              size="small"
              value={targets?.[dir] || OPEN}
              inputProps={{ 'aria-label': t`Profile ${{ path: dir }} installs into` }}
              onChange={(e) => target(dir, e.target.value)}
            >
              <MenuItem value={OPEN}>{t`Open profile`}</MenuItem>
              {(profiles ?? []).map((p) => (
                <MenuItem key={p.id} value={p.id}>
                  {p.name}
                </MenuItem>
              ))}
              {targets?.[dir] && profiles && !profiles.some((p) => p.id === targets[dir]) ? (
                <MenuItem value={targets[dir]}>{t`Deleted profile`}</MenuItem>
              ) : null}
            </Select>
            <Button
              size="small"
              aria-label={t`Remove ${{ name: dir }}`}
              onClick={() => persist(() => save(folders.filter((d) => d !== dir)), failed)}
            >
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

export { ExtraModsFolders }
