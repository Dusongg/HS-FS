package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
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

const outputDir string = "./output"
const transferFile string = "./transfer.json"

func _prase(path string) {
	serial_num := 0
	//bug处理：文件路径不合法情况
	dirs := strings.Split(path, ",") //bug:路径名带有,
	for _, dir := range dirs {
		log.Println(dir)
	}

	for _, dir := range dirs {
		num := 0

		dir = addEscapeBackslash(dir) //转义
		//if d, _ := os.Stat(path); !d.IsDir() {
		//	//报错
		//}
		log.Println("start prasing:   " + dir)
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
					err = os.Mkdir(outputDir, 0755)
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
				serial_num++

				err = os.WriteFile(outputPath, []byte(codeContent), 0644)
				if err != nil {
					return fmt.Errorf("failed to write to file %s: %v", outputPath, err)
				}
				num++
				if num%5000 == 0 {
					log.Println("Number of files currently parsed: " + strconv.Itoa(num))
				}
			}
			return nil
		})
		log.Println("directory:  " + dir + "   total: " + strconv.Itoa(num))
		if err != nil {
			log.Printf("end Error: %v\n", err)
		}

	}
}

type Hsdoc struct {
	XMLName xml.Name `xml:"hsdoc"`
	Code    string   `xml:"code"`
}

func clearOutputDir() error {
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return nil
	}
	files, err := os.ReadDir(outputDir)
	if err != nil {
		return err
	}
	for _, file := range files {
		err = os.RemoveAll(filepath.Join(outputDir, file.Name()))
		if err != nil {
			return err
		}
	}
	err = os.RemoveAll(outputDir)
	if err != nil {
		return err
	}
	err = os.Mkdir(outputDir, 0755)
	if err != nil {
		return err
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
	data, err := os.ReadFile(transferFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &transfer)
	if err != nil {
		return err
	}
	return nil
}
