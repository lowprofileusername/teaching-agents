const str=(v,n=100000)=>typeof v==='string'?v.slice(0,n):'';
const source=s=>({id:str(s.id,300),title:str(s.title,500),teacher:str(s.teacher,100),occurrences:(s.occurrences||[]).slice(0,30).map(t=>({t0:Number(t.t0),t1:Number(t.t1)}))});
export function cleanTurn(t){return {role:t.role==='user'?'user':'assistant',expert_name:str(t.expert_name,80),text:str(t.text),reasoning:str(t.reasoning),status:str(t.status,80)||'complete',request_scope:str(t.request_scope,1000),retrieval:t.retrieval?{found:Number(t.retrieval.found)||0,sent:Number(t.retrieval.sent)||0}:null,at:str(t.at,80),review_attempts:(t.review_attempts||[]).slice(0,2).map(x=>({answer:str(x.answer),truncated:!!x.truncated})),sources:(t.sources||[]).slice(0,30).map(source)};}
export function branchSnapshot(turns,history,library,scope,material){return {
 version:1,library:str(library,3000),scope:str(scope,200),turns:turns.slice(-1000).map(cleanTurn),
 history:history.filter(x=>['user','assistant'].includes(x.role)).slice(-8).map(x=>({role:x.role,content:str(x.content,19000)})),
 material:material?{file:str(material.file,500),selected_unit:str(material.selected_unit,300),scope:str(material.scope,1000),text:str(material.text,18000)}:null
};}
export function exportConversation(turns,format,{library='',includeReasoning=false}={}){
 const records=turns.map(cleanTurn).map(t=>({...t,reasoning:includeReasoning?t.reasoning:''}));
 if(format==='json')return JSON.stringify({version:1,exported_at:new Date().toISOString(),library,turns:records},null,2);
 const lines=[format==='md'?'# 教学资料助手对话':'教学资料助手对话',`资料库：${library}`,'说明：AI回答不是原始证据；来源条目是检索命中，不代表全部被引用。',''];
 for(const t of records){lines.push(`${format==='md'?'## ':''}${t.role==='user'?'用户':t.expert_name?'参考'+t.expert_name+'视角（AI）':'资料助手'}${t.status==='visual_review'?'（AI图片审阅）':t.status==='reviewed_text'?'（AI文字审阅）':t.status==='structure_review'?'（结构建议，未修改原文件）':t.status==='local_reply'?'（工作台回应）':t.status!=='complete'?`（${t.status}，非完整回答）`:''}`,t.request_scope?'本次范围：'+t.request_scope:'',t.text,'');
  for(const r of t.review_attempts||[])lines.push('未通过校验的模型原始返回'+(r.truncated?'（截断）':'')+'：',r.answer,'');
  if(t.reasoning)lines.push('接口返回的思考文本（非事实证据）：',t.reasoning,'');
  if(t.sources.length){lines.push('检索来源：');for(const s of t.sources)lines.push(`- ${s.teacher} · ${s.id} · ${s.title}${s.occurrences.length?' · '+s.occurrences.map(x=>`${x.t0}–${x.t1}秒`).join('；'):''}`);lines.push('');}
 }
 return lines.join('\n');
}
export async function consumeChatStream(response,onEvent){
 if(!response.headers.get('content-type')?.includes('text/event-stream')){const out=await response.json();onEvent('done',out);return out;}
 const reader=response.body.getReader(),decoder=new TextDecoder();let buffer='',event='message',data=[],finished=null;
 const dispatch=()=>{if(!data.length){event='message';return;}const obj=JSON.parse(data.join('\n'));data=[];const type=event;event='message';if(type==='error')throw Error(obj.error||'模型接口返回错误');onEvent(type,obj);if(type==='done')finished=obj;};
 const line=s=>{if(s==='')dispatch();else if(s.startsWith('event:'))event=s.slice(6).trim();else if(s.startsWith('data:'))data.push(s.slice(5).replace(/^ /,''));};
 try{while(!finished){const {value,done}=await reader.read();if(done){buffer+=decoder.decode();if(buffer)line(buffer.replace(/\r$/,''));dispatch();break;}buffer+=decoder.decode(value,{stream:true});let i;while((i=buffer.indexOf('\n'))>=0){const l=buffer.slice(0,i).replace(/\r$/,'');buffer=buffer.slice(i+1);line(l);if(finished)break;}}if(!finished)throw Error('连接已结束，但未收到完整回答；已显示内容仅供参考。');return finished;}
 finally{try{await reader.cancel();}catch{}reader.releaseLock();}
}
