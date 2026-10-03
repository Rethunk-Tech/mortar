import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Dialog, List, ListItemButton, ListItemIcon, ListItemText, TextField } from '@mui/material'
import {
  Download,
  FolderPlus,
  Inbox,
  Keyboard,
  Package,
  Play,
  RefreshCw,
  Settings,
  Share2,
  User,
} from 'lucide-react'
import { type KeyboardEvent, type ReactNode, useEffect, useRef, useState } from 'react'
import { NewProfileDialog } from '../game/NewProfileDialog.tsx'
import { paper } from '../mods/paper.ts'
import type { SettingsSection } from '../nav/store.ts'
import { userModEntries } from '../profiles/count.ts'
import { useProfiles } from '../profiles/store.ts'
import { mergeBindings, SHORTCUTS, type ShortcutId } from '../settings/shortcuts.ts'
import { useSettings } from '../settings/store.ts'
import { buildPaletteItems } from './items.ts'
import { matchPaletteItems, type PaletteItem } from './match.ts'
import { runPaletteItem } from './run.ts'
import { useCommandPalette } from './store.ts'

function iconFor(item: PaletteItem): ReactNode {
  if (item.kind === 'profile') {
    return <User size={16} />
  }
  if (item.kind === 'mod') {
    return <Package size={16} />
  }
  if (item.kind === 'settings') {
    return <Settings size={16} />
  }
  if (item.kind === 'shortcut') {
    return <Keyboard size={16} />
  }
  if (item.id === 'action:play') {
    return <Play size={16} />
  }
  if (item.id === 'action:updates') {
    return <RefreshCw size={16} />
  }
  if (item.id === 'action:downloads') {
    return <Download size={16} />
  }
  if (item.id === 'action:import') {
    return <Inbox size={16} />
  }
  if (item.id === 'action:share') {
    return <Share2 size={16} />
  }
  return <FolderPlus size={16} />
}

function PaletteRows({
  shown,
  currentId,
  onPick,
}: {
  shown: PaletteItem[]
  currentId: string | undefined
  onPick: (id: string) => void
}) {
  return (
    <List dense={true} sx={{ maxHeight: 420, overflow: 'auto', py: 1 }} role="listbox">
      {shown.map((item) => (
        <ListItemButton
          key={item.id}
          selected={item.id === currentId}
          onClick={() => onPick(item.id)}
          role="option"
          aria-selected={item.id === currentId}
        >
          <ListItemIcon sx={{ minWidth: 32, color: 'rgba(225,225,230,0.95)' }}>
            {iconFor(item)}
          </ListItemIcon>
          <ListItemText
            primary={item.label}
            secondary={item.hint}
            slotProps={{ primary: { noWrap: true }, secondary: { noWrap: true } }}
          />
          {item.shortcut ? (
            <kbd style={{ color: 'rgba(225,225,230,0.95)', fontSize: 12 }}>{item.shortcut}</kbd>
          ) : null}
        </ListItemButton>
      ))}
    </List>
  )
}

function usePaletteWindow(open: boolean) {
  const searchRef = useRef<HTMLInputElement>(null)
  useEffect(() => {
    if (!open) {
      return
    }
    const focus = requestAnimationFrame(() => {
      searchRef.current?.focus()
    })
    const onEsc = (e: globalThis.KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        e.stopPropagation()
        useCommandPalette.getState().setOpen(false)
      }
    }
    globalThis.addEventListener('keydown', onEsc, true)
    return () => {
      cancelAnimationFrame(focus)
      globalThis.removeEventListener('keydown', onEsc, true)
    }
  }, [open])
  return searchRef
}

function paletteSections(i18n: I18n) {
  return [
    { id: 'general' as const, label: i18n._(msg`General`) },
    { id: 'appearance' as const, label: i18n._(msg`Appearance`) },
    { id: 'data' as const, label: i18n._(msg`Data`) },
    { id: 'nexus' as const, label: i18n._(msg`Nexus Mods`) },
    { id: 'updates' as const, label: i18n._(msg`Updates`) },
    { id: 'shortcuts' as const, label: i18n._(msg`Shortcuts`) },
    { id: 'about' as const, label: i18n._(msg`About`) },
  ]
}

function shortcutLabels(i18n: I18n): Partial<Record<ShortcutId, string>> {
  return {
    'command-palette': i18n._(msg`Open the command palette`),
    'filter-mods': i18n._(msg`Focus the search`),
    play: i18n._(msg`Play the open profile`),
    'check-updates': i18n._(msg`Check for mod updates`),
    'open-settings': i18n._(msg`Open Settings`),
    dismiss: i18n._(msg`Close dialog or clear selection`),
    'select-all-mods': i18n._(msg`Select all mods`),
    'mod-up': i18n._(msg`Focus the previous mod`),
    'mod-down': i18n._(msg`Focus the next mod`),
    'mod-toggle': i18n._(msg`Toggle the focused mod`),
    'mod-details': i18n._(msg`Open focused mod details`),
    'mod-remove': i18n._(msg`Remove the focused mod`),
  }
}

export function CommandPalette() {
  const { t, i18n } = useLingui()
  const open = useCommandPalette((s) => s.open)
  const creating = useCommandPalette((s) => s.creating)
  const profiles = useProfiles((s) => s.profiles)
  const openId = useProfiles((s) => s.openId)
  const [query, setQuery] = useState('')
  const [index, setIndex] = useState(0)
  const searchRef = usePaletteWindow(open)
  const sections: { id: SettingsSection; label: string }[] = paletteSections(i18n)
  const shortcutLabelMap = shortcutLabels(i18n)
  const bindings = mergeBindings(useSettings((s) => s.shortcuts))
  const profile = profiles.find((p) => p.id === openId)
  const mods = userModEntries(profile?.entries).flatMap((entry) =>
    (entry.mods ?? []).map((mod) => ({
      key: entry.key,
      uniqueId: mod.uniqueId,
      name: mod.name,
    })),
  )
  const shown = matchPaletteItems(
    buildPaletteItems({
      profiles: profiles.map((p) => ({ id: p.id, name: p.name })),
      mods,
      sections,
      shortcuts: SHORTCUTS.map((row) => ({ ...row, keys: bindings[row.id] })),
      shortcutLabels: shortcutLabelMap,
      labels: {
        play: t`Play`,
        updates: t`Check for mod updates`,
        downloads: t`Open Downloads`,
        import: t`Import`,
        share: t`Share`,
        newProfile: t`New profile`,
        streamOverlay: t`Stream overlay`,
        profileHint: t`Open profile`,
        modHint: t`Open mod`,
        settingsHint: t`Settings`,
        tabs: {
          mods: t`Go to Mods`,
          problems: t`Go to Problems`,
          'load-order': t`Go to Load order`,
          saves: t`Go to Saves`,
          notes: t`Go to Notes`,
          console: t`Go to Console`,
          performance: t`Go to Performance`,
        },
        toggle: (name) => t`Toggle ${name}`,
      },
    }),
    query,
  )
  const current = shown[Math.min(index, Math.max(shown.length - 1, 0))]
  const reset = () => {
    setQuery('')
    setIndex(0)
  }
  const close = () => {
    useCommandPalette.getState().setOpen(false)
    reset()
  }
  const pick = (id: string) => {
    runPaletteItem(id)
    close()
  }
  const onKey = (e: KeyboardEvent) => {
    if (e.key === 'Escape') {
      e.preventDefault()
      close()
      return
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setIndex((i) => (shown.length === 0 ? 0 : (i + 1) % shown.length))
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      setIndex((i) => (shown.length === 0 ? 0 : (i - 1 + shown.length) % shown.length))
      return
    }
    if (e.key === 'Enter' && current) {
      e.preventDefault()
      pick(current.id)
    }
  }
  return (
    <>
      <Dialog
        open={open}
        onClose={close}
        transitionDuration={0}
        onKeyDown={onKey}
        slotProps={{
          paper: { ...paper, sx: { ...paper.sx, width: 560, maxWidth: 'calc(100% - 48px)' } },
        }}
      >
        <TextField
          inputRef={searchRef}
          autoFocus={true}
          fullWidth={true}
          value={query}
          onChange={(e) => {
            setQuery(e.target.value)
            setIndex(0)
          }}
          placeholder={t`Search actions and destinations`}
          slotProps={{ htmlInput: { 'aria-label': t`Search actions and destinations` } }}
          sx={{ px: 2, pt: 2, '& .MuiInputBase-root': { userSelect: 'text' } }}
        />
        <PaletteRows shown={shown} currentId={current?.id} onPick={pick} />
      </Dialog>
      <NewProfileDialog
        open={creating}
        onClose={() => useCommandPalette.getState().setCreating(false)}
      />
    </>
  )
}
