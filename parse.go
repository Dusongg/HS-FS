package main

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var outputDir string = "./output"

func prase(path string) {

	//bug处理：文件路径不合法情况
	dirs := strings.Split(path, ",")
	for _, dir := range dirs {
		log.Println(dir)
	}
	//dirs := []string{
	//	"D:\\1_hundsun代码\\DevCodes\\经纪业务运营平台V21\\业务逻辑",
	//	"D:\\1_hundsun代码\\DevCodes\\经纪业务运营平台V21\\原子",
	//}

	for _, dir := range dirs {
		num := 0

		dir = addEscapeBackslash(dir)
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

				codeContent := filterCommentedCode(hsdoc.Code)

				if _, err := os.Stat(outputDir); os.IsNotExist(err) {
					err = os.Mkdir(outputDir, 0755)
					if err != nil {
						return fmt.Errorf("failed to create output directory: %v", err)
					}
				}
				outputFileName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)) + ".code.txt"
				outputPath := filepath.Join(outputDir, outputFileName)

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
	//lines := strings.Split(code, "\n")
	//var uncommentedLines []string
	//commentRegex := regexp.MustCompile(`^\s*(//|--)`)
	//for _, line := range lines {
	//	if !commentRegex.MatchString(line) {
	//		//去除行内注释
	//		if idx := strings.IndexAny(line, "//"); idx != -1 {
	//			line = line[:idx] // 删除注释及其后面的内容
	//		} else if idx2 := strings.IndexAny(line, "--"); idx2 != -1 {
	//			line = line[:idx2]
	//		}
	//		uncommentedLines = append(uncommentedLines, line)
	//	}
	//}
	//return strings.Join(uncommentedLines, "\n")
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
