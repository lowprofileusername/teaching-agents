# 给 Codex 的 AI 赛道深度融入任务书

> 用途：用户是第七届 AI 赛道选手。把已核实的最新资料（李静/宋晓菲两场 AI 赛道冠军课程实例 + 第六届获奖案例与复盘新闻）真正接入工作台，让 AI 赛道生成更专业。
> 前置：**先 pull 最新 main，并先执行 `research/给Codex的撤回与修正指令_事实性断言核查.md`**（该指令优先级最高，本任务书与之冲突处以撤回指令为准）。
> 数据我已经建好并校验：`knowledge/ai_track_deep.json`（2 场冠军课程实例 + 3 场方法支撑，共 19 个 move；7 条第六届报道，每条带出处）。**你不需要重写数据内容，只需接线。**

---

## ===== 从这里开始全部复制 =====

请先 pull 最新 main，并执行完 `research/给Codex的撤回与修正指令_事实性断言核查.md` 后，再做本任务。用户是第七届人工智能赛道选手，需要把 AI 赛道做成工作台的强项。我已建好 `knowledge/ai_track_deep.json`，分三层：`priority_sessions`＝**两场AI赛道冠军课程实例（李静《流体力学》、宋晓菲《酶工程》，来自用户36场库，最优先）**；`support_sessions`＝通用方法支撑（于歆杰、汪琼、刘学莘）；`sixth_edition_reports`＝第六届AI赛道获奖案例与复盘新闻。每条带 source_locator 或 source_url，JSON 已校验通过。**数据内容不要改，你只做下面四件事：**

### 任务 1：加入 embed 与只读检索

1. `builtin.go:16` 的 `//go:embed` 列表加入 `knowledge/ai_track_deep.json`。
2. 在 `builtin_references.go` 参照 `ai_playbook.json` 的加载方式：
   - `installBuiltinReferences` 里 `readReferenceJSON("knowledge/ai_track_deep.json", &deep)`
   - 为 `priority_sessions`（2 场：李静/宋晓菲）、`support_sessions`（3 场：于歆杰/汪琼/刘学莘）和 `sixth_edition_reports`（7 条）注册只读检索条目，ID 前缀用 `builtin:aideep:`（如 `builtin:aideep:lijing_ai_track`、`builtin:aideep:songxiaofei_ai_track`、`builtin:aideep:gao_yue_ai_fit`）
   - `Kind` 用 `ai_migration`；`Authority` 写明："专家方法＝36场迁移建议非官方标准；第六届案例＝公开报道转述非报告全文、非用户成效"
   - `SourceFile` = `knowledge/ai_track_deep.json`，`SourceURL` 用 `repoSource(...)`
   - 对 `sixth_edition_reports` 保留每条的 `source_url`、`source_nature`、`verification` 字段，供生成时标注出处
3. 确保 `builtinSearch` 能按检索词命中这些条目（当前搜索函数用 `queryAI(q)` 判断 AI 查询，AI 赛道问题应能命中）。

### 任务 2：AI 赛道生成时优先引用两场AI赛道冠军课程实例

在 `generationPrompt` 的 AI 赛道分支（`p.AI` 为真时）注入一段"深度资料引用要求"，仅对 architect / video-planner 生效：

1. 从 `ai_track_deep.json` 的 `priority_sessions`（**李静、宋晓菲两场AI赛道冠军课程实例**）中，按报告章节主题选 1–2 个最贴切的 move 作为"可选教学机制"，要求 `references` 至少引用一条 `builtin:aideep:` 条目。
2. **李静的'让流场看得见'使命句、一页总装映射页、四词骨架、质疑AI正式环节，宋晓菲的行业流程三阶段、黑帽环节、自评×第三方双联证据、成效三层叙事，优先级最高**——它们来自同为AI赛道的国赛一等奖得主本人，最可直接对标。注意李静那场'成效证据偏少'是短板，引用其机制时须提示用户补齐成效数据。
3. `support_sessions` 作为方法论支撑按需引用：于歆杰审查三基准/分类治理、汪琼思维实验室/作业归属判定对应 AI 必要性与治理；刘学莘赛制判据（生成问题、反乱炖、题目别用 AI 起、第六届 AI 命中率 33.33%）用于"问题导向/创新特色/反思"段和 PPT 记忆点，但**必须标注"个人评审经验，非官方评分条文"**。
4. 引用第六届案例时（如宋晓菲、李静、辛惠娟、石拓、高越、傅毅），必须带三重标注：
   - `source_nature`（如"第三方公众号转述""校方官网培训报道"）
   - "往届公开报道，不是你的课程事实"
   - verification 为 `transcribed` 的，不得写成获奖者逐字原话，只能写"据公开报道/复盘转述"
5. 高越的'三维痛点定位法''AI适配度象限'只被报道提及名称——**不得编造其具体维度或定义**，只能写"获奖者提及此法，细节见其本人分享"。
6. 与 `ai_playbook.json` 去重：同一条打法（如张静拒绝直答、辛页盲评）不要在两个库里重复引用。

### 任务 3：AI 赛道审阅顾问也能对照深度资料

`specialistPrompt` 里 `ai` 顾问（AI融合顾问）在 AI 赛道时，追加一条检查顺序："是否像李静/宋晓菲那样有可对标的具体机制（使命句、映射页、黑帽环节、双联证据）；是否体现学生判断力与核验留痕（可对照于歆杰审查三基准、汪琼思维实验室/作业归属）；是否规避 AI 空转（AI出题—AI作答—AI批改）；引用往届案例是否标明公开报道而非自身成效；是否只讲设计没给成效数据（李静场的已知短板）"。

### 任务 4：测试（保持原有全绿）

1. 扩充 `ai_track_test.go`：验证 `ai_track_deep.json` 已进入 embed 且能被 `builtinSearch("AI 审查三基准")` / `builtinSearch("第六届 人工智能 一等奖")` / `builtinSearch("李静 流体力学")` 命中；验证 2 场 priority + 3 场 support + 7 条报道的 ID 都能取到；验证出处字段齐全。
2. 扩充 `generate_test.go`：AI 赛道 architect 生成的 references 至少含一个 `builtin:aideep:` id；把赛道切成新文科后，不得出现 `builtin:aideep:` 的 AI 专属条目（深度库只在 AI 赛道注入）。
3. 反向用例：引用 `verification: transcribed` 案例时生成文本若写成"某某原话"应被拒；编造'AI适配度象限'具体维度应被拒或至少不出具体定义。
4. `go test ./...` 全绿，原 10 PASS、`TestBuilderTrackSwitchAndTwoVideoRequirements`、撤回指令新增的断言都不回退。

### 交付

开独立分支（如 `feat/ai-track-deep`），给我分支名/PR；列出改动文件、新增 embed 路径、新检索 ID 前缀、AI 赛道引用规则、测试命令与结果。做不到说明原因，不要假装完成。

## ===== 复制结束 =====

---

## 给你的补充说明（不用发给 Codex）

### 数据出处（可回查）

- **李静（第一优先）**：`00_视频样本总账.md｜lijing`（人工智能赛道，6.6min，L2，证据C）、`02_专家方法库_36场.md` §六A组（质疑AI正式环节；警惕：创新设计占时过高、成效证据偏少）、`03_PPT设计案例库.md` 案例06/07/13/14/19/20/21/22（含OCR实证）、`07_证据核验与六类误判.md`（百分比零命中场次）
- **宋晓菲（第一优先）**：`00_视频样本总账.md｜songxiaofei`（A组，25.8min，L2，证据B，引用需标注）、`02_专家方法库_36场.md` §六A组（行业流程比喻课前课中课后）、`03_PPT设计案例库.md` 案例01–05/08–12/15–18/23/24（含OCR实证）、`04_AI教学创新迁移清单.md` §九
- 于歆杰：`00_视频样本总账.md｜yuxinjie`（30.1 分钟，证据 B，可直接引用）、`04_AI教学创新迁移清单.md` §三/§九、`02_专家方法库_36场.md` §六C组
- 汪琼：`00_视频样本总账.md｜wangqiong`（26.1 分钟，引用需标注）、`04_AI教学创新迁移清单.md` §九、`02_专家方法库_36场.md` §六C组
- 刘学莘：`01_评委判据与赛制.md` 全文（带时间戳）
- 石拓/辛惠娟复盘：`http://mp.weixin.qq.com/s?__biz=MzkwMjIxMTU0NA==&mid=2247733026&idx=2&sn=3c38d85eedd11b19a1b75f548132c8b8`（第三方转述，非逐字）
- 辛惠娟给AI纠错：`http://www.jyb.cn/rmtzgjyb/202609/t20260909_2111518818.html`（中国教育新闻网）
- 高越三维痛点/AI适配度象限：`https://jwc.gzhmu.edu.cn/info/1159/15911.htm`（仅提名称，未展开，不得编造）
- 傅毅 AI 空转：`https://gjjmxy.lixin.edu.cn/szdw/szdwjs/159727.htm`
- 李晨瑜：`https://xhsysu.edu.cn/info/1233/16643.htm`（基础课程组，非 AI 赛道，引用须标明）
- 宋晓菲：`https://jwc.nbu.edu.cn/info/1151/18396.htm`（校方官网，verified）

### 为什么这样做

- **数据归数据，代码归代码**：JSON 里的出处由我逐条核实，Codex 只接线，避免它自己编内容。
- **延续撤回指令的纪律**：所有新增内容都带 source_nature 和 verification，transcribed 的明确不能写成原话——这正是上一轮出的错。
- **两场优先**：用户明确要求 AI 赛道优先分配的是**李静、宋晓菲两场AI赛道冠军课程实例**，所以它们放在 `priority_sessions` 最前，生成时优先级最高；于歆杰/汪琼/刘学莘降为 `support_sessions` 方法论支撑。（此前误记为于歆杰、汪琼，已按用户纠正改回。）
