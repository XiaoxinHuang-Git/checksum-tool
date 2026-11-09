# checksum-tool

![lang](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)
![license](https://img.shields.io/badge/License-MIT-yellow.svg)
![platforms](https://img.shields.io/badge/platform-Windows%20%7C%20Linux-lightgrey.svg)

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
FAILED   config/app.php              (hash mismatch)
MISSING  vendor/cache/file.cache     (清单里有但目录里没找到)
EXTRA    temp/backup.sql            (目录里有但清单里没有)
合计 128　一致 125　变更 1　缺失 1　多余 1
校验不通过
```

`-v` 参数会打印每个文件的校验结果（不只是异常项）：

```bash
checksum-tool verify -v ./package/SHA256_clearlist.txt
```

`-strict` 参数要求目录里没有清单之外的文件才通过（比如要确认交付包解压后没有多余的 .DS_Store）：

```bash
checksum-tool verify -strict ./package/SHA256_clearlist.txt
```

### 比对两次清单的差异

```bash
checksum-tool diff ./v1.0/SHA256.txt ./v1.1/SHA256.txt
```

输出：

```
ADDED   bin/new-service
MODIFIED  config/app.php
REMOVED old/deprecated-library.php
```

## 清单格式

与 `sha256sum -c` 兼容，格式：

```
SHA256_HASH  RELATIVE_PATH
```

示例：

```
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  config/app.php
021baeb0f19f2f84c7a5b0c4d6e8f1a2b3c4d5e6f7890abcdef1234567890abcdef  vendor/autoload.php
```

对方没有本工具时，用系统自带命令也能复核：

```bash
cd ./package && sha256sum -c SHA256_clearlist.txt
```

## 版本

- v0.1.0  初版，支持 gen / verify / diff 三个子命令

## License

MIT
