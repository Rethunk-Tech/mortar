import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { SetByKey } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { DROPBOX, SYNCTHING } from '../../brand/productNames.ts'
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
      description={t`Profile settings, mod lists and configs sync through this folder (with ${{ app: SYNCTHING }}, ${{ otherApp: DROPBOX }} or a NAS); each computer downloads the mod files itself. Each game has its own subfolder, or its own folder or no sync, set in the game's settings.`}
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
