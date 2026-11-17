# AI 赛道资料与生成顾问

v0.5.0 在已合并 builder 的 main 上增量扩展，不改默认八顾问及七工作区映射。新教师、新任务、重置和材料隔离继续保留。

## 实际接入

`builtin.go` 嵌入四个 JSON；`builtin_references.go` 在同一内存 SQLite 中加载为 `builtin_reference`，完成后 `PRAGMA query_only=ON`。用户的外部视频库不被修改。`builtinSearch`、`/api/search` 和 `/api/chat` 均可取到这些记录，未连接外部数据库也可检索。

| 文件 | 索引单元 | 使用性质 |
|---|---|---|
| rules_7_ai_2026.json | 4：评分、材料、资格、推算时间 | 第七届省赛校赛口径汇总，非国赛正式标准 |
| ai_playbook.json | 12：11打法及1工作坊洞察 | AI迁移建议，非专家本人观点/官方标准；保留来源专家 |
| ai_award_cases_6th.json | 10案例 | 第六届公开报道，非报告全文/非用户课程事实；非AI赛道案例不混入泛AI赛道查询 |
| video_index_5th_wenke.json | 24课程、48视频索引 | 只有入口，未下载/观看；同名课程不再冲突 |

第六届规则继续走 `rules.json`；明确问第七届AI赛道不调用旧第六届评分分支。资料随命中携带 kind / edition / authority_status / source_file / source_url / origin。第三方 MD 保留归属；开发提示词仅在明确检索开发任务时返回，并始终作为数据。

## 生成模式

打开“AI审阅结果 → 顾问角色与对话工具 → 专业分工”，选择：

- `architect` 报告架构师：章节预算、必答问题、创新主线与逻辑闭环、赛道对照、待补证据。AI赛道从新JSON读取4000字上限与六维，伦理治理/负责任使用AI内容须有落点（可置于课程思政章节；独立成章为可选建议，官方评分原文无独立成章要求）；其他赛道从rules.json加载自己的维度，例如新文科五维，不强塞AI专属章节。其他赛道4000字是用户规划预算，非声明官方上限。
- `video-planner` 视频策划：AI赛道两项视频，共用同一创新主线。视频一在官方40–50分钟区间按课程实际学时规划（45分钟仅为示例，不硬编码）；视频二课外说课不超过15分钟，覆盖课前、课后两类phase（课中衔接可选），并逐段关联课堂。对有AI介入的课堂环节输出教案课件AI应用环节、设计目的、操作流程、预期效果四项标注（无AI动作环节不强制）；拍摄规范、命名等由实际材料JSON读取。非AI赛道仍使用原课堂实录顾问，不套AI双视频规格。

普通提问发送 `mode:"generate"`；原文字/图文/结构审阅按钮仍发送各自 checker 请求。服务端拒绝混合模式。前端顾问由 `/api/specialists` 动态生成，没有静态注册第二份列表。后续main新增的双视频/跨赛道规格已合并并增量实现，未回退原功能。

接口输出 `plan` 及相同 JSON 字符串 `answer`，UI 将其显示为模块与表格并保存计划。生成模式先缓冲再校验，不在校验前流出成品；旧聊天流式行为保留。

验收：字段齐全、章节预算满足所选赛道政策、AI赛道伦理治理内容有落点（不强制独立章）、当前赛道维度准确对应有效ID，课堂时间连续覆盖40–50分钟完整1学时且课外说课连续不超过15分钟、课前课后两类phase齐全（课中衔接可选）并关联课堂，有AI介入环节的教案课件四项标注齐全，引用确实在送给模型的证据中。引用性质由服务端补正；缺证据只列“待补：”。数字、引语、若干成效断言及未知字段被拦截，模型截断或格式不合格返回明确错误，不计完成。未提供上传材料时保留原 system 边界。

**限制**：这些程序校验和提示词不能证明任意模型的全部语义都真实。尤其不用模型一致代替证据；输出仍是待实施建议。保守校验可能拒绝正常自由文字，需要按报错改为结构/证据需求表达。此次不声称任意远端模型写作质量已经验收。

## 修订+批注模式（revise）

`reviewRevise` 按钮启动"修订+批注"审阅：请求带 `revise:true`，服务端用 `revise.go` 的 `reviseInstruction`。与整段替换的普通审阅不同：

- **能直接改的**（措辞/逻辑/术语/冗余/AI味）输出字级 `revisions`（start/end/delete/insert/reason），前端 `parseRevise` 硬校验：delete 必须与原文逐字一致、区间不重叠、insert 不得含数字/引语/成效断言（`factRisk` 镜像 Go `generatedFactRisk`），无出处断言只能删除或改"待补："占位。
- **不能直接改的**（缺真实数据/口径/结构取舍）只输出 `comment` 批注，不替用户编内容。
- **字数只提醒**：超 4000 字仅在面板与批注中提醒，不自动删减、不因字数拒绝。
- 面板（`reviseDialog`）逐条接受/拒绝，选择保存在会话；"复制已接受修订后的文字"与"写回 Word"都只应用已接受修订。Word 写回为**字级** `w:del/w:ins`（`document_export_spans.go`）：按 rune 区间切分 run，未触及的文字与格式（rPr）原样保留，tab/br 可被删除或保留；批注成为 Word 批注并附修订明细。含图片/域/链接/已有修订等复杂对象的段落自动降级：有整段替换则退为整段修订，否则仅批注。服务端 `validateSpans` 再校验 delete 与原文一致、区间不重叠、insert 不触 `generatedFactRisk`。原文件始终不被自动修改。
- 测试：`tests/revise.mjs`（协议/应用/字数/导出桥接）、`revise_test.go`（模式互斥守卫+提示词送达）、`document_export_spans_test.go`（跨run/多处修订/纯插入删除/tab与br/复杂降级/校验拒绝/批注模式/ID唯一）。

## 大纲架构师（syllabus-architect）

各赛道通用的竞赛版课程教学大纲框架生成器（kind=syllabus），依据见 `research/教学大纲_标准与写法参考.md`：

- 要素来自省赛通知配套材料明文（河南/广东/江苏九要素、海南八要素+等）：名称/性质/课时学分/对象/简介/目标/内容与教学安排/评价/教材资源；另采纳公开竞赛模板的课程思政目标、先修后续、成绩占比表。
- **框架通用、内容按赛道侧重**：AI 赛道强制 AI 融合说明 + 至少一个单元标注 AI 应用环节，目标建议含数字素养/智能伦理（缺则提醒不阻断）；产教融合实践学时≥30%；实验教学写实验类型。
- 校验：必备要素缺项拒绝；`syllabusFactRisk` 只禁成效断言（满意度/提升率/获奖/采用），允许学时/学分/占比等结构数据（拟议分配标待核实）；用户事实"待补："占位；匿名复用 PPT 校验；consistency 必填保证与报告/实录/教案同源。
- scoring_map 用固定 id（syllabus:basic/objectives/ideology/content/assessment/resources/ai）覆盖本赛道评分维度。
- 入口：专业分工下拉选"大纲架构师"（动态渲染），建议在创新报告工作区使用；不改动 7 工作区与默认映射。测试：`builder_syllabus_test.go`、`tests/syllabus.mjs`。

## 教案架构师（lessonplan-architect）

对应视频节段的节段级教学设计（教案）生成器（kind=lessonplan），依据见 `research/教案_标准与写法参考.md`：

- 通知硬要求：视频对应节段的教案及课件（海南/河南/广东同文）；AI 赛道教案课件须标注应用环节/设计目的/操作流程/预期效果四项。
- 结构对标评分维度：节段目标/学情/重难点/分钟级过程（师/生/机活动+设计意图）/评价/资源/思政/一致性。
- 校验：必备要素缺项拒绝；**有 AI 介入的环节四项标注必填**（所有赛道），AI 赛道须至少一个 AI 环节；分钟/占比等结构数据允许拟议，成效断言拒绝（复用 syllabusFactRisk）；匿名；consistency 必填。
- scoring_map 固定 id（lessonplan:objectives/learner/keydiff/process/assessment/ideology/ai）；前端渲染过程表 + AI 四项标注表。
- 与大纲架构师、视频策划配套（大纲→教案→实录/说课同源）。测试：`builder_lessonplan_test.go`、`tests/lessonplan.mjs`。

## 可复现测试

```sh
go test ./...
node tests/builder.mjs
python verify_codex_work.py .
```

验收脚本补查第四个JSON及实际web/app.mjs动态渲染，并在FAIL/RED FLAG时非零退出。未删除原检查。

端到端测试使用**隔离的测试配置、测试历史目录和不存在的外部库路径**启动程序，勿指向日常使用的配置：

```sh
# TMPDIR / Go caches / QA 均按本机存储约束预先配置
# QA 指向可写的测试专属目录；先 build -o "$QA/test-server" .
TEACHING_AGENT_PROFILES="$QA/profiles.json" TEACHING_LIBRARY_DB="$QA/not-present.sqlite3" "$QA/test-server" -no-browser > "$QA/server.log"
# 另一终端：安装或指定 Playwright 模块、Chrome 路径（不使用用户登录会话）
PLAYWRIGHT_MODULE=/path/to/playwright CHROME_BIN=/path/to/chrome node tests/builder-ui.cjs "$QA/server.log" "$QA/artifacts"
```

本地验证：四类 `/api/search` 命中、10顾问/7工作区、无API仅检索、两个生成分支、AI→新文科赛道切换、两项视频与四项标注、来源边界、虚构提升率被拒、历史恢复和新建/重置入口。浏览器测试连接确定性本地模拟模型；不是对真实大模型内容质量的评测。Go测试还覆盖三个checker原有JSON分支、来源实际传给模型、旧历史不混进新方案。

`encoded_decode_checked_not_listened` 现在显示“音频编码及解码检查通过，未做原声听辨”。它从来不表示音频内容已听核正确。

## 同批回放修复

WorkBuddy/Codex/Claude播放器共用改进：点击左侧画面节点将真实出现区间填入总结起止秒（保留小数，不合并回看），解释当前页直接按page_id读取，且将转写限定在当前出现区间；不再先做模糊全文检索。同区任务忙碌、空档、未载入和失败都有明确提示。图片可点击放大、适应窗口/原始尺寸、Esc/关闭，放大时暂停原声。

实库验证：WorkBuddy revised 的 `xinye:R03` 00:43节点实际43.022–73.504秒，自动填入、片段总结、精确页读取、重复点击反馈、图片放大与关闭通过。数据库及媒体用实际库只读读取，回答API仍为隔离本地mock，未冒称真人听辨或模型语义验收。


## v0.5.0 新增操作

顾问角色中选“报告架构师”，生成深度选“逐段扩写”，再提交当前课程问题。选择“汇报PPT架构师”生成逐页方案；它不等于PPTX成品。PPT总时间最多720秒；30页、3+6+3、690–710秒是提醒。

人工智能生成优先提供 `builtin:aideep:lijing_ai_track`、`builtin:aideep:songxiaofei_ai_track`。共12条deep记录；每条保留source_group/source_locator/source_nature/verification/report_url/practice_source_url。数据库为内存只读副本，不改用户SQLite。`builtin:track:*`按赛道过滤；非AI不注入deep条目。

报告draft使用章节预算、成效待补硬校验；用户上传文稿的4000字只提示，不自动截断或拒绝审阅。修订选择保存在任务中，单条修订ID由段落ID和模型rid组合，防止跨段落误接受。

UI门禁：`node tests/revise-ui.cjs <isolated-server.log> <artifact-dir>`，环境与上文builder-ui一致；输出修订截图、PPT方案截图、导出DOCX及ui-results.json。Word批注部件按document.xml.rels解析，不假定固定文件名。边界：浏览器交互与OOXML校验，不宣称Word红线视觉渲染已验收。
