# 给 Codex 的下一步增强指导：报告深化 + PPT 生成

> 用途：AI 赛道 builder 已验收通过（feat/ai-track-builder，10 PASS，CI 双平台 success）。这是**下一阶段**的增量需求，聚焦你还没做的"报告深化"和"PPT 制作"。
> 用法：复制「=====」之间内容粘贴到 Codex。**强调：在现有 builder 基础设施上扩展，不要另起炉灶、不要破坏已通过的 10 PASS。**
> 依据资料（已在仓库）：`research/AI赛道_报告PPT视频_全网备赛资料.md`、`knowledge/rules_7_ai_2026.json`。

---

## ===== 从这里开始全部复制 =====

你刚完成的 AI 赛道 builder（architect + video-planner）我已验收通过：10 PASS、0 FAIL，macOS/Windows CI 都 success，`TestBuilderTrackSwitchAndTwoVideoRequirements` 写得很到位。现在是**下一阶段增量需求**，聚焦两个你还没覆盖的能力：**报告深化**和**PPT 制作**。请在现有 builder 基础设施上扩展，保持已通过的测试全绿。

### 背景：现在的缺口

我核对了你的代码：
- `isBuilder()` 只认 architect / video-planner，`BuilderPlan` 只有 sections/scores/timeline/demo/annotations，没有 slides 字段
- 汇报与答辩工作区仍是纯 checker（main.go 的 instruction 里对 PPT 只说"修改按具体问题—影响—修改方法"），没有任何"从零生成 PPT"能力
- 你没碰过 web/presentation 相关代码

也就是说：**报告能搭骨架但不能逐段成文，PPT 完全不能生成**。用户最需要的第三件事仍是空白。

### 任务 A：architect 增加"逐段扩写"模式（报告深化）

保留现有"骨架生成"，新增一个可选的深度模式，请求带 `depth:"full"` 时对骨架的每一节扩写成可直接用的正文草稿：

1. `BuilderPlan` 的每个 `BuilderSection` 增加可选字段 `draft`（该节正文草稿）。**不要新增顶层字段**，避免在 `validateBuilderPlan` 的 `DisallowUnknownFields` 下破坏现有测试。
2. 扩写规则（写进 `generationPrompt` 的 architect 分支，depth=full 时）：每节按"痛点证据→改革思路→具体做法→可补证据位"展开；**正文里所有成效数字、学生原话、获奖信息必须留空为"待补：XXX"占位**，沿用你已有的 `generatedFactRisk` 正则做硬校验——草稿里出现任何具体成效数字直接拒绝。
3. 字数按该节 `word_budget` 控制，合计不超过 `rules_7_ai_2026.json` 的 `word_limit`（4000）。
4. 复用你现有赛道感知：AI 赛道才扩写"AI 伦理与治理"章节；其他赛道按 `policy.Dimensions` 扩写，不串 AI 内容。
5. 加测试：depth=full 生成后，检查每节有 draft、draft 内无编造成效数字、字数预算不超、新文科无 AI 伦理节。

### 任务 B：新增 builder 顾问 `presentation-architect`（PPT 生成，本次核心）

这是新能力，复用你建好的整套 builder 基础设施（`isBuilder`、`generate` 模式、`validateBuilderPlan`、`BuilderPlan`、embed 检索、`completeGeneration`）：

1. **注册**：`specialists.go` 的 `teachingSpecialists` 加 `{"presentation-architect", "汇报PPT架构师", ...}`；`isBuilder()` 加入该 ID。前端下拉走 `/api/specialists` 动态渲染，不用改前端。
2. **输出结构**：给 `BuilderPlan` 增加一个 `slides` 数组（同样作为可选字段，不破坏现有必填字段校验）。每张 slide 至少含：`id`、`page_no`、`title`（观点写在标题上）、`section`（属于 3+6+3 哪段）、`bullets`（要点）、`visual`（需要的图/表/截图类型）、`evidence_source`（引用本轮检索到的 material id，不得虚构）、`speaker_note`（讲稿要点）、`duration_sec`（该页讲述秒数）。
3. **生成规则**（写进 `generationPrompt` 的 presentation-architect 分支，复用 `builderPolicyFor` 的赛道/届次感知）：
   - **硬性上限只有官方的 ≤12 分钟（720 秒）**：`duration_sec` 合计必须 ≤720，这是唯一能用来拒绝方案的时长规则。
   - **3+6+3**（课程与痛点 3 分钟 / 创新内容 6 分钟 / 成效与推广 3 分钟）与 **≤30 页**来自公开专家经验（见 `research/AI赛道_报告PPT视频_全网备赛资料.md`），属建议而非官方评分条文；生成时尽量满足，**不得因 711–720 秒而拒绝**。
   - **690–710 秒收尾、末页留一页"总结图"可跳过**只是综合得出的经验目标，实现为正向提示或 warning，**不得写进 `validateBuilderPlan` 的拒绝条件**。
   - 每页一个观点且**写在 title 上**；bullets 不超过 4 条。
   - **可视化**：`visual` 字段按每页实际需要标注（流程图/截图/对比图等），由用户按自己课程内容决定。**不要预设任何"标准架构图"或固定图样**——官方评分标准和可追溯的专家原文都没有指定 PPT 必须画哪种图。有出处的相关要求只有一条：众师云 12799 要求**成果报告**插"整体创新逻辑框架图"（那是报告，不是 PPT）。
   - **数据纪律**：slides 里不得出现学生人数、分数、提升率、获奖等成效断言（沿用 `generatedFactRisk` 校验）；`evidence_source` 只能引用本轮实际检索到的 source id；引用专家打法标注"迁移建议非专家本人观点"，获奖案例标注"往届公开报道，不是你的课程事实"。
   - 提到"匿名"要求：演讲者页/页脚/截图 placeholders 里不得出现姓名、学校、院系。
4. **校验扩展**：`validateBuilderPlan` 在 role=presentation-architect 时，**只做有官方出处的硬校验**：slides 非空、`duration_sec` 合计 ≤720、每张 slide 必填字段齐全、`evidence_source` 都在 available 里。页数≤30、3+6+3 三段有页、690–710 秒目标一律作为 warning 列表返回，**不得作为拒绝条件**。其他 role 的校验路径保持原样。
5. **测试**：新建或扩充 `generate_test.go`，加 `TestPresentationArchitectPlan`：验证总时长 ≤720、证据源合法，且 30 页/3+6+3/690–710 秒只产生 warning 不拒绝；写 2 个破坏性用例（时长 721 秒、`evidence_source` 引用不存在的 id）必须被拒。确认新增 role 不影响原 10 PASS。

### 任务 C：三份材料一致性检查（可选，有余力再做）

新增一个 checker 能力（可放在现有 `evidence` 或 `consistency` 顾问，不必新顾问）：上传报告 + PPT 讲稿 + 视频脚本后，检查三份材料里的课程名称、核心术语、数据口径、创新主线是否一致，不一致处列出位置。只做对照提示，不虚构。

### 通用要求（不变）

- 保留 feat 分支已通过的全部能力和测试；新功能是加法，`go test ./...` 必须继续全绿（CI 双平台）。
- 仍然开分支（可继续用 feat/ai-track-builder 或新建 feat/ppt-builder），完成后给我分支名和 PR。
- 交付列出：改了哪些文件、新顾问 ID、新增字段如何校验、测试结果。做不到说明原因，不要假装完成。

### 自检

完成后按 `research/如何验收Codex的改造.md` 跑原有测试，再补测新的两个能力：① architect 的 depth=full 逐段扩写不编数字；② presentation-architect 生成总时长 ≤12 分钟（≤720 秒）的 PPT 方案，30 页/3+6+3/690–710 秒只是经验目标、不硬卡。

## ===== 复制结束 =====

---

## 给你的补充说明（不用发给 Codex）

### 为什么这样设计
- **复用而非重写**：Codex 已经建好了 builder 全套基建（验证器、embed、检索、generate 路由、赛道感知）。新增 PPT 顾问只是往这套基建里加一个 role 和一个可选字段，改动小、风险低、不会破坏已通过的 10 PASS。
- **PPT 生成 vs PPT 审阅**：现有 presentation 工作区是"你上传 PPT 后逐页挑错"，这是 checker；新顾问是"从零规划整份 PPT 的页序、要点、讲稿、时长"，这是 builder。两者互补，不冲突。
- **不预设图样**：此前助手自拟的两张架构图示例（出处不明）已按用户要求删除。PPT 可视化由用户按自己课程决定；指导里不再预设任何"标准图样"，避免把无出处的内容变成事实或硬性规则。

### 数据纪律贯穿始终
任务 A 的逐段扩写和任务 B 的 PPT 生成都复用了 Codex 已有的 `generatedFactRisk` 正则——这是它自己写的"不编造成效数字"硬校验。我要求继续复用，是为了防止新增能力反而把"不虚构"这条底线冲垮。

### 落地建议
如果一次发三个任务怕改动太大，可以先发**任务 B**（PPT 生成，你最需要、也最独立），验收通过后再发 A 和 C。任务 B 单独就能让 AGENT 帮你规划整份 12 分钟汇报 PPT。
