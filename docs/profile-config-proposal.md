# Profile/Config 架构方案

## 背景

qkbox 需要一�?Profile/Config 系统，让用户创建、导入、编辑、使用代理配置�?
当前实现基于 Snapshot 模型（Draft �?Snapshot �?激活），引入了较大的复杂度。本方案提出一个更简洁的架构，参考了 singcast、sing-box-windows、FlClash、Clash Verge 等项目的做法�?
---

## 1. 核心模型

```
Profile = sing-box JSON（不透明存储�?```

**没有 Snapshot。没�?Draft/Snapshot 分离。没�?Managed/Raw 区分。一�?Profile 就是一个配置�?*

```
┌─────────────────────────────────────────────────�?�?                   用户操作                       �?�?                                                 �?�? 导入 URI ─�?configbuild ──�?                    �?�? 选模�?  ─�?configbuild ──┼→ Validate �?content �?�? JSON 编辑 ────────────────�?                    �?�? 订阅刷新 ─�?fetch+parse ──�?                    �?�?                                                 �?�? 点击"使用" ─�?activateProfile                   �?�?                 �?编排�?                        �?�?                   stop old runtime               �?�?                   platform prepare               �?�?                   start new runtime              �?�?                   persist active_profile_id       �?�?                 �?失败                           �?�?                   rollback + 报错                �?�?                                                 �?�? 运行�?                                         �?�?   选节�?─�?SelectOutbound()   (不改配置)        �?�?   切模�?─�?SetClashMode()     (不改配置)        �?�?   测延�?─�?URLTest()          (不改配置)        �?�?                                                 �?└─────────────────────────────────────────────────�?```

---

## 2. 存储

### 数据库：4 张表

```sql
CREATE TABLE profiles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    content TEXT NOT NULL,           -- sing-box JSON，权威验证后写入
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE profile_subscriptions (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL REFERENCES profiles(id),
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    update_policy TEXT NOT NULL,     -- "manual" (v1.0)
    last_status TEXT NOT NULL,       -- "pending" / "updated" / "failed"
    last_error TEXT,
    last_checked_at INTEGER,
    last_updated_at INTEGER,
    content_sha256 TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE data_assets (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL,              -- "rule_set" / "geo_site" / "geo_ip" / "srsc"
    name TEXT NOT NULL,
    source_url TEXT NOT NULL,
    status TEXT NOT NULL,
    cache_key TEXT,
    version TEXT,
    content_sha256 TEXT,
    size_bytes INTEGER,
    last_error TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
-- settings 包含:
--   active_profile_id: 当前引擎使用�?Profile
--   window_state: 窗口位置/大小
--   ...
```

**Profile 本身不存 `source_url`�?* 订阅来源�?`profile_subscriptions.profile_id` 关联。是否可刷新 = 是否存在 subscription row。避免双真相�?
### 删除的存储组�?
| 删除 | 原因 |
|------|------|
| `encrypted_content` �?| Profile 直接�?content，不需要单独的内容�?|
| `snapshots` �?| 不做 Snapshot |
| `runtime_state` �?| active_profile_id �?settings 中即�?|
| `ContentCodec`（AES-256-GCM�?| 不做字段级加�?|
| `FileKeyStore` + `master.key` | 不做字段级加�?|
| `content_address` / SHA-256 引用 | 不需要间接引�?|

**7 �?�?4 表�?* 删除的是 Snapshot 和加密间接层，保留所有独立实体�?
如果后续需要磁盘级加密，可以对整个 SQLite 文件加密（或依赖 OS �?BitLocker/FileVault），不需要字段级 AES�?
---

## 3. 验证策略

**核心原则：所有写�?`profiles.content` 的路径都必须权威验证。`profiles.content` �?runnable config 的边界�?*

```
URI 导入:   configbuild �?singboxadapter.Validate() �?通过 �?写入 content
订阅刷新:   fetch+parse  �?singboxadapter.Validate() �?通过 �?写入 content
JSON 保存:  编辑器内�?  �?singboxadapter.Validate() �?通过 �?写入 content
模板生成:   configbuild  �?singboxadapter.Validate() �?通过 �?写入 content
```

**验证失败 = 不写�?content�?* 返回 diagnostic 给用户�?
编辑器中�?dirty text 保持在内存中�?Save" 意味着进入持久化边界，必须先验证通过�?
激活时仍然做最终的启动检查（引擎可能因运行时原因失败，如端口占用、TUN 权限），但这是最后一道防线，不是唯一一道�?
### singboxadapter.Validate()

调用 `box.New()` 进行完整�?schema 校验。返回结构化 diagnostic 列表。这�?sing-box 的权威校验，不是自写�?schema checker�?
---

## 4. Profile 生命周期

### 4.1 创建 Profile

三条路径，产出相同：一�?sing-box JSON 字符串，验证后存�?Profile.content�?
**路径 A：URI 导入**

```
URI 文本
  �?uriparse: 检测格�?�?解析各协�?�?[]ParsedOutbound
  �?configbuild: 模板 + 节点 �?完整 sing-box JSON
  �?singboxadapter.Validate()
  �?通过 �?创建 Profile，存�?content
```

支持的输入格式：
- sing-box JSON（直通）
- Base64 节点列表（decode �?URI 解析�?- 单行 URI（ss:// vmess:// vless:// trojan:// hysteria2:// tuic://�?
支持等级�?- Supported：ss, trojan, hysteria2
- Experimental：vmess（标准格式）, vless（基础参数�?- Unsupported with Diagnostic：vless (reality/xtls), tuic

**路径 B：模板生�?*

```
选择模板（规则模�?/ 全局代理 / 直连�?  �?configbuild: 模板 + 空节点列�?�?完整 sing-box JSON
  �?singboxadapter.Validate()
  �?通过 �?创建 Profile，存�?content
```

模板生成的配置自�?`clash_mode` 路由规则，为运行时模式切换做准备�?
**路径 C：用户手�?*

```
JSON 编辑�?�?用户输入 �?保存
  �?singboxadapter.Validate()
  �?通过 �?写入 Profile.content
  �?失败 �?返回 diagnostic，不写入
```

### 4.2 编辑 Profile

```
JSON 编辑�?�?修改 �?保存
  �?singboxadapter.Validate()
  �?通过 �?写入 Profile.content
       �?如果�?Profile �?active �?�?编排层重启引�?            �?引擎启动失败 �?回滚到保存前�?content + 报错
  �?失败 �?返回 diagnostic，不写入，引擎不受影�?```

**没有 Draft/Snapshot 分离�?* 编辑直接修改 Profile.content�?
### 4.3 激�?Profile

编排�?service 层，不在 EngineController 内部�?
```
func ActivateProfile(profileID):
    // 1. 捕获旧状�?    oldActiveID := getSetting("active_profile_id")
    oldTarget := nil
    if oldActiveID != "":
        oldTarget = loadRuntimeStartTarget(oldActiveID)

    // 2. 构造新启动目标
    // loadRuntimeStartTarget 内部完成�?    //   load content �?Validate �?extract capabilities �?platform prepare
    newTarget := loadRuntimeStartTarget(profileID)
    if newTarget.error != nil:
        return newTarget.error

    // 3. 恢复系统代理（如果之�?qkbox 拥有�?    proxy.RestoreIfOwned()

    // 4. 停止旧引�?    if engine.State() == STARTED:
        engine.Stop()

    // 5. 启动新引�?    err := engine.Start(newTarget)

    // 6. 启动失败 �?回滚
    if err != nil:
        if oldTarget != nil:
            // 回滚使用同一�?runtime start preparation path�?            // best-effort，失败不覆盖原始错误�?            rollbackTarget := loadRuntimeStartTarget(oldActiveID)
            if rollbackTarget.error == nil:
                _ = engine.Start(rollbackTarget)
        return err

    // 7. 持久�?    setSetting("active_profile_id", profileID)
    return nil
```

**关键设计�?*
- `RuntimeStartTarget` �?snapshot 语义迁移�?profile 语义：`ProfileID + ConfigJSON + RequiredCapabilities`
- engine 不持有配置状态。旧配置�?DB 读取
- 编排逻辑�?service 层，EngineController 只负�?start/stop 状态机
- rollback 也走同一�?validate/prepare/start 路径；best-effort 启动失败不覆盖原始错�?
### 4.4 删除 Profile

```
func DeleteProfile(profileID):
    if getSetting("active_profile_id") == profileID:
        return error("不能删除正在使用�?Profile")
    deleteProfile(profileID)
    deleteSubscriptionsByProfile(profileID)
```

---

## 5. 订阅

### 5.1 模型

订阅是独立实体，通过 `profile_id` 指向 Profile�?
```go
type ProfileSubscription struct {
    ID            string
    ProfileID     string    // 指向目标 Profile
    Name          string
    URL           string
    UpdatePolicy  string    // "manual" (v1.0)
    LastStatus    string    // "pending" / "updated" / "failed"
    LastError     string
    LastCheckedAt int64
    LastUpdatedAt int64
    ContentSHA256 string    // 用于变更检�?    CreatedAt     int64
    UpdatedAt     int64
}
```

**Profile 本身不存 source_url�?* 是否可刷�?= `profile_subscriptions` 中是否有�?profile_id 的行。无双真相�?
### 5.2 创建订阅

```
用户输入 URL + 名称
  �?fetch + parse + configbuild
  �?singboxadapter.Validate()
  �?通过 �?在同一事务中创�?Profile(content) + Subscription(profile_id)
  �?失败 �?不创�?Profile，不创建 Subscription
```

`profiles.content` �?`NOT NULL` 且必须是已验证的 runnable config，因此订阅创建没�?�?Profile"中间态�?
### 5.3 刷新订阅

```
func RefreshSubscription(subID):
    sub := loadSubscription(subID)
    oldContent := loadProfileContent(sub.ProfileID)

    // 1. Fetch
    rawContent := fetchRemote(sub.URL)

    // 2. Parse + Build
    newContent, report := parseAndBuild(rawContent)
    // report 包含：成功数、跳过数、不支持�?URI 列表

    // 3. 验证（写�?content 前必须通过�?    err := singboxadapter.Validate(newContent)
    if err != nil:
        updateSubscriptionStatus(subID, "failed", err.Error())
        return err

    // 4. 写入
    updateProfileContent(sub.ProfileID, newContent)
    updateSubscriptionStatus(subID, "updated")

    // 5. 如果�?Profile 是当前激活的，自动重启引�?    if getSetting("active_profile_id") == sub.ProfileID:
        // 编排层：停止旧引�?�?启动新引�?        err := activateProfileInternal(sub.ProfileID, newContent, oldContent)
        if err != nil:
            // 刷新导致引擎挂了 �?回滚 content
            updateProfileContent(sub.ProfileID, oldContent)
            updateSubscriptionStatus(subID, "failed", err.Error())
            return err

    return nil
```

### 5.4 订阅 �?Profile

- 一�?Profile 可以没有订阅（用户手�?模板生成/URI 导入�?- 一�?Profile 最多有一个订阅（1:1�?- 订阅刷新 = 重新获取 + 验证 + 替换 content + 可能重启引擎
- 下一次刷新会覆盖用户手动编辑�?content（这是预期行为——订阅是内容来源�?
---

## 6. 运行时操作（不改配置�?
所有运行时操作通过 `singboxadapter` �?Go API 实现�?
### 6.1 节点列表

两个数据源按 tag 合并�?
```
�?1: content 解析（静态）
  JSON parse Profile.content �?提取 outbounds
  识别已知节点类型（shadowsocks/vmess/vless/trojan/hysteria2/tuic�?  提取 tag, type, server, port
  复杂结构（嵌�?selector 等）�?diagnostic，不强制解析

�?2: 运行�?groups（动态，引擎运行时可用）
  engine.OutboundGroups() �?每组的成�?+ 当前选中 + urltest 状�?  engine.URLTest() �?延迟数据

合并: �?tag 关联
  静态信息（type/server/port）来�?content 解析
  动态信息（selected/delay/group membership）来自运行时
```

**两个数据源互补：**
- content parser 提供 inactive profile 的节点信�?- 运行�?groups 提供 active profile 的实时状�?- 不持久化节点信息，每次按需解析

### 6.2 代理组切�?
```
engine.SelectOutbound(groupTag, outboundTag)
  �?运行时切换，不改配置文件
```

### 6.3 延迟测试

```
engine.URLTest(groupTag) �?延迟结果
```

### 6.4 模式切换（新增）

实现方式�?*嵌入�?ClashServer + 本地接口断言**，不启用 HTTP API�?
```go
// singboxadapter 内部
type clashModeController interface {
    Mode() string
    ModeList() []string
    SetMode(string)
}

func (a *Adapter) SetClashMode(mode string) error {
    // newBox 创建 sing-box 后，从同一�?sing-box context 捕获 embedded ClashServer
    // managedBox 持有�?controller
    // 调用 SetMode()
    // 如果不可�?�?返回 capability unsupported
}
```

`newBox` 创建 box 后，从同一�?sing-box context 捕获 embedded `ClashServer`，type assertion 到本�?`clashModeController`，并保存�?`managedBox` 中。`SetClashMode` 只调用这个内�?controller，不寻找或启动外�?HTTP API�?
这和现有 `SelectOutbound` 的实现模式一致：通过本地接口断言调用 sing-box 内部能力，不暴露 HTTP API�?
**前提�?* 配置中有 `clash_mode` 匹配规则。configbuild 生成的模板保证这一点�?
### 6.5 连接管理

已有实现，不变�?
### 6.6 流量统计

已有实现，不变�?
---

## 7. IPC 协议变更

### 删除的方法（8 个）

```
- validateProfileDraft
- getProfileDiagnostics
- createProfileSnapshot
- activateProfileSnapshot
- getActiveSnapshot
- listSnapshots
- rollbackToSnapshot
- getActiveProfile
```

### 新增的方�?
```
+ activateProfile           激�?Profile �?编排层启动引擎（失败自动回滚�?+ engine.reloadActiveProfile 重新加载当前 active Profile（失败自动回滚）
+ engine.setClashMode       运行时模式切�?```

### 保留的方�?
```
Profile CRUD:     createProfile, updateProfile, deleteProfile, listProfiles, getProfile
订阅 CRUD:        asset.* (8 个方法，不变)
引擎:             engine.start, engine.stop, engine.reloadActiveProfile, engine.getStatus, engine.subscribe*,
                  engine.getRuntimeCapabilities, engine.listGroups, engine.selectOutbound,
                  engine.urlTest, engine.closeConnection, engine.closeAllConnections
平台:             platform.* (7 个方法，不变)
诊断:             diagnostics.* (2 个方法，不变)
窗口:             window.attach (不变)
Hello:            hello (不变)
```

### activateProfile

```go
type ActivateProfileRequest struct {
    ProfileID string `json:"profile_id"`
}

type ActivateProfileReply struct {
    ActiveProfileID string `json:"active_profile_id"`
}
```

引擎启动失败时返回结构化错误（包�?sing-box 的错误信息），引擎自动回滚�?
### engine.setClashMode

```go
type SetClashModeRequest struct {
    Mode string `json:"mode"` // "rule" / "global" / "direct"
}

type SetClashModeReply struct{}
```

---

## 8. 前端 UX 简�?
### Profile 页面�? 面板 �?3 区域

**当前�? 面板）：**
```
1. Profile 列表
2. JSON 编辑�?3. 验证状�?4. Active Runtime Target
5. Snapshot 列表
6. 订阅管理
7. Data Assets
```

**新方案（3 区域）：**
```
┌──────────────────────────────────────�?�?1. Profile 列表                       �?�?   - 名称 + active 标记               �?�?   - 创建/删除                        �?�?   - 点击 = 选择并激�?               �?├──────────────────────────────────────�?�?2. JSON 编辑�?                       �?�?   - 编辑选中 Profile �?content      �?�?   - 保存 = Validate + 写入 content   �?�?   - Validate 失败 �?展示 diagnostic  �?�?   - 如果�?Profile �?active         �?�?     保存成功后引擎自动重�?          �?├──────────────────────────────────────�?�?3. 订阅管理                           �?�?   - 订阅列表（URL + 状态）           �?�?   - 添加/刷新/删除                   �?�?   - 刷新 = Validate + 替换 + 重启    �?└──────────────────────────────────────�?```

### 删除�?UI 元素

- Snapshot 列表、创�?激�?回滚按钮
- Active Runtime Target 面板
- validation_status 独立面板（验证结果内联到保存/刷新流程中）
- "fork on edit" 警告

### 代理页面

节点列表（content 解析 + 运行时合并）、代理组切换、延迟测试、快速启停。模式切换按钮（规则/全局/直连）�?
### 数据资产

迁移到诊断页面的进阶区域。不属于 Profile 核心流程�?
---

## 9. 和参考项目的对比

| 维度 | singcast | sing-box-windows | Clash Verge | qkbox（本方案�?|
|------|----------|------------------|-------------|----------------|
| 配置格式 | Clash YAML | sing-box JSON | Clash YAML | sing-box JSON |
| 配置存储 | YAML 文件 | JSON 文件 | YAML 文件 | SQLite content 字段 |
| Snapshot | �?| �?| �?| **�?* |
| 节点编辑 | 不做 | 不做 | 不做 | 不做 |
| 模式切换 | Clash API HTTP | Clash API HTTP | Clash API HTTP | singboxadapter 内部 Go API |
| 订阅刷新 | 替换文件 | 替换+重生成骨�?| 替换文件 | 验证 + 替换 content + 自动重启 |
| 回滚 | 重新导入 | backup 文件 | �?| 引擎启动失败自动回滚 |
| 配置校验 | 核心 checkConfig | 自行验证 | 核心验证 | **所有写入路径必须验�?* |
| 加密 | �?| �?| �?| 无（后续可加文件级） |

---

## 10. 模板引擎

模板�?configbuild 的能力，不是 Profile 的属性。模板一次性生成完整配置，之后配置就是普�?Profile�?
### 三个模板

**规则模式（默认）�?*
```json
{
  "inbounds": [
    { "type": "mixed", "listen": "127.0.0.1", "listen_port": 7890 }
  ],
  "outbounds": [
    { "type": "selector", "tag": "proxy", "outbounds": ["auto", "节点1", "节点2", ...] },
    { "type": "urltest", "tag": "auto", "outbounds": ["节点1", "节点2", ...] },
    { "tag": "direct", "type": "direct" },
    { "tag": "block", "type": "block" },
    ...用户节点...
  ],
  "route": {
    "rules": [
      { "clash_mode": "global", "outbound": "proxy" },
      { "clash_mode": "direct", "outbound": "direct" },
      { "rule_set": ["geosite-cn", "geoip-cn"], "outbound": "direct" },
      { "rule_set": ["geosite-geolocation-!cn"], "outbound": "proxy" }
    ],
    "final": "proxy",
    "rule_set": [...]
  },
  "dns": { ... }
}
```

**全局代理�?*
```json
{
  "route": {
    "rules": [
      { "clash_mode": "direct", "outbound": "direct" }
    ],
    "final": "proxy"
  }
}
```

**直连模式�?*
```json
{
  "route": {
    "final": "direct"
  }
}
```

模板中的 `clash_mode` 规则保证运行时模式切换可以工作�?
---

## 11. 删除清单

### 存储�?
| 删除 | 文件 |
|------|------|
| `encrypted_content` �?| `persistence/migrations.go` |
| `snapshots` �?| `persistence/migrations.go` |
| `runtime_state` �?| `persistence/migrations.go` |
| `ContentCodec` | `core/qkboxd/content_codec.go` |
| `FileKeyStore` | `internal/crypto/` |
| Snapshot repository | `persistence/snapshot_repository.go` |
| Content repository | `persistence/content_repository.go` |

### 服务�?
| 删除 | 文件 |
|------|------|
| `SnapshotService` | `core/qkboxd/snapshot_service.go` |
| draft/content 间接方法 | `core/qkboxd/profile_service.go` |

### IPC �?
| 删除 | 方法 |
|------|------|
| 8 �?snapshot 方法 | `shared/api/methods.go`, `internal/ipc/server.go`, `internal/ipc/wire.go`, `internal/ipc/client.go` |
| snapshot DTO | `shared/api/snapshot.go`, `shared/model/snapshot.go` |

### 前端

| 删除 | 文件 |
|------|------|
| Snapshot 列表 UI | ProfilesView snapshot 面板 |
| Active Runtime Target | ProfilesView runtime target 面板 |
| Validation 独立面板 | ProfilesView validation 面板 |
| snapshot 相关 state | `lib/state/profile.svelte.ts` |

---

## 12. 新增清单

| 新增 | 位置 | 说明 |
|------|------|------|
| `activateProfile` IPC | service.go + IPC 全链�?| 替代 8 �?snapshot 方法 |
| `SetClashMode` | singboxadapter | 嵌入�?ClashServer + 接口断言 |
| `engine.setClashMode` IPC | service.go + IPC 全链�?| 模式切换通道 |
| `Validate()` | singboxadapter | `box.New()` 权威校验，返�?diagnostic |
| Profile.content �?| persistence | 直接�?JSON |
| content 解析�?| internal �?core | �?JSON 提取 outbounds 用于节点列表 |
| 模式切换 UI | 代理页面 + 托盘 | 规则/全局/直连 |
| `RuntimeStartTarget` 演化 | engine.go | �?snapshot 语义�?profile 语义 |

---

## 13. 迁移路径

1. **数据库迁�?*：profiles 表新�?content 列。将 active snapshot �?content 复制到对�?profile。删�?snapshot/encrypted_content/runtime_state 表�?2. **IPC 迁移**：删�?8 �?snapshot 方法，新�?activateProfile。更�?MethodRegistry�?3. **服务�?*：删�?SnapshotService + ContentCodec。改�?profile_service 为纯 CRUD + activate 编排�?4. **singboxadapter**：新�?Validate() �?SetClashMode()�?5. **前端迁移**：ProfilesView �?7 面板重构�?3 区域。删�?snapshot 相关 state�?6. **清理**：删�?FileKeyStore、加密相关代码�?
---

## 14. 设计决策记录

| 决策 | 选择 | 原因 |
|------|------|------|
| Snapshot 概念 | 删除 | 行业不用，增加复杂度不增加安全�?|
| Managed/Raw 区分 | 取消 | 统一 Profile，Subscription 是独�?channel |
| Profile.source_url | 不放�?profiles �?| 避免双真相，�?subscription row 决定 |
| 配置加密 | 删除（字段级�?| master key 明文存储保护意义有限 |
| 验证策略 | 所有写入路径必须验�?| content �?runnable config 边界 |
| 节点信息 | 按需�?content 解析，不持久�?| + 运行�?groups 合并 |
| 模式切换 | singboxadapter 内部 Go API | 嵌入�?ClashServer + 接口断言，不暴露 HTTP |
| RuntimeStartTarget | �?snapshot 迁移�?profile 语义 | 编排�?service 层，engine 不持有配�?|
| 订阅刷新 | 验证 + 替换 content + 自动重启 | �?Clash Verge 一�?|
| 回滚机制 | 编排层启动失败自动回�?| 旧内容从 DB 读取，不依赖 engine 状�?|
| 配置格式 | sing-box JSON 原生 | 不做 Clash 转换 |
