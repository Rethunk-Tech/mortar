type ViewMode = 'grid' | 'list'

const isViewMode = (v: unknown): v is ViewMode => v === 'list' || v === 'grid'

export type { ViewMode }
export { isViewMode }
