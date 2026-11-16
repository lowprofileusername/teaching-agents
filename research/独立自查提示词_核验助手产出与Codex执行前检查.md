# 独立自查提示词：核验助手产出（含 Codex 执行前自查）

> 用途一：交给**另一个独立 AGENT**，让它从 GitHub 拉取后，核验助手（我）刚写的文档与 JSON 是不是瞎编的。
> 用途二：交给 **Codex**，要求它在执行 AI 赛道深度融入任务**之前**先自查一遍，确认无误再动手。
> 仓库：`https://github.com/lowprofileusername/teaching-agents`（读 main 分支即可，无需写代码）

---

## ===== 提示词 A：给独立 AGENT（自查助手产出）=====

你是一个独立核验员。请 clone 或 pull `https://github.com/lowprofileusername/teaching-agents` 的 main 分支，**只做核查，不要修改任何文件**。目标是判断助手刚写的资料是否"有迹可循"，还是自己编的。

### 核验对象

1. `knowledge/ai_track_deep.json`
2. `research/给Codex的AI赛道深度融入任务书.md`
3. `research/AI赛道深度资料_两场冠军课程与第六届案例.md`
4. `research/AI赛道专题资料_第六届获奖案例与36场专家方法.md`
5. `research/给Codex的撤回与修正指令_事实性断言核查.md`
6. `research/README.md`、`research/资料总览与Codex使用说明.md` 中与上述资料相关的条目
7. 分支 `feat/ai-track-builder` 上的 builder 代码：`builder.go`、`builder_policy.go`、`builtin.go`、`builtin_references.go`、`specialists.go`、`generate_test.go`、`ai_track_test.go`（用 `git show origin/feat/ai-track-builder:<path>` 读取）

### 可用依据（都在仓库里）

- `research/source-evidence/` 下有 7 个原始依据文件（00 总账 / 01 评委判据 / 02 方法库 / 03 PPT案例库 / 04 迁移清单 / 05 答辩 / 07 证据核验）。所有 `source_locator` 都应能在这里回查。
- `knowledge/rules_7_ai_2026.json`（含官方来源 URL）、`research/third-party/第七届河南_省赛通知及评分标准.pdf`（官方评分原文）。
- 第六届案例的 `source_url` 是外部链接，可用 WebFetch 打开核验。

### 逐项核查清单（每项给 PASS / FAIL / 无法核验 + 证据）

**A. JSON 数据真实性**
1. `priority_sessions` 是否确为李静、宋晓菲两场？到 `source-evidence/00_视频样本总账.md` 查 `lijing` 与 `songxiaofei` 两条：李静是否标注"人工智能"赛道、6.6min、L2、证据C；宋晓菲是否 25.8min、L2、证据B、引用需标注。若对不上＝FAIL。
2. 对 `priority_sessions` 和 `support_sessions` 里**每一条 move**，按它的 `source_locator` 到 `source-evidence/` 对应文件里找原文：
   - 李静"让流场看得见"→ 03 文件案例06；"四词骨架"→案例07；"一页总装映射页"→案例14；"质疑AI正式环节"→02 文件 §六A组李静行；"五步流程图人机分工"→案例20；"证据偏少/百分比零命中"→02 与 07 文件。
   - 宋晓菲"行业流程比喻"→02 文件 §六A组宋晓菲行；"黑帽环节"→04 文件 §九 + 03 文件案例16；"低成本工具/可辅可练可验"→04 文件 §九 + 03 案例08/23；"自评×第三方双联证据"→03 案例17 + 07 文件 P30；"成效三层叙事/学生原话"→03 案例09/12；"诚实局限"→05 文件。
   - 于歆杰三基准/分类治理→04 文件 §三/§九；汪琼思维实验室→02 文件 §六C组、作业归属→04 文件 §九；刘学莘判据→01 文件。
   - **找不到原文或描述对不上的，逐条列出判 FAIL。**
3. `sixth_edition_reports` 7 条：每条的 `source_url` 用 WebFetch 打开，核对 `practice`/`quote` 是否真的出现在页面里；核对 `verification` 标注是否合理（校方官网/主流媒体可 verified，第三方公众号转述应 transcribed）。**特别注意：高越的"三维痛点定位法/AI适配度象限"页面是否只提名称、没给具体维度**——如果 JSON 里编造了具体维度＝FAIL。
4. 检查 JSON 里有没有凭空出现的成效数字（学生人数、提升率、分数、获奖归因到课程）——有＝FAIL。

**B. 文档与 JSON 一致性**
5. `AI赛道深度资料_两场冠军课程与第六届案例.md` 的每一条，是否能在 `ai_track_deep.json` 里找到对应项，且与 source-evidence 一致？有无文档写了但 JSON 没有、或两者矛盾？
6. 任务书里"优先引用李静/宋晓菲、于歆杰等作支撑"的说法，是否与 JSON 的 `priority_sessions`/`support_sessions` 结构一致？
7. 撤回指令里说的 5 处问题（独立伦理章、说课三段phase、每环节四项标注、案例无出处、45分钟硬编码），是否与 `feat/ai-track-builder` 分支源码实际情况相符？逐条到源码里找证据：
   - `builder.go` 里是否真有 `(policy.AI && !ethics)` 硬校验？
   - 是否真有 `t.End > 45` / `p.Duration != 45`？
   - 说课 phase 是否真强制三段？
   - `ai_award_cases_6th.json` 10 条是否真无 `source_url`？
   - 官方评分 .doc（写 URL，或用河南PDF）里是否真没有"独立伦理章节"要求？
   - **如果撤回指令描述的源码问题根本不存在，说明助手在编造问题＝FAIL。**

**C. "两场主角"判断是否武断**
8. 助手先说是于歆杰/汪琼、后按用户纠正改为李静/宋晓菲。请独立判断：单看 `source-evidence/00_视频样本总账.md`，李静和宋晓菲是否确实属于 AI 赛道/AI 一等奖得主？于歆杰、汪琼是否主旨报告而非参赛课程实例？据此判断现在的结论是否合理，还是为了迎合用户改口。

### 输出格式

| 项 | 结论 | 证据（文件路径+原文片段） |
|---|---|---|

最后给一句总判定：**"助手产出全部有据可查" / "发现 N 处瞎编或无法核验"**，并逐条列出问题。不要客气，发现问题就明确指出。

## ===== 提示词 A 结束 =====

---

## ===== 提示词 B：给 Codex（执行前自查）=====

在执行 `research/给Codex的AI赛道深度融入任务书.md` **之前**，先做下面自查，把结果贴给我，确认后才开始改代码：

1. pull 最新 main，读 `knowledge/ai_track_deep.json` 和 `research/source-evidence/` 下 7 个依据文件。
2. 逐条核对 `priority_sessions`（李静、宋晓菲）和 `support_sessions`（于歆杰、汪琼、刘学莘）里每个 move 的 `source_locator`，是否能在 source-evidence 对应文件里找到原文。**任何一条找不到，停下来告诉我，不要继续，也不要自己"合理补写"。**
3. 核对 `sixth_edition_reports` 7 条的 `source_url` 与 `verification`：`transcribed` 的不得当原话；高越的"AI适配度象限"不得编造具体维度。
4. 核对 `ai_track_deep.json` 里没有任何编造的成效数字（人数/提升率/分数/获奖归因）。
5. 确认已执行 `research/给Codex的撤回与修正指令_事实性断言核查.md` 的 5 项撤回：独立伦理章改有落点即可、说课课前课后为硬要求、标注只覆盖有AI动作环节、案例补 verified/source_url、时长改 40–50 分钟。
6. 把自查结果按"PASS/FAIL + 证据位置"列成表发给我。**有任何 FAIL，先报告，等我确认再动手；全部 PASS 才开始接线。**

## ===== 提示词 B 结束 =====

---

## 补充说明（不用发给 AGENT）

- 我上传 `research/source-evidence/` 是因为发现 `source_locator` 指向的原始文件原本都不在 GitHub，独立 AGENT 无从核验——这是"有迹可循"的必要前提。
- 自查设计成"独立 AGENT 查助手 + Codex 执行前自查"双保险，且不修改文件，避免自查过程污染数据。
- 第 8 项是专门测"是否为了迎合用户改口而动摇事实判断"——如果独立核验认为李静/宋晓菲确实更匹配 AI 赛道，说明这次更正是对的；如果认为于歆杰/汪琼也合理，那只是分类口径不同，不算瞎编，但要在结论里说清。
