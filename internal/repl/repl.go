package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/YuleBest/pvzpak/internal/pak"
)

type PakFileSystem struct {
	Files       []pak.FileInfo
	CurrentPath string
}

func NewPakFileSystem(files []pak.FileInfo) *PakFileSystem {
	return &PakFileSystem{
		Files:       files,
		CurrentPath: "/",
	}
}

func (fs *PakFileSystem) ResolvePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return fs.CurrentPath
	}

	if strings.HasPrefix(path, "/") {
		normalized := fs.NormalizePath(path)
		if normalized == "" {
			return "/"
		}
		return normalized
	}

	var resultParts []string
	if fs.CurrentPath != "/" {
		resultParts = strings.Split(fs.CurrentPath[1:], "/")
	}

	for _, part := range strings.Split(path, "/") {
		part = strings.TrimSpace(part)
		if part == "" || part == "." {
			continue
		} else if part == ".." {
			if len(resultParts) > 0 {
				resultParts = resultParts[:len(resultParts)-1]
			}
		} else {
			resultParts = append(resultParts, part)
		}
	}

	if len(resultParts) == 0 {
		return "/"
	}
	return "/" + strings.Join(resultParts, "/")
}

func (fs *PakFileSystem) NormalizePath(path string) string {
	var parts []string
	for _, part := range strings.Split(path, "/") {
		part = strings.TrimSpace(part)
		if part == "" || part == "." {
			continue
		} else if part == ".." {
			if len(parts) > 0 {
				parts = parts[:len(parts)-1]
			}
		} else {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "/" + strings.Join(parts, "/")
}

func (fs *PakFileSystem) GetEntriesAtPath(targetPath string) ([]string, []*pak.FileInfo) {
	resolvedPath := fs.ResolvePath(targetPath)

	var directories []string
	var files []*pak.FileInfo

	currentPrefix := ""
	if resolvedPath != "/" {
		currentPrefix = resolvedPath[1:]
	}

	for i := range fs.Files {
		filePath := fs.Files[i].FileName

		if currentPrefix == "" {
			if slashPos := strings.Index(filePath, "\\"); slashPos >= 0 {
				dirName := filePath[:slashPos]
				if !contains(directories, dirName) {
					directories = append(directories, dirName)
				}
			} else {
				files = append(files, &fs.Files[i])
			}
		} else {
			normalizedPrefix := strings.ReplaceAll(currentPrefix, "/", "\\")
			if strings.HasPrefix(filePath, normalizedPrefix) {
				remaining := filePath[len(normalizedPrefix):]
				if strings.HasPrefix(remaining, "\\") {
					remaining = remaining[1:]
				} else if remaining != "" {
					continue
				}

				if slashPos := strings.Index(remaining, "\\"); slashPos >= 0 {
					dirName := remaining[:slashPos]
					if dirName != "" && !contains(directories, dirName) {
						directories = append(directories, dirName)
					}
				} else if remaining != "" {
					files = append(files, &fs.Files[i])
				}
			}
		}
	}

	sort.Slice(directories, func(i, j int) bool {
		return strings.ToLower(directories[i]) < strings.ToLower(directories[j])
	})

	sort.Slice(files, func(i, j int) bool {
		nameA := files[i].FileName
		if idx := strings.LastIndex(nameA, "\\"); idx >= 0 {
			nameA = nameA[idx+1:]
		}
		nameB := files[j].FileName
		if idx := strings.LastIndex(nameB, "\\"); idx >= 0 {
			nameB = nameB[idx+1:]
		}
		return strings.ToLower(nameA) < strings.ToLower(nameB)
	})

	return directories, files
}

func (fs *PakFileSystem) ChangeDirectory(path string) error {
	targetPath := fs.ResolvePath(path)

	if targetPath == "/" {
		fs.CurrentPath = "/"
		return nil
	}

	parentPath := "/"
	if pos := strings.LastIndex(targetPath, "/"); pos > 0 {
		parentPath = targetPath[:pos]
	} else if pos == 0 {
		parentPath = "/"
	}

	dirName := targetPath[strings.LastIndex(targetPath, "/")+1:]

	parentDirs, _ := fs.GetEntriesAtPath(parentPath)
	for _, d := range parentDirs {
		if d == dirName {
			fs.CurrentPath = targetPath
			return nil
		}
	}

	return fmt.Errorf("目录不存在: %s", path)
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func FormatFileInfo(file *pak.FileInfo, formatStr string) string {
	fullPathUnix := strings.ReplaceAll(file.FileName, "\\", "/")
	parts := strings.Split(fullPathUnix, "/")
	fileName := parts[len(parts)-1]
	dirPath := ""
	if len(parts) > 1 {
		dirPath = strings.Join(parts[:len(parts)-1], "/")
	}

	result := formatStr
	result = strings.ReplaceAll(result, "$path", fullPathUnix)
	result = strings.ReplaceAll(result, "$name", fileName)
	result = strings.ReplaceAll(result, "$dir", dirPath)
	result = strings.ReplaceAll(result, "$size", fmt.Sprintf("%d", file.ZSize))
	result = strings.ReplaceAll(result, "$osize", fmt.Sprintf("%d", file.Size))
	return result
}

func FormatDirInfo(dirPath string, formatStr string) string {
	dirPathUnix := strings.ReplaceAll(dirPath, "\\", "/")
	parts := strings.Split(dirPathUnix, "/")
	dirName := parts[len(parts)-1]
	parentPath := ""
	if len(parts) > 1 {
		parentPath = strings.Join(parts[:len(parts)-1], "/")
	}

	result := formatStr
	result = strings.ReplaceAll(result, "$path", dirPathUnix)
	result = strings.ReplaceAll(result, "$name", dirName)
	result = strings.ReplaceAll(result, "$dir", parentPath)
	result = strings.ReplaceAll(result, "$size", "<DIR>")
	result = strings.ReplaceAll(result, "$osize", "<DIR>")
	return result
}

func ParseCommandLine(input string) (string, string, bool) {
	if idx := strings.Index(input, " > "); idx >= 0 {
		return strings.TrimSpace(input[:idx]), strings.TrimSpace(input[idx+3:]), true
	}
	return strings.TrimSpace(input), "", false
}

func ParseCommandArgs(input string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false

	for _, ch := range input {
		switch {
		case ch == '"':
			inQuotes = !inQuotes
		case (ch == ' ' || ch == '\t') && !inQuotes:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

type OutputBuffer struct {
	Lines []string
}

func NewOutputBuffer() *OutputBuffer {
	return &OutputBuffer{Lines: []string{}}
}

func (ob *OutputBuffer) Writeln(line string) {
	ob.Lines = append(ob.Lines, line)
}

func GlobMatch(text, pattern string) bool {
	return globMatchRecursive([]rune(text), []rune(pattern), 0, 0)
}

func globMatchRecursive(text, pattern []rune, tIdx, pIdx int) bool {
	if pIdx >= len(pattern) {
		return tIdx >= len(text)
	}
	if tIdx >= len(text) {
		for _, c := range pattern[pIdx:] {
			if c != '*' {
				return false
			}
		}
		return true
	}

	switch pattern[pIdx] {
	case '*':
		if globMatchRecursive(text, pattern, tIdx, pIdx+1) {
			return true
		}
		for i := tIdx; i < len(text); i++ {
			if globMatchRecursive(text, pattern, i+1, pIdx+1) {
				return true
			}
		}
		return false
	case '?':
		return globMatchRecursive(text, pattern, tIdx+1, pIdx+1)
	case '[':
		for j := pIdx + 1; j < len(pattern); j++ {
			if pattern[j] == ']' {
				charClass := pattern[pIdx+1 : j]
				if matchesCharClass(text[tIdx], charClass) {
					return globMatchRecursive(text, pattern, tIdx+1, j+1)
				}
				return false
			}
		}
		return text[tIdx] == pattern[pIdx] && globMatchRecursive(text, pattern, tIdx+1, pIdx+1)
	default:
		return text[tIdx] == pattern[pIdx] && globMatchRecursive(text, pattern, tIdx+1, pIdx+1)
	}
}

func matchesCharClass(ch rune, charClass []rune) bool {
	if len(charClass) == 0 {
		return false
	}
	negated := charClass[0] == '!'
	charsToCheck := charClass
	if negated {
		charsToCheck = charClass[1:]
	}

	matched := false
	for i := 0; i < len(charsToCheck); i++ {
		if i+2 < len(charsToCheck) && charsToCheck[i+1] == '-' {
			if ch >= charsToCheck[i] && ch <= charsToCheck[i+2] {
				matched = true
				break
			}
			i += 2
		} else if ch == charsToCheck[i] {
			matched = true
			break
		}
	}

	if negated {
		return !matched
	}
	return matched
}

func ListDirectoryToBuffer(fs *PakFileSystem, targetPath string, output *OutputBuffer) {
	directories, files := fs.GetEntriesAtPath(targetPath)

	for _, dir := range directories {
		output.Writeln(dir)
	}
	for _, file := range files {
		fileName := file.FileName
		if idx := strings.LastIndex(fileName, "\\"); idx >= 0 {
			fileName = fileName[idx+1:]
		}
		output.Writeln(fileName)
	}

	if len(directories) == 0 && len(files) == 0 {
		output.Writeln("目录为空")
	}
}

func FindAllFilesInPathToBuffer(fs *PakFileSystem, basePath string, formatStr string, output *OutputBuffer) {
	resolvedPath := fs.ResolvePath(basePath)
	prefix := ""
	if resolvedPath != "/" {
		prefix = resolvedPath[1:]
	}

	for i := range fs.Files {
		filePath := fs.Files[i].FileName
		include := false

		if prefix == "" {
			include = true
		} else {
			normalizedPrefix := strings.ReplaceAll(prefix, "/", "\\")
			if strings.HasPrefix(filePath, normalizedPrefix) {
				remaining := filePath[len(normalizedPrefix):]
				if strings.HasPrefix(remaining, "\\") || remaining == "" {
					include = true
				}
			}
		}

		if include {
			output.Writeln(FormatFileInfo(&fs.Files[i], formatStr))
		}
	}
}

func FindByNameToBuffer(fs *PakFileSystem, filename string, formatStr string, output *OutputBuffer) {
	currentPrefix := ""
	if fs.CurrentPath != "/" {
		currentPrefix = fs.CurrentPath[1:]
	}

	var foundFiles []*pak.FileInfo
	foundDirs := make(map[string]bool)

	for i := range fs.Files {
		filePath := fs.Files[i].FileName
		fileInCurrentPath := false

		if currentPrefix == "" {
			fileInCurrentPath = true
		} else {
			normalizedPrefix := strings.ReplaceAll(currentPrefix, "/", "\\")
			if strings.HasPrefix(filePath, normalizedPrefix) {
				if len(filePath) == len(normalizedPrefix) || filePath[len(normalizedPrefix)] == '\\' {
					fileInCurrentPath = true
				}
			}
		}

		if fileInCurrentPath {
			relativePath := filePath
			if currentPrefix != "" {
				normalizedPrefix := strings.ReplaceAll(currentPrefix, "/", "\\")
				remaining := filePath[len(normalizedPrefix):]
				relativePath = strings.TrimPrefix(remaining, "\\")
			}

			pathParts := strings.Split(relativePath, "\\")
			fileBasename := pathParts[len(pathParts)-1]
			if fileBasename == filename {
				foundFiles = append(foundFiles, &fs.Files[i])
			}

			for _, part := range pathParts[:len(pathParts)-1] {
				if part == filename {
					dirParts := []string{}
					for _, p := range pathParts {
						dirParts = append(dirParts, p)
						if p == filename {
							break
						}
					}
					fullDirPath := strings.Join(dirParts, "\\")
					if currentPrefix != "" {
						fullDirPath = strings.ReplaceAll(currentPrefix, "/", "\\") + "\\" + fullDirPath
					}
					foundDirs[fullDirPath] = true
				}
			}
		}
	}

	var sortedDirs []string
	for d := range foundDirs {
		sortedDirs = append(sortedDirs, d)
	}
	sort.Strings(sortedDirs)

	for _, dir := range sortedDirs {
		output.Writeln(FormatDirInfo(dir, formatStr))
	}
	for _, file := range foundFiles {
		output.Writeln(FormatFileInfo(file, formatStr))
	}
}

func FindByPatternToBuffer(fs *PakFileSystem, pattern string, formatStr string, output *OutputBuffer) {
	var searchPattern string
	if strings.HasPrefix(pattern, "/") {
		searchPattern = pattern[1:]
	} else {
		if fs.CurrentPath == "/" {
			searchPattern = pattern
		} else {
			searchPattern = fs.CurrentPath[1:] + "/" + pattern
		}
	}
	normalizedPattern := strings.ReplaceAll(searchPattern, "/", "\\")

	for i := range fs.Files {
		if GlobMatch(fs.Files[i].FileName, normalizedPattern) {
			output.Writeln(FormatFileInfo(&fs.Files[i], formatStr))
		}
	}
}

func FindByRegexToBuffer(fs *PakFileSystem, regexPattern string, formatStr string, output *OutputBuffer) {
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		output.Writeln(fmt.Sprintf("正则表达式错误: %s", err))
		return
	}

	allDirs := make(map[string]bool)
	for _, file := range fs.Files {
		unixPath := strings.ReplaceAll(file.FileName, "\\", "/")
		pathParts := strings.Split(unixPath, "/")
		for i := 0; i < len(pathParts)-1; i++ {
			dirPath := strings.Join(pathParts[:i+1], "/")
			allDirs[dirPath] = true
		}
	}

	var matchedDirs []string
	for dir := range allDirs {
		if re.MatchString(dir) {
			matchedDirs = append(matchedDirs, dir)
		}
	}
	sort.Strings(matchedDirs)
	for _, dir := range matchedDirs {
		output.Writeln(FormatDirInfo(strings.ReplaceAll(dir, "/", "\\"), formatStr))
	}

	for _, file := range fs.Files {
		unixPath := strings.ReplaceAll(file.FileName, "\\", "/")
		if re.MatchString(unixPath) {
			output.Writeln(FormatFileInfo(&file, formatStr))
		}
	}
}

func ShowPakInfoToBuffer(dataLen int, files []pak.FileInfo, output *OutputBuffer) {
	output.Writeln(fmt.Sprintf("PAK 文件大小: %.2f MB", float64(dataLen)/1024.0/1024.0))
	output.Writeln(fmt.Sprintf("文件数量: %d", len(files)))

	var totalCompressed uint32
	var totalUncompressed uint32
	for _, f := range files {
		totalCompressed += f.ZSize
		totalUncompressed += f.Size
	}

	output.Writeln(fmt.Sprintf("压缩总大小: %d bytes", totalCompressed))
	if totalUncompressed > 0 {
		output.Writeln(fmt.Sprintf("原始总大小: %d bytes", totalUncompressed))
		ratio := float64(totalCompressed) / float64(totalUncompressed) * 100.0
		output.Writeln(fmt.Sprintf("压缩率: %.1f%%", ratio))
	}
}

func RunBatchCommands(pakPath string, commands []string) error {
	data, err := os.ReadFile(pakPath)
	if err != nil {
		return err
	}

	encrypted := pak.DetectEncryption(data)
	if encrypted {
		pak.CryptData(data)
	}

	pakInfo, _, err := pak.ParsePakInfo(data)
	if err != nil {
		return err
	}

	fs := NewPakFileSystem(pakInfo.FileInfoLibrary)

	for _, cmdStr := range commands {
		cmd, redirectFile, hasRedirect := ParseCommandLine(cmdStr)
		parts := ParseCommandArgs(cmd)

		if len(parts) == 0 {
			continue
		}

		output := NewOutputBuffer()
		ExecuteCommand(fs, data, encrypted, parts, output)

		if hasRedirect {
			if err := os.WriteFile(redirectFile, []byte(strings.Join(output.Lines, "\n")), 0644); err != nil {
				return err
			}
		} else {
			for _, line := range output.Lines {
				fmt.Println(line)
			}
		}
	}

	return nil
}

func ExecuteCommand(fs *PakFileSystem, data []byte, encrypted bool, parts []string, output *OutputBuffer) {
	if len(parts) == 0 {
		return
	}
	command := parts[0]

	switch command {
	case "help", "h":
		ShowHelpToBuffer(output)
	case "ls", "dir":
		targetPath := ""
		if len(parts) > 1 {
			targetPath = parts[1]
		}
		ListDirectoryToBuffer(fs, targetPath, output)
	case "cd":
		if len(parts) > 1 {
			if err := fs.ChangeDirectory(parts[1]); err != nil {
				output.Writeln(fmt.Sprintf("错误: %s", err))
			}
		} else {
			fs.CurrentPath = "/"
		}
	case "find":
		ExecuteFindCommand(fs, data, parts, output)
	case "info":
		ShowPakInfoToBuffer(len(data), fs.Files, output)
	default:
		output.Writeln(fmt.Sprintf("未知命令: %s. 输入 'help' 查看可用命令", command))
	}
}

func ExecuteFindCommand(fs *PakFileSystem, data []byte, parts []string, output *OutputBuffer) {
	var formatStr string = "$path"
	var searchType string
	var searchValue string
	var showHelp bool
	var parseError bool
	var extractDir string

	for i := 1; i < len(parts); i++ {
		switch parts[i] {
		case "-help", "--help":
			showHelp = true
		case "-name":
			if i+1 < len(parts) {
				searchType = "name"
				searchValue = parts[i+1]
				i++
			} else {
				output.Writeln("错误: -name 需要指定文件名")
				parseError = true
			}
		case "-filter":
			if i+1 < len(parts) {
				searchType = "filter"
				searchValue = parts[i+1]
				i++
			} else {
				output.Writeln("错误: -filter 需要指定模式")
				parseError = true
			}
		case "-match":
			if i+1 < len(parts) {
				searchType = "match"
				searchValue = parts[i+1]
				i++
			} else {
				output.Writeln("错误: -match 需要指定正则表达式")
				parseError = true
			}
		case "-format":
			if i+1 < len(parts) {
				formatStr = parts[i+1]
				i++
			} else {
				output.Writeln("错误: -format 需要指定格式字符串")
				parseError = true
			}
		case "-extract":
			if i+1 < len(parts) {
				extractDir = parts[i+1]
				i++
			} else {
				output.Writeln("错误: -extract 需要指定目标目录")
				parseError = true
			}
		default:
			output.Writeln(fmt.Sprintf("未知参数: %s", parts[i]))
			parseError = true
		}
	}

	if showHelp {
		ShowFindHelp(output)
	} else if parseError {
		return
	} else if extractDir != "" {
		count, err := ExtractFilteredFiles(fs, data, extractDir, searchType, searchValue)
		if err != nil {
			output.Writeln(fmt.Sprintf("提取失败: %s", err))
		} else {
			output.Writeln(fmt.Sprintf("成功提取 %d 个文件到: %s", count, extractDir))
		}
	} else {
		switch searchType {
		case "name":
			FindByNameToBuffer(fs, searchValue, formatStr, output)
		case "filter":
			FindByPatternToBuffer(fs, searchValue, formatStr, output)
		case "match":
			FindByRegexToBuffer(fs, searchValue, formatStr, output)
		case "":
			FindAllFilesInPathToBuffer(fs, fs.CurrentPath, formatStr, output)
		default:
			output.Writeln("用法:")
			output.Writeln("  find [-format \"格式\"]                    列出当前目录下所有文件")
			output.Writeln("  find -name <filename> [-format \"格式\"]   查找指定文件名")
			output.Writeln("  find -filter <pattern> [-format \"格式\"]  根据通配符查找文件")
			output.Writeln("  find -match <regex> [-format \"格式\"]     根据正则表达式查找文件")
			output.Writeln("支持的通配符: * ? [abc] [a-z] [!abc]")
			output.Writeln("格式变量:")
			output.Writeln("  $path   - 文件完整路径")
			output.Writeln("  $name   - 文件名（不含路径）")
			output.Writeln("  $dir    - 目录路径")
			output.Writeln("  $size   - 文件大小（压缩后）")
			output.Writeln("  $osize  - 原始文件大小")
			output.Writeln("示例: find -format \"$path -- $size bytes\"")
		}
	}
}

func ShowHelpToBuffer(output *OutputBuffer) {
	output.Writeln("可用命令:")
	output.Writeln("  help, h             显示此帮助信息")
	output.Writeln("  ls [path]           列出目录内容 (支持相对/绝对路径)")
	output.Writeln("  cd <path>           切换目录 (支持 .., ./, ../, /abs/path, rel/path)")
	output.Writeln("  find                列出当前目录下所有文件")
	output.Writeln("  find -help          显示find命令详细帮助")
	output.Writeln("  find -name <filename>    查找指定文件名")
	output.Writeln("  find -filter <pattern>   根据通配符查找文件")
	output.Writeln("  find -match <regex>      根据正则表达式查找文件")
	output.Writeln("    支持通配符: * ? [abc] [a-z] [!abc]")
	output.Writeln("    示例: find -filter /compiled/* 或 find -filter *.jpg")
	output.Writeln("  info                显示PAK文件信息")
	output.Writeln("  exit, quit, q       退出程序")
	output.Writeln("  [command] > file.txt     重定向输出到文件")
}

func ShowFindHelp(output *OutputBuffer) {
	output.Writeln("FIND - 文件查找命令")
	output.Writeln("")
	output.Writeln("用法:")
	output.Writeln("  find [选项]")
	output.Writeln("")
	output.Writeln("选项:")
	output.Writeln("  -help, --help             显示此帮助信息")
	output.Writeln("  -name <文件名>            按确切文件名查找")
	output.Writeln("  -filter <模式>            按通配符模式查找")
	output.Writeln("  -match <正则表达式>       按正则表达式查找")
	output.Writeln("  -format <格式字符串>      自定义输出格式")
	output.Writeln("  -extract <目录>           将筛选的文件解包到指定目录")
	output.Writeln("")
	output.Writeln("通配符:")
	output.Writeln("  *                        匹配任意数量的字符")
	output.Writeln("  ?                        匹配单个字符")
	output.Writeln("  [abc]                    匹配方括号中的任意一个字符")
	output.Writeln("  [a-z]                    匹配指定范围内的字符")
	output.Writeln("  [!abc]                   匹配不在方括号中的字符")
	output.Writeln("")
	output.Writeln("格式变量:")
	output.Writeln("  $path                    文件的完整路径")
	output.Writeln("  $name                    文件名（不含路径）")
	output.Writeln("  $dir                     文件所在目录路径")
	output.Writeln("  $size                    文件大小（压缩后，字节）")
	output.Writeln("  $osize                   原始文件大小（字节）")
}

func ExtractFilteredFiles(fs *PakFileSystem, pakData []byte, extractDir string, searchType string, searchValue string) (int, error) {
	_, headerSize, err := pak.ParsePakInfo(pakData)
	if err != nil {
		return 0, err
	}

	filteredFiles := filterFiles(fs, searchType, searchValue)

	if len(filteredFiles) == 0 {
		return 0, nil
	}

	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return 0, err
	}

	fileOffset := headerSize
	extractedCount := 0

	for i := range fs.Files {
		shouldExtract := false
		for _, ff := range filteredFiles {
			if ff.FileName == fs.Files[i].FileName {
				shouldExtract = true
				break
			}
		}

		if shouldExtract {
			if fileOffset+int(fs.Files[i].ZSize) > len(pakData) {
				return 0, fmt.Errorf("文件 %s 数据超出PAK文件边界", fs.Files[i].FileName)
			}

			fileData := pakData[fileOffset : fileOffset+int(fs.Files[i].ZSize)]
			osPath := strings.ReplaceAll(fs.Files[i].FileName, "\\", string(filepath.Separator))
			outputFilePath := filepath.Join(extractDir, osPath)
			if err := pak.EnsureDirectoryExists(outputFilePath); err != nil {
				return 0, err
			}
			if err := os.WriteFile(outputFilePath, fileData, 0644); err != nil {
				return 0, err
			}
			extractedCount++
		}

		fileOffset += int(fs.Files[i].ZSize)
	}

	return extractedCount, nil
}

func filterFiles(fs *PakFileSystem, searchType, searchValue string) []*pak.FileInfo {
	switch searchType {
	case "name":
		return filterFilesByName(fs, searchValue)
	case "filter":
		return filterFilesByPattern(fs, searchValue)
	case "match":
		return filterFilesByRegex(fs, searchValue)
	default:
		return filterFilesInCurrentPath(fs)
	}
}

func filterFilesByName(fs *PakFileSystem, filename string) []*pak.FileInfo {
	var result []*pak.FileInfo
	for i := range fs.Files {
		unixPath := strings.ReplaceAll(fs.Files[i].FileName, "\\", "/")
		parts := strings.Split(unixPath, "/")
		if parts[len(parts)-1] == filename {
			result = append(result, &fs.Files[i])
		}
	}
	return result
}

func filterFilesByPattern(fs *PakFileSystem, pattern string) []*pak.FileInfo {
	var result []*pak.FileInfo
	for i := range fs.Files {
		unixPath := strings.ReplaceAll(fs.Files[i].FileName, "\\", "/")
		if GlobMatch(unixPath, pattern) {
			result = append(result, &fs.Files[i])
		}
	}
	return result
}

func filterFilesByRegex(fs *PakFileSystem, regexPattern string) []*pak.FileInfo {
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil
	}
	var result []*pak.FileInfo
	for i := range fs.Files {
		unixPath := strings.ReplaceAll(fs.Files[i].FileName, "\\", "/")
		if re.MatchString(unixPath) {
			result = append(result, &fs.Files[i])
		}
	}
	return result
}

func filterFilesInCurrentPath(fs *PakFileSystem) []*pak.FileInfo {
	prefix := ""
	if fs.CurrentPath != "/" {
		prefix = fs.CurrentPath[1:]
	}
	var result []*pak.FileInfo
	for i := range fs.Files {
		filePath := fs.Files[i].FileName
		if prefix == "" {
			result = append(result, &fs.Files[i])
		} else {
			normalizedPrefix := strings.ReplaceAll(prefix, "/", "\\")
			if strings.HasPrefix(filePath, normalizedPrefix) {
				remaining := filePath[len(normalizedPrefix):]
				if strings.HasPrefix(remaining, "\\") || remaining == "" {
					result = append(result, &fs.Files[i])
				}
			}
		}
	}
	return result
}
