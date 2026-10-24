import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { PickFolder } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { SetByKey } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { space } from '../../theme/density.ts'
import { persist } from '../persist.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { separator, split } from './pathList.ts'

function WatchFolders() {
  const { t } = useLingui()
  const folders = split(useSettings((s) => s.watchFolders ?? ''))
  const save = (next: string[]) => SetByKey('watchFolders', next.join(separator), '')
  const failed = t`Could not save that setting`
  const add = () =>
    persist(async () => {
      const dir = await PickFolder(t`Folder to watch`)
      if (dir && !folders.includes(dir)) {
        await save([...folders, dir])
      }
    }, failed)
  return (
    <SettingRow
      label={t`Also watch these folders`}
      description={t`New archives in these folders are offered for install. Subfolders are not watched. For a mod's build output, use the game's Extra mods folders.`}
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, alignItems: 'flex-start' }}>
        {folders.map((dir) => (
          <Box key={dir} sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
            <Typography noWrap={true} title={dir} sx={{ maxWidth: 360 }}>
              {dir}
            </Typography>
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

export { WatchFolders }
