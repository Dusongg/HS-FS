package main

import (
	"fmt"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"strconv"
)

func main() {
	mw := &MyMainWindow{}
	subwd := MySubWindow{}

	if err := (MainWindow{
		Title: "hs_file_searcher",
		// 指定窗口的大小
		MinSize:  Size{},
		AssignTo: &mw.MainWindow,
		Layout: VBox{
			MarginsZero: true,
		},
		Children: []Widget{
			HSplitter{
				MaxSize: Size{0, 50},

				Children: []Widget{
					Label{Text: "目录 / 文件: "},
					LineEdit{
						AssignTo: &mw.file_or_directory,
						MaxSize:  Size{Height: 10, Width: 1},
					},
					PushButton{AssignTo: &mw.load, Text: "解析原子文件与逻辑服务文件内容"},
				},
			},
			HSplitter{
				MaxSize: Size{0, 50},
				Children: []Widget{
					Label{Text: "查找目标: "},
					LineEdit{
						AssignTo: &mw.target,
						MaxSize:  Size{Height: 10, Width: 1},
					},
					Label{Text: "匹配模式: "},
					RadioButtonGroup{
						Buttons: []RadioButton{
							RadioButton{
								Name:     "all",
								Text:     "直接匹配",
								Value:    "1",
								AssignTo: &mw.type_direct_match,
							},
							RadioButton{
								Name:     "all",
								Text:     "正则匹配",
								Value:    "2",
								AssignTo: &mw.type_Regular_match,
							},
						},
					},
					PushButton{AssignTo: &mw.run, Text: "运行"},
				},
			},

			VSplitter{
				Children: []Widget{
					TextEdit{
						AssignTo: &mw.result,
						ReadOnly: true,
						HScroll:  true,
						VScroll:  true},
				},
			},
			Label{Text: "查询结果数量： ", AssignTo: &mw.numLabel},
		},
	}.Create()); err != nil {
		return
	}

	mw.type_direct_match.Clicked().Attach(func() {
		go func() {
			mw.SetType(mw.type_direct_match.Value())
		}()
	})
	mw.type_Regular_match.Clicked().Attach(func() {
		go func() {
			mw.SetType(mw.type_Regular_match.Value())
		}()
	})

	mw.load.Clicked().Attach(func() {
		if err := (Dialog{
			AssignTo: &subwd.Dialog,
			Size:     Size{Width: 200, Height: 150},
			Layout:   VBox{},

			Children: []Widget{
				Label{Text: "输入待解析得目录或文件(以英文逗号分割)"},
				LineEdit{
					AssignTo: &subwd.prase_path,
					MaxSize:  Size{Height: 20, Width: 1},
				},
				PushButton{
					Text: "Reload",
					OnClicked: func() {
						err := clearOutputDir()
						if err != nil {
							fmt.Printf("Error clearing output directory %s: %v\n", outputDir, err)
							walk.MsgBox(subwd, "提示", "Error clearing output directory", walk.MsgBoxIconWarning)
							subwd.Accept()
						}
						general_append(&subwd)
					},
				},
				PushButton{
					Text: "Append",
					OnClicked: func() {
						general_append(&subwd)
					},
				},
				PushButton{
					Text: "Cancel",
					OnClicked: func() {
						subwd.Accept()
					},
				},
			},
		}.Create(mw)); err != nil {
			return
		}
		subwd.Run()
	})

	mw.run.Clicked().Attach(func() {
		switch {
		case mw.file_or_directory.Text() == "":
			walk.MsgBox(mw, "提示", "请输入目标所在文件或目录", walk.MsgBoxIconWarning)
			return
		case mw.target.Text() == "":
			walk.MsgBox(mw, "提示", "请输入查找目标", walk.MsgBoxIconWarning)
			return
		case mw.match_mode == 0:
			walk.MsgBox(mw, "提示", "请选择匹配模式", walk.MsgBoxIconWarning)
			return
		}
		mw.search()
	})

	mw.Run()

}

func general_append(subwd *MySubWindow) {
	if subwd.prase_path.Text() == "" {
		walk.MsgBox(subwd, "提示", "请输入目录或文件", walk.MsgBoxIconWarning)
		return
	}
	prase(subwd.prase_path.Text())
	subwd.Accept()
}

type MySubWindow struct {
	*walk.Dialog
	prase_path *walk.LineEdit
}

type MyMainWindow struct {
	*walk.MainWindow

	type_direct_match  *walk.RadioButton
	type_Regular_match *walk.RadioButton
	run                *walk.PushButton

	file_or_directory *walk.LineEdit
	load              *walk.PushButton
	target            *walk.LineEdit

	out_num *walk.LineEdit
	result  *walk.TextEdit

	match_mode int
	typeLabel  *walk.Label
	numLabel   *walk.Label
}

func (this *MyMainWindow) SetType(type_id interface{}) {
	this.match_mode, _ = strconv.Atoi(type_id.(string))
}
func (this *MyMainWindow) search() {
	var ret []string = _search(this.file_or_directory.Text(), this.target.Text(), this.match_mode)
	var result string
	for _, v := range ret {
		//bug:多个结果换行显示
		result += v + "\n"
	}
	this.result.SetText(result)
	this.numLabel.SetText(strconv.Itoa(len(result)))
}
