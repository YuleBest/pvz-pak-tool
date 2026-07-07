package pak

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func UnpackPak(inputPath, outputDir string) error {
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("输入文件不存在: %s", inputPath)
	}

	if filepath.Ext(inputPath) != ".pak" {
		return fmt.Errorf("输入文件必须是 .pak 文件")
	}

	if _, err := os.Stat(outputDir); err == nil {
		empty, err := IsDirectoryEmpty(outputDir)
		if err != nil {
			return err
		}
		if !empty {
			return fmt.Errorf("输出目录不为空: %s", outputDir)
		}
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	fmt.Printf("正在解包: %s\n", inputPath)
	fmt.Printf("输出目录: %s\n", outputDir)

	data, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	encrypted := DetectEncryption(data)
	if encrypted {
		CryptData(data)
	}

	pakInfo, headerSize, err := ParsePakInfo(data)
	if err != nil {
		return err
	}

	fmt.Println("PAK 文件信息:")
	ShowPakInfoSimple(len(data), pakInfo.FileInfoLibrary)
	fmt.Println()

	fileOffset := headerSize
	for index, fileInfo := range pakInfo.FileInfoLibrary {
		if index%100 == 0 {
			fmt.Printf("正在解包: %d/%d\n", index+1, len(pakInfo.FileInfoLibrary))
		}

		if fileOffset+int(fileInfo.ZSize) > len(data) {
			return fmt.Errorf("文件 %s 数据超出PAK文件边界", fileInfo.FileName)
		}

		fileData := data[fileOffset : fileOffset+int(fileInfo.ZSize)]

		osPath := strings.ReplaceAll(fileInfo.FileName, "\\", string(filepath.Separator))
		outputFilePath := filepath.Join(outputDir, osPath)
		if err := EnsureDirectoryExists(outputFilePath); err != nil {
			return err
		}

		if err := os.WriteFile(outputFilePath, fileData, 0644); err != nil {
			return err
		}

		fileOffset += int(fileInfo.ZSize)
	}

	fmt.Printf("解包完成！提取了 %d 个文件\n", len(pakInfo.FileInfoLibrary))
	return nil
}

func DetectEncryption(data []byte) bool {
	if len(data) < 8 {
		return false
	}
	magic := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
	return magic != Magic
}
