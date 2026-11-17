# v0.5.0 阶段0与阶段1自查

日期：2026-11-17T08:51:52.229510

基线 main=c282b75。Go/Node在main执行；verify脚本要求非main，因此在同一提交新建feat/v050-ai-deep执行，无代码变更。来源抽查核对仓库基准及独立核验报告，不冒称重新观看视频或重新核验全部外链。

| 项目 | 结果 | 证据位置 |
|---|---|---|
|阶段0 / 合并与文件|PASS|stage0-reverified.json; main=c282b75，PR1 merged|
|李静 / 使命、映射、质疑AI、短板|PASS|research/source-evidence/03_PPT设计案例库.md 案例06/14/20；02:122；07:197|
|宋晓菲 / 三阶段、黑帽、双联证据、三层成效|PASS|02:125；04§九；03案例17（20家单位）、12；05:86|
|于歆杰 / 三基准、分类治理|PASS|04§三及§九23:51–26:11|
|汪琼 / 思维实验室、归属|PASS|02:152；04§九23:00–24:21|
|刘学莘 / 生成问题、反乱炖、数据|PASS|01:59/80/82/128；个人评审经验并非官方条文|
|7条第六届报道 / 来源与性质|PASS|knowledge/ai_track_deep.json sixth_edition_reports；对照独立核验报告A0/A。李静机制另有practice_source_url；石拓/高越/傅毅transcribed仅转述；高越不展开维度|
|禁止虚构用户成效|PASS|ai_track_deep全文检查：现有数字均归属案例或讲者统计，未归到用户课程|
|5项撤回保留|PASS|builder.go、builder_policy.go、builtin_references.go；TestBuilderAuditRetractionRegressions|
|Go全量回归|PASS|stage1-go.log: ok lijing-assistant 8.358s|
|builder/revise Node|PASS|stage1-builder.log / stage1-revise.log: PASS|
|静态与编译验收|PASS|stage1-verify.log: 13 PASS / 0 FAIL / 0 RED FLAG|
