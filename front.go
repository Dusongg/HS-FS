package main

import (
	"fmt"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	NONE_MATCH  = "NONE_MATCH"
	EXACT_MATCH = "EXACT_MATCH"
	REGEX_MATCH = "REGEX_MATCH"
)

func main() {

	transfer, err := loadTransferFromFile() //from parse.go  ,最开始无法加载
	log.Printf("transfer size :%d", len(transfer))
	if err != nil {
		log.Println(err)
	}
	//results_list := NewResultsListModel(nil) //from search.go
	tablemodel := NewResultInfoModel()

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
						//bug
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

			TableView{
				AssignTo:         &mw.res_view,
				Model:            tablemodel,
				AlternatingRowBG: true,
				ColumnsOrderable: true,
				OnCurrentIndexChanged: func() {
					if index := mw.res_view.CurrentIndex(); index > -1 {
						target_file := extractLastBracketContent(tablemodel.results[index].call_chain) //拿掉调用链的最后一个函数
						log.Printf("open : %s", target_file)

						if source_file, exists := transfer[target_file]; exists {
							cmd := exec.Command("cmd", "/c", "start", "", source_file)
							if err := cmd.Run(); err != nil {
								walk.MsgBox(mw, "报错", err.Error(), walk.MsgBoxIconError)
							}
						} else {
							walk.MsgBox(mw, "报错", fmt.Sprintf("can not find source file of: %s", tablemodel.results[index]), walk.MsgBoxIconError)
						}
					}
				},
				Columns: []TableViewColumn{
					TableViewColumn{
						DataMember: "call_chain",
						Width:      700,
					},
					TableViewColumn{
						DataMember: "target_row_nums",
						Width:      230,
					},
				},

				OnMouseDown: func(x, y int, button walk.MouseButton) {
				},
				OnMouseMove: func(x, y int, button walk.MouseButton) {
					//index := mw.res_list.CurrentIndex()

				},
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
			MinSize:  Size{Width: 500, Height: 200},
			Layout:   VBox{},

			Children: []Widget{

				Composite{
					Layout: Grid{Columns: 10},
					Children: []Widget{
						LineEdit{
							Text:     "输入待解析得目录或文件(以英文逗号分割)",
							AssignTo: &subwd.prase_path,
							MaxSize:  Size{Width: 450, Height: 20},
						},
						PushButton{
							//修改：只有第一次点击才清空
							OnMouseDown: func(x, y int, button walk.MouseButton) {
								subwd.prase_path.SetText("")
							},
							Text: "Browser",
							OnClicked: func() {
								browser(subwd)
							},
						},
					},
				},

				Composite{
					Layout: HBox{},
					Children: []Widget{
						PushButton{
							Text: "Reload",
							OnClicked: func() {
								parse(subwd, transfer, true)
							},
						},
						PushButton{
							Text: "Append",
							OnClicked: func() {
								parse(subwd, transfer, false)
							},
						},
						PushButton{
							Text: "Cancel",
							OnClicked: func() {
								subwd.Accept()
							},
						},
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
		mw.search(tablemodel)

	})

	mw.Run()

}

func parse(subwd *MySubWindow, transfer map[string]string, Reload bool) {
	//清空transfer
	transfer = make(map[string]string)
	log.Println("清空之前的transfer")

	if subwd.prase_path.Text() == "" {
		walk.MsgBox(subwd, "提示", "请输入目录或文件", walk.MsgBoxIconWarning)
		return
	}
	if Reload {
		walk.MsgBox(subwd, "提示", "正在清除目录中的文件", walk.MsgBoxIconWarning)
		//先清除解析目录再判断有没有输入文件路径
		err := clearOutputDir()
		if err != nil {
			log.Printf("Error clearing output directory %s: %v\n", outputDir, err)
			walk.MsgBox(subwd, "提示", "Error clearing output directory", walk.MsgBoxIconWarning)
		}
	}

	_prase(subwd.prase_path.Text(), transfer)
	if err := reloadTransferToFile(transfer); err != nil {
		log.Printf("Error reloading transfer to file: %v\n", err)
	}
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

	out_num  *walk.LineEdit
	res_view *walk.TableView

	match_mode string
	typeLabel  *walk.Label
	numLabel   *walk.Label
}

func (this *MyMainWindow) search(tablemodel *ResultInfoModel) {
	result_list, target_row_nums, err := _search(this.file_or_directory.Text(), this.target.Text(), this.match_mode)
	if err != nil {
		log.Printf("final Error :%v\n", err)
		tablemodel.UpdateItems(nil, nil)
	} else {
		this.numLabel.SetText(strconv.Itoa(len(result_list)))
		tablemodel.UpdateItems(result_list, target_row_nums)
	}

}

type ResultInfo struct {
	call_chain      string
	target_row_nums string
}

type ResultInfoModel struct {
	walk.SortedReflectTableModelBase
	results []ResultInfo
}

var _ walk.ReflectTableModel = new(FileInfoModel)

func NewResultInfoModel() *ResultInfoModel {
	return new(ResultInfoModel)
}
func (m *ResultInfoModel) Items() interface{} {
	return m.results
}

func (m *ResultInfoModel) RowCount() int {
	return len(m.results)
}
func (m *ResultInfoModel) Value(row, col int) interface{} {
	if col == 0 {
		return m.results[row].call_chain
	} else if col == 1 {
		return m.results[row].target_row_nums
	}
	return nil
}

//func (m *ResultInfoModel) Value(row, col int) interface{} {
//	if col == 0 {
//		return m.results[row] // 返回整个字符串
//	}
//	return nil
//}

func (m *ResultInfoModel) UpdateItems(call_chains []string, rows []string) {
	for id, cc := range call_chains {
		item := ResultInfo{
			call_chain:      cc,
			target_row_nums: rows[id],
		}
		m.results = append(m.results, item)
	}
	log.Println("has flush")
	for _, line := range m.results {
		log.Println(line)
	}
	m.PublishRowsReset()
}

func (this *MyMainWindow) SetType(mode string) {
	this.match_mode = mode
}

func extractLastBracketContent(line string) string {
	re := regexp.MustCompile(`\[[^\]]*\]`)
	matches := re.FindAllString(line, -1)
	if len(matches) == 0 {
		log.Println("no brackets found")
	}
	return matches[len(matches)-1]
}
