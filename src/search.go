package main

import (
	"bufio"
	"fmt"
	"github.com/lxn/walk"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type SearchResultInfo struct {
	CallChain     []string
	TargetRowNums []string
	Errs          []string
}

func _search(file_or_directory string, target string, mode int, mw *MyMainWindow) *SearchResultInfo {
	bitmap := NewBitmap(100)
	path, err := os.Stat(file_or_directory)
	if err != nil {
		walk.MsgBox(mw, "警告", err.Error(), walk.MsgBoxIconError)
		return nil
	}
	if path.IsDir() {
		return directory_dfs(file_or_directory, target, mode, bitmap)
	} else {
		return file_dfs(file_or_directory, target, mode, bitmap)
	}
}

func directory_dfs(directory string, target string, mode int, bmp *Bitmap) *SearchResultInfo {
	result := &SearchResultInfo{}
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		//log.Println("start at: " + path)
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
			func_name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			intput_filename := outputDir + "/" + func_name + ".code.txt"
			_func_name := fmt.Sprintf("[%s]", func_name)
			//for _, result := range file_dfs(intput_filename, target, mode) {
			//	results = append(results, strings.TrimSuffix(path, filepath.Base(path)) + ": " + result)
			//}

			if tv, exist := transfer[_func_name]; exist && !bmp.IsSet(tv.SerialNumber) {
				bmp.Set(transfer[_func_name].SerialNumber)
				ret := file_dfs(intput_filename, target, mode, bmp)
				if len(ret.Errs) != 0 {
					result.Errs = append(result.Errs, ret.Errs...)
				}
				result.CallChain = append(result.CallChain, ret.CallChain...)
				result.TargetRowNums = append(result.TargetRowNums, ret.TargetRowNums...)
				bmp.Clear(tv.SerialNumber)
			}

		}
		return nil
	})
	if err != nil {
		log.Println(err)
		return nil
	}
	return result
}

func file_dfs(_filepath string, target string, mode int, bmp *Bitmap) *SearchResultInfo {
	//log.Println("now in: " + _filepath)
	result := &SearchResultInfo{}
	var matches []string //该文件调用的原子或业务逻辑

	var regex *regexp.Regexp
	switch mode {
	//精准匹配
	case EXACT_MATCH:
		regex = regexp.MustCompile("\\b" + regexp.QuoteMeta(target) + "\\b")
	//正则模糊匹配
	case REGEX_MATCH:
		regex = regexp.MustCompile(target)
	}

	file, err := os.Open(_filepath)
	if err != nil {
		result.Errs = append(result.Errs, fmt.Sprintf("Error opening file : %v\n", err))
		return result
	}
	defer file.Close()

	func_name := fmt.Sprintf("[%s]", strings.TrimSuffix(filepath.Base(_filepath), ".code.txt")) //[AF_xxx|AS_xx|LF_xx....]
	bmp.Set(transfer[func_name].SerialNumber)

	func_regex := regexp.MustCompile(`\[(AS|AF|AP|LF|LS)_[^]]+\]`)
	is_found := false
	scanner := bufio.NewScanner(file)
	lineNumber := 1
	seen := make(map[string]bool) //去重
	var ans_lines []int           //当前文件匹配到target所在行
	var first_matches_lines []int
	var foundString string

	for scanner.Scan() {
		line := scanner.Text()
		if regex.MatchString(line) {
			is_found = true
			ans_lines = append(ans_lines, lineNumber)
			if mode == REGEX_MATCH {
				foundString = regex.FindString(line)
			}
		}

		//考虑每一行只有一个[AS|AF|AP|LF|LS]
		submatch := func_regex.FindString(line)
		if submatch != "" && !seen[submatch] {
			seen[submatch] = true
			first_matches_lines = append(first_matches_lines, lineNumber)
			matches = append(matches, submatch)
		}
		//TODO考虑每一行有多个[AS|AF|AP|LF|LS]
		//submatches := M_regex.FindAllString(line, -1)
		//if submatches != nil {
		//	matches = append(matches, submatches...)
		//	log.Println(len(submatches))
		//}
		lineNumber++
	}

	var row string
	if is_found {
		for _, line := range ans_lines {
			row += fmt.Sprintf("<%d>", line)
		}
		if mode == EXACT_MATCH {
			result.CallChain = append(result.CallChain, func_name)
		} else if mode == REGEX_MATCH {
			result.CallChain = append(result.CallChain, fmt.Sprintf("%s::%s", func_name, foundString))
		}
		result.TargetRowNums = append(result.TargetRowNums, row)
	}

	for id, match_func_name := range matches {
		next_file := outputDir + "/" + match_func_name[1:len(match_func_name)-1] + ".code.txt"
		//dfs
		if tv, exist := transfer[match_func_name]; exist && !bmp.IsSet(tv.SerialNumber) {
			bmp.Set(tv.SerialNumber)
			rets := file_dfs(next_file, target, mode, bmp)
			if rets.Errs != nil {
				for _, e := range rets.Errs {
					//那个文件在哪一行调用那个方法导致报错
					result.Errs = append(result.Errs, fmt.Sprintf("%s<%d> -> %s", func_name, first_matches_lines[id], e))
				}
			}

			if len(rets.CallChain) != len(rets.TargetRowNums) {
				LOG.Fatalf("程序内部错误，callchainSize:%d, tartgetrownumsSize: %d", len(rets.CallChain), len(rets.TargetRowNums))
			}
			for i, call_chain := range rets.CallChain {
				result.CallChain = append(result.CallChain, func_name+" -> "+call_chain)
				result.TargetRowNums = append(result.TargetRowNums, rets.TargetRowNums[i])
			}
			bmp.Clear(tv.SerialNumber)
		} else {
			if !exist {
				result.Errs = append(result.Errs, fmt.Sprintf("%s<%d> -> %s was not found", func_name, first_matches_lines[id], match_func_name))
			} else {
				result.Errs = append(result.Errs, fmt.Sprintf("%s<%d> -> %s has been visited", match_func_name, first_matches_lines[id], func_name))

			}
		}

	}

	if err := scanner.Err(); err != nil {
		log.Println("Error reading file:", err)
	}

	return result
}
