import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Alert, Menu } from '@mui/material'
import { usePopoverAnchor } from '@rethunk/react-load-hooks'
import { Bug, Ellipsis, FolderOpen, Pencil, ScrollText, SquareCode, Trash2 } from 'lucide-react'
import { useState } from 'react'
import type { ProjectInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  Forget,
  OpenEditor,
  OpenProject,
  SetWatch,
  Trash as TrashProject,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { StartDebugging } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import type { DevProject } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { persist, saveSetting } from '../settings/persist.ts'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { IconAction } from '../shell/IconAction.tsx'
import { MenuAction, MenuCheck } from '../shell/MenuAction.tsx'
import { MenuRule } from '../shell/TitleMenu.tsx'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useTrustAsk } from './builds.ts'
import { EditProjectDialog } from './EditProjectDialog.tsx'
import { useIsAsset } from './kinds.ts'
import { showModLog } from './modCosts.ts'
import { gitSummary } from './projectSummary.ts'
import { ReleaseDialog } from './ReleaseDialog.tsx'
import { ReleaseItems } from './ReleaseItems.tsx'
import { reportTrashFailed } from './trashFailed.ts'
import { UploadDialog } from './UploadDialog.tsx'

// Asking before a project leaves the list, and whether its folder goes to the trash with it.
function RemoveProject({
  project,
  summary,
  open,
  onClose,
  onConfirm,
}: {
  project: DevProject
  summary: string
  open: boolean
  onClose: () => void
  onConfirm: (trash: boolean) => void
}) {
  const { t } = useLingui()
  const [trashFolder, setTrashFolder] = useState(false)
  const close = () => {
    onClose()
    setTrashFolder(false)
  }
  return (
    <ConfirmationDialog
      open={open}
      tone={trashFolder ? 'danger' : 'warning'}
      title={t`Remove ${{ name: project.name }} from the list?`}
      body={
        trashFolder
          ? t`Its folder moves to the trash, where you can restore it, and the mod leaves the profile.`
          : t`Its folder stays where it is; Mortar stops building it and installing it into the profile.`
      }
      confirmLabel={trashFolder ? t`Remove and move to trash` : t`Remove from list`}
      onCancel={close}
      onConfirm={() => {
        close()
        onConfirm(trashFolder)
      }}
      checkbox={{
        label: t`Also move its folder to the trash`,
        checked: trashFolder,
        onChange: setTrashFolder,
      }}
    >
      {trashFolder && summary !== '' ? (
        <Alert variant="compact" severity="warning">
          {t`Work that is only in this folder goes with it: ${{ summary }}.`}
        </Alert>
      ) : null}
    </ConfirmationDialog>
  )
}

// The menu's rows: what a row opens is the caller's, and its dialogs live in ProjectMenu below.
function MenuRows({
  game,
  project,
  info,
  code,
  anchor,
  onClose,
  open,
}: {
  game: string
  project: DevProject
  info: ProjectInfo | undefined
  // A project made only of files has no build, plugin or log.
  code: boolean
  anchor: HTMLElement | null
  onClose: () => void
  open: Openers
}) {
  const { t, i18n } = useLingui()
  const act = (fn: () => unknown) => () => {
    onClose()
    fn()
  }
  // One start with the game's Mono debugger open; the toast says where to attach.
  const debug = () =>
    StartDebugging(game, project.profile)
      .then((address) =>
        useToasts.getState().push({
          kind: 'info',
          title: i18n._(msg`Starting for debugging`),
          body: i18n._(msg`Attach your editor's debugger to ${{ address }}.`),
        }),
      )
      .catch(reportError(t`Could not start the game`))
  return (
    <Menu anchorEl={anchor} open={anchor !== null} onClose={onClose}>
      <MenuAction
        icon={<Pencil size={16} aria-hidden={true} />}
        label={t`Edit details…`}
        disabled={!info}
        reason={t`Reading the project…`}
        onClick={act(open.edit)}
      />
      <MenuAction
        icon={<SquareCode size={16} aria-hidden={true} />}
        label={t`Open in editor`}
        onClick={act(() =>
          OpenEditor(game, project.dir).catch(reportError(t`Could not open the editor`)),
        )}
      />
      <MenuAction
        icon={<FolderOpen size={16} aria-hidden={true} />}
        label={t`Open folder`}
        onClick={act(() =>
          OpenProject(game, project.dir).catch(reportError(t`Could not open the folder`)),
        )}
      />
      {code ? <MenuRule /> : null}
      {code ? (
        <MenuAction
          icon={<Bug size={16} aria-hidden={true} />}
          label={t`Start for debugging`}
          onClick={act(debug)}
        />
      ) : null}
      {code ? (
        <MenuAction
          icon={<ScrollText size={16} aria-hidden={true} />}
          label={t`Show log`}
          disabled={!info}
          onClick={act(() => info && showModLog(info))}
        />
      ) : null}
      <MenuRule />
      <ReleaseItems
        act={act}
        released={info?.released}
        onRelease={open.release}
        onUpload={open.upload}
      />
      {code ? <MenuRule /> : null}
      {code ? (
        <MenuCheck
          checked={Boolean(project.watch)}
          label={t`Build on save`}
          onClick={() => saveSetting(() => SetWatch(game, project.dir, !project.watch))}
        />
      ) : null}
      <MenuRule />
      <MenuAction
        icon={<Trash2 size={16} aria-hidden={true} />}
        label={t`Remove from list`}
        tone="error"
        onClick={act(open.remove)}
      />
    </Menu>
  )
}

interface Openers {
  edit: () => void
  release: () => void
  upload: (version: string) => void
  remove: () => void
}

/**
 * A project card's ⋯ button with its menu and the dialogs its rows open. Its own component, so opening one redraws
 * this and not the card's name, tile and chips.
 */
export function ProjectMenu({
  game,
  project,
  info,
  name,
}: {
  game: string
  project: DevProject
  info: ProjectInfo | undefined
  // The name the card shows, which the release's dialogs title themselves with.
  name: string
}) {
  const { t } = useLingui()
  const { anchor, show, hide } = usePopoverAnchor()
  const [editing, setEditing] = useState(false)
  const [releasing, setReleasing] = useState(false)
  // The version a local release offers to upload; empty when none is open.
  const [uploading, setUploading] = useState('')
  const [removing, setRemoving] = useState(false)
  // A project made only of files has no build, plugin or log.
  const code = !useIsAsset(project.dir)
  return (
    <>
      <IconAction
        label={t`More actions`}
        icon={<Ellipsis size={18} />}
        menu={true}
        onClick={show}
      />
      <MenuRows
        game={game}
        project={project}
        info={info}
        code={code}
        anchor={anchor}
        onClose={hide}
        open={{
          edit: () => setEditing(true),
          release: () =>
            project.untrusted
              ? useTrustAsk.setState({ ask: { game, dir: project.dir, pack: false } })
              : setReleasing(true),
          upload: setUploading,
          remove: () => setRemoving(true),
        }}
      />
      <RemoveProject
        project={project}
        summary={gitSummary(info?.git)}
        open={removing}
        onClose={() => setRemoving(false)}
        onConfirm={(trash) => {
          if (trash) {
            TrashProject(game, project.dir).catch(reportTrashFailed)
          } else {
            persist(() => Forget(game, project.dir), t`Could not remove the project`)
          }
        }}
      />
      {editing && info ? (
        <EditProjectDialog
          game={game}
          dir={project.dir}
          info={info}
          onClose={() => setEditing(false)}
        />
      ) : null}
      {releasing ? (
        <ReleaseDialog
          game={game}
          dir={project.dir}
          name={name}
          onClose={() => setReleasing(false)}
          onLocal={(released) => {
            setReleasing(false)
            setUploading(released)
          }}
        />
      ) : null}
      {uploading ? (
        <UploadDialog
          game={game}
          dir={project.dir}
          name={name}
          version={uploading}
          onClose={() => setUploading('')}
        />
      ) : null}
    </>
  )
}
