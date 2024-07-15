package filetree

import "fmt"

type UnionFileTree struct {
	trees []ReadWriter
}

func NewUnionFileTree() *UnionFileTree {
	return &UnionFileTree{
		trees: make([]ReadWriter, 0),
	}
}

func (u *UnionFileTree) PushTree(t ReadWriter) {
	u.trees = append(u.trees, t)
}

func (u *UnionFileTree) Squash() (ReadWriter, error) {
	switch len(u.trees) {
	case 0:
		return New(), nil
	case 1:
		// important: use clone over copy to reduce memory footprint. If callers need a distinct tree they can call
		// copy after the fact.
		return u.trees[0].Clone()
	}

	var squashedTree ReadWriter
	var err error
	for layerIdx, refTree := range u.trees {
		if layerIdx == 0 {
			// important: use clone over copy to reduce memory footprint. If callers need a distinct tree they can call
			// copy after the fact.
			squashedTree, err = refTree.Clone()
			if err != nil {
				return nil, err
			}
			continue
		}

		if err = squashedTree.Merge(refTree); err != nil {
			return nil, fmt.Errorf("unable to squash layer=%d : %w", layerIdx, err)
		}
	}
	return squashedTree, nil
}
