# Architecture Decisions

本文档记录 qkbox v1.0 的长期架构决策。它不是讨论稿，而是实现和 review 的判定基准。

Roadmap 只能引用这里的结论，不应重新论证或引入第二套模型。

---

## AD-001: 工程正统性与恰足性

**Decision**

qkbox 的实现优先追求正确边界、最小必要复杂度和可维护的工程形状。

**Rules**

- 不为明确不支持的行为提供兜底路径。
- 不保留冗余数据真相。
- 不为尚未进入产品合同的未来能力预埋抽象。
- 内部私有边界默认可信，外部和持久化边界必须严格。
- 如果旧模型已经错误，允许大规模重构，而不是在错误模型上兼容演进。

**Consequences**

- 代码审查优先看边界是否真实、数据真相是否唯一、错误是否在正确层处理。
- 临时桥接只允许用于一次性迁移，不允许成为长期运行路径。

---

## AD-002: qkbox 是唯一用户入口

**Decision**

用户只直接启动 `qkbox`。其他二进制是私有 helper。

**Rules**

- `qkbox` 是托盘、IPC server、runtime coordinator 和本地 runtime owner。
- `qkbox-window` 是按需 spawn 的 GUI helper，只由 `qkbox` 启动。
- `qkbox-provider` 是提权 helper，只由 qkbox runtime 编排启动。
- 直接运行 `qkbox-window` 或 `qkbox-provider` 是 unsupported behavior。

**Consequences**

- 包装产物不得把 helper 暴露为用户入口。
- helper 缺少 direct-launch fallback 不是 bug。
- IPC handshake 失败应直接失败，不反向启动主程序。

---

## AD-003: Profile 是配置边界

**Decision**

一个 Profile 就是一份可运行的 sing-box JSON 配置。

**Rules**

- `profiles.content` 是持久化配置边界。
- `profiles.content` 必须是完整 runnable config。
- Profile ID 是用户选择和运行时激活的身份。
- 模板、URI、订阅、JSON 编辑只是产生 Profile content 的输入方式。

**Consequences**

- 激活行为以 Profile 为单位，不以导入来源、模板来源或中间产物为单位。
- Profile 本身不保存节点解析结果作为权威数据。

---

## AD-004: 删除 Profile Snapshot 模型

**Decision**

不保留 Profile Snapshot、Draft/Snapshot 分离或 snapshot activation。

**Rules**

- 不存在 `SnapshotService`。
- 不存在 profile snapshot repository。
- 不存在 `active_snapshot_id`。
- 不存在 `createProfileSnapshot`、`activateProfileSnapshot`、`rollbackToSnapshot` 等 profile snapshot IPC。
- JSON 编辑器中的 dirty text 是前端内存态，不是持久化 draft。

**Consequences**

- Save 的含义是 validate 成功后写入 `profiles.content`。
- Rollback 是 runtime activation 失败后的 service 编排，不是 profile snapshot 回滚功能。
- 代码中仍可存在其他领域的 snapshot 概念，例如 OS proxy snapshot、traffic snapshot；它们不能和 Profile Snapshot 混用。

---

## AD-005: 删除 Managed/Raw Profile 区分

**Decision**

不区分 Managed Profile 和 Raw Profile。

**Rules**

- 所有 Profile 都是同一种实体。
- URI 导入和订阅刷新最终也只写入 `profiles.content`。
- JSON 编辑不会把 Profile fork 成另一种类型。
- 不持久化“解析出的节点列表”作为 Profile 元数据。

**Consequences**

- 节点列表来自 content 解析和运行时状态，不来自 profile mode。
- 用户看到的是一组 Profile，而不是多套 Profile 类型。

---

## AD-006: 订阅是通道，不是 Profile 身份

**Decision**

订阅关系由 `profile_subscriptions.profile_id` 承载，Profile 本身不保存订阅来源。

**Rules**

- `profiles` 表没有 `source_url`。
- 一个可刷新 Profile 由是否存在 subscription row 决定。
- `profile_subscriptions.url` 是订阅通道的来源。
- `data_assets.source_url` 只属于数据资产下载，不属于 Profile。

**Consequences**

- 不存在 Profile source 和 Subscription source 两套真相。
- 删除 Profile 必须同时删除指向它的 subscription row。
- 刷新订阅只更新目标 Profile content，不改变 Profile 身份。

---

## AD-007: 所有持久化配置写入必须权威验证

**Decision**

任何写入 `profiles.content` 的路径都必须先通过 `singboxadapter.Validate()`。

**Rules**

- URI import: parse/build -> validate -> persist。
- Subscription refresh: fetch/parse/build -> validate -> persist。
- JSON save: dirty text -> validate -> persist。
- Template create: build -> validate -> persist。
- 验证失败不写入数据库。

**Consequences**

- `profiles.content` 始终是已通过 sing-box 权威校验的 runnable config。
- 轻量 JSON 检查只能用于即时反馈，不是持久化边界。
- diagnostic 来自 sing-box 权威错误，并在 adapter 层翻译为可展示结构。

---

## AD-008: service 层编排 runtime activation

**Decision**

Profile activation 由 qkboxd service 层编排，而不是由 engine 持有配置或决定业务语义。

**Rules**

Activation 顺序固定：

1. 读取旧 active profile。
2. 加载新 Profile content。
3. validate。
4. 提取 runtime capabilities。
5. stop old runtime。
6. platform prepare。
7. start new runtime。
8. 持久化 `active_profile_id`。
9. 失败时恢复 proxy，并 best-effort 重启旧 profile。

**Consequences**

- rollback 走同一套 validate/prepare/start 路径。
- 运行时失败是 runtime 失败，不会把无效 content 持久化。
- `RuntimeStartTarget` 使用 profile 语义：`ProfileID + ConfigJSON + RequiredCapabilities`。

---

## AD-009: engine 不持有配置

**Decision**

engine 只管理 runtime owner 生命周期和状态，不拥有 Profile content。

**Rules**

- engine 接收 `RuntimeStartTarget`。
- engine 不读取数据库。
- engine 不解析 Profile。
- engine 不保存 ConfigJSON 作为业务状态。

**Consequences**

- service 是业务编排层。
- engine 可以服务本地 runtime、provider runtime 和未来 platform runtime。
- runtime status 暴露 active profile identity，而不是 snapshot identity。

---

## AD-010: 运行时控制不修改配置

**Decision**

节点选择、模式切换、测速和连接观测属于 runtime control，不修改 `profiles.content`。

**Rules**

- `SelectOutbound(groupTag, outboundTag)` 只作用于当前 runtime。
- `SetClashMode(mode)` 只作用于当前 runtime。
- `URLTest(groupTag)` 只触发运行时测试。
- 这些操作不写 Profile content。

**Consequences**

- Profile 保存的是配置定义，不保存用户的临时运行时选择。
- 需要持久化偏好时必须作为独立设置进入产品合同，不能偷偷写回 config。

---

## AD-011: Clash mode 只使用嵌入式控制面

**Decision**

qkbox 不对外暴露 Clash HTTP API。模式切换通过 embedded ClashServer 的本地接口完成。

**Rules**

- sing-box config 可包含 experimental clash api 以创建内部 ClashServer。
- `newBox` 捕获同一个 sing-box context 内的 embedded `ClashServer`。
- `managedBox` 保存本地 `clashModeController`。
- `SetClashMode` 通过接口断言调用本地 controller。
- 不启动、不扫描、不暴露外部 HTTP API。

**Consequences**

- UI 可提供规则、全局、直连切换。
- 不引入外部控制端口的安全面。

---

## AD-012: 节点列表有静态和动态两种来源

**Decision**

节点列表由 Profile content 静态解析和 active runtime groups 动态状态合成。

**Rules**

- inactive profile: 只从 content 解析 group/outbound。
- active profile: content 解析结果按 tag 合并 runtime group 状态。
- 合并 key 是 sing-box tag。
- 不把合成结果持久化为 Profile 元数据。

**Consequences**

- inactive Profile 仍能展示结构。
- active Profile 能展示当前选择、延迟和运行时状态。
- 运行时状态消失时不污染配置。

---

## AD-013: 不做字段级配置加密

**Decision**

v1.0 不保留 `encrypted_content`、`ContentCodec`、`FileKeyStore` 或 `master.key`。

**Rules**

- Profile content 直接存 SQLite。
- 数据库权限依赖 OS 用户边界。
- 若未来需要磁盘加密，应作为整体数据库加密或 OS 加密能力评估，不做局部字段加密补丁。

**Consequences**

- Profile 存储路径更直接。
- 少一套密钥生命周期和损坏恢复问题。

---

## AD-014: 一次性迁移，不长期兼容旧模型

**Decision**

从当前代码进入新架构时，只允许一次性迁移旧数据，不允许长期保留旧 API 或旧表作为兼容路径。

**Rules**

- 迁移完成后删除旧表、旧 repo、旧 service、旧 IPC、旧前端 state。
- 测试应覆盖迁移结果和新模型行为。
- 文档和代码都不能继续把旧模型描述为可用路径。

**Consequences**

- 重构会较大，但最终代码形状干净。
- 后续 Milestone 不需要理解两套 Profile 生命周期。
