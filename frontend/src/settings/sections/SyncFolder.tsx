import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { PickFolder } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { SetByKey } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useToasts } from '../../toasts/store.ts'
import { persist } from '../persist.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

export function SyncFolder() {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const folder = useSettings((s) => s.syncFolder ?? '')
  const save = (dir: string) =>
    persist(() => SetByKey('syncFolder', dir, ''), push, t`Could not save that setting`)
  return (
    <SettingRow
      label={t`Sync folder`}
      description={t`Profile settings, mod lists and configs are written here and offered to your other machines that use the same folder (through Syncthing, Dropbox or a NAS). Mod files are not copied; each machine gets them from their sources. Off by default.`}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <Typography noWrap={true} title={folder} sx={{ maxWidth: 360 }}>
          {folder || t`Off`}
        </Typography>
        <Button
          variant="outlined"
          onClick={() =>
            persist(
              async () => {
                const dir = await PickFolder(t`Sync folder`)
                if (dir) {
                  await SetByKey('syncFolder', dir, '')
                }
              },
              push,
              t`Could not save that setting`,
            )
          }
        >
          {t`Choose…`}
        </Button>
        {folder ? <Button onClick={() => save('')}>{t`Turn off`}</Button> : null}
      </Box>
    </SettingRow>
  )
}
