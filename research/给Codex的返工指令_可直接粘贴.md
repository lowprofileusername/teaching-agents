# 给 Codex 的返工指令（直接复制「=====」之间全部内容粘贴到聊天窗口）

> 背景：你已发布 v0.4.8，但那是"新老师项目与工作区重置"功能。我核对了代码：**AI 赛道 builder 改造一项都没做**。下面是返工要求，请照做。

---

## ===== 从这里开始全部复制 =====

我核对了刚发布的 v0.4.8，发现你合并了我的研究资料并发了版，但**没有执行我此前给你的改造任务**。验收脚本在最新版 main 上跑出 0 PASS / 4 FAIL / 8 RED FLAG——等于完全未改造。请现在按下面清单补齐，这是真正的任务（v0.4.8 的新项目/重置功能保留，不要回退）。

### 先确认现状（我已替你核过，你可直接采信）
- `builtin.go:16` 的 embed 仍是 `knowledge/rules.json knowledge/rules_6_2026.pdf knowledge/experts.json`，三个新 JSON 未内置
- `specialists.go:18` 的 `teachingSpecialists` 仍是 8 个顾问，无 architect / video-planner
- `main.go:192/195/198` 只有三个审阅型 sys 分支，无 generate 模式
- `web/app.mjs:267` 顾问下拉是从 `/api/specialists` 动态渲染，所以**只要 specialists.go 加了顾问，前端自动出现，不用改前端**
- 研究资料已全部在 main：`knowledge/rules_7_ai_2026.json`、`ai_playbook.json`、`ai_award_cases_6th.json`、`video_index_5th_wenke.json`，以及 `research/资料总览与Codex使用说明.md`（先读这份）、`research/给Codex的提示词.md`（详细任务书）、`research/如何验收Codex的改造.md`（验收标准）

### 任务 1：真正内置四个 JSON（不是只塞进 embed）
1. `builtin.go:16` 的 `//go:embed` 加入四个文件：`knowledge/rules_7_ai_2026.json`、`knowledge/ai_playbook.json`、`knowledge/ai_award_cases_6th.json`、`knowledge/video_index_5th_wenke.json`
2. **关键**：参照现有 `builtinLibrary()`（builtin.go:45）的只读加载 + `builtinRuleSearch()`（builtin.go:105）的检索方式，为这四个文件写实际加载与检索代码。只加 embed 不写读取 = 未完成——验收脚本会检查每个文件名在 .go 文件中的出现次数，≤1 次直接判 RED FLAG。
3. 届次/性质感知：用户问第七届 AI 赛道时命中 `rules_7_ai_2026.json` 并标注"省赛校赛口径汇总，非国赛正式标准"；问第六届仍走原有 `rules.json`。四类资料回答时要能区分：第六届规则=往届参考、第七届规则=非正式标准、打法库=AI迁移建议（非专家本人观点/非官方标准）、获奖案例=公开报道（非报告全文/非用户成效）。

### 任务 2：新增两个 builder 顾问（specialists.go）
在 `teachingSpecialists` 数组加两个角色，字段风格与现有一致（ID/Name/Purpose/Checks/Deliverable/Keywords）：
1. `architect`（报告架构师）：从零生成 4000 字报告骨架与逐段字数预算。Checks 含痛点可溯源、唯一创新主线、6 维评分落点、成效有前后数据、AI 伦理治理单列。Deliverable 强制产出：①一句话创新主线 ②逻辑闭环图描述 ③独立"AI 伦理与治理"章节 ④6 维评分对照表 ⑤需用户补充的证据清单。引用 `ai_award_cases_6th.json` 案例作结构参考时，必须标注"往届公开报道，不是你的课程事实"。
2. `video-planner`（视频策划）：把 2–3 个创新点映射成 45 分钟课堂时刻表（时间段/教学环节/AI介入动作/要拍到的学生行为证据/对应报告举措）。引用 `ai_playbook.json` 打法时标注来源专家并说明"迁移建议非专家本人观点"。
前端不用改（动态渲染），但要确认新顾问在下拉里可选。

### 任务 3：加 generate 生成模式（main.go）
- 在 chat 请求结构加 `mode:"generate"` 字段；在 `main.go:198` 审阅分支之后增加生成模式分支，输出结构化 JSON：章节骨架 / 字数预算 / 每节必答问题 / 待补证据清单 / 6 维对照表。
- **硬性红线：只输出"用户需要补充什么证据"的清单，绝不替用户编造学生人数、前后测、提升率、学生原话、第三方采用。** 这一条违反就是失败。
- 保持现有输出纪律：结构化分支不加 Markdown 围栏/前言，字段完整；未提供课程事实时不得声称已阅读用户材料（沿用 main.go 末尾 systemText 边界语句）。

### 任务 4：测试与交付
- 保留 v0.4.8 及之前所有能力（8 顾问、7 工作区、三个 checker 审阅行为不变），新功能是加法。
- 跑 `go test ./...`，为 architect / video-planner / generate 模式补测试（参考 `specialists_test.go`、`context_pipeline_test.go` 风格）。
- **开独立分支**（如 `feat/ai-track-builder`），不要直接提交 main；完成后给我 PR 链接。
- 交付时列出：改了哪些文件、新顾问 ID、四个 JSON 如何 embed 与检索、测试命令与结果。做不到的步骤说明原因，不要假装完成。

### 自检
交付前对照 `research/如何验收Codex的改造.md` 逐条自测。注意：验收脚本 `verify_codex_work.py` 检查的是"当前分支非 main"，所以你要在 feat 分支上、或告诉我分支名让我切过去跑。

## ===== 复制结束 =====

---

## 给你的补充说明（不用发给 Codex）

### 为什么这份和上一份不同
- 上一份《给Codex的提示词.md》是"从零改造"任务书，Codex 显然把它当成背景资料合并了，没当任务执行
- 这一份开头直接点明"你发了 v0.4.8 但改造一项没做"，用验收脚本的具体结果（0 PASS / FAIL 项）形成压力，并明确"v0.4.8 功能保留、不回退"，避免它推翻现有版本
- 补了两个它上次可能没注意的细节：前端顾问是动态渲染不用改（减少它乱改前端）、四个 JSON 而非三个（新增了视频索引）

### 我的 MD 改了哪些
- 新建了本文件《给Codex的返工指令_可直接粘贴.md》
- 《给Codex的提示词.md》《如何验收Codex的改造.md》《资料总览与Codex使用说明.md》**内容不用改**——它们是正确的，只是 Codex 没执行。不过我把最新的验收结果（0 PASS）记进了工作日志备查。
- 这份返工指令要不要传 GitHub？建议传——这样 Codex 在仓库里能直接看到，且和你上传的验收标准放一起形成闭环。你确认我就传。
