# checksum-tool

![lang](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)
![license](https://img.shields.io/badge/License-MIT-yellow.svg)
![platforms](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey.svg)
![CI](https://github.com/XiaoxinHuang-Git/checksum-tool/actions/workflows/build.yml/badge.svg)

交付物完整性校验工具：生成和校验 SHA256 清单，用于确认交付包在传输、拷贝、解压之后没有被漏发、错发或改动。

- 单个 Go 二进制，无外部依赖，Windows / Linux / macOS 直接运行
- 清单格式与 `sha256sum` 兼容，对方没有本工具时也能用系统自带命令复核
- 支持检测变更、缺失，以及清单之外多出来的文件

## 使用场景

| 场景 | 做法 |
|---|---|
| 交付前 | 在交付目录执行 `gen`，把清单一起打包给对方 |
| 对方复核 | 解压后执行 `verify`，确认与清单一致 |
| 增量发布 | 只对新包执行 `gen`，并与上一次清单比对差异 |
| 部署位核对 | 在目标服务器上执行 `verify`，确认落地文件与交付包一致 |

## 安装

方式一：从 [Releases](https://github.com/XiaoxinHuang-Git/checksum-tool/releases) 下载（推荐）

在 Release 页面下载对应平台的二进制文件，放到 PATH 目录下即可。支持：
- Linux amd64 / arm64
- Windows amd64 (.exe)
- macOS amd64 / arm64

方式二：自行编译

```bash
go build -o checksum-tool .
```

方式三：交叉编译 Linux 版本

```bash
GOOS=linux GOARCH=amd64 go build -o checksum-tool-linux .
```

Windows PowerShell 下：

```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o checksum-tool-linux .
```

## 用法

### 生成清单

```bash
checksum-tool gen -dir ./package
```

输出：

```
清单文件: /data/package/SHA256_clearlist.txt
文件数:   128
```

自定义输出文件名：

```bash
checksum-tool gen -dir ./package -o SHA256_20261107.txt
```

### 校验清单

```bash
checksum-tool verify ./package/SHA256_clearlist.txt
```

通过时：

```
✅ 清单一致，128 个文件全部通过校验
```

发现问题时会列出具体差异：

```
❌ 校验失败，发现 3 处差异：
  - MODIFIED:  config/app.yaml (hash mismatch)
  - MISSING:   uploads/logo.png
  - EXTRA:     temp/debug.log (不在清单中)
```

## 清单格式

生成的清单文件内容与 `sha256sum` 兼容：

```
a1b2c3d4...f1e2  package/index.html
b2c3d4e5...f2a3  package/js/app.js
...
```

对方可以直接用系统命令复核：

```bash
sha256sum -c SHA256_clearlist.txt
```

## CLI 帮助

```
checksum-tool <command> [flags]

Commands:
  gen     生成清单
  verify  校验清单

Gen Flags:
  -dir string   要扫描的目录 (required)
  -o string     输出文件路径 (默认 <dir>/SHA256_clearlist.txt)
  -recursive    递归扫描子目录 (默认 true)

Verify Flags:
  -strict       严格模式：清单外的文件也视为异常
```

## 构建

打 tag 自动触发多平台构建并发布到 Release：

```bash
git tag v1.0.0
git push origin v1.0.0
```

## License

MIT