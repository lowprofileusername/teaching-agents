export const tasks=[
 {id:'overview',title:'备赛概览',subtitle:'从真实教学问题出发，确定材料的论证主线。',action:'请根据我的备赛信息，列出当前最需要明确的教学问题、已有证据和下一步材料准备顺序。不要编造未提供的课程情况。'},
 {id:'report',title:'创新报告',subtitle:'按目录和正文分章，检查问题、举措与成效的对应关系。',action:'请审阅本次提供的报告正文。先给最重要的修改意见，再按原段落列出原文、具体问题、修改理由、建议替换文字。不要把目录当正文，不要新增未经证实的数据。'},
 {id:'lesson',title:'课堂实录',subtitle:'定位真实发生的教学行为，检查创新是否落实到学生学习。',action:'请按所选赛道的课堂实录参考维度审阅本次提供的材料。按时间段、教师行为、学生行为、评价反馈、问题、改进建议组织。转写无法证明的视觉/收音/全班参与维度标为证据不足，不虚构时间点，不因抽样判整课合格。'},
 {id:'presentation',title:'汇报与答辩',subtitle:'把报告压成清楚的证据叙事，准备评委追问。',action:'请审阅本次提供的汇报材料：逐页检查主张与证据、叙事顺序、信息量、讲述时间，并给可替换标题与可能的评委追问。没有画面不能判断版式、配色、字号。'},
 {id:'consistency',title:'综合核查',subtitle:'把报告、课堂实录与PPT放在同一条证据链上。',action:'请建立痛点—目标—举措—学生产物—评价—成效的对应表，并检查本次已提供材料之间的课程名称、样本、数字和实施过程是否一致。未提供的材料标未对照，不推定矛盾或一致。'},
 {id:'replay',title:'专家讲座',subtitle:'检索专家与课程目录，回看讲座原声、画面与转写，并就讲座提问。',action:'请基于资料库查找与我的课程问题有关的专家案例，解释可借鉴的方法、实施条件及不适用情形；或基于本轮提供的讲座片段解答问题，不冒称已听核原声。'}
];
export function taskByID(id){return tasks.find(x=>x.id===id)||tasks[0]}
export function rubricFor(catalog,track,task){const g=catalog?.groups.find(g=>g.tracks.includes(track));return g?{group:g,part:g.parts[task]||null}:null}
export function reviewContext({edition='7',track='',course='',task='overview'},catalog){
 const t=taskByID(task),rubric=rubricFor(catalog,track,task);
 return {task:t.title,edition,track:track||'未选赛道',course,rule_status:catalog?.future_note||'相应届次正式规则尚未提供',reference:rubric?{edition:6,source_url:catalog.source.url,publisher:catalog.source.publisher,focus:rubric.group.focus,part:rubric.part,video_requirements:task==='lesson'?rubric.group.video_requirements:undefined}:null};
}
export function taskQuestion(id){return taskByID(id).action}
