import type { ImportRequest } from './store.ts'
import type { Tab } from './useImportFlow.ts'

/** Fills the import dialog from what opened it: a collection update, another manager's profile, a link, a nearby
 * Mortar's payload or a file. */
export async function seedPreview(
  request: ImportRequest,
  flow: {
    setTab: (tab: Tab) => void
    setText: (text: string) => void
    previewLink: (value: string) => Promise<void>
    previewFile: (file: string) => Promise<void>
    previewData: (data: string) => Promise<void>
    previewExternal: (value: NonNullable<ImportRequest['external']>) => Promise<void>
    previewCollectionUpdate: () => Promise<void>
  },
) {
  flow.setTab(request.tab === 'link' ? 'link' : 'file')
  if (request.collectionUpdate) {
    await flow.previewCollectionUpdate()
    return
  }
  if (request.external) {
    await flow.previewExternal(request.external)
    return
  }
  if (request.seed && request.tab === 'link') {
    flow.setText(request.seed)
    await flow.previewLink(request.seed)
    return
  }
  if (request.seed && request.tab === 'data') {
    await flow.previewData(request.seed)
    return
  }
  if (request.seed) {
    await flow.previewFile(request.seed)
  }
}
