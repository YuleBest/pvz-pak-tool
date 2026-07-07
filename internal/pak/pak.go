package pak

import (
	"fmt"
	"io"
)

type FileInfo struct {
	FileName string
	ZSize    uint32
	Size     uint32
	FileTime uint64
}

type PakInfo struct {
	Magic           uint32
	Version         uint32
	FileInfoLibrary []FileInfo
	Compress        *bool
	Pc              bool
	Win             bool
}

const (
	Magic           uint32 = 0xBAC04AC0
	Version         uint32 = 0x0
	InfoEnd         byte   = 0x80
	DefaultFileTime uint64 = 129146222018596744
)

func NewPakInfo() PakInfo {
	return PakInfo{
		Magic:           Magic,
		Version:         Version,
		FileInfoLibrary: []FileInfo{},
		Compress:        nil,
		Pc:              true,
		Win:             true,
	}
}

func ParsePakInfo(data []byte) (PakInfo, int, error) {
	var pos int
	pakInfo := NewPakInfo()

	magic, err := ReadU32LE(data, &pos)
	if err != nil {
		return pakInfo, 0, err
	}
	if magic != Magic {
		return pakInfo, 0, fmt.Errorf("invalid PAK magic: expected 0x%08X, got 0x%08X", Magic, magic)
	}

	version, err := ReadU32LE(data, &pos)
	if err != nil {
		return pakInfo, 0, err
	}
	pakInfo.Version = version

	for {
		if pos >= len(data) {
			return pakInfo, 0, io.ErrUnexpectedEOF
		}
		flag := data[pos]
		pos++

		if flag == InfoEnd {
			break
		} else if flag != 0 {
			return pakInfo, 0, fmt.Errorf("invalid file flag: 0x%02X at position %d (expected 0x00 or 0x80)", flag, pos-1)
		}

		if pakInfo.Compress == nil {
			savedPos := pos

			if pos >= len(data) {
				return pakInfo, 0, io.ErrUnexpectedEOF
			}
			nameLen := int(data[pos])
			pos += 1 + nameLen

			pos += 4

			if pos+12 < len(data) {
				pos += 4
				pos += 8

				if pos < len(data) && (data[pos] == 0 || data[pos] == InfoEnd) {
					pakInfo.Compress = boolPtr(true)
				} else {
					pakInfo.Compress = boolPtr(false)
				}
			} else {
				pakInfo.Compress = boolPtr(false)
			}

			pos = savedPos
		}

		fileName, err := ReadStringByU8Head(data, &pos)
		if err != nil {
			return pakInfo, 0, err
		}

		zSize, err := ReadU32LE(data, &pos)
		if err != nil {
			return pakInfo, 0, err
		}

		var size uint32
		if pakInfo.Compress != nil && *pakInfo.Compress {
			size, err = ReadU32LE(data, &pos)
			if err != nil {
				return pakInfo, 0, err
			}
		}

		fileTime, err := ReadU64LE(data, &pos)
		if err != nil {
			return pakInfo, 0, err
		}

		pakInfo.FileInfoLibrary = append(pakInfo.FileInfoLibrary, FileInfo{
			FileName: fileName,
			ZSize:    zSize,
			Size:     size,
			FileTime: fileTime,
		})
	}

	return pakInfo, pos, nil
}

func ShowPakInfoSimple(dataLen int, files []FileInfo) {
	fmt.Printf("  PAK 文件大小: %.2f MB\n", float64(dataLen)/1024.0/1024.0)
	fmt.Printf("  文件数量: %d\n", len(files))
}

func boolPtr(b bool) *bool {
	return &b
}
