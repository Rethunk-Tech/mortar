import { HistoryPanel } from './HistoryPanel.tsx'

export function HistoryDialog({
  profileId,
  open,
  onClose,
}: {
  profileId: string
  open: boolean
  onClose: () => void
}) {
  return <HistoryPanel profileId={profileId} open={open} onClose={onClose} />
}
