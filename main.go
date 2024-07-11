package main

import (
	"fmt"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"strconv"
)

func main() {
	mw := &MyMainWindow{}
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
					Label{Text: "路径 / 文件: "},
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
					Label{Text: "      功能:   "},
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
			mw.SetType(mw.type_direct_match.Value(), "直接匹配")
		}()
	})
	mw.type_Regular_match.Clicked().Attach(func() {
		go func() {
			mw.SetType(mw.type_Regular_match.Value(), "正则匹配")
		}()
	})

	mw.load.Clicked().Attach(func() {
		var subWindow *walk.Dialog
		var prase_path *walk.LineEdit
		Dialog{
			AssignTo: &subWindow,
			Size:     Size{Width: 200, Height: 150},
			Layout:   VBox{},

			Children: []Widget{
				Label{Text: "重新解析耗时较长，是否要继续"},
				PushButton{
					Text: "继续",
					OnClicked: func() {
						var subWindow2 *walk.Dialog
						Dialog{
							AssignTo: &subWindow2,
							Size:     Size{Width: 200, Height: 150},
							Layout:   VBox{},

							Children: []Widget{
								Label{Text: "输入待解析得目录(多个目录用英文逗号分割)"},
								LineEdit{
									AssignTo: &prase_path,
									MaxSize:  Size{Height: 20, Width: 1},
								},
								PushButton{
									Text: "OK",
									OnClicked: func() {
										prase(prase_path.Text())
										subWindow2.Accept()
									},
								},
							},
						}.Run(mw)
						subWindow.Accept()
					},
				},
				PushButton{
					Text: "退出",
					OnClicked: func() {
						subWindow.Accept()
					},
				},
			},
		}.Run(mw)
	})

	mw.run.Clicked().Attach(func() {
		if mw.match_mode == 0 {
			walk.MsgBox(mw, "提示", "请选择搜索模式", walk.MsgBoxIconWarning)
			return
		}
		mw.search()
	})

	mw.Run()

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

func (this *MyMainWindow) SetType(type_id interface{}, str string) {
	fmt.Println(type_id)
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
