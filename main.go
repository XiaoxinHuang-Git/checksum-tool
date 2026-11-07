// checksum-tool 用于生成与校验交付物的 SHA256 清单。
// 典型场景：交付包发出前生成清单，对方收到后复核文件是否完整、是否被改动。
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const defaultManifest = "SHA256_clearlist.txt"

type entry struct {
	hash string
	path string
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "gen":
		err = runGen(os.Args[2:])
	case "verify":
		err = runVerify(os.Args[2:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `checksum-tool - 交付物 SHA256 清单生成与校验

用法:
  checksum-tool gen    [-dir DIR] [-o FILE]             生成清单
  checksum-tool verify [-dir DIR] [-v] [-strict] FILE   校验清单

参数:
  gen
    -dir DIR    待生成清单的目录，默认当前目录
    -o FILE     输出文件，默认 DIR/`+defaultManifest+`

  verify
    -dir DIR    校验根目录，默认清单文件所在目录
    -v          打印每个文件的校验结果（默认只打印异常项）
    -strict     目录中存在清单之外的文件时判定为失败

说明:
  生成时跳过 .git 目录与清单文件自身，按相对路径排序输出。
  清单格式与 sha256sum 兼容，可直接用 sha256sum -c 复核。

退出码:
  0  全部一致
  1  存在差异或缺失
  2  参数错误
`)
}

func runGen(args []string) error {
	fset := flag.NewFlagSet("gen", flag.ContinueOnError)
	dir := fset.String("dir", ".", "待生成清单的目录")
	out := fset.String("o", "", "输出文件")
	if err := fset.Parse(args); err != nil {
		return err
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		return err
	}
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s 不是目录", root)
	}

	output := *out
	if output == "" {
		output = filepath.Join(root, defaultManifest)
	} else if !filepath.IsAbs(output) {
		output = filepath.Join(root, output)
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return err
	}

	entries, err := collect(root, output)
	if err != nil {
		return err
	}
	if err := writeManifest(output, entries); err != nil {
		return err
	}

	fmt.Printf("清单文件: %s\n文件数:   %d\n", output, len(entries))
	return nil
}

func runVerify(args []string) error {
	fset := flag.NewFlagSet("verify", flag.ContinueOnError)
	dir := fset.String("dir", "", "校验根目录")
	verbose := fset.Bool("v", false, "打印每个文件的结果")
	strict := fset.Bool("strict", false, "存在清单之外的文件时判定失败")
	if err := fset.Parse(args); err != nil {
		return err
	}
	if fset.NArg() != 1 {
		return errors.New("verify 需要指定清单文件")
	}

	manifest, err := filepath.Abs(fset.Arg(0))
	if err != nil {
		return err
	}
	entries, err := readManifest(manifest)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("%s 中没有有效记录", manifest)
	}

	root := *dir
	if root == "" {
		root = filepath.Dir(manifest)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return err
	}

	known := make(map[string]bool, len(entries))
	var changed, missing int

	for _, e := range entries {
		target := filepath.Join(root, filepath.FromSlash(e.path))
		known[filepath.Clean(target)] = true

		sum, err := hashFile(target)
		switch {
		case err != nil:
			missing++
			fmt.Printf("MISSING  %s\n", e.path)
		case !strings.EqualFold(sum, e.hash):
			changed++
			fmt.Printf("FAILED   %s\n", e.path)
		default:
			if *verbose {
				fmt.Printf("OK       %s\n", e.path)
			}
		}
	}

	extra, err := findExtra(root, known, manifest)
	if err != nil {
		return err
	}
	for _, p := range extra {
		fmt.Printf("EXTRA    %s\n", p)
	}

	fmt.Printf("\n清单文件: %s\n校验目录: %s\n", manifest, root)
	fmt.Printf("合计 %d　一致 %d　变更 %d　缺失 %d　多余 %d\n",
		len(entries), len(entries)-changed-missing, changed, missing, len(extra))

	if changed > 0 || missing > 0 || (*strict && len(extra) > 0) {
		return errors.New("校验未通过")
	}
	fmt.Println("校验通过")
	return nil
}

// collect 遍历 root 下所有普通文件并计算 SHA256，跳过 skip 与 .git 目录。
func collect(root, skip string) ([]entry, error) {
	var entries []entry

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}

		abs, err := filepath.Abs(p)
		if err != nil {
			return err
		}
		if abs == skip {
			return nil
		}

		sum, err := hashFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		entries = append(entries, entry{hash: sum, path: filepath.ToSlash(rel)})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })
	return entries, nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeManifest(path string, entries []entry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, e := range entries {
		if _, err := fmt.Fprintf(w, "%s  %s\n", e.hash, e.path); err != nil {
			return err
		}
	}
	return w.Flush()
}

func readManifest(path string) ([]entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []entry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		if e, ok := parseLine(sc.Text()); ok {
			entries = append(entries, e)
		}
	}
	return entries, sc.Err()
}

// parseLine 解析一行清单，兼容 sha256sum 的两种格式：
// "<hash>  <path>" 与 "<hash> *<path>"。
func parseLine(line string) (entry, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") || len(line) < 67 {
		return entry{}, false
	}

	sum := line[:64]
	if _, err := hex.DecodeString(sum); err != nil {
		return entry{}, false
	}

	rest := strings.TrimLeft(line[64:], " ")
	rest = strings.TrimPrefix(rest, "*")
	if rest == "" {
		return entry{}, false
	}
	return entry{hash: strings.ToLower(sum), path: rest}, true
}

func findExtra(root string, known map[string]bool, manifest string) ([]string, error) {
	manifestAbs, err := filepath.Abs(manifest)
	if err != nil {
		return nil, err
	}

	var extra []string
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}

		abs, err := filepath.Abs(p)
		if err != nil {
			return err
		}
		if abs == manifestAbs || known[filepath.Clean(abs)] {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		extra = append(extra, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(extra)
	return extra, nil
}
