package conc

func TryDoN[T any](n int, c chan T, do func(t T)) {
	for i := 0; i < n; i++ {
		TryDo(c, do)
	}
}

func TryDo[T any](c chan T, do func(t T)) {
	select {
	case r := <-c:
		do(r)
	default:
	}
}

func Check[T any](c chan T) (T, bool) {
	select {
	case r := <-c:
		return r, true
	default:
	}
	var t T
	return t, false
}

func TrySend[T any](v T, c chan T) bool {
	select {
	case c <- v:
		return true
	default:
	}
	return false
}
