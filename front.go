package main

import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"log"
	"strconv"
	"strings"
)

const (
	NONE_MATCH  = "NONE_MATCH"
	EXACT_MATCH = "EXACT_MATCH"
	REGEX_MATCH = "REGEX_MATCH"
)

func main() {
	walk.AppendToWalkInit(func() {
		walk.FocusEffect, _ = walk.NewBorderGlowEffect(walk.RGB(0, 63, 255))
		walk.InteractionEffect, _ = walk.NewDropShadowEffect(walk.RGB(63, 63, 63))
		walk.ValidationErrorEffect, _ = walk.NewBorderGlowEffect(walk.RGB(255, 0, 0))
	})

	mw := &MyMainWindow{}
	subwd := &MySubWindow{}

	if err := (MainWindow{
		Title: "hs_file_searcher",
		// 指定窗口的大小
		MinSize:  Size{},
		AssignTo: &mw.MainWindow,
		Layout: VBox{
			MarginsZero: true,
		},
		OnDropFiles: func(files []string) {
			mw.file_or_directory.SetText(strings.Join(files, "\r\n"))
		},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 3},
				Children: []Widget{
					Label{
						Text: "目录 / 文件: ",
					},
					LineEdit{
						OnMouseDown: func(x, y int, button walk.MouseButton) {
							mw.file_or_directory.SetText("")
						},
						Text:     "Drop or Paste files here",
						AssignTo: &mw.file_or_directory,
					},
					PushButton{
						Text: "Browser",
						OnClicked: func() {
							browser(mw)
						},
					},
				},
			},
			Composite{
				Layout: Grid{Columns: 10},
				Children: []Widget{
					Label{
						Text: "查找目标: ",
					},
					LineEdit{
						AssignTo: &mw.target,
					},
					Label{
						Text: "匹配模式: ",
					},
					RadioButtonGroup{
						Buttons: []RadioButton{
							{
								Name:     "exact_match",
								Text:     "精确匹配",
								AssignTo: &mw.type_exact_match,
							},
							{
								Name:     "regular_match",
								Text:     "正则匹配",
								AssignTo: &mw.type_regular_match,
							},
						},
					},
				},
			},
			//下拉选项案例
			//HSplitter{
			//	Children: []Widget{
			//		Label{
			//			Text: "Preferred Food:",
			//		},
			//		ComboBox{
			//			Editable: true,
			//			Value:    Bind("PreferredFood"),
			//			Model:    []string{"Fruit", "Grass", "Fish", "Meat"},
			//		},
			//	},
			//},

			TextEdit{

				AssignTo: &mw.result,
				ReadOnly: true,
				HScroll:  true,
				VScroll:  true,
			},
			Label{Text: "查询结果数量： ", AssignTo: &mw.numLabel},

			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{AssignTo: &mw.run, Text: "Run"},

					PushButton{AssignTo: &mw.load, Text: "Prase"},
					PushButton{Text: "Cancel", OnClicked: func() {
						mw.Close()
					}},
				},
			},
		},
	}.Create()); err != nil {
		return
	}

	mw.type_exact_match.Clicked().Attach(func() {
		go func() {
			mw.SetType(EXACT_MATCH)
		}()
	})
	mw.type_regular_match.Clicked().Attach(func() {
		go func() {
			mw.SetType(REGEX_MATCH)
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
							log.Printf("Error clearing output directory %s: %v\n", outputDir, err)
							walk.MsgBox(subwd, "提示", "Error clearing output directory", walk.MsgBoxIconWarning)
							subwd.Accept()
						}
						general_append(subwd)
					},
				},
				PushButton{
					Text: "Append",
					OnClicked: func() {
						general_append(subwd)
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
		case mw.match_mode == NONE_MATCH:
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

	type_exact_match   *walk.RadioButton
	type_regular_match *walk.RadioButton
	run                *walk.PushButton

	file_or_directory *walk.LineEdit
	load              *walk.PushButton
	target            *walk.LineEdit

	out_num *walk.LineEdit
	result  *walk.TextEdit

	match_mode string
	typeLabel  *walk.Label
	numLabel   *walk.Label
}

func (this *MyMainWindow) search() {
	var ret []string = _search(this.file_or_directory.Text(), this.target.Text(), this.match_mode)
	var result string
	for _, v := range ret {
		//bug:多个结果换行显示
		result = result + v + "\r\n"

	}
	this.result.SetText(result)
	this.numLabel.SetText(strconv.Itoa(len(result)))
	log.Println(result)
}

func (this *MyMainWindow) SetType(mode string) {
	this.match_mode = mode
}
