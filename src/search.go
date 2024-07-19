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
	"sync"
)

type SearchResultInfo struct {
	CallChain     []string
	TargetRowNums []string
	Errs          []string
}

func Search_(searchScope string, target string, mode int, mw *MyMainWindow) *SearchResultInfo {
	bitmap := NewBitmap(100)
	path, err := os.Stat(searchScope)
	if err != nil {
		walk.MsgBox(mw, "警告", err.Error(), walk.MsgBoxIconError)
		return nil
	}
	if path.IsDir() {
		return asyncDirectoryDFS(searchScope, target, mode, bitmap)
	} else {
		return fileDFS(searchScope, target, mode, bitmap)
	}
}

func asyncDirectoryDFS(searchScope string, target string, mode int, bitmap *Bitmap) *SearchResultInfo {
	subDirs, err := os.ReadDir(searchScope)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	results := make(chan *SearchResultInfo, len(subDirs))

	for _, subDir := range subDirs {
		if strings.HasPrefix(subDir.Name(), ".") {
			continue
		}
		wg.Add(1)
		if subDir.IsDir() {
			go func(searchScope string, target string, mode int, bitmap *Bitmap) {
				defer wg.Done()
				result := directoryDFS(searchScope, target, mode, bitmap)
				results <- result
			}(filepath.Join(searchScope, subDir.Name()), target, mode, bitmap)
		} else {
			go func(searchScope string, target string, mode int, bitmap *Bitmap) {
				defer wg.Done()
				result := fileDFS(searchScope, target, mode, bitmap)
				results <- result
			}(filepath.Join(searchScope, subDir.Name()), target, mode, bitmap)
		}
	}

	go func() {
		wg.Wait()
		close(results)
	}()
	finalResults := &SearchResultInfo{}
	for result := range results {
		if len(result.Errs) != 0 {
			finalResults.Errs = append(finalResults.Errs, result.Errs...)
		}
		finalResults.CallChain = append(finalResults.CallChain, result.CallChain...)
		finalResults.TargetRowNums = append(finalResults.TargetRowNums, result.TargetRowNums...)
	}
	return finalResults
}

func directoryDFS(directory string, target string, mode int, bmp *Bitmap) *SearchResultInfo {
	result := &SearchResultInfo{}
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		//LOG.Println("start at: " + path)
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
			funcName := fmt.Sprintf("[%s]", strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
			inputFilename := outputDir + "/" + funcName[1:len(funcName)-1] + ".code.txt"
			//for _, result := range file_dfs(intput_filename, target, mode) {
			//	results = append(results, strings.TrimSuffix(path, filepath.Base(path)) + ": " + result)
			//}

			if tv, exist := transfer[funcName]; exist && !bmp.IsSet(tv.SerialNumber) {
				bmp.Set(transfer[funcName].SerialNumber)
				ret := fileDFS(inputFilename, target, mode, bmp)
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

func fileDFS(_filepath string, target string, mode int, bmp *Bitmap) *SearchResultInfo {
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

	funcName := fmt.Sprintf("[%s]", strings.TrimSuffix(filepath.Base(_filepath), ".code.txt")) //[AF_xxx|AS_xx|LF_xx....]
	bmp.Set(transfer[funcName].SerialNumber)

	funcRegex := regexp.MustCompile(`\[(AS|AF|AP|LF|LS)_[^]]+\]`)
	isFound := false
	scanner := bufio.NewScanner(file)
	lineNumber := 1
	seen := make(map[string]bool) //去重
	var ansLines []int            //当前文件匹配到target所在行
	var firstMatchesLines []int
	var foundString string

	for scanner.Scan() {
		line := scanner.Text()
		if regex == nil {
			return nil
		}
		if regex.MatchString(line) {
			isFound = true
			ansLines = append(ansLines, lineNumber)
			if mode == REGEX_MATCH {
				foundString = regex.FindString(line)
			}
		}

		//考虑每一行只有一个[AS|AF|AP|LF|LS]
		submatch := funcRegex.FindString(line)
		if submatch != "" && !seen[submatch] {
			seen[submatch] = true
			firstMatchesLines = append(firstMatchesLines, lineNumber)
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
	if isFound {
		for _, line := range ansLines {
			row += fmt.Sprintf("<%d>", line)
		}
		if mode == EXACT_MATCH {
			result.CallChain = append(result.CallChain, funcName)
		} else if mode == REGEX_MATCH {
			result.CallChain = append(result.CallChain, fmt.Sprintf("%s::%s", funcName, foundString))
		}
		result.TargetRowNums = append(result.TargetRowNums, row)

		//LOG.Println(result.CallChain)
	}

	for id, matchedFuncName := range matches {
		nextFile := outputDir + "/" + matchedFuncName[1:len(matchedFuncName)-1] + ".code.txt"
		//dfs
		if tv, exist := transfer[matchedFuncName]; exist && !bmp.IsSet(tv.SerialNumber) {
			bmp.Set(tv.SerialNumber)
			rets := fileDFS(nextFile, target, mode, bmp)
			if rets.Errs != nil {
				for _, e := range rets.Errs {
					//那个文件在哪一行调用那个方法导致报错
					result.Errs = append(result.Errs, fmt.Sprintf("%s<%d> -> %s", funcName, firstMatchesLines[id], e))
				}
			}

			if len(rets.CallChain) != len(rets.TargetRowNums) {
				LOG.Fatalf("程序内部错误，callchainSize:%d, tartgetrownumsSize: %d", len(rets.CallChain), len(rets.TargetRowNums))
			}
			for i, callChain := range rets.CallChain {
				result.CallChain = append(result.CallChain, funcName+" -> "+callChain)
				result.TargetRowNums = append(result.TargetRowNums, rets.TargetRowNums[i])
			}
			bmp.Clear(tv.SerialNumber)
		} else {
			if !exist {
				result.Errs = append(result.Errs, fmt.Sprintf("%s<%d> -> %s was not found", funcName, firstMatchesLines[id], matchedFuncName))
			} else {
				result.Errs = append(result.Errs, fmt.Sprintf("%s<%d> -> %s has been visited", matchedFuncName, firstMatchesLines[id], funcName))

			}
		}

	}

	if err := scanner.Err(); err != nil {
		log.Println("Error reading file:", err)
	}

	return result
}
