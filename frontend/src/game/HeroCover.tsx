import { useLingui } from '@lingui/react/macro'
import { Box, IconButton, ListItemIcon, ListItemText, Menu, MenuItem } from '@mui/material'
import { ImageOff, ImagePlus } from 'lucide-react'
import { useEffect, useState } from 'react'
import { PickImage } from '../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  ClearCover,
  Covers,
  SetCover,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

// The first of the profile's covers that loads (picked, Nexus, Steam), else a solid tone.
export function HeroCover({ game, profile }: { game: string; profile: Profile }) {
  // Kept with the `updated` it was read for: the profile's cover and mods change with it.
  const [covers, setCovers] = useState<{ stamp: string; list: string[] } | null>(null)
  const [failed, setFailed] = useState<{ stamp: string; urls: string[] }>({ stamp: '', urls: [] })
  const updated = String(profile.updated)
  useEffect(() => {
    let live = true
    Covers(game, profile.id)
      .then((list) => {
        if (live) {
          setCovers({ stamp: updated, list: list ?? [] })
        }
      })
      .catch(reportUnexpected)
    return () => {
      live = false
    }
  }, [game, profile.id, updated])
  if (covers === null) {
    return <Box sx={{ width: '100%', height: '100%', bgcolor: 'rgb(44,44,54)' }} />
  }
  const skip = failed.stamp === covers.stamp ? failed.urls : []
  const src = covers.list.find((c) => !skip.includes(c))
  return src ? (
    <Box
      component="img"
      src={src}
      alt=""
      onError={() => setFailed({ stamp: covers.stamp, urls: [...skip, src] })}
      sx={{ width: '100%', height: '100%', objectFit: 'cover', display: 'block' }}
    />
  ) : (
    <Box sx={{ width: '100%', height: '100%', bgcolor: 'rgb(44,44,54)' }} />
  )
}

export function CoverButton({ game, profile }: { game: string; profile: Profile }) {
  const { t } = useLingui()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const replace = useProfiles((s) => s.replace)
  const close = () => setAnchor(null)
  const choose = async () => {
    close()
    const path = await PickImage(t`Choose cover image`)
    if (!path) {
      return
    }
    try {
      replace(await SetCover(game, profile.id, path))
    } catch (e) {
      useToasts
        .getState()
        .push({ kind: 'error', title: t`Could not use that image`, body: errorMessage(e) })
    }
  }
  const automatic = async () => {
    close()
    replace(await ClearCover(game, profile.id))
  }
  return (
    <>
      <IconButton
        aria-label={t`Cover image`}
        aria-haspopup="menu"
        onClick={(e) => setAnchor(e.currentTarget)}
        size="small"
      >
        <ImagePlus size={16} />
      </IconButton>
      <Menu open={anchor !== null} anchorEl={anchor} onClose={close}>
        <MenuItem onClick={() => choose().catch(reportUnexpected)}>
          <ListItemIcon sx={{ color: 'inherit' }}>
            <ImagePlus size={16} />
          </ListItemIcon>
          <ListItemText>{t`Choose cover image…`}</ListItemText>
        </MenuItem>
        <MenuItem disabled={!profile.cover} onClick={() => automatic().catch(reportUnexpected)}>
          <ListItemIcon sx={{ color: 'inherit' }}>
            <ImageOff size={16} />
          </ListItemIcon>
          <ListItemText>{t`Use the automatic cover`}</ListItemText>
        </MenuItem>
      </Menu>
    </>
  )
}
