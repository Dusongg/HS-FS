package main

import (
	"fmt"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	NONE_MATCH  = "NONE_MATCH"
	EXACT_MATCH = "EXACT_MATCH"
	REGEX_MATCH = "REGEX_MATCH"
)

var LOG = log.New(os.Stdout, "INFO: ", log.LstdFlags|log.Lshortfile)

var transfer = make(map[string]transferValue)

func init() {
	//TODO:transfer应该是服务端开始就创建的，后面将他搬离front部分，  考虑将这部分用redis实现
	LOG.Println("Initializing transfer")
	err := loadTransferFromFile() //from parse.go  ,最开始无法加载
	if err != nil {
		LOG.Printf("laod transferfile err: %v\n", err)
	}
	LOG.Printf("transfer size :%d", len(transfer))

}

func main() {
	//窗口样式
	walk.AppendToWalkInit(func() {
		walk.FocusEffect, _ = walk.NewBorderGlowEffect(walk.RGB(0, 63, 255))
		walk.InteractionEffect, _ = walk.NewDropShadowEffect(walk.RGB(63, 63, 63))
		walk.ValidationErrorEffect, _ = walk.NewBorderGlowEffect(walk.RGB(255, 0, 0))
	})

	mw := &MyMainWindow{}
	replace_subwd := &MySubWindow{}
	results_table := NewResultInfoModel()
	errs_table := NewErrInfoModel()

	if err := (MainWindow{
		Title: "HS-FS",
		// 指定窗口的大小
		MinSize:  Size{Width: 500, Height: 640},
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
						Text:     preSearchPath,
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
					Label{Text: "查找目标: "},
					LineEdit{AssignTo: &mw.target},
					Label{Text: "匹配模式: "},
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
								OnClicked: func() {
									walk.MsgBox(mw, "提示", "该功能尚未实现", walk.MsgBoxIconWarning)
								},
							},
						},
					},
				},
			},

			TableView{
				AssignTo: &mw.res_view,
				Model:    results_table,
				MinSize:  Size{Width: 500, Height: 350},

				AlternatingRowBG: true,
				ColumnsOrderable: true,
				OnCurrentIndexChanged: func() {
					if index := mw.res_view.CurrentIndex(); index > -1 {
						target_file := extractLastBracketContent(results_table.results[index].call_chain) //拿掉调用链的最后一个函数
						LOG.Printf("open : %s", target_file)

						if transfer_value, exists := transfer[target_file]; exists {
							cmd := exec.Command("cmd", "/c", "start", "", transfer_value.OriginPath)
							if err := cmd.Run(); err != nil {
								walk.MsgBox(mw, "报错", err.Error(), walk.MsgBoxIconError)
							}
						} else {
							walk.MsgBox(mw, "报错", fmt.Sprintf("can not find source file of: %s", results_table.results[index]), walk.MsgBoxIconError)
						}
					}
				},
				Columns: []TableViewColumn{
					TableViewColumn{
						DataMember: "调用链",
						Width:      700,
					},
					TableViewColumn{
						DataMember: "目标所在行",
						Width:      300,
					},
				},
			},

			TableView{
				AssignTo:         &mw.err_view,
				Model:            errs_table,
				AlternatingRowBG: true,
				ColumnsOrderable: true,
				MinSize:          Size{Width: 500, Height: 150},

				Columns: []TableViewColumn{
					TableViewColumn{
						Width:      1000,
						DataMember: "报错",
					},
				},
			},

			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{Text: "查询结果数量： ", AssignTo: &mw.numLabel},
					HSpacer{Size: 10},
					Label{Text: "报错数量： ", AssignTo: &mw.errLable},
					HSpacer{},

					CheckBox{
						Name:     "exact_match",
						Text:     "重新解析生成",
						AssignTo: &mw.reload,
					},

					PushButton{AssignTo: &mw.run, Text: "Run"},
					PushButton{AssignTo: &mw.set, Text: "Set"},
					PushButton{
						Text:     "Quit",
						AssignTo: &mw.quit,
					},
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
	mw.reload.Clicked().Attach(func() {
		if mw.reload.Checked() {
			mw.is_reload = true
			LOG.Printf("Whether to reload: %t\n", mw.is_reload)
		} else {
			mw.is_reload = false
			LOG.Printf("Whether to reload: %t\n", mw.is_reload)

		}
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

		if parseDir == "" {
			walk.MsgBox(mw, "提示", "请先设置解析路径", walk.MsgBoxIconWarning)
			runsubwd(replace_subwd, mw)
		} else if files, _ := os.ReadDir(outputDir); len(files) == 0 {
			walk.MsgBox(mw, "提示", "output文件夹为空，正在为您自动解析", walk.MsgBoxIconWarning)
			parse(mw, false) //bug：subwd窗口没有打开，提示信息需要在wm
		} else if mw.is_reload {
			parse(mw, true)
		}
		mw.search(results_table, errs_table)
	})

	mw.set.Clicked().Attach(func() {
		runsubwd(replace_subwd, mw)
	})

	mw.quit.Clicked().Attach(func() {
		path := filepath.Join(ROOT_DIR, SAVE_pre_searchPath)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			CreateAndLoadOutputDir()
		}
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		defer file.Close()
		if err != nil {
			walk.MsgBox(mw, "提示", fmt.Sprintf("无法打开文件:%s , 更新失败", path), walk.MsgBoxIconWarning)
			return
		}
		LOG.Printf("记录上一次搜索路径：%s\n", mw.file_or_directory.Text())
		_, err = file.WriteString(mw.file_or_directory.Text())
		if err != nil {
			LOG.Printf("向 %s 写入 %s 错误 : %v\n, 更新失败", path, replace_subwd.output_path.Text(), err)
			return
		}
		mw.Close()
	})

	mw.Run()

}

func runsubwd(replace_subwd *MySubWindow, mw *MyMainWindow) {
	if err := (Dialog{
		AssignTo: &replace_subwd.Dialog,
		MinSize:  Size{Width: 700, Height: 200},
		Layout:   VBox{},

		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 10},
				Children: []Widget{
					Label{Text: "预处理的文件夹(搜索范围)"},
					LineEdit{
						//TODO:下拉查看
						Text:     parseDir,
						AssignTo: &replace_subwd.parse_path,
						MaxSize:  Size{Width: 450, Height: 20},
					},
					PushButton{
						Text:     "保存更改",
						AssignTo: &replace_subwd.parse_path_save,
					},
					PushButton{
						Text: "Browser",
						OnClicked: func() {
							browser(replace_subwd)
						},
					},
				},
			},
			Composite{
				Layout: Grid{Columns: 10},
				Children: []Widget{
					Label{Text: "更改输出文件路径: "},
					LineEdit{
						Text:     "当前路径: " + outputDir,
						AssignTo: &replace_subwd.output_path,
						MaxSize:  Size{Width: 450, Height: 20},
					},
					PushButton{
						Text:     "保存更改",
						AssignTo: &replace_subwd.output_path_save,
					},
				},
			},
			Composite{
				Layout: Grid{Columns: 10},
				Children: []Widget{
					PushButton{
						Text: "生成文件",
						OnClicked: func() {
							parse(mw, false)
							replace_subwd.Accept()
						},
					},
					PushButton{
						Text: "退出",
						OnClicked: func() {
							if parseDir != replace_subwd.parse_path.Text() {
								save_parse_path(replace_subwd, mw)
							}
							replace_subwd.Accept()

						},
					},
				},
			},
		},
	}.Create(mw)); err != nil {
		return
	}

	replace_subwd.output_path_save.Clicked().Attach(func() {
		path := filepath.Join(ROOT_DIR, SAVE_outputDir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			CreateAndLoadOutputDir()
		}
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		defer file.Close()
		if err != nil {
			walk.MsgBox(mw, "提示", fmt.Sprintf("无法打开文件:%s , 更新失败", path), walk.MsgBoxIconWarning)
			return
		}
		LOG.Printf("更改路径：%s\n", replace_subwd.output_path.Text())
		_, err = file.WriteString(replace_subwd.output_path.Text())
		if err != nil {
			LOG.Printf("向 %s 写入 %s 错误 : %v\n, 更新失败", path, replace_subwd.output_path.Text(), err)
			return
		}
		outputDir = replace_subwd.output_path.Text()
	})
	replace_subwd.parse_path_save.Clicked().Attach(func() {
		save_parse_path(replace_subwd, mw)
	})

	replace_subwd.Run()
}

func save_parse_path(subwd *MySubWindow, mw *MyMainWindow) {
	path := filepath.Join(ROOT_DIR, SAVE_parseDir)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		CreateAndLoadParseDir()
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	defer file.Close()
	if err != nil {
		walk.MsgBox(mw, "提示", fmt.Sprintf("无法打开文件:%s , 更新失败", path), walk.MsgBoxIconWarning)
		return
	}
	LOG.Printf("更改路径：%s\n", subwd.parse_path.Text())
	_, err = file.WriteString(subwd.parse_path.Text())
	if err != nil {
		LOG.Printf("向 %s 写入 %s 错误 : %v\n, 更新失败", path, subwd.parse_path.Text(), err)
		return
	}
	parseDir = subwd.parse_path.Text()
}

func parse(mw *MyMainWindow, Reload bool) {
	if parseDir == "" {
		walk.MsgBox(mw, "提示", "请输入目录或文件", walk.MsgBoxIconWarning)
		return
	}
	if Reload {
		LOG.Println("正在清除output文件夹")
		//TODO：改界面
		walk.MsgBox(mw, "提示", "正在清除目录中的文件", walk.MsgBoxIconWarning)
		//先清除解析目录再判断有没有输入文件路径
		err := clearOutputDir()
		if err != nil {
			LOG.Printf("Error clearing output directory %s: %v\n", outputDir, err)
			walk.MsgBox(mw, "提示", "Error clearing output directory", walk.MsgBoxIconWarning)
		}
	}

	//清空transfer
	transfer = make(map[string]transferValue)
	LOG.Println("清空transfer")
	//TODO:改界面
	walk.MsgBox(mw, "提示", "解析过程可能耗时较长，请耐心等待", walk.MsgBoxIconInformation)

	_prase(parseDir)
	if err := reloadTransferToFile(); err != nil {
		LOG.Printf("Error reloading transfer to file: %v\n", err)
	}
	walk.MsgBox(mw, "提示", "预处理解析完成", walk.MsgBoxIconInformation)
}

type MySubWindow struct {
	*walk.Dialog

	parse_path      *walk.LineEdit
	parse_path_save *walk.PushButton

	output_path      *walk.LineEdit
	output_path_save *walk.PushButton
}

type MyMainWindow struct {
	*walk.MainWindow

	type_exact_match   *walk.RadioButton
	type_regular_match *walk.RadioButton
	run                *walk.PushButton

	file_or_directory *walk.LineEdit
	set               *walk.PushButton
	target            *walk.LineEdit

	out_num  *walk.LineEdit
	res_view *walk.TableView
	err_view *walk.TableView

	match_mode string
	typeLabel  *walk.Label
	numLabel   *walk.Label
	errLable   *walk.Label

	reload    *walk.CheckBox
	is_reload bool

	quit *walk.PushButton
}

func (this *MyMainWindow) search(result_table *ResultInfoModel, errs_table *ErrInfoModel) {
	//TODO:处理不合法路径
	result := _search(this.file_or_directory.Text(), this.target.Text(), this.match_mode)
	this.numLabel.SetText("查询结果数量：" + strconv.Itoa(len(result.CallChain)))
	this.errLable.SetText("报错数量： " + strconv.Itoa(len(result.Errs)))
	LOG.Printf("search complete, results nums: %d, err nums: %d", len(result.CallChain), len(result.Errs))

	result_table.UpdateItems(result.CallChain, result.TargetRowNums)
	errs_table.UpdateItems(result.Errs)
}

type ResultInfo struct {
	call_chain      string
	target_row_nums string
}

type ResultInfoModel struct {
	walk.SortedReflectTableModelBase
	results []*ResultInfo
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
func (m *ResultInfoModel) UpdateItems(call_chains []string, rows []string) {
	m.results = nil //清空之前的
	for id, cc := range call_chains {
		item := &ResultInfo{
			call_chain:      cc,
			target_row_nums: rows[id],
		}
		m.results = append(m.results, item)
	}
	m.PublishRowsReset()
}

type ErrInfo struct {
	err_info string
}
type ErrInfoModel struct {
	walk.SortedReflectTableModelBase
	errs []*ErrInfo
}

func NewErrInfoModel() *ErrInfoModel {
	return new(ErrInfoModel)
}
func (m *ErrInfoModel) Items() interface{} {
	return m.errs
}

func (m *ErrInfoModel) RowCount() int {
	return len(m.errs)
}
func (m *ErrInfoModel) Value(row, col int) interface{} {
	if col == 0 {
		return m.errs[row].err_info
	}
	return nil
}
func (m *ErrInfoModel) UpdateItems(errs []string) {
	m.errs = nil
	for _, err := range errs {
		item := &ErrInfo{
			err_info: err,
		}
		m.errs = append(m.errs, item)
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
		LOG.Println("no brackets found")
	}
	return matches[len(matches)-1]
}
