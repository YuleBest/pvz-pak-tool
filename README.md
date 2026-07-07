# PVZ PAK Tool

一个植物大战僵尸 PAK 文件的解包和打包工具（Go 语言移植版）

## 功能特性

- **解包 PAK 文件** - 将 .pak 文件解包到指定目录
- **打包目录** - 将目录打包为 .pak 文件
- **交互式浏览器** - REPL 模式下浏览 PAK 文件内容
- **批处理模式** - 支持命令行批量操作
- **文件搜索** - 支持通配符和正则表达式搜索文件

## 安装

确保已安装 Go 1.24+ 环境：

```bash
git clone https://github.com/YuleBest/pvz-pak-tool.git
cd pvz-pak-tool
go build -o pkt ./cmd/pkt
```

编译后的可执行文件为 `pkt`。

## 使用方法

### 基本用法

```bash
# 解包 PAK 文件到目录
./pkt game.pak -o extracted_files/

# 将目录打包为 PAK 文件
./pkt game_files/ -o game.pak
```

### 交互式模式命令

```bash
# 进入交互式浏览模式
./pkt game.pak
```

在 REPL 模式下，支持以下命令：

- `ls [path]` - 列出当前目录或指定路径的文件
- `cd <path>` - 切换到指定目录
- `find` - 列出当前目录下所有文件
- `find -name <filename>` - 按文件名精确查找
- `find -filter <pattern>` - 通配符搜索（支持 \* ? [abc] [a-z] [!abc]）
- `find -match <regex>` - 正则表达式搜索
- `find -extract <dir>` - 搜索并提取文件到指定目录
- `info` - 显示 PAK 文件信息
- `help` - 显示帮助信息
- `exit` - 退出程序

### 批处理模式

```bash
./pkt game.pak -c "ls" -c "find -filter *.xml"
```

### 高级功能

- 支持输出重定向：`ls > filelist.txt`
- 支持自定义格式化输出：`find -format "$path -- $size bytes"`
- 自动检测 PAK 文件加密并解密

## 格式说明

PAK 文件格式采用小端序：

| 偏移 | 大小 | 说明 |
|------|------|------|
| 0 | 4 | 魔数 0xBAC04AC0（加密后为 0x4D37BD37） |
| 4 | 4 | 版本号 |
| 8+ | 变长 | 文件条目列表，每条以 0x00 开头 |
| 结束 | 1 | 结束标志 0x80 |
| 之后 | 变长 | 文件数据 |

文件名采用 GBK 编码，路径使用 Windows 反斜杠风格。PAK 文件整体可通过 XOR 0xF7 加解密。

## 项目结构

```
.
├── cmd
│   └── pkt
│       └── main.go       # CLI 命令行入口
├── internal
│   ├── pak
│   │   ├── pack.go       # 目录打包逻辑 (package pak)
│   │   ├── pak.go        # PAK 格式解析和结构体定义 (package pak)
│   │   ├── pak_test.go   # PAK 解析与打包单元测试
│   │   ├── unpack.go     # PAK 解包逻辑 (package pak)
│   │   └── utils.go      # 工具函数（GBK 编解码、加解密等）(package pak)
│   └── repl
│       ├── repl.go       # REPL 交互式模式和模拟文件系统 (package repl)
│       └── repl_test.go  # 交互式与模拟文件系统测试
├── test
│   └── main.pak          # 测试数据
├── go.mod
├── go.sum
└── README.md
```

## 测试

```bash
go test ./...
```

## 参考

本项目是 [axh-xecoy/pvz-pak-tool](https://github.com/axh-xecoy/pvz-pak-tool) Rust 版本的 Go 语言复刻。

## 许可证

MIT
