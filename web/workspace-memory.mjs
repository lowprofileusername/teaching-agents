import {cleanTurn} from './conversation.mjs';
export function encodeSession(s){
 const out={};for(const key of ['uploaded','reviewItems','history','unit','section','attachMaterial','question','scope','expert','specialist','jobState','status','view'])if(s[key]!==undefined)out[key]=s[key];
 out.conversation=(s.conversation||[]).map(cleanTurn);out.reviewResults=[...(s.reviewResults||new Map())];return out;
}
export function decodeSession(s){
 if(!s||typeof s!=='object'||!Array.isArray(s.conversation)||!Array.isArray(s.reviewResults))throw Error('工作区记录结构无效');
 if(!Array.isArray(s.history||[])||!Array.isArray(s.reviewItems||[])||s.reviewResults.some(x=>!Array.isArray(x)||typeof x[0]!=='string'||!Array.isArray(x[1])))throw Error('审阅进度结构无效');
 if(s.uploaded&&(!Array.isArray(s.uploaded.units)||s.uploaded.units.some(u=>typeof u.text!=='string')))throw Error('材料快照无效');
 const out={...s,reviewResults:new Map(s.reviewResults),conversation:s.conversation.map(cleanTurn)};
 if(s.jobState==='running'){out.jobState='interrupted';out.status='上次任务中断，已完成批次已保留；点击审阅继续剩余批次，不会自动重发请求。'}
 for(const t of out.conversation)if(t.status==='generating')t.status='interrupted';return out;
}
export function createWorkspaceMemory({request,snapshot,restore,indicator}){
 let id=crypto.randomUUID(),revision=0,last='',enabled=false,timer=null,chain=Promise.resolve(),conflict=false;
 const tell=t=>indicator.textContent=t;
 function flush(){if(!enabled)return Promise.resolve();clearTimeout(timer);chain=chain.catch(()=>{}).then(async()=>{
  if(conflict)throw Error('工作记录冲突，请在历史记录中另存副本');
  const state=snapshot(),raw=JSON.stringify(state);if(raw===last)return;
  tell('正在保存工作记录…');try{const x=await request({action:'save',id,revision,title:state.project?.course||'未命名备赛项目',state});revision=x.revision;last=raw;tell('已保存 '+new Date(x.updated).toLocaleTimeString()+' · 可关闭后恢复')}
  catch(e){if(e.message.includes('另一窗口'))conflict=true;tell('未保存：'+e.message);throw e}
 });return chain}
 function changed(){if(!enabled)return;clearTimeout(timer);timer=setTimeout(()=>flush().catch(()=>{}),900)}
 async function load(sessionID,copy=false){await flush();const x=await request({action:'load',id:sessionID});restore(x.state);id=copy?crypto.randomUUID():x.id;revision=copy?0:x.revision;conflict=false;last='';tell('已恢复材料、问答与已完成批次；未自动继续模型请求');await flush()}
 async function start(fresh){const x=await request({action:'list'});if(!fresh&&x.sessions.length){const s=await request({action:'load',id:x.sessions[0].id});restore(s.state);id=s.id;revision=s.revision;tell('已恢复上次工作；中断请求不会自动重发')};enabled=true;changed();return x}
 return{start,flush,changed,load,list:()=>request({action:'list'}),async importState(state){await flush();restore(state);id=crypto.randomUUID();revision=0;conflict=false;last='';await flush()},async saveCopy(){id=crypto.randomUUID();revision=0;conflict=false;last='';await flush()},async fresh(){await flush();id=crypto.randomUUID();revision=0;last='';conflict=false}};
}
