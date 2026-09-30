import { create } from 'zustand'

export type SettingsSection = 'appearance' | 'about'

export const useSettingsDialog = create<{
  open: boolean
  section: SettingsSection
  openSettings: (section?: SettingsSection) => void
  close: () => void
  setSection: (section: SettingsSection) => void
}>((set) => ({
  open: false,
  section: 'appearance',
  openSettings: (section) => set((s) => ({ open: true, section: section ?? s.section })),
  close: () => set({ open: false }),
  setSection: (section) => set({ section }),
}))

export const openSettings = (section?: SettingsSection) =>
  useSettingsDialog.getState().openSettings(section)
