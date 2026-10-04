import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Tab,
  Tabs,
  TextField,
  Typography,
} from '@mui/material'
import { Pencil } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  ReadConfig,
  ReadContentSchema,
  WriteConfig,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { MONO } from '../theme/theme.ts'
import { errorDetails } from '../toasts/errorKind.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { PresetsButton } from './ConfigPresets.tsx'
import { applyCPSchema, parseCPSchema } from './configFields.ts'
import { type ConfigNode, parseConfig, setAt, stringifyConfig } from './configForm.ts'
import { Fields } from './configFormUi.tsx'
import { MenuHint, MenuPages } from './configMenuUi.tsx'
import { useGmcmMenu } from './useGmcmMenu.ts'
import { useLocked } from './useLocked.ts'

const text = { fontSize: 13 } as const
const noWrap = { whiteSpace: 'nowrap' } as const
const cpFor = 'Pathoschild.ContentPatcher'

function openTarget() {
  const { game, openId } = useProfiles.getState()
  return game && openId ? { game: game.id, id: openId } : null
}

function parseSchema(raw: string): ReturnType<typeof parseCPSchema> {
  try {
    return parseCPSchema(JSON.parse(raw) as unknown)
  } catch {
    return {}
  }
}

function fetchDoc(mod: Mod) {
  const target = openTarget()
  if (!target) {
    return Promise.reject(new Error('no profile'))
  }
  const schemaP =
    mod.contentPackFor === cpFor
      ? ReadContentSchema(target.game, target.id, mod.key, mod.uniqueId).catch(() => '{}')
      : Promise.resolve('{}')
  return Promise.all([ReadConfig(target.game, target.id, mod.key, mod.uniqueId), schemaP])
}

function useConfigDoc(mod: Mod, open: boolean) {
  const locked = useLocked()
  const [tree, setTree] = useState<ConfigNode | null>(null)
  const [jsonDraft, setJsonDraft] = useState('')
  const [tab, setTab] = useState(0)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)
  const [dirty, setDirty] = useState(false)
  const [discardOpen, setDiscardOpen] = useState(false)
  const readGen = useRef(0)
  useEffect(() => {
    readGen.current += 1
    if (!open) {
      setTree(null)
      setJsonDraft('')
      setError('')
      setSaved(false)
      setDirty(false)
      setTab(0)
      return
    }
    setSaved(false)
    setDirty(false)
    const seq = readGen.current
    fetchDoc(mod)
      .then(([raw, schemaRaw]) => {
        if (seq !== readGen.current) {
          return
        }
        setError('')
        setJsonDraft(raw)
        setTree(applyCPSchema(parseConfig(raw), parseSchema(schemaRaw)))
      })
      .catch((e: unknown) => {
        if (seq !== readGen.current) {
          return
        }
        setTree(null)
        setError(errorDetails(e))
      })
  }, [open, mod.key, mod.uniqueId, mod.contentPackFor, mod])
  const applyJson = (): boolean => {
    try {
      setTree(parseConfig(jsonDraft))
      setError('')
      return true
    } catch (e: unknown) {
      setError(errorDetails(e))
      return false
    }
  }
  const save = () => {
    const seq = readGen.current
    const target = openTarget()
    if (!target) {
      return
    }
    let body = jsonDraft
    if (tab === 1) {
      if (!tree) {
        return
      }
      body = stringifyConfig(tree)
    } else if (tab === 2 && !applyJson()) {
      return
    } else if (tab === 0) {
      return
    }
    WriteConfig(target.game, target.id, mod.key, mod.uniqueId, body)
      .then(() => {
        if (seq === readGen.current) {
          setSaved(true)
          setDirty(false)
          setJsonDraft(body)
        }
      })
      .catch(reportUnexpected)
  }
  return {
    locked,
    tree,
    jsonDraft,
    tab,
    error,
    saved,
    dirty,
    discardOpen,
    setDiscardOpen,
    setTab,
    setJsonDraft,
    applyJson,
    save,
    markTree: (path: readonly number[], next: ConfigNode) => {
      setSaved(false)
      setDirty(true)
      setTree((cur) => (cur ? setAt(cur, path, next) : cur))
    },
    markJson: (value: string) => {
      setSaved(false)
      setDirty(true)
      setJsonDraft(value)
    },
  }
}

function EditorDialog({
  open,
  tree,
  jsonDraft,
  tab,
  error,
  saved,
  discardOpen,
  locked,
  menu,
  onClose,
  onDiscard,
  onKeep,
  onSave,
  onTab,
  onJson,
  onTree,
  onOpenJson,
}: {
  open: boolean
  tree: ConfigNode | null
  jsonDraft: string
  tab: number
  error: string
  saved: boolean
  discardOpen: boolean
  locked: boolean
  menu: ReturnType<typeof useGmcmMenu>
  onClose: () => void
  onDiscard: () => void
  onKeep: () => void
  onSave: () => void
  onTab: (tab: number) => void
  onJson: (value: string) => void
  onTree: (path: readonly number[], next: ConfigNode) => void
  onOpenJson: () => void
}) {
  const { t } = useLingui()
  return (
    <>
      <Dialog open={open} onClose={onClose} fullWidth={true} maxWidth="sm">
        <DialogTitle>{t`Edit config.json`}</DialogTitle>
        <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
          {error ? (
            <Typography role="alert" sx={text}>
              {error}
            </Typography>
          ) : null}
          <Tabs
            value={tab}
            onChange={(_, next: number) => onTab(next)}
            aria-label={t`Settings view`}
          >
            <Tab label={t`Menu`} />
            <Tab label={t`Form`} />
            <Tab label={t`JSON`} />
          </Tabs>
          {tab === 0 && menu.capture ? (
            <MenuPages
              capture={menu.capture}
              pageId={menu.pageId}
              drafts={menu.drafts}
              result={menu.result}
              onPage={menu.setPageId}
              onChange={menu.change}
              onDiscard={menu.discard}
            />
          ) : null}
          {tab === 0 && !menu.capture ? <MenuHint /> : null}
          {tab === 1 ? <Fields tree={tree} onChange={onTree} onOpen={onOpenJson} /> : null}
          {tab === 2 ? (
            <TextField
              size="small"
              fullWidth={true}
              multiline={true}
              minRows={12}
              value={jsonDraft}
              onChange={(e) => onJson(e.target.value)}
              slotProps={{
                htmlInput: { 'aria-label': t`config.json` },
                input: { sx: { fontFamily: MONO, fontSize: 13 } },
              }}
            />
          ) : null}
        </DialogContent>
        <DialogActions>
          {saved ? <Typography sx={{ mr: 'auto', ...text }}>{t`Saved`}</Typography> : null}
          <Button onClick={onClose} sx={noWrap}>{t`Close`}</Button>
          <DisabledReason title={t`Stop the game to change mods.`} disabled={locked}>
            <Button onClick={onSave} disabled={locked || (tab === 1 && !tree)} sx={noWrap}>
              {t`Save`}
            </Button>
          </DisabledReason>
        </DialogActions>
      </Dialog>
      <ConfirmDialog
        open={discardOpen}
        title={t`Discard changes?`}
        confirmLabel={t`Discard`}
        color="error"
        onCancel={onKeep}
        onConfirm={onDiscard}
      />
    </>
  )
}

function ConfigEditor({ mod, open, onClose }: { mod: Mod; open: boolean; onClose: () => void }) {
  const doc = useConfigDoc(mod, open)
  const menu = useGmcmMenu(mod.uniqueId, open)
  const close = () => {
    if (doc.dirty) {
      doc.setDiscardOpen(true)
      return
    }
    onClose()
  }
  return (
    <EditorDialog
      open={open}
      tree={doc.tree}
      jsonDraft={doc.jsonDraft}
      tab={doc.tab}
      error={doc.error}
      saved={doc.saved}
      discardOpen={doc.discardOpen}
      locked={doc.locked}
      menu={menu}
      onClose={close}
      onDiscard={onClose}
      onKeep={() => doc.setDiscardOpen(false)}
      onSave={() => {
        if (doc.tab === 0) {
          menu.save()
          return
        }
        doc.save()
      }}
      onTab={(next) => {
        if (next === 2 && doc.tree) {
          doc.setJsonDraft(stringifyConfig(doc.tree))
        }
        if (next === 1 && doc.tab === 2 && !doc.applyJson()) {
          return
        }
        doc.setTab(next)
      }}
      onJson={doc.markJson}
      onTree={doc.markTree}
      onOpenJson={() => {
        if (doc.tree) {
          doc.setJsonDraft(stringifyConfig(doc.tree))
        }
        doc.setTab(2)
      }}
    />
  )
}

function EditConfigButton({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const [open, setOpen] = useState(false)
  return (
    <>
      <Button
        size="small"
        variant="outlined"
        startIcon={<Pencil size={14} />}
        onClick={() => setOpen(true)}
        sx={noWrap}
      >
        {t`Edit…`}
      </Button>
      <PresetsButton mod={mod} />
      <ConfigEditor mod={mod} open={open} onClose={() => setOpen(false)} />
    </>
  )
}

export { EditConfigButton }
