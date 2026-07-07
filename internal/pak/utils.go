package pak

import (
	"encoding/binary"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func ReadU32LE(data []byte, pos *int) (uint32, error) {
	if *pos+4 > len(data) {
		return 0, io.ErrUnexpectedEOF
	}
	v := binary.LittleEndian.Uint32(data[*pos : *pos+4])
	*pos += 4
	return v, nil
}

func ReadU64LE(data []byte, pos *int) (uint64, error) {
	if *pos+8 > len(data) {
		return 0, io.ErrUnexpectedEOF
	}
	v := binary.LittleEndian.Uint64(data[*pos : *pos+8])
	*pos += 8
	return v, nil
}

func ReadStringByU8Head(data []byte, pos *int) (string, error) {
	if *pos >= len(data) {
		return "", io.ErrUnexpectedEOF
	}
	length := int(data[*pos])
	*pos++
	if *pos+length > len(data) {
		return "", io.ErrUnexpectedEOF
	}
	decoded, _ := simplifiedchinese.GBK.NewDecoder().Bytes(data[*pos : *pos+length])
	*pos += length
	return string(decoded), nil
}

func WriteStringByU8Head(w io.Writer, s string) error {
	encoded, _ := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(s))
	if len(encoded) > 255 {
		return io.ErrShortWrite
	}
	if _, err := w.Write([]byte{byte(len(encoded))}); err != nil {
		return err
	}
	_, err := w.Write(encoded)
	return err
}

func CryptData(data []byte) {
	const key byte = 0xF7
	for i := range data {
		data[i] ^= key
	}
}

func EnsureDirectoryExists(filePath string) error {
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

func IsDirectoryEmpty(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
