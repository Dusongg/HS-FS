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
const SAVE_outputDir string = "outputdir.txt"
const SAVE_parseDir string = "parse.txt"
const SAVE_pre_target string = "pre_target.txt"
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

func _prase(mw_parse *ProcessMW, total int) {
	serial_num := 0
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
					SerialNumber: serial_num,
					OriginPath:   path,
				}
				mw_parse.Synchronize(func() {
					mw_parse.progressBar.SetValue(serial_num)
					mw_parse.schedule.SetText(fmt.Sprintf("%.2f%%", float64(serial_num)/float64(total)*100))
				})
				serial_num++

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

func clearOutputDir(mw_clean *ProcessMW, total int) error {
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
	where_output := filepath.Join(ROOT_DIR, SAVE_outputDir)
	if _, err := os.Stat(where_output); os.IsNotExist(err) {
		file, err := os.Create(where_output)
		if err != nil {
			LOG.Printf("failed to create output_path file: %v", err)
		}
		defer file.Close()
		file.WriteString("D:\\HS-FS\\output")
	}

	file, err := os.OpenFile(filepath.Join(ROOT_DIR, SAVE_outputDir), os.O_RDWR|os.O_CREATE, 0644)
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
	where_prase := filepath.Join(ROOT_DIR, SAVE_parseDir)
	if _, err := os.Stat(where_prase); os.IsNotExist(err) {
		file, err := os.Create(where_prase)
		if err != nil {
			LOG.Printf("failed to create parse_path file: %v", err)
		}

		defer file.Close()
	}
	parse_dir, err := os.Open(filepath.Join(ROOT_DIR, SAVE_parseDir))
	if err != nil {
		LOG.Printf("failed to open file: %v", err)
	}
	defer parse_dir.Close()
	scanner := bufio.NewScanner(parse_dir)
	for scanner.Scan() {
		parseDir = scanner.Text()
	}
	LOG.Println("Parse dir:", parseDir)

}

func CreateOrLoadPreSearchDir() {
	where_presearch := filepath.Join(ROOT_DIR, SAVE_pre_searchPath)
	if _, err := os.Stat(where_presearch); os.IsNotExist(err) {
		file, err := os.Create(where_presearch)
		if err != nil {
			LOG.Printf("failed to create pre_search_path file: %v", err)
		}
		defer file.Close()
	}
	pre_search_file, err := os.Open(filepath.Join(ROOT_DIR, SAVE_pre_searchPath))
	if err != nil {
		LOG.Printf("failed to open file: %v", err)
	}
	defer pre_search_file.Close()
	scanner := bufio.NewScanner(pre_search_file)
	for scanner.Scan() {
		preSearchPath = scanner.Text()
	}
	LOG.Println("previous search path:", preSearchPath)
}

func CreateOrLoadPreTarget() {
	where_pretarget := filepath.Join(ROOT_DIR, SAVE_pre_target)
	if _, err := os.Stat(where_pretarget); os.IsNotExist(err) {
		file, err := os.Create(where_pretarget)
		if err != nil {
			LOG.Printf("failed to create pre_search_path file: %v", err)
		}
		defer file.Close()
	}
	pre_target_file, err := os.Open(filepath.Join(ROOT_DIR, SAVE_pre_target))
	if err != nil {
		LOG.Printf("failed to open file: %v", err)
	}
	defer pre_target_file.Close()
	scanner := bufio.NewScanner(pre_target_file)
	for scanner.Scan() {
		pre_targets = append(pre_targets, scanner.Text())
	}
	LOG.Println("previous targets:", pre_targets)
}
