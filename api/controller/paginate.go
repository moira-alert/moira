package controller

// applyPagination returns entities[page*size:(page+1)*size] if possible.
// If bad page and size are given or out of range than empty slice []T is returned.
func applyPagination[T any](page, size, total int64, entities []T) []T {
	if page < 0 || (page > 0 && size < 0) {
		return make([]T, 0)
	}

	if page >= 0 && size >= 0 {
		start := page * size
		end := start + size

		if start >= total {
			return make([]T, 0)
		} else {
			if end > total {
				end = total
			}

			return entities[start:end]
		}
	}

	return entities
}
