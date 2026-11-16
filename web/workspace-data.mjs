export function documentText(doc){return doc?.raw_text??doc?.units?.map(u=>u.text).join('\n')??''}
export function timestamp(s){const p=s.replace(',','.').split(':').map(Number);return p.length===3?p[0]*3600+p[1]*60+p[2]:p.length===2?p[0]*60+p[1]:NaN}
export function subtitleCues(text){const lines=text.replace(/\r/g,'').split('\n'),out=[];for(let i=0;i<lines.length;i++){const m=lines[i].match(/((?:\d+:)?\d{2}:\d{2}[.,]\d{3})\s*-->\s*((?:\d+:)?\d{2}:\d{2}[.,]\d{3})/);if(!m)continue;const start=timestamp(m[1]),end=timestamp(m[2]);let content=[];while(i+1<lines.length&&lines[i+1].trim()&&!lines[i+1].includes('-->'))content.push(lines[++i]);out.push({id:'cue'+(out.length+1),start,end,text:content.join('\n'),invalid:end<=start})}return out}
export function timelineAudit(cues){let overlap=0,gaps=[];for(let i=1;i<cues.length;i++){if(cues[i].start<cues[i-1].end)overlap++;if(cues[i].start-cues[i-1].end>30)gaps.push({start:cues[i-1].end,end:cues[i].start})}return{count:cues.length,span:cues.length?Math.max(...cues.map(c=>c.end))-Math.min(...cues.map(c=>c.start)):0,invalid:cues.filter(c=>c.invalid).length,overlap,gaps}}
export function numericMentions(doc,kind){const out=[];for(const u of doc?.units||[]){const re=/\d+(?:\.\d+)?\s*(?:课时|小时|分钟|%|％|人|名|次|分|项|届|年)/g;for(const m of u.text.matchAll(re)){out.push({kind,file:doc.name,unit:u.label,value:m[0].replace(/\s/g,''),context:u.text.slice(Math.max(0,m.index-22),m.index+m[0].length+22)});if(out.length>=300)return out}}return out}
export function slideStats(doc){return(doc?.units||[]).map(u=>({no:u.no,title:u.text.split(/\n/).find(s=>s.trim())?.slice(0,80)||u.label,chars:Array.from(u.text.replace(/\s/g,'')).length,empty:!u.text.trim()}))}
export const chainLabels=['真实问题','课程目标','创新举措','学生产物','评价依据','成效证据'];
export const slideRoles=['开场','目录','课程定位','问题与学情','创新举措','评价设计','成效与推广','总结','附录','待确认'];
export function slideOutline(doc){
 const units=doc?.units||[],frequency=new Map(),lines=units.map(u=>u.text.split(/\n/).map(x=>x.trim()).filter(Boolean));
 for(const ls of lines)for(const t of new Set(ls))frequency.set(t,(frequency.get(t)||0)+1);
 return units.map((u,i)=>{const clean=lines[i].filter(t=>!(units.length>=4&&(frequency.get(t)||0)>=Math.max(3,Math.ceil(units.length*.4)))&&!/^\d+[.、．]?$/.test(t));
  const title=clean.find(t=>t.length>=10&&t.length<=95&&/[：:｜]/.test(t))||clean.find(t=>t.length>=5&&t.length<=95)||clean[0]||u.label;
  const candidate=[title,...clean.slice(0,8)].join(' ');let role='待确认';
  if(lines[i].some(t=>/^(附[.．录]|备份页|备用页)/.test(t)))role='附录';
  else if(/请.*(批评|指正)|THANKS|谢谢/i.test(candidate))role='总结';
  else if(i===0)role='开场';
  else if(/目录|CONTENTS/i.test(candidate)||(clean.filter(t=>/^0\d\.$/.test(t)).length>=3))role='目录';
  else if(/成效|推广|复测|反思|成果/.test(title))role='成效与推广';
  else if(/评价|量规|评分|成绩构成/.test(title))role='评价设计';
  else if(/举措|模式|伴学|一企|教学案例|思政.*融|项目/.test(title))role='创新举措';
  else if(/学情|痛点|真实问题|数据证据/.test(title))role='问题与学情';
  else if(/课程|平台条件|发展历程/.test(title))role='课程定位';
  return{no:u.no,title,role,basis:'按可提取文字推断；请确认，非原PPT结构认证'};
 });
}
