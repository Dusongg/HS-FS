package main

import (
	"bufio"
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
	"sync"
	"time"
)

const (
	NONE_MATCH  = 0
	EXACT_MATCH = 1
	REGEX_MATCH = 2
)

const SAVE_pre_searchPath string = "pre_search.txt"

var pre_targets []string
var history_mutex sync.Mutex
var LOG = log.New(os.Stdout, "INFO: ", log.LstdFlags|log.Lshortfile)
var transfer = make(map[string]transferValue)

func init() {
	LOG.Println("Initializing transfer")
	err := loadTransferFromFile() //from parse.go  ,最开始无法加载
	if err != nil {
		LOG.Printf("laod transferfile err: %v\n", err)
	}
	LOG.Printf("transfer size :%d", len(transfer))

	CreateOrLoadPreTarget()
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
		OnSizeChanged: func() {
			if len(pre_targets) > 5 {
				pre_targets = pre_targets[:5]
			}
			mw.target.SetModel(pre_targets)
		},
		Children: []Widget{
			//第一行：搜索路径
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
			//第二行：关键词
			Composite{
				Layout: Grid{Columns: 10},
				Children: []Widget{
					Label{Text: "查找目标: "},
					ComboBox{
						Editable: true,
						AssignTo: &mw.target,
						Model:    pre_targets,
						OnEditingFinished: func() {
							new_target := mw.target.Text()
							LOG.Println("新增一条搜索记录: " + new_target)
							pre_targets = append([]string{new_target}, pre_targets...)
							if len(pre_targets) > 5 {
								pre_targets = pre_targets[:5]
							}
							LOG.Println("当前pre_targets:", pre_targets)
							mw.target.SetModel(pre_targets)
							mw.target.SetText(new_target)
							go func() {
								saveHistoryTarget(new_target)
							}()
						},
					},

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
							},
						},
					},
				},
			},
			//结果表
			TableView{
				AssignTo: &mw.res_view,
				Model:    results_table,
				MinSize:  Size{Width: 500, Height: 350},

				AlternatingRowBG: true,
				ColumnsOrderable: true,
				//TODO:选择打开解析前或者解析后的问题
				OnCurrentIndexChanged: func() {
					if index := mw.res_view.CurrentIndex(); index > -1 {
						target_file := extractLastBracketContent(results_table.results[index].call_chain) //拿掉调用链的最后一个函数(带有[])

						var openFileWD *walk.Dialog
						if err := (Dialog{
							AssignTo: &openFileWD,
							MinSize:  Size{Width: 700, Height: 200},
							Layout:   VBox{},
							Children: []Widget{
								PushButton{
									Text: "打开解析前的文件",
									OnClicked: func() {
										LOG.Printf("open : %s", target_file)
										if transfer_value, exists := transfer[target_file]; exists {
											cmd := exec.Command("cmd", "/c", "start", "", transfer_value.OriginPath)
											if err := cmd.Run(); err != nil {
												walk.MsgBox(mw, "报错", err.Error(), walk.MsgBoxIconError)
											}
										} else {
											walk.MsgBox(mw, "报错", fmt.Sprintf("can not find source file of: %s", results_table.results[index]), walk.MsgBoxIconError)
										}
									},
								},
								PushButton{
									Text: "打开解析后的文件",
									OnClicked: func() {
										path := filepath.Join(outputDir, target_file[1:len(target_file)-1]+".code.txt")
										cmd := exec.Command("cmd", "/c", "start", "", path)
										if err := cmd.Run(); err != nil {
											walk.MsgBox(mw, "报错", err.Error(), walk.MsgBoxIconError)
										}
									},
								},
							},
						}.Create(mw)); err != nil {
							return
						}
						openFileWD.Run()
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
			//错误表
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
			//最后一行
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{Text: "查询结果数量： ", AssignTo: &mw.numLabel},
					HSpacer{Size: 10},
					Label{Text: "报错数量： ", AssignTo: &mw.errLable},
					HSpacer{Size: 10},
					Label{Text: "搜索总耗时： ", AssignTo: &mw.timeLable},
					HSpacer{},

					CheckBox{
						Name:     "exact_match",
						Text:     "重新解析生成",
						AssignTo: &mw.reload,
					},

					PushButton{AssignTo: &mw.run, Text: "Run"},
					PushButton{AssignTo: &mw.set, Text: "Settings"},
					PushButton{AssignTo: &mw.quit, Text: "Quit"},
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

		if outputDir == "" {
			walk.MsgBox(mw, "提示", "output文件路径错误，请重新设置（默认：D:\\HS-FS\\output）", walk.MsgBoxIconWarning)
			runsubwd(replace_subwd, mw)

		}

		if parseDir == "" {
			walk.MsgBox(mw, "提示", "请先设置待解析文件的路径", walk.MsgBoxIconWarning)
			runsubwd(replace_subwd, mw)
		} else if files, _ := os.ReadDir(outputDir); len(files) == 0 {
			walk.MsgBox(mw, "提示", "output文件夹为空，正在为您自动解析", walk.MsgBoxIconWarning)
			parse(mw, false)
		} else if mw.is_reload {
			parse(mw, true)
		} else if len(transfer) == 0 {
			walk.MsgBox(mw, "提示", "依赖文件被意外删除，需要重新加载", walk.MsgBoxIconWarning)
			parse(mw, true)
		}

		save_pre_searchPath(mw)

		mw.search(results_table, errs_table)
	})

	mw.set.Clicked().Attach(func() {
		runsubwd(replace_subwd, mw)
	})

	mw.quit.Clicked().Attach(func() {
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
					Label{Text: "带解析的文件路径(搜索范围)"},
					LineEdit{
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
					Label{Text: "解析后的输出路径："},
					LineEdit{
						Text:     outputDir,
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
							if parseDir != replace_subwd.parse_path.Text() { //表示没有点save
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
		save_output_path(replace_subwd, mw)
	})
	replace_subwd.parse_path_save.Clicked().Attach(func() {
		save_parse_path(replace_subwd, mw)
	})

	replace_subwd.Run()
}

func save_parse_path(subwd *MySubWindow, mw *MyMainWindow) {
	path := filepath.Join(ROOT_DIR, SAVE_parseDir)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		CreateOrLoadParseDir()
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
		files, _ := os.ReadDir(outputDir)
		LOG.Printf("正在清除output文件夹， 文件数量：%d\n", len(files))

		mw_clean := new(ProcessMW)
		MainWindow{
			AssignTo: &mw_clean.MainWindow,
			Title:    "正在清除output文件夹",
			Size:     Size{500, 200},
			Layout:   VBox{},
			Children: []Widget{
				Composite{
					Layout: Grid{Columns: 1},
					Children: []Widget{
						Label{
							Text:      fmt.Sprintf("正在清除 %s 中的文件", outputDir),
							Alignment: AlignHNearVNear,
						},
					},
				},
				Composite{
					Layout: Grid{Columns: 2},
					Children: []Widget{
						ProgressBar{
							AssignTo: &mw_clean.progressBar,
							MinValue: 0,
							MaxValue: len(files),
							OnSizeChanged: func() {
								if mw_clean.progressBar.Value() == len(files)-1 {
									mw_clean.Close()
								}
							},
						},
						Label{AssignTo: &mw_clean.schedule},
					},
				},
			},
		}.Create()
		mw_clean.Show()
		go func() {
			err := clearOutputDir(mw_clean, len(files))
			if err != nil {
				LOG.Printf("Error clearing output directory %s: %v\n", outputDir, err)
				walk.MsgBox(mw, "提示", "Error clearing output directory", walk.MsgBoxIconWarning)
			}
			mw_clean.Close()
		}()
		mw_clean.Run()
	}

	//清空transfer
	transfer = make(map[string]transferValue)
	LOG.Println("清空transfer")

	files_num, err := countFiles()
	if err != nil {
		LOG.Printf("count files err: %v\n", err)
	}
	LOG.Printf("带解析的文件数量， 文件数量：%d\n", files_num)

	startTime := time.Now()
	mw_prase := new(ProcessMW)
	MainWindow{
		AssignTo: &mw_prase.MainWindow,
		Title:    "正在预处理待搜索的文件，请耐心等待",
		Size:     Size{500, 200},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 1},
				Children: []Widget{
					Label{
						Text:      fmt.Sprintf("正在向 %s 写入预处理后的文件", outputDir),
						Alignment: AlignHNearVNear,
						MinSize:   Size{50, 10},
					},
				},
			},
			Composite{
				Layout: Grid{Columns: 2},
				Children: []Widget{
					ProgressBar{
						AssignTo: &mw_prase.progressBar,
						MinValue: 0,
						MaxValue: files_num,
						OnSizeChanged: func() {
							if mw_prase.progressBar.Value() == files_num-1 {
								mw_prase.Close()
								walk.MsgBox(mw, "提示", fmt.Sprintf("预处理解析完成, 总耗时: %s", time.Since(startTime)), walk.MsgBoxIconInformation)
							}
						},
					},
					Label{AssignTo: &mw_prase.schedule},
				},
			},
		},
	}.Create()
	mw_prase.Show()
	go _prase(mw_prase, files_num)
	mw_prase.Run()

	if err := reloadTransferToFile(); err != nil {
		LOG.Printf("Error reloading transfer to file: %v\n", err)
	}
}

type ProcessMW struct {
	*walk.MainWindow
	schedule    *walk.Label
	progressBar *walk.ProgressBar
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
	target            *walk.ComboBox

	out_num  *walk.LineEdit
	res_view *walk.TableView
	err_view *walk.TableView

	match_mode int
	typeLabel  *walk.Label
	numLabel   *walk.Label
	errLable   *walk.Label
	timeLable  *walk.Label

	reload    *walk.CheckBox
	is_reload bool

	quit *walk.PushButton
}

func (this *MyMainWindow) search(result_table *ResultInfoModel, errs_table *ErrInfoModel) {
	if IsVaildPath(this.file_or_directory.Text()) == false {
		walk.MsgBox(this, "报错", "搜索路径不合法", walk.MsgBoxIconWarning)
		return
	}
	startTime := time.Now()
	result := _search(this.file_or_directory.Text(), this.target.Text(), this.match_mode, this)
	this.numLabel.SetText("查询结果数量： " + strconv.Itoa(len(result.CallChain)))
	this.errLable.SetText("报错数量： " + strconv.Itoa(len(result.Errs)))
	this.timeLable.SetText("搜索总耗时： " + time.Since(startTime).String())
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

func (this *MyMainWindow) SetType(mode int) {
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

func countFiles() (int, error) {
	var count int
	dirs := strings.Split(parseDir, ",")
	for _, dir := range dirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir // 跳过整个目录
			}
			if !info.IsDir() {
				count++
			}
			return nil
		})
		if err != nil {
			return count, err
		}
	}
	return count, nil
}

func saveHistoryTarget(new_target string) {
	history_mutex.Lock()
	defer history_mutex.Unlock()
	history_path := filepath.Join(ROOT_DIR, SAVE_pre_target)
	file, err := os.OpenFile(history_path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		LOG.Println("无法打开文件:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("读取文件出错:", err)
		return
	}
	lines = append([]string{new_target}, lines...)

	if len(lines) > 20 {
		lines = lines[:10]
	}
	file.Truncate(0)
	file.Seek(0, 0)
	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			fmt.Println("写入文件出错:", err)
			return
		}
	}
	writer.Flush()
}

func save_output_path(replace_subwd *MySubWindow, mw *MyMainWindow) {
	path := filepath.Join(ROOT_DIR, SAVE_outputDir)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		CreateOrLoadOutputDir()
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
	LOG.Println("outputDir changed --> " + outputDir)
}

func save_pre_searchPath(mw *MyMainWindow) {
	//preSearch_mutex.Lock()
	//defer preSearch_mutex.Unlock()
	path := filepath.Join(ROOT_DIR, SAVE_pre_searchPath)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		CreateOrLoadOutputDir()
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
		LOG.Printf("向 %s 写入 %s 错误 : %v\n, 更新失败", path, mw.file_or_directory.Text(), err)
		return
	}
}

func IsVaildPath(searchPath string) bool {
	cleanedPath := filepath.Clean(searchPath)
	absPath, err := filepath.Abs(cleanedPath)
	if err != nil {
		LOG.Printf("absPath error: %v", err)
		return false
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		LOG.Printf("searchPath Not Found: %s", searchPath)
		return false
	}
	return true
}
