export const isBuilder=id=>['architect','video-planner'].includes(id);
const text=s=>String(s??'').replace(/[<>]/g,'').replace(/\|/g,'／').replace(/\r?\n/g,' ');
const list=a=>(a||[]).map(s=>'- '+text(s)).join('\n');
export function renderBuilderPlan(p){
 const lines=['## '+(p.kind==='report'?'报告架构方案':'课堂视频策划'),text(p.boundary),'','### 一句话创新主线',text(p.innovation_line),'','### 逻辑闭环',text(p.logic_loop),''];
 if(p.kind==='report'){
  lines.push('### 章节骨架与字数预算',`预算合计：${p.word_budget_total} 字（计划值）`,'');
  for(const s of p.sections)lines.push(`#### ${text(s.title)} · ${s.word_budget} 字`,'必答问题：',list(s.questions),'','待补证据：',list(s.evidence_needed),'');
 }else{
  lines.push('### 课堂时刻表',`总时长：${p.duration_minutes} 分钟（本次策划参数）`,'','| 时间段 | 教学环节 | AI介入动作 | 要拍到的学生行为证据 | 对应报告举措 |','| --- | --- | --- | --- | --- |');
  for(const t of p.timeline)lines.push(`| ${t.start_minute}–${t.end_minute}分 | ${text(t.stage)} | ${text(t.ai_action)} | ${text(t.student_evidence)} | ${text(t.report_measure)} |`);
 }
 const positions=new Map([...(p.sections||[]).map(s=>[s.id,s.title]),...(p.timeline||[]).map(t=>[t.id,`${t.start_minute}–${t.end_minute}分 ${t.stage}`])]);
 lines.push('','### 六维评分对照',p.boundary?.includes('第六届六维')?'第六届规则，仅作往届参考。':'省赛校赛口径汇总，非国赛正式标准。','','| 维度 | 对应位置 | 待补证据 |','| --- | --- | --- |');
 for(const s of p.scoring_map)lines.push(`| ${text(s.dimension)} | ${text(s.section_ids.map(id=>positions.get(id)||id).join('；'))} | ${text(s.evidence_needed.join('；'))} |`);
 lines.push('','### 需要补充的证据',list(p.evidence_needed),'','### 资料依据与使用边界');
 for(const r of p.references)lines.push(`- ${text(r.source_id)}：${text(r.use)}。${text(r.notice)}`);
 return lines.join('\n');
}
