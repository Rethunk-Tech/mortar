import type { Diff } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/syncsvc/models.ts'

/** A diff and the revision of the offer it was read for. */
export interface ShownDiff {
  diff: Diff
  revision: string
}

/** The revision an answer targets: the one on screen when a diff is open, so Apply never takes a newer payload. */
export const revisionToAnswer = (offerRevision: string, shown: ShownDiff | null) =>
  shown?.revision ?? offerRevision

/** An open diff that belongs to an older revision of the offer. */
export const diffIsStale = (offerRevision: string, shown: ShownDiff | null) =>
  shown !== null && shown.revision !== offerRevision
