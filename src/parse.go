package main

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type transferValue struct {
	SerialNumber int
	OriginPath   string
	FunctionNum  string
}

const (
	ROOT_DIR       string = "D:\\HS-FS"
	OUTPUTDIR_DOC  string = "outputdir.txt"
	PARSEDIR_DOC   string = "parse.txt"
	PRE_TARGET_DOC string = "pre_target.txt"
	transferFile   string = "D:\\HS-FS\\transfer.json"
)

var (
	preSearchPaths []string
	preTargets     []string
	outputDir      string
	parseDir       string
	transfer       = make(map[string]transferValue)
)

func init() {
	if _, err := os.Stat(ROOT_DIR); os.IsNotExist(err) {
		err := os.Mkdir(ROOT_DIR, 0755)
		if err != nil {
			ERROR.Printf("failed to create output directory: %v\n", err)
		}
	}
	CreateOrLoadPreSearchDir()
	CreateOrLoadPreTarget()
	CreateOrLoadOutputDir()
	CreateOrLoadParseDir()

	LOG.Println("Initializing transfer")
	err := loadTransferFromFile() //from parse.go  ,最开始无法加载
	if err != nil {
		ERROR.Printf("load transferfile err: %v\n", err)
	}
	LOG.Printf("transfer size :%d\n", len(transfer))
}

func Parse_(parseWd *ProcessWd, total int) {
	serialNum := 0
	//bug处理：文件路径不合法情况
	dirs := strings.Split(parseDir, ",") //bug:路径名带有,
	for _, dir := range dirs {
		LOG.Println(dir)
	}

	for _, dir := range dirs {
		num := 0

		dir = addEscapeBackslash(dir) //转义
		//if d, _ := os.Stat(path); !d.IsDir() {
		//	//报错
		//}
		LOG.Println("start parsing:   " + dir)
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir // 跳过整个目录
			}
			if !info.IsDir() {
				data, err := os.ReadFile(path)

				if err != nil {
					return fmt.Errorf("failed to read file %s: %v", path, err)
				}

				var hsdoc Hsdoc
				err = xml.Unmarshal(data, &hsdoc)
				if err != nil {
					ERROR.Printf("failed to unmarshal XML from file %s: %v\n", path, err)
					return filepath.SkipDir
				}

				codeContent := hsdoc.Code
				if !isWithComments {
					codeContent = filterCommentedCode(hsdoc.Code) //去注释
				}

				if _, err := os.Stat(outputDir); os.IsNotExist(err) {
					err = os.MkdirAll(outputDir, 0755)
					if err != nil {
						return fmt.Errorf("failed to create output directory: %v", err)
					}
				}
				baseWithoutExt := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
				outputFileName := baseWithoutExt + ".code.txt"

				outputPath := filepath.Join(outputDir, outputFileName)

				transfer[fmt.Sprintf("[%s]", baseWithoutExt)] = transferValue{
					SerialNumber: serialNum,
					OriginPath:   path,
					FunctionNum:  hsdoc.Basic.InnerBasic.ObjectId,
				}
				parseWd.Synchronize(func() {
					parseWd.progressBar.SetValue(serialNum)
					err := parseWd.schedule.SetText(fmt.Sprintf("%.2f%%", float64(serialNum)/float64(total)*100))
					if err != nil {
						ERROR.Println(err)
					}
				})
				serialNum++

				err = os.WriteFile(outputPath, []byte(codeContent), 0644)
				if err != nil {
					return fmt.Errorf("failed to write to file %s: %v", outputPath, err)
				}
				num++
				if num%5000 == 0 {
					LOG.Println("Number of files currently parsed: " + strconv.Itoa(num))
				}
			}
			return nil
		})
		if err != nil {
			ERROR.Printf("end Error: %v\n", err)
			return
		}
		LOG.Println("directory:  " + dir + "   total: " + strconv.Itoa(num))

	}
}

type InnerBasic struct {
	ObjectId string `xml:"objectId,attr"`
}
type OuterBasic struct {
	InnerBasic InnerBasic `xml:"basic"`
}
type Hsdoc struct {
	XMLName xml.Name   `xml:"hsdoc"`
	Code    string     `xml:"code"`
	Basic   OuterBasic `xml:"basic"`
}

func clearOutputDir(cleanWd *ProcessWd, total int) error {
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return nil
	}
	files, err := os.ReadDir(outputDir)
	if err != nil {
		return fmt.Errorf("failed to read output directory: %v", err)
	}
	for i, file := range files {
		err = os.RemoveAll(filepath.Join(outputDir, file.Name()))
		if err != nil {
			return fmt.Errorf("failed to remove file %v: %v", file.Name(), err)
		}
		LOG.Printf("remove file %d\n", i)
		cleanWd.Synchronize(func() {
			cleanWd.progressBar.SetValue(i)
			err := cleanWd.schedule.SetText(fmt.Sprintf("%.2f%%", float64(i+1)/float64(total)*100))
			if err != nil {
				ERROR.Println(err)
			}
		})
	}
	err = os.RemoveAll(outputDir)
	if err != nil {
		return fmt.Errorf("failed to remove output directory: %v", err)
	}
	err = os.Mkdir(outputDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}
	return nil
}

func filterCommentedCode(content string) string {
	re := regexp.MustCompile(`//.*|/\*(.|\n)*?\*/|--.*`)
	content = re.ReplaceAllString(content, "")
	return content
}

func addEscapeBackslash(path string) string {
	var builder strings.Builder
	for _, char := range path {
		if char == '\\' {
			builder.WriteRune(char)
		}
		builder.WriteRune(char)
	}
	return builder.String()
}

func reloadTransferToFile() error {
	transferJson, err := json.Marshal(transfer)
	if err != nil {
		return err
	}
	return os.WriteFile(transferFile, transferJson, 0644)
}
func loadTransferFromFile() error {
	_, err := os.Open(transferFile)
	if err != nil {
		return fmt.Errorf("failed to open transfer file: %v", err)
	}

	data, err := os.ReadFile(transferFile)
	if err != nil {
		return fmt.Errorf("failed to read transfer file: %v", err)
	}
	err = json.Unmarshal(data, &transfer)
	if err != nil {
		return fmt.Errorf("failed to unmarshal transfer file: %v", err)
	}
	return nil
}

func CreateOrLoadOutputDir() {
	outputSavePath := filepath.Join(ROOT_DIR, OUTPUTDIR_DOC)
	if _, err := os.Stat(outputSavePath); os.IsNotExist(err) {
		file, err := os.Create(outputSavePath)
		if err != nil {
			LOG.Printf("failed to create output_path file: %v", err)
		}
		defer file.Close()
		_, err = file.WriteString("D:\\HS-FS\\output")
		if err != nil {
			ERROR.Println(err)
		}

	}

	file, err := os.OpenFile(filepath.Join(ROOT_DIR, OUTPUTDIR_DOC), os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		LOG.Printf("failed to open file: %v", err)
	}

	defer file.Close()

	//outputdir.txt为空的情况
	if scanner := bufio.NewScanner(file); !scanner.Scan() {
		file.WriteString("D:\\HS-FS\\output")
	}
	file.Seek(0, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		outputDir = scanner.Text()
	}
	LOG.Println("Output dir:", outputDir)

}

func CreateOrLoadParseDir() {
	parseSavePath := filepath.Join(ROOT_DIR, PARSEDIR_DOC)
	if _, err := os.Stat(parseSavePath); os.IsNotExist(err) {
		file, err := os.Create(parseSavePath)
		if err != nil {
			ERROR.Printf("failed to create parse_path file: %v\n", err)
		}

		defer file.Close()
	}
	file, err := os.Open(filepath.Join(ROOT_DIR, PARSEDIR_DOC))
	if err != nil {
		ERROR.Printf("failed to open file: %v\n", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parseDir = scanner.Text()
	}
	LOG.Println("Parse dir:", parseDir)

}

func CreateOrLoadPreSearchDir() {
	presearchSavePath := filepath.Join(ROOT_DIR, PRE_SEARCHPATH_DOC)
	if _, err := os.Stat(presearchSavePath); os.IsNotExist(err) {
		file, err := os.Create(presearchSavePath)
		if err != nil {
			ERROR.Printf("failed to create pre_search_path file: %v\n", err)
		}
		defer file.Close()
	}
	file, err := os.Open(filepath.Join(ROOT_DIR, PRE_SEARCHPATH_DOC))
	if err != nil {
		ERROR.Printf("failed to open file: %v\n", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		preSearchPaths = append(preSearchPaths, scanner.Text())
	}
	LOG.Println("previous search path:", preSearchPaths)
}

func CreateOrLoadPreTarget() {
	pretargetSavePath := filepath.Join(ROOT_DIR, PRE_TARGET_DOC)
	if _, err := os.Stat(pretargetSavePath); os.IsNotExist(err) {
		file, err := os.Create(pretargetSavePath)
		if err != nil {
			ERROR.Printf("failed to create pre_search_path file: %v\n", err)
		}
		defer file.Close()
	}
	file, err := os.Open(filepath.Join(ROOT_DIR, PRE_TARGET_DOC))
	if err != nil {
		ERROR.Printf("failed to open file: %v\n", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		preTargets = append(preTargets, scanner.Text())
	}
	LOG.Println("previous targets:", preTargets)
}

//func GetTransfer() {
//	serialNum := 0
//	dirs := strings.Split(parseDir, ",") //bug:路径名带有,
//	for _, dir := range dirs {
//		LOG.Println(dir)
//	}
//	for _, dir := range dirs {
//		dir = addEscapeBackslash(dir) //转义
//		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
//			if err != nil {
//				return err
//			}
//			if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
//				return filepath.SkipDir // 跳过整个目录
//			}
//			if !info.IsDir() {
//				baseWithoutExt := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
//				transfer[fmt.Sprintf("[%s]", baseWithoutExt)] = transferValue{
//					SerialNumber: serialNum,
//					OriginPath:   path,
//				}
//				serialNum++
//			}
//			return nil
//		})
//		if err != nil {
//			ERROR.Printf("end Error: %v\n", err)
//			return
//		}
//		transferJson, err := json.Marshal(transfer)
//		if err != nil {
//			ERROR.Println(err)
//			return
//		}
//		err = os.WriteFile(transferFile, transferJson, 0644)
//		if err != nil {
//			ERROR.Println(err)
//		}
//	}
//}

var sema = make(chan struct{}, 20)
var gSerialNum = 0
var mutex sync.Mutex

func GetTransfer() {
	var n sync.WaitGroup
	dirs := strings.Split(parseDir, ",")
	for _, dir := range dirs {
		dir = addEscapeBackslash(dir)
		n.Add(1)
		go walkDir(dir, &n)
	}
	n.Wait()
	transferJson, err := json.Marshal(transfer)
	if err != nil {
		ERROR.Println(err)
		return
	}
	err = os.WriteFile(transferFile, transferJson, 0644)
	if err != nil {
		ERROR.Println(err)
	}
}

func walkDir(dir string, n *sync.WaitGroup) {
	defer n.Done()
	for _, entry := range dirents(dir) {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			n.Add(1)
			go walkDir(path, n)
		} else {
			data, err := os.ReadFile(path)
			if err != nil {
				ERROR.Printf("failed to read file %s: %v\n", path, err)
				continue
			}
			var hsdoc Hsdoc
			err = xml.Unmarshal(data, &hsdoc)
			if err != nil {
				ERROR.Printf("failed to unmarshal XML from file %s: %v\n", path, err)
				continue
			}

			baseWithoutExt := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			mutex.Lock()
			transfer[fmt.Sprintf("[%s]", baseWithoutExt)] = transferValue{
				SerialNumber: gSerialNum,
				OriginPath:   path,
				FunctionNum:  hsdoc.Basic.InnerBasic.ObjectId,
			}
			gSerialNum++
			mutex.Unlock()
		}
	}
}

func dirents(dir string) []os.DirEntry {
	sema <- struct{}{}
	defer func() { <-sema }()
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "du1: %v\n", err)
		return nil
	}
	return entries
}
