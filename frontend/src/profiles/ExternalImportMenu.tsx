import { useLingui } from '@lingui/react/macro'
import { Button, Dialog, DialogContent, DialogTitle, Menu, MenuItem } from '@mui/material'
import { Download } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { SourceInfo } from '../../bindings/github.com/Rethunk-AI/mortar/internal/migrate/models.ts'
import {
  ExternalPreview,
  ExternalSources,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { openImport } from '../share/store.ts'
import { reportUnexpected } from '../toasts/report.ts'

export function ExternalImportMenu({ game }: { game: string }) {
  const { t } = useLingui()
  const [sources, setSources] = useState<SourceInfo[]>([])
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const [source, setSource] = useState<SourceInfo | null>(null)

  useEffect(() => {
    let current = true
    ExternalSources(game)
      .then((detected) => {
        if (current) {
          setSources(detected ?? [])
        }
      })
      .catch((error) => {
        if (current) {
          setSources([])
          reportUnexpected(error)
        }
      })
    return () => {
      current = false
    }
  }, [game])

  if (sources.length === 0) {
    return null
  }

  return (
    <>
      <Button
        variant="outlined"
        color="inherit"
        startIcon={<Download size={16} />}
        onClick={(event) => setAnchor(event.currentTarget)}
        sx={{ height: 40, px: 2, fontSize: 14, whiteSpace: 'nowrap' }}
      >
        {t`Import from…`}
      </Button>
      <Menu anchorEl={anchor} open={anchor !== null} onClose={() => setAnchor(null)}>
        {sources.map((item) => (
          <MenuItem
            key={item.kind}
            onClick={() => {
              setAnchor(null)
              setSource(item)
            }}
          >
            {t`Import from ${item.name}…`}
          </MenuItem>
        ))}
      </Menu>
      <Dialog open={source !== null} onClose={() => setSource(null)}>
        <DialogTitle>{t`Choose a profile to import`}</DialogTitle>
        <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, minWidth: 360 }}>
          {(source?.profiles ?? []).map((profile) => (
            <Button
              key={profile.id}
              onClick={() => {
                if (!source) {
                  return
                }
                ExternalPreview(game, source.kind, profile.id)
                  .then((preview) => {
                    openImport({ external: preview })
                    setSource(null)
                  })
                  .catch(reportUnexpected)
              }}
              sx={{ justifyContent: 'flex-start', textTransform: 'none' }}
            >
              {profile.name}
            </Button>
          ))}
        </DialogContent>
      </Dialog>
    </>
  )
}
