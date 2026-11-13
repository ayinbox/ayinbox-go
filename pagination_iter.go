//go:build go1.23

package ayinbox

import (
	"context"
	"iter"
)

// All returns an iterator over the remaining rows, for use with range on
// Go 1.23 and later. An error ends the walk and is yielded with a zero row.
//
//	for c, err := range client.Campaigns.ListAutoPaging(nil).All(ctx) {
//		if err != nil { ... }
//	}
func (p *Pager[T]) All(ctx context.Context) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for p.Next(ctx) {
			if !yield(p.Current(), nil) {
				return
			}
		}
		if err := p.Err(); err != nil {
			var zero T
			yield(zero, err)
		}
	}
}
