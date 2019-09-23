package vgrouter

type QueryUpdater interface {
	QueryUpdate()
}

type QueryUpdaterRef struct {
	QueryUpdater // embed QueryUpdater
}

func (h *QueryUpdaterRef) QueryUpdaterSet(o QueryUpdater) {
	h.QueryUpdater = o
}

type QueryUpdaterSetter interface {
	QueryUpdaterSet(QueryUpdater)
}
