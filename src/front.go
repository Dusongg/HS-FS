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

const PRE_SEARCHPATH_DOC string = "pre_search.txt"

var preTargets []string
var historyMutex sync.Mutex
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
	settingWd := &MySubWindow{}
	resultsTable := NewResultInfoModel()
	errsTable := NewErrInfoModel()

	if err := (MainWindow{
		Title: "HS-FS",
		// 指定窗口的大小
		MinSize:  Size{Width: 500, Height: 640},
		AssignTo: &mw.MainWindow,
		Layout: VBox{
			MarginsZero: true,
		},
		OnDropFiles: func(files []string) {
			mw.searchScope.SetText(strings.Join(files, "\r\n"))
		},
		OnSizeChanged: func() {
			if len(preTargets) > 5 {
				preTargets = preTargets[:5]
			}
			mw.target.SetModel(preTargets)
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
						AssignTo: &mw.searchScope,
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
						Model:    preTargets,
						OnEditingFinished: func() {
							newTarget := mw.target.Text()
							LOG.Println("新增一条搜索记录: " + newTarget)
							preTargets = append([]string{newTarget}, preTargets...)
							if len(preTargets) > 5 {
								preTargets = preTargets[:5]
							}
							LOG.Println("当前preTargets:", preTargets)
							mw.target.SetModel(preTargets)
							mw.target.SetText(newTarget)
							go func() {
								saveHistoryTarget(newTarget)
							}()
						},
					},

					Label{Text: "匹配模式: "},
					RadioButtonGroup{
						Buttons: []RadioButton{
							{
								Name:     "exact_match",
								Text:     "精确匹配",
								AssignTo: &mw.exactMatchRB,
							},
							{
								Name:     "regular_match",
								Text:     "正则匹配",
								AssignTo: &mw.regularMatchRB,
							},
						},
					},
				},
			},
			//结果表
			TableView{
				AssignTo: &mw.resView,
				Model:    resultsTable,
				MinSize:  Size{Width: 500, Height: 350},

				AlternatingRowBG: true,
				ColumnsOrderable: true,
				OnCurrentIndexChanged: func() {
					if index := mw.resView.CurrentIndex(); index > -1 {
						targetFile := extractLastBracketContent(resultsTable.results[index].callChain) //拿掉调用链的最后一个函数(带有[])

						var openFileWD *walk.Dialog
						if err := (Dialog{
							AssignTo: &openFileWD,
							MinSize:  Size{Width: 700, Height: 200},
							Layout:   VBox{},
							Children: []Widget{
								PushButton{
									Text: "打开解析前的文件",
									OnClicked: func() {
										LOG.Printf("open : %s", targetFile)
										if transferValue, exists := transfer[targetFile]; exists {
											cmd := exec.Command("cmd", "/c", "start", "", transferValue.OriginPath)
											if err := cmd.Run(); err != nil {
												walk.MsgBox(mw, "报错", err.Error(), walk.MsgBoxIconError)
											}
										} else {
											walk.MsgBox(mw, "报错", fmt.Sprintf("can not find source file of: %s", resultsTable.results[index]), walk.MsgBoxIconError)
										}
									},
								},
								PushButton{
									Text: "打开解析后的文件",
									OnClicked: func() {
										path := filepath.Join(outputDir, targetFile[1:len(targetFile)-1]+".code.txt")
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
				AssignTo:         &mw.errView,
				Model:            errsTable,
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
					Label{Text: "报错数量： ", AssignTo: &mw.errLabel},
					HSpacer{Size: 10},
					Label{Text: "搜索总耗时： ", AssignTo: &mw.timeLabel},
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

	mw.exactMatchRB.Clicked().Attach(func() {
		go func() {
			mw.SetType(EXACT_MATCH)
		}()
	})
	mw.regularMatchRB.Clicked().Attach(func() {
		go func() {
			mw.SetType(REGEX_MATCH)
		}()
	})
	mw.reload.Clicked().Attach(func() {
		if mw.reload.Checked() {
			mw.isReload = true
			LOG.Printf("Whether to reload: %t\n", mw.isReload)
		} else {
			mw.isReload = false
			LOG.Printf("Whether to reload: %t\n", mw.isReload)

		}
	})

	mw.run.Clicked().Attach(func() {
		switch {
		case mw.searchScope.Text() == "":
			walk.MsgBox(mw, "提示", "请输入目标所在文件或目录", walk.MsgBoxIconWarning)
			return
		case mw.target.Text() == "":
			walk.MsgBox(mw, "提示", "请输入查找目标", walk.MsgBoxIconWarning)
			return
		case mw.matchPattern == NONE_MATCH:
			walk.MsgBox(mw, "提示", "请选择匹配模式", walk.MsgBoxIconWarning)
			return
		}

		if outputDir == "" {
			walk.MsgBox(mw, "提示", "output文件路径错误，请重新设置（默认：D:\\HS-FS\\output）", walk.MsgBoxIconWarning)
			runSettingWd(settingWd, mw)

		}

		if parseDir == "" {
			walk.MsgBox(mw, "提示", "请先设置待解析文件的路径", walk.MsgBoxIconWarning)
			runSettingWd(settingWd, mw)
		} else if files, _ := os.ReadDir(outputDir); len(files) == 0 {
			walk.MsgBox(mw, "提示", "output文件夹为空，正在为您自动解析", walk.MsgBoxIconWarning)
			parse(mw, false)
		} else if mw.isReload {
			parse(mw, true)
		} else if len(transfer) == 0 {
			walk.MsgBox(mw, "提示", "依赖文件被意外删除，需要重新加载", walk.MsgBoxIconWarning)
			parse(mw, true)
		}

		savePreSearchpath(mw)

		mw.search(resultsTable, errsTable)
	})

	mw.set.Clicked().Attach(func() {
		runSettingWd(settingWd, mw)
	})

	mw.quit.Clicked().Attach(func() {
		mw.Close()
	})

	mw.Run()

}

func runSettingWd(settingWd *MySubWindow, mw *MyMainWindow) {
	if err := (Dialog{
		AssignTo: &settingWd.Dialog,
		MinSize:  Size{Width: 700, Height: 200},
		Layout:   VBox{},

		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 10},
				Children: []Widget{
					Label{Text: "带解析的文件路径(搜索范围)"},
					LineEdit{
						Text:     parseDir,
						AssignTo: &settingWd.parsePathEdit,
						MaxSize:  Size{Width: 450, Height: 20},
					},
					PushButton{
						Text:     "保存更改",
						AssignTo: &settingWd.parsePathSave,
					},
					PushButton{
						Text: "Browser",
						OnClicked: func() {
							browser(settingWd)
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
						AssignTo: &settingWd.outputPathEdit,
						MaxSize:  Size{Width: 450, Height: 20},
					},
					PushButton{
						Text:     "保存更改",
						AssignTo: &settingWd.outputPathSave,
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
							settingWd.Accept()
						},
					},
					PushButton{
						Text: "退出",
						OnClicked: func() {
							if parseDir != settingWd.parsePathEdit.Text() { //表示没有点save
								saveParsePath(settingWd, mw)
							}
							settingWd.Accept()

						},
					},
				},
			},
		},
	}.Create(mw)); err != nil {
		return
	}

	settingWd.outputPathSave.Clicked().Attach(func() {
		saveOutputPath(settingWd, mw)
	})
	settingWd.parsePathSave.Clicked().Attach(func() {
		saveParsePath(settingWd, mw)
	})

	settingWd.Run()
}

func saveParsePath(subwd *MySubWindow, mw *MyMainWindow) {
	path := filepath.Join(ROOT_DIR, PARSEDIR_DOC)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		CreateOrLoadParseDir()
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	defer file.Close()
	if err != nil {
		walk.MsgBox(mw, "提示", fmt.Sprintf("无法打开文件:%s , 更新失败", path), walk.MsgBoxIconWarning)
		return
	}
	LOG.Printf("更改路径：%s\n", subwd.parsePathEdit.Text())
	_, err = file.WriteString(subwd.parsePathEdit.Text())
	if err != nil {
		LOG.Printf("向 %s 写入 %s 错误 : %v\n, 更新失败", path, subwd.parsePathEdit.Text(), err)
		return
	}
	parseDir = subwd.parsePathEdit.Text()
}

func parse(mw *MyMainWindow, Reload bool) {
	if parseDir == "" {
		walk.MsgBox(mw, "提示", "请输入目录或文件", walk.MsgBoxIconWarning)
		return
	}
	if Reload {
		files, _ := os.ReadDir(outputDir)
		LOG.Printf("正在清除output文件夹， 文件数量：%d\n", len(files))

		cleanWd := new(ProcessWd)
		MainWindow{
			AssignTo: &cleanWd.MainWindow,
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
							AssignTo: &cleanWd.progressBar,
							MinValue: 0,
							MaxValue: len(files),
							OnSizeChanged: func() {
								if cleanWd.progressBar.Value() == len(files)-1 {
									cleanWd.Close()
								}
							},
						},
						Label{AssignTo: &cleanWd.schedule},
					},
				},
			},
		}.Create()
		cleanWd.Show()
		go func() {
			err := clearOutputDir(cleanWd, len(files))
			if err != nil {
				LOG.Printf("Error clearing output directory %s: %v\n", outputDir, err)
				walk.MsgBox(mw, "提示", "Error clearing output directory", walk.MsgBoxIconWarning)
			}
			cleanWd.Close()
		}()
		cleanWd.Run()
	}

	//清空transfer
	transfer = make(map[string]transferValue)
	LOG.Println("清空transfer")

	filesNum, err := countFiles()
	if err != nil {
		LOG.Printf("count files err: %v\n", err)
	}
	LOG.Printf("带解析的文件数量， 文件数量：%d\n", filesNum)

	startTime := time.Now()
	parseWd := new(ProcessWd)
	err = MainWindow{
		AssignTo: &parseWd.MainWindow,
		Title:    "正在预处理待搜索的文件，请耐心等待",
		Size:     Size{Width: 500, Height: 200},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 1},
				Children: []Widget{
					Label{
						Text:      fmt.Sprintf("正在向 %s 写入预处理后的文件", outputDir),
						Alignment: AlignHNearVNear,
						MinSize:   Size{Width: 50, Height: 10},
					},
				},
			},
			Composite{
				Layout: Grid{Columns: 2},
				Children: []Widget{
					ProgressBar{
						AssignTo: &parseWd.progressBar,
						MinValue: 0,
						MaxValue: filesNum,
						OnSizeChanged: func() {
							if parseWd.progressBar.Value() == filesNum-1 {
								parseWd.Close()
								walk.MsgBox(mw, "提示", fmt.Sprintf("预处理解析完成, 总耗时: %s", time.Since(startTime)), walk.MsgBoxIconInformation)
							}
						},
					},
					Label{AssignTo: &parseWd.schedule},
				},
			},
		},
	}.Create()
	if err != nil {
		return
	}
	parseWd.Show()
	go Parse_(parseWd, filesNum)
	parseWd.Run()

	if err := reloadTransferToFile(); err != nil {
		LOG.Printf("Error reloading transfer to file: %v\n", err)
	}
}

type ProcessWd struct {
	*walk.MainWindow
	schedule    *walk.Label
	progressBar *walk.ProgressBar
}

type MySubWindow struct {
	*walk.Dialog

	parsePathEdit *walk.LineEdit
	parsePathSave *walk.PushButton

	outputPathEdit *walk.LineEdit
	outputPathSave *walk.PushButton
}

type MyMainWindow struct {
	*walk.MainWindow
	//第一行
	searchScope *walk.LineEdit
	//第二行
	target         *walk.ComboBox
	matchPattern   int
	exactMatchRB   *walk.RadioButton
	regularMatchRB *walk.RadioButton
	//结果表与错误表
	resView *walk.TableView
	errView *walk.TableView
	//最后一行
	numLabel  *walk.Label
	errLabel  *walk.Label
	timeLabel *walk.Label
	reload    *walk.CheckBox
	isReload  bool
	run       *walk.PushButton
	set       *walk.PushButton
	quit      *walk.PushButton
}

func (this *MyMainWindow) search(resultTable *ResultInfoModel, errsTable *ErrInfoModel) {
	if IsValidPath(this.searchScope.Text()) == false {
		walk.MsgBox(this, "报错", "搜索路径不合法", walk.MsgBoxIconWarning)
		return
	}
	startTime := time.Now()
	result := Search_(this.searchScope.Text(), this.target.Text(), this.matchPattern, this)
	this.numLabel.SetText("查询结果数量： " + strconv.Itoa(len(result.CallChain)))
	this.errLabel.SetText("报错数量： " + strconv.Itoa(len(result.Errs)))
	this.timeLabel.SetText("搜索总耗时： " + time.Since(startTime).String())
	LOG.Printf("search complete, results nums: %d, err nums: %d", len(result.CallChain), len(result.Errs))

	resultTable.UpdateItems(result.CallChain, result.TargetRowNums)
	errsTable.UpdateItems(result.Errs)
}

type ResultInfo struct {
	callChain     string
	targetRowNums string
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
		return m.results[row].callChain
	} else if col == 1 {
		return m.results[row].targetRowNums
	}
	return nil
}
func (m *ResultInfoModel) UpdateItems(callChains []string, rows []string) {
	m.results = nil //清空之前的
	for id, cc := range callChains {
		item := &ResultInfo{
			callChain:     cc,
			targetRowNums: rows[id],
		}
		m.results = append(m.results, item)
	}
	m.PublishRowsReset()
}

type ErrInfo struct {
	errInfo string
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
		return m.errs[row].errInfo
	}
	return nil
}
func (m *ErrInfoModel) UpdateItems(errs []string) {
	m.errs = nil
	for _, err := range errs {
		item := &ErrInfo{
			errInfo: err,
		}
		m.errs = append(m.errs, item)
	}
	m.PublishRowsReset()
}

func (this *MyMainWindow) SetType(mode int) {
	this.matchPattern = mode
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
	historyMutex.Lock()
	defer historyMutex.Unlock()
	historyTargetFilePath := filepath.Join(ROOT_DIR, PRE_TARGET_DOC)
	file, err := os.OpenFile(historyTargetFilePath, os.O_RDWR|os.O_CREATE, 0644)
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

func saveOutputPath(settingWd *MySubWindow, mw *MyMainWindow) {
	path := filepath.Join(ROOT_DIR, OUTPUTDIR_DOC)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		CreateOrLoadOutputDir()
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	defer file.Close()
	if err != nil {
		walk.MsgBox(mw, "提示", fmt.Sprintf("无法打开文件:%s , 更新失败", path), walk.MsgBoxIconWarning)
		return
	}
	LOG.Printf("更改路径：%s\n", settingWd.outputPathEdit.Text())
	_, err = file.WriteString(settingWd.outputPathEdit.Text())
	if err != nil {
		LOG.Printf("向 %s 写入 %s 错误 : %v\n, 更新失败", path, settingWd.outputPathEdit.Text(), err)
		return
	}
	outputDir = settingWd.outputPathEdit.Text()
	LOG.Println("outputDir changed --> " + outputDir)
}

func savePreSearchpath(mw *MyMainWindow) {
	//preSearch_mutex.Lock()
	//defer preSearch_mutex.Unlock()
	path := filepath.Join(ROOT_DIR, PRE_SEARCHPATH_DOC)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		CreateOrLoadOutputDir()
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	defer file.Close()
	if err != nil {
		walk.MsgBox(mw, "提示", fmt.Sprintf("无法打开文件:%s , 更新失败", path), walk.MsgBoxIconWarning)
		return
	}
	LOG.Printf("记录上一次搜索路径：%s\n", mw.searchScope.Text())
	_, err = file.WriteString(mw.searchScope.Text())
	if err != nil {
		LOG.Printf("向 %s 写入 %s 错误 : %v\n, 更新失败", path, mw.searchScope.Text(), err)
		return
	}
}

func IsValidPath(searchPath string) bool {
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
