export const isBuilder=id=>['architect','video-planner','presentation-architect','syllabus-architect'].includes(id);
const text=s=>String(s??'').replace(/[<>]/g,'').replace(/\|/g,'／').replace(/\r?\n/g,' ');
const list=a=>(a||[]).map(s=>'- '+text(s)).join('\n');
export function renderBuilderPlan(p){
 const lines=['## '+(p.kind==='report'?'报告架构方案':p.kind==='presentation'?'汇报PPT方案':p.kind==='syllabus'?'课程教学大纲方案':'课堂视频策划'),text(p.boundary),'','### 一句话创新主线',text(p.innovation_line),'','### 逻辑闭环',text(p.logic_loop),''];
 if(p.kind==='report'){
  lines.push('### 章节骨架与字数预算',`预算合计：${p.word_budget_total} 字（计划值）`,'');
  for(const s of p.sections){lines.push(`#### ${text(s.title)} · ${s.word_budget} 字`,'必答问题：',list(s.questions),'','待补证据：',list(s.evidence_needed),'');if(s.draft)lines.push('正文草稿：',s.draft,'');}
 }else if(p.kind==='presentation'){
  lines.push('### 逐页方案',`总时长：${p.slides.reduce((n,s)=>n+s.duration_sec,0)} 秒`);
  for(const s of p.slides)lines.push('',`#### 第 ${s.page_no} 页 · ${text(s.title)}`,`环节：${text(s.section)} · ${s.duration_sec} 秒`,list(s.bullets),'可视化：'+text(s.visual),'讲稿：'+text(s.speaker_note),'资料：'+text(s.evidence_source));
  if(p.warnings?.length)lines.push('','### 建议提醒（不阻断）',list(p.warnings));
 }else if(p.kind==='syllabus'&&p.syllabus){
  const y=p.syllabus;
  lines.push('### 基本信息','','| 要素 | 内容 |','| --- | --- |',`| 课程名称 | ${text(y.course_name)} |`,`| 课程性质 | ${text(y.course_nature)} |`,`| 学分/学时 | ${text(y.credits_hours)} |`,`| 授课对象 | ${text(y.audience)} |`);
  if(y.prerequisites)lines.push(`| 先修/后续课程 | ${text(y.prerequisites)} |`);
  lines.push('','### 课程简介',text(y.overview),'','### 课程目标',list(y.objectives),'','### 课程思政目标',text(y.ideology_politics),'','### 教学内容与学时安排','','| 单元 | 学时 | 重难点 | AI应用环节 |','| --- | --- | --- | --- |');
  for(const u of y.content_schedule||[])lines.push(`| ${text(u.unit)} | ${text(u.hours)} | ${text(u.key_points)} | ${text(u.ai_stage||'—')} |`);
  lines.push('','### 课程评价与成绩构成',text(y.assessment),'','### 教材与教学资源',list(y.resources));
  if(y.ai_integration)lines.push('','### AI融合说明',text(y.ai_integration));
  lines.push('','### 与报告/实录/教案的一致性',text(y.consistency));
  if(p.warnings?.length)lines.push('','### 建议提醒（不阻断）',list(p.warnings));
 }else{
  lines.push('### 课堂时刻表',`总时长：${p.duration_minutes} 分钟（本次策划参数）`,'','| 时间段 | 教学环节 | AI介入动作 | 要拍到的学生行为证据 | 对应报告举措 |','| --- | --- | --- | --- | --- |');
  for(const t of p.timeline)lines.push(`| ${t.start_minute}–${t.end_minute}分 | ${text(t.stage)} | ${text(t.ai_action)} | ${text(t.student_evidence)} | ${text(t.report_measure)} |`);
 }
 if(p.kind==='lesson'){
  lines.push('','### 视频二：课外教学展示说课',`计划时长：${p.demo_duration_minutes} 分钟（上限十五分钟）`,'与课堂实录使用同一创新主线；以下是拟议脚本与拟采集证据。');
  for(const d of p.demo_script||[])lines.push('',`#### ${d.start_minute}–${d.end_minute}分 · ${text(d.phase)}`,text(d.narration),'','AI动作：'+text(d.ai_action),'','拟展示的界面或学生产物：'+text(d.student_evidence),'','衔接课堂环节：'+text(d.class_segment_id));
  lines.push('','### 配套教案与课件四项标注','','| AI应用环节 | 设计目的 | 操作流程 | 预期效果 |','| --- | --- | --- | --- |');
  for(const a of p.lesson_annotations||[])lines.push(`| ${text(a.segment_id)} | ${text(a.purpose)} | ${text(a.operation)} | ${text(a.expected_effect)} |`);
 }
 if(p.requirements){
  const r=p.requirements;lines.push('','### 材料规格与共同约束',text(r.notice));
  if(r.report)lines.push('',`报告：字数上限 ${r.report.word_limit} 字，查重不高于 ${r.report.similarity_limit_percent}%，${r.report.anonymous?'匿名提交':'按所给要求提交'}；格式 ${text(r.report.format)}。`);
  if(p.kind==='lesson'&&r.video)for(const [key,label] of [['lesson_record','课堂实录'],['extra_demo','课外说课'],['lesson_plan_annotation','教案与课件'],['naming','两项视频命名']])lines.push('',label+'：'+text(r.video[key]));
 }
 const positions=new Map([...(p.slides||[]).map(s=>[s.id,`第${s.page_no}页 ${s.title}`]),...(p.sections||[]).map(s=>[s.id,s.title]),...(p.timeline||[]).map(t=>[t.id,`${t.start_minute}–${t.end_minute}分 ${t.stage}`]),...(p.syllabus?[['syllabus:basic','基本信息'],['syllabus:objectives','课程目标'],['syllabus:ideology','思政目标'],['syllabus:content','内容与学时'],['syllabus:assessment','课程评价'],['syllabus:resources','教材资源'],['syllabus:ai','AI融合']]:[])]);
 lines.push('',`### ${p.scoring_map.length}维评分对照`,text(p.requirements?.notice||p.boundary),'','| 维度 | 对应位置 | 待补证据 |','| --- | --- | --- |');
 for(const s of p.scoring_map)lines.push(`| ${text(s.dimension)} | ${text(s.section_ids.map(id=>positions.get(id)||id).join('；'))} | ${text(s.evidence_needed.join('；'))} |`);
 lines.push('','### 需要补充的证据',list(p.evidence_needed),'','### 资料依据与使用边界');
 for(const r of p.references)lines.push(`- ${text(r.source_id)}：${text(r.use)}。${text(r.notice)}`);
 return lines.join('\n');
}
