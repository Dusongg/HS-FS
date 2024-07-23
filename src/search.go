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

var (
	memo     = make(map[string]*SearchResultInfo)
	memoLock sync.RWMutex
)

func Search_(searchScope string, target string, mode int, mw *MyMainWindow) *SearchResultInfo {
	memo = make(map[string]*SearchResultInfo)
	path, err := os.Stat(searchScope)
	if err != nil {
		walk.MsgBox(mw, "警告", err.Error(), walk.MsgBoxIconError)
		return nil
	}
	if path.IsDir() {
		return asyncDirectoryDFS(searchScope, target, mode)
	} else {
		if !originalSearch {
			searchScope = filepath.Join(ROOT_DIR, filepath.Base(searchScope)+".code.txt")
		}
		return fileDFS(searchScope, target, mode)

	}
}

func asyncDirectoryDFS(searchScope string, target string, mode int) *SearchResultInfo {
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
				result := directoryDFS(searchScope, target, mode)
				results <- result
			}(filepath.Join(searchScope, subDir.Name()), target, mode, NewBitmap(DEFAULTMAPSIZE))
		} else {
			go func(searchScope string, target string, mode int, bitmap *Bitmap) {
				defer wg.Done()
				if !originalSearch {
					searchScope = filepath.Join(ROOT_DIR, filepath.Base(searchScope)+".code.txt")
				}
				result := fileDFS(searchScope, target, mode)
				results <- result
			}(filepath.Join(searchScope, subDir.Name()), target, mode, NewBitmap(DEFAULTMAPSIZE))
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

func directoryDFS(directory string, target string, mode int) *SearchResultInfo {
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
			path = outputDir + "/" + funcName[1:len(funcName)-1] + ".code.txt"
			//for _, result := range file_dfs(intput_filename, target, mode) {
			//	results = append(results, strings.TrimSuffix(path, filepath.Base(path)) + ": " + result)
			//}

			if _, exist := transfer[funcName]; exist {
				ret := fileDFS(path, target, mode)
				if len(ret.Errs) != 0 {
					result.Errs = append(result.Errs, ret.Errs...)
				}
				result.CallChain = append(result.CallChain, ret.CallChain...)
				result.TargetRowNums = append(result.TargetRowNums, ret.TargetRowNums...)
			}

		}
		return nil
	})
	if err != nil {
		ERROR.Println("open dir when search: ", err)
		return nil
	}
	return result
}

// _filepath: outputDir + "/" + funcName[1:len(funcName)-1] + ".code.txt" 或 .aservice_design....
func fileDFS(_filepath string, target string, mode int) *SearchResultInfo {
	memoLock.RLock()
	if pre, exist := memo[_filepath]; exist {
		memoLock.RUnlock()
		return pre
	} else {
		memoLock.RUnlock()
	}
	result := &SearchResultInfo{}
	var matches []string //该文件调用的原子或业务逻辑

	var regex *regexp.Regexp
	switch mode {
	//精准匹配
	case EXACT_MATCH:
		regex = regexp.MustCompile(regexp.QuoteMeta(target))
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

	var funcName string
	if originalSearch {
		funcName = fmt.Sprintf("[%s]", strings.TrimSuffix(filepath.Base(_filepath), filepath.Ext(_filepath)))
	} else {
		funcName = fmt.Sprintf("[%s]", strings.TrimSuffix(filepath.Base(_filepath), ".code.txt")) //[AF_xxx|AS_xx|LF_xx....]
	}

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
	}

	for id, matchedFuncName := range matches {
		var nextFile string
		if transfer == nil {
			FATAL.Fatalf("transfer is nil")
		}
		if originalSearch {
			if _, exist := transfer[matchedFuncName]; exist {
				nextFile = transfer[matchedFuncName].OriginPath
			}
		} else {
			nextFile = outputDir + "/" + matchedFuncName[1:len(matchedFuncName)-1] + ".code.txt"
		}
		//dfs && !bmp.IsSet(tv.SerialNumber)
		if _, exist := transfer[matchedFuncName]; exist && matchedFuncName != funcName {
			rets := fileDFS(nextFile, target, mode)
			if rets.Errs != nil {
				for _, e := range rets.Errs {
					//那个文件在哪一行调用那个方法导致报错
					result.Errs = append(result.Errs, fmt.Sprintf("%s<%d> -> %s", funcName, firstMatchesLines[id], e))
				}
			}

			if len(rets.CallChain) != len(rets.TargetRowNums) {
				FATAL.Fatalf("程序内部错误，callchainSize:%d, tartgetrownumsSize: %d", len(rets.CallChain), len(rets.TargetRowNums))
			}
			for i, callChain := range rets.CallChain {
				result.CallChain = append(result.CallChain, funcName+" -> "+callChain)
				result.TargetRowNums = append(result.TargetRowNums, rets.TargetRowNums[i])
			}
		} else {
			if !exist {
				result.Errs = append(result.Errs, fmt.Sprintf("%s<%d> -> %s was not found", funcName, firstMatchesLines[id], matchedFuncName))
			} else {
				//result.Errs = append(result.Errs, fmt.Sprintf("%s<%d> -> %s has been visited", matchedFuncName, firstMatchesLines[id], funcName))
			}
		}

	}

	if err := scanner.Err(); err != nil {
		ERROR.Println("Error reading file:", err)
	}

	memoLock.Lock()
	memo[_filepath] = result
	memoLock.Unlock()
	return result
}

func asyncSerach(searchScope string, target string, mode int) *SearchResultInfo {
	memo = make(map[string]*SearchResultInfo)
	_, err := os.Stat(searchScope)
	if err != nil {
		return nil
	}

	var n sync.WaitGroup
	resultChan := make(chan *SearchResultInfo, 100)
	results := &SearchResultInfo{}

	n.Add(1)
	go walkDirSearch(searchScope, target, mode, &n, resultChan)

	go func() {
		for ret := range resultChan {
			results.CallChain = append(results.CallChain, ret.CallChain...)
			results.TargetRowNums = append(results.TargetRowNums, ret.TargetRowNums...)
			results.Errs = append(results.Errs, ret.Errs...)
		}
	}()
	n.Wait()
	return results
}

func walkDirSearch(dir string, target string, mode int, n *sync.WaitGroup, resultChan chan *SearchResultInfo) {
	defer n.Done()
	for _, entry := range direntsSearch(dir) {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			n.Add(1)
			go walkDirSearch(path, target, mode, n, resultChan)
		} else {
			if !originalSearch {
				path = filepath.Join(outputDir, strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))+".code.txt")
			}
			resultChan <- fileDFS(path, target, mode)
		}
	}
}

func direntsSearch(dir string) []os.DirEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "du1: %v\n", err)
		return nil
	}
	return entries
}
