package game

// CatalogChangedEvent is emitted with the new serial when a catalog fetched after start replaces the one Mortar
// started with, so the window reads its game list again.
const CatalogChangedEvent = "catalog:changed"

// CatalogStaleEvent is emitted after a catalog refresh with a CatalogStale payload when the catalog in use was signed
// more than components.StaleAfter ago, so the window can offer a retry. CatalogRetryEvent comes back from the window.
const (
	CatalogStaleEvent = "catalog:stale"
	CatalogRetryEvent = "catalog:retry"
)

// CatalogStale is the CatalogStaleEvent payload: when the catalog in use was signed.
type CatalogStale struct {
	Issued string `json:"issued"`
}
