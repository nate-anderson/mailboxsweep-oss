package batch

func WithLimit[T any](slice []T, limit int) [][]T {
	res := [][]T{}

	if len(slice) < limit {
		return [][]T{slice}
	}

	chunk := make([]T, 0, limit)
	for _, item := range slice {
		chunk = append(chunk, item)
		if len(chunk) == limit {
			res = append(res, chunk)
			chunk = make([]T, 0, limit)
		}
	}

	if len(chunk) > 0 {
		res = append(res, chunk)
	}

	return res
}
