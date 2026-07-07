package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/YuleBest/pvzpak/internal/pak"
	"github.com/YuleBest/pvzpak/internal/repl"
)

type Cli struct {
	Input    string
	Output   string
	Commands []string
}

func main() {
	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}

	var input string
	var output string
	var commands []string
	args := os.Args[1:]

	i := 0
	for i < len(args) {
		arg := args[i]
		if arg == "-o" || arg == "--output" {
			if i+1 < len(args) {
				output = args[i+1]
				i += 2
			} else {
				fmt.Fprintf(os.Stderr, "错误: -o/--output 需要指定值\n")
				os.Exit(1)
			}
		} else if arg == "-c" || arg == "--command" {
			if i+1 < len(args) {
				commands = append(commands, args[i+1])
				i += 2
			} else {
				fmt.Fprintf(os.Stderr, "错误: -c/--command 需要指定值\n")
				os.Exit(1)
			}
		} else if arg == "-h" || arg == "--help" {
			showUsage()
			return
		} else if arg == "-v" || arg == "--version" {
			fmt.Printf("PVZ PAK Tool %s\n", Version)
			return
		} else if arg[0] == '-' {
			fmt.Fprintf(os.Stderr, "错误: 未知参数: %s\n", arg)
			os.Exit(1)
		} else if input == "" {
			input = arg
			i++
		} else {
			fmt.Fprintf(os.Stderr, "错误: 多余的参数: %s\n", arg)
			os.Exit(1)
		}
	}

	if input == "" {
		showUsage()
		os.Exit(1)
	}

	cli := Cli{
		Input:    input,
		Output:   output,
		Commands: commands,
	}

	inputInfo, err := os.Stat(cli.Input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 输入路径不存在: %s\n", cli.Input)
		os.Exit(1)
	}

	if cli.Output != "" {
		if inputInfo.IsDir() {
			if err := pak.PackToPak(cli.Input, cli.Output); err != nil {
				fmt.Fprintf(os.Stderr, "错误: %s\n", err)
				os.Exit(1)
			}
		} else if filepath.Ext(cli.Input) == ".pak" {
			if err := pak.UnpackPak(cli.Input, cli.Output); err != nil {
				fmt.Fprintf(os.Stderr, "错误: %s\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "错误: 无法识别的输入类型\n")
			fmt.Fprintf(os.Stderr, "  - 打包: 输入应为目录\n")
			fmt.Fprintf(os.Stderr, "  - 解包: 输入应为 .pak 文件\n")
			os.Exit(1)
		}
	} else if len(cli.Commands) > 0 {
		if filepath.Ext(cli.Input) == ".pak" {
			if err := repl.RunBatchCommands(cli.Input, cli.Commands); err != nil {
				fmt.Fprintf(os.Stderr, "错误: %s\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "错误: 批处理模式需要 .pak 文件作为输入\n")
			os.Exit(1)
		}
	} else if inputInfo.IsDir() {
		fmt.Fprintf(os.Stderr, "错误: 打包目录需要指定输出PAK文件\n")
		fmt.Fprintf(os.Stderr, "用法: %s <目录> -o <输出.pak文件>\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	} else if filepath.Ext(cli.Input) == ".pak" {
		fmt.Fprintf(os.Stderr, "错误: 解包需要指定输出目录\n")
		fmt.Fprintf(os.Stderr, "用法: %s <输入.pak文件> -o <输出目录>\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	} else {
		fmt.Fprintf(os.Stderr, "错误: 无法识别的输入类型\n")
		os.Exit(1)
	}
}

func showUsage() {
	fmt.Fprintf(os.Stderr, "PVZ PAK文件操作工具 - 植物大战僵尸资源包管理器\n\n")
	fmt.Fprintf(os.Stderr, "用法:\n")
	fmt.Fprintf(os.Stderr, "  %s <INPUT> [-o|--output <OUTPUT>] [-c|--command <COMMAND>]...\n", filepath.Base(os.Args[0]))
	fmt.Fprintf(os.Stderr, "  %s -v|--version\n", filepath.Base(os.Args[0]))
	fmt.Fprintf(os.Stderr, "  %s -h|--help\n\n", filepath.Base(os.Args[0]))
	fmt.Fprintf(os.Stderr, "参数:\n")
	fmt.Fprintf(os.Stderr, "  INPUT             输入文件或目录 (.pak文件将被解包，目录将被打包)\n")
	fmt.Fprintf(os.Stderr, "  -o, --output      输出路径（目录或.pak文件）\n")
	fmt.Fprintf(os.Stderr, "  -c, --command     要执行的命令（可多次使用）\n")
	fmt.Fprintf(os.Stderr, "  -v, --version     显示版本信息\n")
	fmt.Fprintf(os.Stderr, "  -h, --help        显示帮助信息\n")
}
