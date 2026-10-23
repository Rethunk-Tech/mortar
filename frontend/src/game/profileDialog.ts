import { create } from 'zustand'

export type ProfileDialogKind =
  | 'edit'
  | 'editLaunch'
  | 'history'
  | 'health'
  | 'compare'
  | 'merge'
  | 'delete'
  | 'savedGroup'
  | 'farm'
  | 'folder'
  | 'backup'
  | 'saveTemplate'
  | 'applyTemplate'
  | 'manageTemplates'

// The profile dialogs the profile menus open. They live in one host above every menu, so a menu can unmount when it
// closes; a menu that held them had to stay mounted, with all its items, on every profile row.
export const useProfileDialog = create<{
  kind: ProfileDialogKind | null
  profileId: string
  open: boolean
  seq: number
}>(() => ({ kind: null, profileId: '', open: false, seq: 0 }))

export function openProfileDialog(kind: ProfileDialogKind, profileId: string) {
  useProfileDialog.setState((s) => ({ kind, profileId, open: true, seq: s.seq + 1 }))
}

export function closeProfileDialog() {
  useProfileDialog.setState({ open: false })
}
