import { reportUnexpected } from '../../toasts/report.ts'
import { loadDetails } from '../nexusDetails.ts'

export async function loadAllDetails(
  game: string,
  ids: number[],
  done: (loading: boolean) => void,
) {
  done(true)
  try {
    for (const id of ids) {
      await loadDetails(game, id)
    }
  } catch (e) {
    reportUnexpected(e)
  } finally {
    done(false)
  }
}
