import { msg } from '@lingui/core/macro'
import { InstallOverlay } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { AnswerOverlay } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useInstall } from './store.ts'

/** Answers an optional file's question: its folder from, laid at to inside its main file's folder. */
export async function chooseOverlay(from: string, to: string) {
  const session = useInstall.getState().remap
  if (!session) {
    return
  }
  useInstall.setState({ remap: null })
  try {
    if (session.queueId) {
      await AnswerOverlay(session.queueId, from, to)
      return
    }
    const res = await InstallOverlay(
      session.game,
      session.profileId,
      session.key,
      from,
      to,
      session.source,
    )
    useProfiles.getState().replace(res.profile)
    const names = (res.added ?? []).join(', ')
    useToasts
      .getState()
      .push({ kind: 'success', title: i18n._(msg`Added ${names} to ${session.profileName}`) })
  } catch (e) {
    toastError(i18n._(msg`Could not add the optional file`), e)
  }
}
