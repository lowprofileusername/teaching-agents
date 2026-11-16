import {reviewSections,parseStructuredJSON} from './document-review.mjs';
import {slideOutline} from './workspace-data.mjs';
export function structureMap(doc,task){
 if(task==='presentation'){const outline=slideOutline(doc);return outline.map((s,i)=>({id:'unit'+s.no,title:`第${s.no}页：${s.title}`,level:1,preview:doc.units[i].text.slice(0,70),suggested_role:s.role}))}
 return reviewSections(doc).map(s=>({id:s.id,title:s.title,level:s.level,preview:s.paragraphs.find(p=>p.id!==s.id)?.text.slice(0,70)||''}));
}
export function structurePrompt(){return '根据完整目录与各单元开头摘录，检查结构顺序和衔接。此处并未提供全文，不臆断未提供段落。只输出JSON：{"overview":"整体结构判断，300字以内","issues":[{"kind":"移位|合并|拆分|衔接|缺口|顺序","source_ids":["实际单元id"],"target_id":"目标单元id；无需目标则空字符串","reason":"具体理由，150字以内"}]}。没有发现问题时issues为空。最多20项；id必须来自structure_map。移位、合并、顺序必须指定实际目标位置。只给建议，不能声称已经修改原文件。'}
export function parseStructurePlan(answer,map){
 const x=parseStructuredJSON(answer,'issues'),ids=new Set(map.map(s=>s.id));
 if(typeof x?.overview!=='string'||!x.overview.trim()||x.overview.length>3000||!Array.isArray(x.issues)||x.issues.length>20)throw Error('结构诊断缺少有效概述或调整清单。');
 for(const r of x.issues){if(!['移位','合并','拆分','衔接','缺口','顺序'].includes(r.kind)||!Array.isArray(r.source_ids)||!r.source_ids.length||!r.source_ids.every(id=>ids.has(id))||typeof r.reason!=='string'||!r.reason.trim()||r.reason.length>3000||typeof r.target_id!=='string'||(r.target_id&&!ids.has(r.target_id))||(['移位','合并','顺序'].includes(r.kind)&&!r.target_id))throw Error('结构调整引用了无效位置或缺少理由。')}
 return x;
}
export function structureText(plan,map){const titles=new Map(map.map(s=>[s.id,s.title]));return '### 整体结构诊断\n\n'+plan.overview+'\n\n依据：全文目录与各单元开头摘录，具体段落仍须后续细审。\n\n'+(plan.issues.length?plan.issues.map((r,i)=>`#### ${i+1}. ${r.kind}\n\n当前位置：${r.source_ids.map(id=>titles.get(id)).join('；')}\n\n${r.target_id?'建议参照/移向：'+titles.get(r.target_id)+'\n\n':''}理由：${r.reason}`).join('\n\n'):'本轮未提出结构调整；不代表全文没有逻辑问题。')}
