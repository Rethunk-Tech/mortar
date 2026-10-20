import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { PickFolder } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { SetByKey } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useCurrentGame } from '../../nav/currentGame.ts'
import { space } from '../../theme/density.ts'
import { persist } from '../persist.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { separator, split } from './pathList.ts'

function ExtraModsFolders() {
  const { t } = useLingui()
  const game = useCurrentGame()
  const folders = split(useSettings((s) => s.games?.[game]?.extraModsFolder ?? ''))
  const save = (next: string[]) => SetByKey('extraModsFolder', next.join(separator), game)
  const failed = t`Could not save that setting`
  const add = () =>
    persist(async () => {
      const dir = await PickFolder(t`Extra mods folder`)
      if (dir && !folders.includes(dir)) {
        await save([...folders, dir])
      }
    }, failed)
  return (
    <SettingRow
      label={t`Extra mods folders`}
      description={t`Folders of unpacked mods to add from the Add menu`}
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, alignItems: 'flex-start' }}>
        {folders.map((dir) => (
          <Box key={dir} sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
            <Typography noWrap={true} title={dir} sx={{ maxWidth: 360 }}>
              {dir}
            </Typography>
            <Button
              size="small"
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
