import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
} from '@mui/material'
import { useState } from 'react'
import { useProfiles } from '../profiles/store.ts'
import { openImport } from '../share/store.ts'
import { useIncomingShares } from './incoming.ts'

export function IncomingPrompt() {
  const { t } = useLingui()
  const incoming = useIncomingShares((state) => state.items[0])
  const removeFirst = useIncomingShares((state) => state.removeFirst)
  const profiles = useProfiles((state) => state.profiles)
  const [choosing, setChoosing] = useState(false)

  if (!incoming) {
    return null
  }

  const link = `mortar://${incoming.game}/p/${incoming.payload}`
  const decline = () => {
    setChoosing(false)
    removeFirst()
  }
  const compare = (profileId: string) => {
    setChoosing(false)
    removeFirst()
    openImport({ profileId, link })
  }

  return (
    <>
      <Dialog open={!choosing} onClose={decline}>
        <DialogTitle>{t`${incoming.sender} sent you ${incoming.profileName} (${incoming.game})`}</DialogTitle>
        <DialogContent />
        <DialogActions>
          <Button onClick={decline}>{t`Decline`}</Button>
          <Button onClick={() => setChoosing(true)} disabled={profiles.length === 0}>
            {t`Compare with a profile…`}
          </Button>
          <Button
            variant="contained"
            onClick={() => {
              removeFirst()
              openImport({ link })
            }}
          >
            {t`Import as new profile`}
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog open={choosing} onClose={() => setChoosing(false)} fullWidth={true} maxWidth="xs">
        <DialogTitle>{t`Compare with a profile…`}</DialogTitle>
        <DialogContent dividers={true}>
          <List disablePadding={true}>
            {profiles.map((profile) => (
              <ListItem key={profile.id} disablePadding={true}>
                <ListItemButton onClick={() => compare(profile.id)}>
                  <ListItemText primary={profile.name} />
                </ListItemButton>
              </ListItem>
            ))}
          </List>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setChoosing(false)}>{t`Cancel`}</Button>
        </DialogActions>
      </Dialog>
    </>
  )
}
