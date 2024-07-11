package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func _search(file_or_directory string, target string, mode int) []string {
	path, err := os.Stat(file_or_directory)
	if err != nil {
		fmt.Println(err)
	}
	if path.IsDir() {
		return directory_dfs(file_or_directory, target, mode)
	} else {
		return file_dfs(file_or_directory, target, mode)
	}
}

func directory_dfs(directory string, target string, mode int) []string {
	var results []string
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}
		if !info.IsDir() {
			results = append(results, file_dfs(filepath.Base(path), target, mode)...)
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	return results
}

func file_dfs(file string, target string, mode int) []string {
	var results []string
	return results
}
