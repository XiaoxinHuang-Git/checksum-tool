# checksum-tool

交付物完整性校验工具：生成和校验 SHA256 清单，用于确认交付包在传输、拷贝、解压之后没有被漏发、错发或改动。

- 单个 Go 二进制，无外部依赖，Windows / Linux 直接运行
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

方式一：自行编译（推荐，无网络依赖）

```bash
go build -o checksum-tool .
```

方式二：交叉编译 Linux 版本

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
清单文件: /data/package/SHA256_clearlist.txt
校验目录: /data/package
合计 128　一致 128　变更 0　缺失 0　多余 0
校验通过
```

失败时只打印异常项：

```
FAILED   conf/app.conf
MISSING  lib/core.jar
EXTRA    logs/debug.log

清单文件: /data/package/SHA256_clearlist.txt
校验目录: /data/package
合计 128　一致 126　变更 1　缺失 1　多余 1
error: 校验未通过
```

常用参数：

| 参数 | 说明 |
|---|---|
| `-dir DIR` | 校验根目录，默认清单文件所在目录 |
| `-v` | 打印每个文件的结果，默认只打印异常项 |
| `-strict` | 目录中存在清单之外的文件时也判定失败 |

`-strict` 适合「交付包必须与清单完全一致」的场景；不加时，日志、缓存这类运行时产生的文件只会告警，不影响结论。

## 清单格式

与 `sha256sum` 输出一致，两个空格分隔：

```
ee7bf3a9b4c6d5c4a74cd3dc779f8385dcadbfde77b57670d411b75f7405fac2  app.conf
fa9329660f850f7f6982bf63cdecb856422811dd9a943d9b000ddc014b17a205  lib/core.jar
```

- 按相对路径排序，同一份文件在不同机器上生成的清单逐字节一致
- 路径统一使用 `/`，Windows 生成的清单可以在 Linux 上直接校验
- 跳过 `.git` 目录与清单文件自身

因此对方可以直接用系统命令复核：

```bash
sha256sum -c SHA256_clearlist.txt
```

## 退出码

| 退出码 | 含义 |
|---|---|
| 0 | 全部一致 |
| 1 | 存在变更或缺失（`-strict` 下多余也计入） |
| 2 | 参数错误 |

适合直接放进部署脚本：

```bash
./checksum-tool verify ./SHA256_clearlist.txt || { echo "交付物校验失败，终止部署"; exit 1; }
```

## 开发

```bash
go test ./...
go vet ./...
```

## 边界说明

本工具校验的是**完整性**：文件有没有被漏掉、改动、替换。

它不提供**防篡改**能力——清单和文件一起被替换时无法发现，需要防篡改请配合签名（GPG、cosign）使用。
