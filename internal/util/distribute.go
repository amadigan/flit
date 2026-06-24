package util

func Distribute[T any](inch <-chan T, outchs ...chan<- T) {
	go func() {
		defer func() {
			for _, ch := range outchs {
				close(ch)
			}
		}()

		for item := range inch {
			for _, ch := range outchs {
				ch <- item
			}
		}
	}()
}
