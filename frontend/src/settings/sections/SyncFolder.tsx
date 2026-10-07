import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { SetByKey } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { space } from '../../theme/density.ts'
import { persist, pickFolderSetting } from '../persist.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

export function SyncFolder() {
  const { t } = useLingui()
  const folder = useSettings((s) => s.syncFolder ?? '')
  const save = (dir: string) =>
    persist(() => SetByKey('syncFolder', dir, ''), t`Could not save that setting`)
  return (
    <SettingRow
      label={t`Sync folder`}
      description={t`Profile settings, mod lists and configs are written here and offered to other machines using the same folder (through Syncthing, Dropbox or a NAS). Mod files are not copied; each machine downloads them from their sources. Off by default.`}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
        <Typography noWrap={true} title={folder} sx={{ maxWidth: 360 }}>
          {folder || t`Off`}
        </Typography>
        <Button
          variant="outlined"
          onClick={() =>
            pickFolderSetting(t`Sync folder`, 'syncFolder', '', t`Could not save that setting`)
          }
        >
          {t`Choose…`}
        </Button>
        {folder ? <Button onClick={() => save('')}>{t`Disable`}</Button> : null}
      </Box>
    </SettingRow>
  )
}
