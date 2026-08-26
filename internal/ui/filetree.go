package ui

import (
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/sagnikc395/open-obsidian/internal/vault"
)

// FileTree wraps widget.Tree for vault files.
type FileTree struct {
	widget.BaseWidget
	tree     *widget.Tree
	store    *vault.Store
	onSelect func(rel string)
}

func NewFileTree(store *vault.Store, onSelect func(rel string)) *FileTree {
	f := &FileTree{store: store, onSelect: onSelect}
	f.ExtendBaseWidget(f)
	f.buildTree()
	return f
}

type treeNode struct {
	name     string
	rel      string // full rel for files; dir path with / suffix for dirs
	isDir    bool
	children []*treeNode
}

func (f *FileTree) buildTree() {
	root := f.buildNodes()
	f.tree = widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			if uid == "" {
				var ids []string
				for _, c := range root.children {
					ids = append(ids, c.rel)
				}
				return ids
			}
			n := findNode(root, uid)
			if n == nil {
				return nil
			}
			var ids []string
			for _, c := range n.children {
				ids = append(ids, c.rel)
			}
			return ids
		},
		func(uid widget.TreeNodeID) bool {
			if uid == "" {
				return true
			}
			n := findNode(root, uid)
			return n != nil && n.isDir
		},
		func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(uid widget.TreeNodeID, branch bool, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			n := findNode(root, uid)
			if n != nil {
				lbl.SetText(n.name)
			} else {
				lbl.SetText(uid)
			}
		},
	)
	f.tree.OnSelected = func(uid widget.TreeNodeID) {
		n := findNode(root, uid)
		if n != nil && !n.isDir && f.onSelect != nil {
			f.onSelect(n.rel)
		}
	}
}

func (f *FileTree) buildNodes() *treeNode {
	root := &treeNode{name: "vault", rel: "", isDir: true}
	files := f.store.Files()
	// also include empty dirs? For MVP only files + implied dirs
	dirMap := map[string]*treeNode{"": root}
	var dirs []string
	for _, rel := range files {
		dir := filepath.Dir(rel)
		if dir == "." {
			dir = ""
		}
		// ensure dir chain
		parts := strings.Split(dir, "/")
		cur := ""
		for _, p := range parts {
			if p == "" {
				continue
			}
			parent := cur
			if cur == "" {
				cur = p
			} else {
				cur = cur + "/" + p
			}
			if _, ok := dirMap[cur]; !ok {
				node := &treeNode{name: p, rel: cur + "/", isDir: true}
				dirMap[cur] = node
				parentNode := dirMap[parent]
				parentNode.children = append(parentNode.children, node)
				dirs = append(dirs, cur)
			}
		}
		parentKey := dir
		if parentKey == "" {
			parentKey = ""
		}
		parent := dirMap[parentKey]
		leaf := &treeNode{name: filepath.Base(rel), rel: rel, isDir: false}
		parent.children = append(parent.children, leaf)
	}
	// sort children
	for _, n := range dirMap {
		sort.Slice(n.children, func(i, j int) bool {
			if n.children[i].isDir != n.children[j].isDir {
				return n.children[i].isDir // dirs first
			}
			return n.children[i].name < n.children[j].name
		})
	}
	return root
}

func findNode(root *treeNode, uid string) *treeNode {
	if root.rel == uid {
		return root
	}
	var search func(n *treeNode) *treeNode
	search = func(n *treeNode) *treeNode {
		for _, c := range n.children {
			if c.rel == uid {
				return c
			}
			if c.isDir {
				if found := search(c); found != nil {
					return found
				}
			}
		}
		return nil
	}
	return search(root)
}

func (f *FileTree) RefreshTree() {
	f.buildTree()
	f.Refresh()
}

func (f *FileTree) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(f.tree)
}
