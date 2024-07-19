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
)

type transferValue struct {
	SerialNumber int
	OriginPath   string
}

const ROOT_DIR string = "D:\\HS-FS"
const OUTPUTDIR_DOC string = "outputdir.txt"
const PARSEDIR_DOC string = "parse.txt"
const PRE_TARGET_DOC string = "pre_target.txt"
const transferFile string = "D:\\HS-FS\\transfer.json"

var preSearchPath string
var outputDir string
var parseDir string

func init() {
	if _, err := os.Stat(ROOT_DIR); os.IsNotExist(err) {
		err := os.Mkdir(ROOT_DIR, 0755)
		if err != nil {
			LOG.Printf("failed to create output directory: %v", err)
		}
	}
	CreateOrLoadOutputDir()
	CreateOrLoadParseDir()
	CreateOrLoadPreSearchDir()

}

func Parse_(mw_parse *ProcessWd, total int) {
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
		LOG.Println("start prasing:   " + dir)
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
					return fmt.Errorf("failed to unmarshal XML from file %s: %v", path, err)
				}

				codeContent := filterCommentedCode(hsdoc.Code) //去注释

				if _, err := os.Stat(outputDir); os.IsNotExist(err) {
					err = os.MkdirAll(outputDir, 0755)
					if err != nil {
						return fmt.Errorf("failed to create output directory: %v", err)
					}
				}
				base_without_ext := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
				outputFileName := base_without_ext + ".code.txt"

				outputPath := filepath.Join(outputDir, outputFileName)

				transfer[fmt.Sprintf("[%s]", base_without_ext)] = transferValue{
					SerialNumber: serialNum,
					OriginPath:   path,
				}
				mw_parse.Synchronize(func() {
					mw_parse.progressBar.SetValue(serialNum)
					mw_parse.schedule.SetText(fmt.Sprintf("%.2f%%", float64(serialNum)/float64(total)*100))
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
		LOG.Println("directory:  " + dir + "   total: " + strconv.Itoa(num))
		if err != nil {
			LOG.Printf("end Error: %v\n", err)
		}

	}
}

type Hsdoc struct {
	XMLName xml.Name `xml:"hsdoc"`
	Code    string   `xml:"code"`
}

func clearOutputDir(mw_clean *ProcessWd, total int) error {
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
		mw_clean.Synchronize(func() {
			mw_clean.progressBar.SetValue(i)
			mw_clean.schedule.SetText(fmt.Sprintf("%.2f%%", float64(i+1)/float64(total)*100))
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
	transfer_json, err := json.Marshal(transfer)
	if err != nil {
		return err
	}
	return os.WriteFile(transferFile, transfer_json, 0644)
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
		file.WriteString("D:\\HS-FS\\output")
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
			LOG.Printf("failed to create parse_path file: %v", err)
		}

		defer file.Close()
	}
	file, err := os.Open(filepath.Join(ROOT_DIR, PARSEDIR_DOC))
	if err != nil {
		LOG.Printf("failed to open file: %v", err)
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
			LOG.Printf("failed to create pre_search_path file: %v", err)
		}
		defer file.Close()
	}
	file, err := os.Open(filepath.Join(ROOT_DIR, PRE_SEARCHPATH_DOC))
	if err != nil {
		LOG.Printf("failed to open file: %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		preSearchPath = scanner.Text()
	}
	LOG.Println("previous search path:", preSearchPath)
}

func CreateOrLoadPreTarget() {
	pretargetSavePath := filepath.Join(ROOT_DIR, PRE_TARGET_DOC)
	if _, err := os.Stat(pretargetSavePath); os.IsNotExist(err) {
		file, err := os.Create(pretargetSavePath)
		if err != nil {
			LOG.Printf("failed to create pre_search_path file: %v", err)
		}
		defer file.Close()
	}
	file, err := os.Open(filepath.Join(ROOT_DIR, PRE_TARGET_DOC))
	if err != nil {
		LOG.Printf("failed to open file: %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		preTargets = append(preTargets, scanner.Text())
	}
	LOG.Println("previous targets:", preTargets)
}
