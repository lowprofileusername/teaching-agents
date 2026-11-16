import {applyRevisions} from './document-review.mjs';

// Build Word write-back edits from accepted span revisions + comments.
// Accepted revisions are applied to the paragraph text (paragraph-absolute
// offsets = part.start + revision.start); the result rides the existing
// DOCX revision export (w:del/w:ins), comments become Word comments.
export function reviseExportEdits(doc,items,decisions){
 if(!doc||doc.ext!=='.docx'||!doc.source_id)throw Error('写回Word需要当前DOCX原件；旧记录可点击"关联同一份Word原件"。PDF/PPT可保留对话意见。');
 const ps=doc.paragraphs||[],byPara=new Map();
 for(const it of items){
  const g=byPara.get(it.paragraph_id)||{parts:[],comments:[],placements:[]};
  g.parts.push(it);
  if(it.comment&&it.comment.trim()&&it.comment.trim()!=='建议保留')g.comments.push(it.comment);
  if(it.placement)g.placements.push('结构建议：'+it.placement.kind+'，目标段落 '+it.placement.target_section_id+'；'+it.placement.reason);
  byPara.set(it.paragraph_id,g);
 }
 const edits=[];
 for(const [pid,g] of byPara){
  const p=ps.find(x=>x.id===pid);if(!p)continue;
  const accepted=[];
  for(const part of g.parts){
   for(const r of part.revisions||[]){
    if(decisions.get(r.rid)===true)accepted.push({...r,start:part.start+r.start,end:part.start+r.end});
   }
  }
  let replacement='';
  if(accepted.length){
   accepted.sort((a,b)=>b.start-a.start);
   let out=p.text;
   for(const r of accepted){
    const a=Array.from(out);
    if(a.slice(r.start,r.end).join('')!==r.delete)throw Error(pid+' 原文与修订区间不一致，请重新审阅。');
    out=a.slice(0,r.start).join('')+r.insert+a.slice(r.end).join('');
   }
   if(out!==p.text)replacement=out;
  }
  const comment=[...g.comments,...g.placements].join('\n')||'建议保留';
  edits.push({paragraph_id:pid,original:p.text,comment,replacement});
 }
 if(!edits.length)throw Error('还没有已完成的修订意见可以写回。');
 return edits;
}

export function reviseCounts(items,decisions){
 let total=0,accepted=0;
 for(const it of items)for(const r of it.revisions||[]){total++;if(decisions.get(r.rid)===true)accepted++}
 return {total,accepted};
}

// Render the accept/reject panel. el is the host DOM helper; decisions is a
// Map(rid -> bool) owned by the session so choices survive task switches.
export function mountReviseDialog({listEl,items,decisions,el,onChange}){
 listEl.replaceChildren();
 if(!items.length){listEl.append(el('p','本范围还没有修订结果。'));return}
 for(const it of items){
  const box=el('section',undefined,'reviseItem');
  box.append(el('h4',`${it.section} · ${it.paragraph_id}${it.start?`（字符 ${it.start}–${it.end}）`:''}`));
  const orig=el('details',undefined,'reviseOriginal');
  orig.append(el('summary','查看原文片段'),el('p',it.text));
  box.append(orig);
  box.append(el('p',it.comment,'reviseComment'));
  if(it.placement)box.append(el('p',`结构建议：${it.placement.kind} → ${it.placement.target_section_id}；${it.placement.reason}`,'fine'));
  if(!(it.revisions||[]).length){
   box.append(el('p','（本段只有批注，没有可直接执行的修订）','fine'));
  }
  for(const r of it.revisions||[]){
   const label=el('label',undefined,'reviseSpan check');
   const cb=el('input');cb.type='checkbox';cb.checked=decisions.get(r.rid)===true;
   cb.onchange=()=>{decisions.set(r.rid,cb.checked);onChange?.()};
   const del=el('span',r.delete||'（插入点）','revDel');
   const ins=el('span',r.insert,'revIns');
   const reason=el('span',r.reason,'fine');
   label.append(cb,del,document.createTextNode(' → '),ins,reason);
   box.append(label);
  }
  listEl.append(box);
 }
}