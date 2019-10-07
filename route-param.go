package vgrouter

// PathParam is parameter key/value pair extracted from a URL path.
type PathParam struct {
	Key   string
	Value string
}

// PathParamList is a slice of PathParam.
type PathParamList []PathParam

// ByName returns the named parameter value or an empty string if not found.
func (ps PathParamList) ByName(name string) string {
	for i := range ps {
		if ps[i].Key == name {
			return ps[i].Value
		}
	}
	return ""
}
