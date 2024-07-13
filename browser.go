// Copyright 2011 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type Directory struct {
	name     string
	parent   *Directory
	children []*Directory
}

func NewDirectory(name string, parent *Directory) *Directory {
	return &Directory{name: name, parent: parent}
}

var _ walk.TreeItem = new(Directory)

func (d *Directory) Text() string {
	return d.name
}

func (d *Directory) Parent() walk.TreeItem {
	if d.parent == nil {
		// We can't simply return d.parent in this case, because the interface
		// value then would not be nil.
		return nil
	}

	return d.parent
}

func (d *Directory) ChildCount() int {
	if d.children == nil {
		// It seems this is the first time our child count is checked, so we
		// use the opportunity to populate our direct children.
		if err := d.ResetChildren(); err != nil {
			log.Print(err)
		}
	}

	return len(d.children)
}

func (d *Directory) ChildAt(index int) walk.TreeItem {
	return d.children[index]
}

func (d *Directory) Image() interface{} {
	return d.Path()
}

func (d *Directory) ResetChildren() error {
	d.children = nil

	dirPath := d.Path()

	if err := filepath.Walk(d.Path(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if info == nil {
				return filepath.SkipDir
			}
		}

		name := info.Name()

		if !info.IsDir() || path == dirPath || shouldExclude(name) {
			return nil
		}

		d.children = append(d.children, NewDirectory(name, d))

		return filepath.SkipDir
	}); err != nil {
		return err
	}

	return nil
}

func (d *Directory) Path() string {
	elems := []string{d.name}

	dir, _ := d.Parent().(*Directory)

	for dir != nil {
		elems = append([]string{dir.name}, elems...)
		dir, _ = dir.Parent().(*Directory)
	}

	return filepath.Join(elems...)
}

type DirectoryTreeModel struct {
	walk.TreeModelBase
	roots []*Directory
}

var _ walk.TreeModel = new(DirectoryTreeModel)

func NewDirectoryTreeModel() (*DirectoryTreeModel, error) {
	model := new(DirectoryTreeModel)

	drives, err := walk.DriveNames()
	if err != nil {
		return nil, err
	}

	for _, drive := range drives {
		switch drive {
		case "A:\\", "B:\\":
			continue
		}

		model.roots = append(model.roots, NewDirectory(drive, nil))
	}

	return model, nil
}

func (*DirectoryTreeModel) LazyPopulation() bool {
	// We don't want to eagerly populate our tree view with the whole file system.
	return true
}

func (m *DirectoryTreeModel) RootCount() int {
	return len(m.roots)
}

func (m *DirectoryTreeModel) RootAt(index int) walk.TreeItem {
	return m.roots[index]
}

type FileInfo struct {
	Name     string
	Size     int64
	Modified time.Time
}

type FileInfoModel struct {
	walk.SortedReflectTableModelBase
	dirPath string
	items   []*FileInfo
}

var _ walk.ReflectTableModel = new(FileInfoModel)

func NewFileInfoModel() *FileInfoModel {
	return new(FileInfoModel)
}

func (m *FileInfoModel) Items() interface{} {
	return m.items
}

func (m *FileInfoModel) SetDirPath(dirPath string) error {
	m.dirPath = dirPath
	m.items = nil

	if err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if info == nil {
				return filepath.SkipDir
			}
		}

		name := info.Name()

		if path == dirPath || shouldExclude(name) {
			return nil
		}

		item := &FileInfo{
			Name:     name,
			Size:     info.Size(),
			Modified: info.ModTime(),
		}

		m.items = append(m.items, item)

		if info.IsDir() {
			return filepath.SkipDir
		}

		return nil
	}); err != nil {
		return err
	}

	m.PublishRowsReset()

	return nil
}

func (m *FileInfoModel) Image(row int) interface{} {
	return filepath.Join(m.dirPath, m.items[row].Name)
}

func shouldExclude(name string) bool {
	switch name {
	case "System Volume Information", "pagefile.sys", "swapfile.sys":
		return true
	}

	return false
}

func browser(wd interface{}) {
	var mainWindow *walk.MainWindow
	var splitter *walk.Splitter
	var treeView *walk.TreeView
	var tableView *walk.TableView
	var text_path *walk.TextEdit

	treeModel, err := NewDirectoryTreeModel()
	if err != nil {
		log.Fatal(err)
	}
	tableModel := NewFileInfoModel()

	if err := (MainWindow{
		AssignTo: &mainWindow,
		Title:    "Walk File Browser Example",
		MinSize:  Size{600, 400},
		Size:     Size{1024, 640},
		Layout:   VBox{MarginsZero: true},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2},
				Children: []Widget{
					TextEdit{
						AssignTo: &text_path,
						MinSize:  Size{Width: 900, Height: 2},
						MaxSize:  Size{Width: 900, Height: 2},
					},
					PushButton{
						Text: "OK",
						OnClicked: func() {
							mainWindow.Close()
						},
					},
				},
			},
			HSplitter{
				AssignTo: &splitter,
				Children: []Widget{
					TreeView{
						AssignTo: &treeView,
						Model:    treeModel,
						OnCurrentItemChanged: func() {
							dir := treeView.CurrentItem().(*Directory)
							if err := tableModel.SetDirPath(dir.Path()); err != nil {
								walk.MsgBox(
									mainWindow,
									"Error",
									err.Error(),
									walk.MsgBoxOK|walk.MsgBoxIconError)
							}
						},
					},
					TableView{
						AssignTo:      &tableView,
						StretchFactor: 2,
						Columns: []TableViewColumn{
							TableViewColumn{
								DataMember: "Name",
								Width:      192,
							},
							TableViewColumn{
								DataMember: "Size",
								Format:     "%d",
								Alignment:  AlignFar,
								Width:      64,
							},
							TableViewColumn{
								DataMember: "Modified",
								Format:     "2006-01-02 15:04:05",
								Width:      120,
							},
						},
						Model: tableModel,
						OnCurrentIndexChanged: func() {
							if index := tableView.CurrentIndex(); index > -1 {
								name := tableModel.items[index].Name
								dir := treeView.CurrentItem().(*Directory)
								path := filepath.Join(dir.Path(), name)

								switch wd.(type) {
								//查询:单个路径
								case *MyMainWindow:
									err := wd.(*MyMainWindow).file_or_directory.SetText(path)
									text_path.SetText(path)
									if err != nil {
										walk.MsgBox(mainWindow, "提示", "路径输入错误", walk.MsgBoxIconWarning)
									}
								//解析：可能多条路径
								case *MySubWindow:
									input_path := wd.(*MySubWindow).prase_path
									if input_path.Text() == "" {
										err := input_path.SetText(path)
										text_path.SetText(path)
										if err != nil {
											walk.MsgBox(mainWindow, "提示", "路径输入错误", walk.MsgBoxIconWarning)
										}
									} else {
										err := input_path.SetText(input_path.Text() + "," + path)
										text_path.SetText(text_path.Text() + "," + path)
										if err != nil {
											walk.MsgBox(mainWindow, "提示", "路径输入错误", walk.MsgBoxIconWarning)

										}
									}
								}
							}

						},
					},
				},
			},
		},
	}.Create()); err != nil {
		log.Fatal(err)
	}

	splitter.SetFixed(treeView, true)
	splitter.SetFixed(tableView, true)

	mainWindow.Run()
}
