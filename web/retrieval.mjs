export const timecode=s=>`${Math.floor(s/60).toString().padStart(2,'0')}:${Math.floor(s%60).toString().padStart(2,'0')}`;
const norm=s=>String(s||'').toLowerCase().replace(/[\s\p{P}\p{S}]/gu,'');
function terms(q){const s=norm(q),out=new Set(s.match(/[a-z0-9]+/g)||[]);for(let i=0;i<s.length-1;i++)out.add(s.slice(i,i+2));return [...out];}
function score(text,q,ts){const s=norm(text);let n=s.includes(norm(q))?25:0;for(const t of ts)if(s.includes(t))n+=t.length>2?3:1;return n;}
export function retrieve(db,q,limit=5){
 const ts=terms(q);const pageMatch=q.match(/(?:R\s*0*|第\s*)(\d{1,2})(?:\s*[页頁])?/i);const explicit=pageMatch?Number(pageMatch[1]):null;
 const pages=db.pages.map(p=>({p,score:score(p.title,q,ts)*3+score(p.page_text,q,ts)*2+score(p.utterances.map(u=>u.text).join(' '),q,ts)+(p.page_no===explicit?1000:0)})).filter(x=>x.score>0).sort((a,b)=>b.score-a.score).slice(0,limit);
 const pageIds=new Set(pages.map(x=>x.p.page_no));
 const raw=db.ocr.filter(o=>o.page_refs.length>0).map(o=>({o,score:score(o.text,q,ts)})).filter(x=>x.score>1).sort((a,b)=>b.score-a.score);const ocr=[];const seen=new Set();
 for(const x of raw){const signature=x.o.page_refs.join(',');if(seen.has(signature))continue;seen.add(signature);ocr.push(x.o);if(ocr.length>=3)break;}
 const evidence=pages.map(({p})=>({reference:`R${String(p.page_no).padStart(2,'0')}`,title:p.title,original_ppt_page:p.original_page_no,occurrences:p.occurrences,page_facts:p.page_text,transcripts_unverified:p.utterances,issues:p.issues,research_analysis:p.analysis?.analysis}));
 return {pages:pages.map(x=>x.p),ocr,evidence:JSON.stringify({scope:db.scope,notice:'R为视频展示单元，不是原PPT页码；转写与OCR均可能有错。原声片段可回听。',pages:evidence,raw_ocr_unverified:ocr.map(o=>({id:o.id,text:o.text,times:o.times.slice(0,8),page_refs:o.page_refs}))})};
}
