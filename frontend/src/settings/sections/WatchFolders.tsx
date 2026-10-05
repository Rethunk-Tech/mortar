import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { PickFolder } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { SetByKey } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useToasts } from '../../toasts/store.ts'
import { persist } from '../persist.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

// Joins the folders the way the setting stores them: one path-list string, ';' on Windows and ':' elsewhere.
const separator = navigator.userAgent.includes('Windows') ? ';' : ':'

function WatchFolders() {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const value = useSettings((s) => s.watchFolders ?? '')
  const folders = value === '' ? [] : value.split(separator)
  const save = (next: string[]) =>
    persist(
      () => SetByKey('watchFolders', next.join(separator), ''),
      push,
      t`Could not save that setting`,
    )
  const add = () =>
    persist(
      async () => {
        const dir = await PickFolder(t`Folder to watch for archives`)
        if (dir && !folders.includes(dir)) {
          await SetByKey('watchFolders', [...folders, dir].join(separator), '')
        }
      },
      push,
      t`Could not save that setting`,
    )
  return (
    <SettingRow
      label={t`Also watch these folders`}
      description={t`New .zip, .7z and .rar files that land in them are offered for install, like the download folder. Subfolders are not watched.`}
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, alignItems: 'flex-start' }}>
        {folders.map((dir) => (
          <Box key={dir} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Typography noWrap={true} title={dir} sx={{ maxWidth: 360 }}>
              {dir}
            </Typography>
            <Button size="small" onClick={() => save(folders.filter((f) => f !== dir))}>
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
