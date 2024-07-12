package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func _search(file_or_directory string, target string, mode string) []string {
	path, err := os.Stat(file_or_directory)
	if err != nil {
		log.Println(err)
	}
	if path.IsDir() {
		return directory_dfs(file_or_directory, target, mode)
	} else {
		return file_dfs(file_or_directory, target, mode)
	}
}

func directory_dfs(directory string, target string, mode string) []string {
	var results []string
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		//跳过svn目录
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}
		if !info.IsDir() {
			//输入类似："D:\1_hundsun代码\DevCodes\经纪业务运营平台V21\业务逻辑\存管\UFT接口管理\服务\LS_UFT接口管理_UFT系统委托同步结果查询.service_design"
			//添加文件目录
			intput_filename := outputDir + "/" + strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)) + ".code.txt"
			//for _, result := range file_dfs(intput_filename, target, mode) {
			//	results = append(results, strings.TrimSuffix(path, filepath.Base(path)) + ": " + result)
			//}

			results = append(results, file_dfs(intput_filename, target, mode)...)
		}
		return nil
	})
	if err != nil {
		log.Printf("Error: %v\n", err)
	}
	return results
}

func file_dfs(_filepath string, target string, mode string) []string {
	var results []string
	var matches []string //该文件调用的原子或业务逻辑

	var regex *regexp.Regexp
	switch mode {
	//精准匹配
	case EXACT_MATCH:
		regex = regexp.MustCompile("\\b" + regexp.QuoteMeta(target) + "\\b")
	//正则模糊匹配
	case REGEX_MATCH:
		//regex = regexp.MustCompile("\\.\\d+")
	}

	M_regex := regexp.MustCompile(`(AS|AF|AP|LF|LS)_[^]]+`)

	file, err := os.Open(_filepath)
	if err != nil {
		log.Printf("Error opening file : %v\n", err)
		return results
	}
	defer file.Close()

	is_found := false
	scanner := bufio.NewScanner(file)
	lineNumber := 1
	var ans_lines []int
	seen := make(map[string]bool) //去重
	for scanner.Scan() {
		line := scanner.Text()
		if regex.MatchString(line) {
			ans_lines = append(ans_lines, lineNumber)
			//debug
			//log.Printf("find target at %s : line<%d>\n", strings.TrimSuffix(filepath.Base(_filepath), filepath.Ext(_filepath)), lineNumber)
			is_found = true
		}

		//考虑每一行只有一个[AS|AF|AP|LF|LS]
		submatch := M_regex.FindString(line)
		if submatch != "" && !seen[submatch] {
			seen[submatch] = true
			matches = append(matches, submatch)
		}
		//submatches := M_regex.FindAllString(line, -1)
		//if submatches != nil {
		//	matches = append(matches, submatches...)
		//	log.Println(len(submatches))
		//}
		lineNumber++
	}

	//log.Println("total : " + strconv.Itoa(len(ans_lines))) //debug
	if is_found {
		var this_file_result = strings.TrimSuffix(filepath.Base(_filepath), ".code.txt") + " in line" //带路径
		for _, line := range ans_lines {
			this_file_result += fmt.Sprintf("<%d>", line)
		}
		results = append(results, this_file_result)
	}

	//dfs
	for _, match := range matches {
		next_file := outputDir + "/" + match + ".code.txt"
		ret_results := file_dfs(next_file, target, mode)
		for _, ret_result := range ret_results {
			results = append(results, strings.TrimSuffix(filepath.Base(_filepath), ".code.txt")+" -> "+ret_result)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Println("Error reading file:", err)
	}

	return results
}

func test() {
	//test
	for _, res := range directory_dfs(`D:\1_hundsun代码\DevCodes\经纪业务运营平台V21\业务逻辑\存管\UFT接口管理`, "hs_strcpy", EXACT_MATCH) {
		log.Println(res)
	}

}
