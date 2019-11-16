package bpa

import (
	"fmt"
)

func BuildBlocks(tasks []*Task, objs []*Object) (blocks []*Block, err error) {
	blocks = make([]*Block, 0)
	var i, j, l int

	// a defined task type may be empty string, tasks which type is empty
	// wouldn't be treated as the same type

	for i = range tasks {
		tk := tasks[i]
		l = len(blocks)

		if l == 0 || tk.Type == "" || blocks[l-1].Type != tk.Type {
			blocks = append(blocks,
				&Block{tk.Type, make([]*Task, 0), make([]*Object, 0), nil})
			l++
		}

		blocks[l-1].Tasks = append(blocks[l-1].Tasks, tk)
	}

	for i = range blocks {
		for j = range objs {
			if objs[j].Type == blocks[i].Type {
				blocks[i].Objects = append(blocks[i].Objects, objs[j])
			}
		}
	}

	for i = 0; i < len(blocks); i++ {
		blocks[i].Index = make([]int, len(blocks[i].Objects))
	}

	return
}

func (block *Block) Clear() (err error) {
	var i int

	if len(block.Index) != len(block.Objects) {
		msg := ".Index doesn't equal to .Object in block \"%v\""
		err = fmt.Errorf(msg, block.GetTaskNames())
		return
	}

	for i = 0; i < len(block.Objects); i++ {
		if block.Index[i] < 0 {
			block.Index[i] = 0
		}

		if block.Index[i] >= len(block.Tasks) {
			block.Index = append(block.Index[:i], block.Index[i+1:]...)
			block.Objects = append(block.Objects[:i], block.Objects[i+1:]...)
			i--
		}
	}

	return
}

func TruncateBlocks(blocks []*Block) (nblock []*Block, err error) {
	var n int

	for n = range blocks {
		if err = blocks[n].Clear(); err != nil {
			return
		}

		if len(blocks[n].Objects) != 0 {
			break
		}
	}

	nblock = blocks[n:]
	return
}

func (block *Block) GetTaskNames() (names []string) {
	names = make([]string, len(block.Tasks))
	var i int

	for i = range block.Tasks {
		names[i] = block.Tasks[i].Name
	}

	return
}
