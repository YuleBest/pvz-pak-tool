package repl

import (
	"os"
	"testing"

	"github.com/YuleBest/pvzpak/internal/pak"
)

const testPakPath = "../../test/main.pak"

func TestPakFileSystem(t *testing.T) {
	data, err := os.ReadFile(testPakPath)
	if err != nil {
		t.Fatalf("Failed to read test PAK file: %v", err)
	}

	if pak.DetectEncryption(data) {
		pak.CryptData(data)
	}

	pakInfo, _, err := pak.ParsePakInfo(data)
	if err != nil {
		t.Fatalf("Failed to parse PAK info: %v", err)
	}

	fs := NewPakFileSystem(pakInfo.FileInfoLibrary)

	dirs, files := fs.GetEntriesAtPath("")
	if len(dirs) == 0 && len(files) == 0 {
		t.Error("Root directory should not be empty")
	}
	t.Logf("Root directory: %d dirs, %d files", len(dirs), len(files))

	if len(dirs) > 0 {
		dirName := dirs[0]
		t.Logf("First directory: %s", dirName)

		err := fs.ChangeDirectory(dirName)
		if err != nil {
			t.Fatalf("Failed to cd to %s: %v", dirName, err)
		}

		if fs.CurrentPath != "/"+dirName {
			t.Errorf("Expected current path /%s, got %s", dirName, fs.CurrentPath)
		}
	}

	backPath := fs.ResolvePath("..")
	if backPath != "/" {
		t.Errorf("Expected '..' to resolve to '/', got '%s'", backPath)
	}

	rootPath := fs.ResolvePath("/")
	if rootPath != "/" {
		t.Errorf("Expected '/' to resolve to '/', got '%s'", rootPath)
	}
}

func TestResolvePath(t *testing.T) {
	pakInfo := pak.NewPakInfo()
	fs := NewPakFileSystem(pakInfo.FileInfoLibrary)

	tests := []struct {
		current string
		input   string
		expect  string
	}{
		{"/", "", "/"},
		{"/", "/", "/"},
		{"/", "foo", "/foo"},
		{"/foo", "bar", "/foo/bar"},
		{"/foo", "..", "/"},
		{"/foo/bar", "..", "/foo"},
		{"/foo", "../bar", "/bar"},
		{"/foo/bar", "../baz", "/foo/baz"},
	}

	for _, tt := range tests {
		fs.CurrentPath = tt.current
		got := fs.ResolvePath(tt.input)
		if got != tt.expect {
			t.Errorf("ResolvePath(%q) with current=%q: expected %q, got %q", tt.input, tt.current, tt.expect, got)
		}
	}
}

func TestGlobMatch(t *testing.T) {
	tests := []struct {
		text    string
		pattern string
		expect  bool
	}{
		{"abc", "abc", true},
		{"abc", "*", true},
		{"abc", "a*", true},
		{"abc", "*c", true},
		{"abc", "a?c", true},
		{"abc", "a??", true},
		{"abc", "??c", true},
		{"abc", "ab?", true},
		{"abc", "a", false},
		{"abc", "ab", false},
		{"abc", "b*", false},
		{"abc", "*d", false},
		{"hello.txt", "*.txt", true},
		{"hello.txt", "*.md", false},
		{"dir/file.txt", "dir/*", true},
		{"dir/file.txt", "*/file.txt", true},
		{"a", "[abc]", true},
		{"d", "[abc]", false},
		{"a", "[a-z]", true},
		{"9", "[a-z]", false},
		{"b", "[!abc]", false},
		{"d", "[!abc]", true},
	}

	for _, tt := range tests {
		got := GlobMatch(tt.text, tt.pattern)
		if got != tt.expect {
			t.Errorf("GlobMatch(%q, %q): expected %v, got %v", tt.text, tt.pattern, tt.expect, got)
		}
	}
}

func TestFormatFileInfo(t *testing.T) {
	file := &pak.FileInfo{
		FileName: "dir\\subdir\\test.txt",
		ZSize:    1024,
		Size:     2048,
		FileTime: pak.DefaultFileTime,
	}

	result := FormatFileInfo(file, "$path")
	expected := "dir/subdir/test.txt"
	if result != expected {
		t.Errorf("FormatFileInfo $path: expected %q, got %q", expected, result)
	}

	result = FormatFileInfo(file, "$name")
	if result != "test.txt" {
		t.Errorf("FormatFileInfo $name: expected 'test.txt', got %q", result)
	}

	result = FormatFileInfo(file, "$size")
	if result != "1024" {
		t.Errorf("FormatFileInfo $size: expected '1024', got %q", result)
	}

	result = FormatFileInfo(file, "$dir")
	if result != "dir/subdir" {
		t.Errorf("FormatFileInfo $dir: expected 'dir/subdir', got %q", result)
	}

	result = FormatFileInfo(file, "$osize")
	if result != "2048" {
		t.Errorf("FormatFileInfo $osize: expected '2048', got %q", result)
	}
}

func TestParseCommandArgs(t *testing.T) {
	tests := []struct {
		input  string
		expect []string
	}{
		{"ls", []string{"ls"}},
		{"ls /path", []string{"ls", "/path"}},
		{"find -name \"test file.txt\"", []string{"find", "-name", "test file.txt"}},
		{"find -format \"$path -- $size\"", []string{"find", "-format", "$path -- $size"}},
	}

	for _, tt := range tests {
		got := ParseCommandArgs(tt.input)
		if len(got) != len(tt.expect) {
			t.Errorf("ParseCommandArgs(%q): expected %d args, got %d: %v", tt.input, len(tt.expect), len(got), got)
			continue
		}
		for i := range got {
			if got[i] != tt.expect[i] {
				t.Errorf("ParseCommandArgs(%q)[%d]: expected %q, got %q", tt.input, i, tt.expect[i], got[i])
			}
		}
	}
}

func TestParseCommandLine(t *testing.T) {
	cmd, file, ok := ParseCommandLine("find > output.txt")
	if cmd != "find" || file != "output.txt" || !ok {
		t.Errorf("ParseCommandLine('find > output.txt'): got (%q, %q, %v)", cmd, file, ok)
	}

	cmd, file, ok = ParseCommandLine("ls /path")
	if cmd != "ls /path" || file != "" || ok {
		t.Errorf("ParseCommandLine('ls /path'): got (%q, %q, %v)", cmd, file, ok)
	}
}
