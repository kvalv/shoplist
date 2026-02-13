package devtools

import (
	"context"
	"os"
	"time"
)

// If file does not exists, no element is sent on the channel
func WatchFile(ctx context.Context, path string) chan struct{} {
	ch := make(chan struct{})
	finfo, err := os.Stat(path)
	if err != nil {
		close(ch)
		return ch
	}
	t0 := finfo.ModTime()

	go func() {
		defer close(ch)
		tick := time.Tick(500 * time.Millisecond)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick:
				finfo, err := os.Stat(path)
				if err != nil {
					continue
				}
				t1 := finfo.ModTime()
				if t1.After(t0) {
					ch <- struct{}{}
					t0 = t1
				}
			}
		}
	}()
	return ch
}
