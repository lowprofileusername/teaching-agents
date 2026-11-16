import {intervalAt,transcriptVersions} from './lecture-timeline.mjs';
import {timecode} from './retrieval.mjs';
export function lectureQuestion(q,context,mode='auto',range=null){
 if(!context?.video)return{error:'请先选择要回看的讲座，随后可直接问“这场讲座讲了什么”。'};
 const {video,entries,utterances,current=0}=context;
 let selected=mode==='auto'?(/这页|当前页|这一页|刚才|这段|这一段|现在/.test(q)?'current':'whole'):mode;
 const explicit=q.match(/(\d{1,2}:\d{2}(?::\d{2})?)\s*(?:到|至|[-—–])\s*(\d{1,2}:\d{2}(?::\d{2})?)/);const seconds=s=>s.split(':').reduce((a,x)=>a*60+Number(x),0);
 if(explicit){range=[seconds(explicit[1]),seconds(explicit[2])];selected='range'}
 const page=intervalAt(entries,current).entry;let t0=0,t1=Number(video.duration);
 if(selected==='current'){t0=page?.t0??Math.max(0,current-30);t1=page?.t1??Math.min(t1,current+30)}
 if(selected==='range'){if(!range||range[0]<0||range[1]<=range[0]||range[1]>t1)return{error:'请输入讲座时长内的有效起止时间。'};[t0,t1]=range}
 const version=context.version||transcriptVersions(utterances)[0],rows=utterances.filter(u=>u.version===version&&u.t0<t1&&u.t1>t0&&!String(u.quality_status).includes('superseded'));
 const pages=entries.filter(e=>e.t0<t1&&e.t1>t0),directory=[...new Map(pages.map(e=>[e.page_id,e])).values()].map(e=>`${e.page_id} ${timecode(e.t0)} ${e.title}`).join('\n');
 const lines=rows.map(u=>{const p=intervalAt(entries,(u.t0+u.t1)/2).entry;return `[${timecode(u.t0)}–${timecode(u.t1)} ${p?.page_id||'无页面标定'}] ${u.text}`});
 const label=selected==='whole'?'整场讲座':`${timecode(t0)}–${timecode(t1)}`;
 return{tag:video.tag,video,range:selected,t0,t1,lines,directory,material:{file:video.teacher+' · '+video.title,selected_unit:label,scope:label+'的资料库转写及页面目录；ASR可用于主题总结，精确引文/数字须核对，不是已听校原话。',text:'页面目录：\n'+directory+'\n\n转写（'+version+'）：\n'+lines.join('\n'),text_truncated:false}};
}
export function lectureBatches(lines,limit=6500){const chunks=[];let current=[];let length=0;for(const line of lines){if(length+line.length>limit&&current.length){chunks.push(current.join('\n'));current=[];length=0}current.push(line);length+=line.length+1}if(current.length)chunks.push(current.join('\n'));return chunks}
