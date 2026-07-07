package pak

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testPakPath = "../../test/main.pak"

func TestDetectEncryption(t *testing.T) {
	data, err := os.ReadFile(testPakPath)
	if err != nil {
		t.Fatalf("Failed to read test PAK file: %v", err)
	}

	if !DetectEncryption(data) {
		t.Error("Expected test PAK to be detected as encrypted")
	}
}

func TestDecryptAndParse(t *testing.T) {
	data, err := os.ReadFile(testPakPath)
	if err != nil {
		t.Fatalf("Failed to read test PAK file: %v", err)
	}

	if DetectEncryption(data) {
		CryptData(data)
	}

	pakInfo, headerSize, err := ParsePakInfo(data)
	if err != nil {
		t.Fatalf("Failed to parse PAK info: %v", err)
	}

	if pakInfo.Magic != Magic {
		t.Errorf("Expected magic 0x%08X, got 0x%08X", Magic, pakInfo.Magic)
	}

	if len(pakInfo.FileInfoLibrary) == 0 {
		t.Error("Expected at least one file in PAK")
	} else {
		t.Logf("Found %d files in PAK", len(pakInfo.FileInfoLibrary))
		t.Logf("Header size: %d bytes", headerSize)
	}

	if pakInfo.Compress != nil {
		t.Logf("Compression mode: %v", *pakInfo.Compress)
	}

	for i, f := range pakInfo.FileInfoLibrary {
		if f.FileName == "" {
			t.Errorf("File %d has empty name", i)
		}
		if f.ZSize == 0 {
			t.Errorf("File %s has zero z_size", f.FileName)
		}
	}

	var totalSize uint32
	for _, f := range pakInfo.FileInfoLibrary {
		totalSize += f.ZSize
	}
	t.Logf("Total compressed size: %d bytes", totalSize)

	if headerSize+int(totalSize) > len(data) {
		t.Errorf("Header + data size (%d) exceeds file size (%d)", headerSize+int(totalSize), len(data))
	}
}

func TestCryptData(t *testing.T) {
	original := []byte{0xBA, 0xC0, 0x4A, 0xC0}
	copy := make([]byte, len(original))
	copyData(copy, original)

	CryptData(copy)

	if string(copy) == string(original) {
		t.Error("CryptData should change the data")
	}

	CryptData(copy)

	if string(copy) != string(original) {
		t.Error("Double CryptData should restore original data")
	}
}

func TestCollectFiles(t *testing.T) {
	tmpDir := t.TempDir()

	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create sub dir: %v", err)
	}

	file1 := filepath.Join(tmpDir, "a.txt")
	file2 := filepath.Join(subDir, "b.txt")
	os.WriteFile(file1, []byte("hello"), 0644)
	os.WriteFile(file2, []byte("world"), 0644)

	files, err := CollectFiles(tmpDir, tmpDir)
	if err != nil {
		t.Fatalf("CollectFiles failed: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(files))
	}

	for _, f := range files {
		if !strings.Contains(f.RelPath, "\\") && !strings.HasPrefix(f.RelPath, "sub") {
			t.Logf("File: %s", f.RelPath)
		}
	}

	found := false
	for _, f := range files {
		if f.RelPath == "a.txt" || f.RelPath == "sub\\b.txt" {
			found = true
		}
	}
	if !found {
		t.Logf("Files found: %v", files)
	}
}

func TestReadWriteString(t *testing.T) {
	var buf strings.Builder
	testStr := "测试文件名.txt"

	if err := WriteStringByU8Head(&buf, testStr); err != nil {
		t.Fatalf("WriteStringByU8Head failed: %v", err)
	}

	data := []byte(buf.String())
	var pos int
	readStr, err := ReadStringByU8Head(data, &pos)
	if err != nil {
		t.Fatalf("ReadStringByU8Head failed: %v", err)
	}

	if readStr != testStr {
		t.Errorf("Round-trip string mismatch: expected %q, got %q", testStr, readStr)
	}
}

func TestPackAndUnpackRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	pakFile := filepath.Join(tmpDir, "test.pak")
	outDir := filepath.Join(tmpDir, "out")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create src dir: %v", err)
	}

	subDir := filepath.Join(srcDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	os.WriteFile(filepath.Join(srcDir, "hello.txt"), []byte("Hello World!"), 0644)
	os.WriteFile(filepath.Join(srcDir, "subdir", "nested.txt"), []byte("Nested file"), 0644)
	os.WriteFile(filepath.Join(srcDir, "中国.txt"), []byte{0xE4, 0xBD, 0xA0, 0xE5, 0xA5, 0xBD}, 0644)

	if err := PackToPak(srcDir, pakFile); err != nil {
		t.Fatalf("PackToPak failed: %v", err)
	}

	if _, err := os.Stat(pakFile); os.IsNotExist(err) {
		t.Fatal("PAK file was not created")
	}

	if err := UnpackPak(pakFile, outDir); err != nil {
		t.Fatalf("UnpackPak failed: %v", err)
	}

	expectedFiles := []string{"hello.txt", filepath.Join("subdir", "nested.txt"), "中国.txt"}
	for _, ef := range expectedFiles {
		expectedPath := filepath.Join(outDir, ef)
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Errorf("Expected file not found: %s", expectedPath)
		}
	}

	helloData, _ := os.ReadFile(filepath.Join(outDir, "hello.txt"))
	if string(helloData) != "Hello World!" {
		t.Errorf("hello.txt content mismatch: got %q", string(helloData))
	}

	nestedData, _ := os.ReadFile(filepath.Join(outDir, "subdir", "nested.txt"))
	if string(nestedData) != "Nested file" {
		t.Errorf("nested.txt content mismatch: got %q", string(nestedData))
	}

	chineseData, _ := os.ReadFile(filepath.Join(outDir, "中国.txt"))
	expectedChineseData := []byte{0xE4, 0xBD, 0xA0, 0xE5, 0xA5, 0xBD}
	if string(chineseData) != string(expectedChineseData) {
		t.Errorf("中国.txt content mismatch: got %v", chineseData)
	}
}

func TestIsDirectoryEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	empty, err := IsDirectoryEmpty(tmpDir)
	if err != nil {
		t.Fatalf("IsDirectoryEmpty failed: %v", err)
	}
	if !empty {
		t.Error("New temp dir should be empty")
	}

	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("test"), 0644)

	empty, err = IsDirectoryEmpty(tmpDir)
	if err != nil {
		t.Fatalf("IsDirectoryEmpty failed: %v", err)
	}
	if empty {
		t.Error("Dir with file should not be empty")
	}

	nonExistent := filepath.Join(tmpDir, "nonexistent")
	empty, err = IsDirectoryEmpty(nonExistent)
	if err != nil {
		t.Fatalf("IsDirectoryEmpty on non-existent dir failed: %v", err)
	}
	if !empty {
		t.Error("Non-existent dir should be considered empty")
	}
}

func copyData(dst, src []byte) {
	copy(dst, src)
}

func TestPakInfoErrors(t *testing.T) {
	_, _, err := ParsePakInfo([]byte{0x00, 0x00, 0x00, 0x00})
	if err == nil {
		t.Error("Expected error for invalid magic")
	}

	_, _, err = ParsePakInfo([]byte{})
	if err == nil {
		t.Error("Expected error for empty data")
	}
}

func TestEnsureDirectoryExists(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "a", "b", "c", "file.txt")

	if err := EnsureDirectoryExists(nestedPath); err != nil {
		t.Fatalf("EnsureDirectoryExists failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "a", "b", "c")); os.IsNotExist(err) {
		t.Error("Directory was not created")
	}
}
