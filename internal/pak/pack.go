package pak

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileEntry struct {
	RelPath string
	AbsPath string
}

func CollectFiles(dir string, baseDir string) ([]FileEntry, error) {
	var files []FileEntry
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		absPath := filepath.Join(dir, entry.Name())
		relPath := strings.ReplaceAll(absPath[len(baseDir):], "/", "\\")
		if relPath[0] == '\\' {
			relPath = relPath[1:]
		}
		if entry.IsDir() {
			subFiles, err := CollectFiles(absPath, baseDir)
			if err != nil {
				return nil, err
			}
			files = append(files, subFiles...)
		} else {
			files = append(files, FileEntry{RelPath: relPath, AbsPath: absPath})
		}
	}
	return files, nil
}

func PackToPak(inputDir, outputPath string) error {
	info, err := os.Stat(inputDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("输入目录不存在: %s", inputDir)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("输入路径必须是目录")
	}

	if filepath.Ext(outputPath) != ".pak" {
		return fmt.Errorf("输出文件必须是 .pak 文件")
	}

	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("输出文件已存在: %s", outputPath)
	}

	fmt.Printf("正在打包: %s\n", inputDir)
	fmt.Printf("输出文件: %s\n", outputPath)

	files, err := CollectFiles(inputDir, inputDir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("输入目录为空")
	}
	fmt.Printf("找到 %d 个文件\n", len(files))

	fileNames := make(map[string]bool)
	for _, f := range files {
		if fileNames[f.RelPath] {
			return fmt.Errorf("发现重复的文件名: %s", f.RelPath)
		}
		fileNames[f.RelPath] = true
	}

	var fileInfos []FileInfo
	for _, f := range files {
		fi, err := os.Stat(f.AbsPath)
		if err != nil {
			return err
		}
		fileSize := fi.Size()
		if fileSize > int64(^uint32(0)) {
			return fmt.Errorf("文件过大 (>4GB): %s", f.RelPath)
		}
		fileInfos = append(fileInfos, FileInfo{
			FileName: f.RelPath,
			ZSize:    uint32(fileSize),
			Size:     uint32(fileSize),
			FileTime: DefaultFileTime,
		})
	}

	pakInfo := NewPakInfo()
	pakInfo.FileInfoLibrary = fileInfos
	pakInfo.Compress = boolPtr(false)

	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	var magicBytes [4]byte
	binary.LittleEndian.PutUint32(magicBytes[:], Magic)
	if _, err := outFile.Write(magicBytes[:]); err != nil {
		return err
	}

	var verBytes [4]byte
	binary.LittleEndian.PutUint32(verBytes[:], Version)
	if _, err := outFile.Write(verBytes[:]); err != nil {
		return err
	}

	for _, fi := range pakInfo.FileInfoLibrary {
		if _, err := outFile.Write([]byte{0}); err != nil {
			return err
		}
		if err := WriteStringByU8Head(outFile, fi.FileName); err != nil {
			return err
		}
		var zsb [4]byte
		binary.LittleEndian.PutUint32(zsb[:], fi.ZSize)
		if _, err := outFile.Write(zsb[:]); err != nil {
			return err
		}
		if pakInfo.Compress != nil && *pakInfo.Compress {
			var sb [4]byte
			binary.LittleEndian.PutUint32(sb[:], fi.Size)
			if _, err := outFile.Write(sb[:]); err != nil {
				return err
			}
		}
		var ftb [8]byte
		binary.LittleEndian.PutUint64(ftb[:], fi.FileTime)
		if _, err := outFile.Write(ftb[:]); err != nil {
			return err
		}
	}

	if _, err := outFile.Write([]byte{InfoEnd}); err != nil {
		return err
	}

	for idx, f := range files {
		if idx%100 == 0 {
			fmt.Printf("正在打包: %d/%d\n", idx+1, len(files))
		}
		fileData, err := os.ReadFile(f.AbsPath)
		if err != nil {
			return err
		}
		if _, err := outFile.Write(fileData); err != nil {
			return err
		}
	}

	outFile.Close()

	pakData, err := os.ReadFile(outputPath)
	if err != nil {
		return err
	}
	CryptData(pakData)
	if err := os.WriteFile(outputPath, pakData, 0644); err != nil {
		return err
	}

	fmt.Printf("打包完成！生成了包含 %d 个文件的PAK\n", len(pakInfo.FileInfoLibrary))

	outFi, _ := os.Stat(outputPath)
	fmt.Printf("输出文件大小: %.2f MB\n", float64(outFi.Size())/1024.0/1024.0)

	return nil
}
