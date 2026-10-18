# 雀魂助手 UI（Majan Helper UI）

> **⚠️ AI 创作的实验品，不保证功能**
>
> 本项目的 UI、鸣牌判断层和文档由 AI 辅助生成/修改，仅用于学习和实验。**不保证正确性、稳定性、兼容性或持续维护**，使用风险自负。
>
> 本项目基于 [EndlessCheng/mahjong-helper](https://github.com/EndlessCheng/mahjong-helper)（MIT）修改，**不是官方项目**，与原作者、雀魂/Mahjong Soul 官方均无关联。
>
> 使用第三方助手可能违反游戏服务条款，请自行判断风险；建议仅用于学习、单机或友人对局。

![主界面](docs/screenshot-main.png)

## 必须搭配 MajsoulMax 使用

本程序本身只负责**接收对局数据、分析并显示 UI**，对局数据来自 [Avenshy/MajsoulMax](https://github.com/Avenshy/MajsoulMax) 的 `helper` 插件。

1. 先按 [MajsoulMax 的 README](https://github.com/Avenshy/MajsoulMax) 安装并配置好 MajsoulMax。
2. 确认 `MajsoulMax/config/settings.yaml` 中：
   ```yaml
   plugin_enable:
     helper: true
   ```
3. 确认 `MajsoulMax/config/settings.helper.yaml` 中的地址是：
   ```yaml
   config:
     api_url: https://localhost:12121/
   ```
4. 启动本程序，再打开雀魂开始一局。

> 不要同时运行原版 `mahjong-helper.exe`，它同样占用 12121 端口。

## 功能

- **92px 大字推荐打牌**，一眼看清
- 前 5 候选：进张、和率、危险度
- **鸣牌判断**：建议吃/碰/杠或建议跳过，并说明理由
  - 无役的副露直接判“建议跳过：无役，鸣牌后不能和牌”
  - 门清优先保持立直；只有向听前进且确定有役，或获得役种后明显更快时才建议鸣
  - 会导致振听的鸣牌直接跳过
- 防守推荐、手牌、副露、宝牌
- 最近 12 巡的 `自家切牌 / 进攻 / 防守` 记录
- 深色 520×760 侧边栏，默认放在游戏窗口右侧

![鸣牌判断](docs/screenshot-call.png)

## 鸣牌判断规则

这一层是**保守的启发式**，不是完整的攻守押し引き AI：

| 情况 | 建议 |
|---|---|
| 鸣牌后无役 | 跳过（无役不能和牌） |
| 鸣牌后向听前进，且有役 | 鸣牌 |
| 已听牌的门清手牌 | 优先保持门清立直，除非打点/进张明显更好 |
| 向听没有前进、速度也没有明显提升 | 跳过，保持门清 |
| 门清无役，鸣牌获得役种且速度明显更快 | 鸣牌 |
| 鸣牌会导致振听 | 跳过 |

UI 会显示每个后切选项的 `断幺 / 役牌 / 三色…` 或红色 `无役`，以及“不鸣”的听牌/进张对比。最终选择仍由你判断。

## 快速开始（Windows 10/11 x64）

### 方式 A：下载 Release

如果本仓库的 **Releases** 页面提供了 `mahjong-helper-ui-win64.zip`：

1. 解压到一个**可写**文件夹。
2. 关闭原版 `mahjong-helper.exe`。
3. 双击 `start-ui.vbs`。
4. 接受数据参考雀魂max项目https://github.com/Avenshy/MajsoulMax

### 方式 B：从源码构建

要求：Go 1.20+（推荐 1.27），Windows x64。

```powershell
cd mahjong-helper-ui
$env:GOPROXY="https://goproxy.cn,direct"
$env:GOTOOLCHAIN="local"
$env:CGO_ENABLED="0"
go build -trimpath -ldflags "-s -w -X main.version=0.3.1-ui" -o mahjong-helper-ui.exe .
```

也可以直接运行仓库里的 `build.ps1`：

```powershell
.\build.ps1
```

构建完成后，`mahjong-helper-ui.exe` 与本目录的 `start-ui.vbs` 在同一个文件夹，双击 `start-ui.vbs` 即可。

> 国内网络可把 `GOPROXY` 换成 `https://goproxy.cn,direct` 或你常用的 Go 模块代理。
>
> 生成的 exe 未签名，Windows Defender 可能会提示；请自行决定是否允许。

## 启动与端口

| 文件/端口 | 作用 |
|---|---|
| `start-ui.vbs` | 隐藏启动助手，并用 Edge 应用窗口打开 UI |
| `start-ui-fallback.cmd` | 备用：最小化控制台 + 默认浏览器 |
| `stop-ui.cmd` | 停止助手 |
| `127.0.0.1:12121` | HTTPS 数据入口，MajsoulMax 插件继续使用 |
| `127.0.0.1:12122` | HTTP UI（`/ui`、`/ws`、`/api/state`） |

默认窗口位置 `(1390, 30)`、尺寸 `520×760`。想改位置，编辑 `start-ui.vbs` 里的 `--window-position`。

如果 12122 被占用，可以用：

```powershell
.\mahjong-helper-ui.exe -majsoul -p 12121 -ui-port 12345
```

然后打开 `http://127.0.0.1:12345/ui`。

## 无游戏自检

```powershell
cd mahjong-helper-ui
.\mahjong-helper-ui.exe -majsoul -p 12123 -ui-port 12124
```

另开一个 PowerShell：

```powershell
# 普通摸牌分析
$body = '{"reset":true,"tiles":"123m456p789s11223z"}'
[System.IO.File]::WriteAllText("$env:TEMP\body.json", $body, (New-Object System.Text.UTF8Encoding($false)))
curl.exe -k -X POST "https://127.0.0.1:12123/analysis" -H "Content-Type: application/json" --data-binary "@$env:TEMP\body.json"

# 鸣牌判断示例：吃 4万，断幺九且向听前进
$body = '{"reset":true,"tiles":"234m567p55m23m45p5s","target":"4m"}'
[System.IO.File]::WriteAllText("$env:TEMP\body.json", $body, (New-Object System.Text.UTF8Encoding($false)))
curl.exe -k -X POST "https://127.0.0.1:12123/analysis" -H "Content-Type: application/json" --data-binary "@$env:TEMP\body.json"
```

打开 `http://127.0.0.1:12124/ui` 查看 UI；`http://127.0.0.1:12124/api/state` 可查看原始 JSON 状态。

## 日志回放（可选）

`tools/replay` 可以把 `gamedata-*.log` 里的历史消息重新发给测试实例：

```powershell
go run ./tools/replay -log "..\log\gamedata-20261016-192334.log" -url "https://127.0.0.1:12123/" -delay 5ms
```

建议用独立端口（如 12123/12124）测试，避免和实时对局冲突。

## 常见问题

**UI 一直显示“等待牌局数据…”**
- 雀魂是否已经开局？
- MajsoulMax 的 `plugin_enable.helper` 是否为 `true`？
- 12121 是否被原版助手占用？先关闭原版。

**启动后没有窗口**
- 双击 `start-ui-fallback.cmd`，用默认浏览器打开。
- 检查 `log/gamedata-*.log` 是否有端口占用或解析错误。

**Edge 窗口尺寸不对**
- Edge 复用了已有进程时可能忽略 `--window-size`。可以给 `start-ui.vbs` 的 Edge 参数加独立配置目录：
  `--user-data-dir=%LOCALAPPDATA%\MajsoulHelperUIEdge`
  代价是首次打开可能多一个引导页。

**想要置顶**
- 当前版本没有置顶；可以用 PowerToys 的 Always on Top（`Win+Ctrl+T`）把窗口钉住。

## 已知限制

- 这是 AI 辅助的实验品，**不保证功能正确**。
- 鸣牌判断是基于役种/向听/进张/振听的启发式规则，不是完整 AI；偏保守，可能错过一些激进但正确的副露。
- 没有做完整的防守押し引き、读牌、立直判断优化。
- 仅保证 Windows x64 的基本使用路径；其他平台请自行调整启动方式。
- UI 的 Edge 应用窗口依赖 Edge 已安装；没有做独立安装包。
- 未签名 exe 可能被安全软件拦截。

## 目录结构

```
.
├── analysis.go / core.go / cli.go ...   # 原 mahjong-helper 核心（MIT）
├── ui_state.go                          # UI 状态 DTO / 发布器
├── ui_call.go                           # 鸣牌判断（无役/振听/速度）
├── ui_server.go                         # HTTP / WebSocket / 内嵌网页
├── web/                                 # 前端页面
├── tools/replay/                        # 日志回放测试工具
├── start-ui.vbs / start-ui-fallback.cmd / stop-ui.cmd
├── docs/                                # 截图、详细用法、原项目 README
├── LICENSE                              # MIT（原项目版权）
└── README.md
```

## 许可与致谢

- 原项目：[EndlessCheng/mahjong-helper](https://github.com/EndlessCheng/mahjong-helper)，MIT License。
- 对局数据接入：[Avenshy/MajsoulMax](https://github.com/Avenshy/MajsoulMax)（独立项目，本仓库不包含其代码）。
- 本仓库的 UI、鸣牌判断和文档修改：AI 辅助生成，按 MIT 协议提供。
- 原项目 LICENSE 保留在根目录 `LICENSE`；上游 README 保存在 `docs/upstream-README.md`。

## 免责声明

本项目仅供学习、研究和单机/友人对局辅助实验使用。作者不对任何账号处罚、游戏结果、数据丢失或其它损失负责。使用第三方助手可能违反游戏服务条款，请自行评估并承担风险。若你不接受这些条件，请不要使用本项目。
