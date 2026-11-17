import {renderBuilderPlan} from './builder.mjs';
const node=(t,x,c)=>{const n=document.createElement(t);if(x!==undefined)n.textContent=x;if(c)n.className=c;return n};
// Companion chat for the desktop teacher: reuses the backend /api/chat (retrieval,
// builder validation and data discipline all server-side). Q&A or generate
// (syllabus/report/lessonplan/presentation) for the AI track.
export function createPetChat({token}){
 const panel=node('div',undefined,'petChat');panel.hidden=true;
 const head=node('header',undefined,'petChatHead');
 const title=node('strong','和教师对话');
 const close=node('button','收起 ✕');close.type='button';
 head.append(title,close);
 const modeRow=node('div',undefined,'petChatMode');
 const mode=node('select');mode.setAttribute('aria-label','对话类型');
 for(const [v,l] of [['','问答'],['syllabus-architect','改大纲'],['architect','报告骨架'],['lessonplan-architect','教案'],['presentation-architect','汇报PPT']]){const o=node('option',l);o.value=v;mode.append(o)}
 modeRow.append(node('span','类型'),mode);
 const log=node('div',undefined,'petChatLog');log.setAttribute('aria-live','polite');
 const status=node('p','问问题，或选类型让她生成大纲/报告/教案方案。','petChatStatus');status.setAttribute('role','status');
 const inputRow=node('div',undefined,'petChatInput');
 const input=node('textarea');input.rows=2;input.placeholder='例如：帮我把大纲的课程目标写得更可测量…';input.setAttribute('aria-label','对教师说的话');
 const send=node('button','发送');send.type='button';
 inputRow.append(input,send);
 panel.append(head,modeRow,log,status,inputRow);
 function append(role,text){const m=node('div',undefined,'petChatMsg '+(role==='user'?'me':'her'));m.append(node('small',role==='user'?'我':'教师'),node('div',text));log.append(m);log.scrollTop=log.scrollHeight}
 async function submit(){
  const q=input.value.trim();if(!q)return;
  const specialist=mode.value,generate=!!specialist;
  append('user',q);input.value='';status.textContent='正在思考…';send.disabled=true;
  try{
   const body={question:q,stream:false,history:[]};
   if(generate){body.mode='generate';body.specialist_id=specialist;body.track='人工智能';body.edition='7'}
   const r=await fetch('/api/chat',{method:'POST',headers:{'Content-Type':'application/json','X-Session-Token':token},body:JSON.stringify(body)});
   const x=await r.json();if(!r.ok)throw Error(x.error||'请求失败');
   append('her',x.plan?renderBuilderPlan(x.plan):(x.answer||'（无回答）'));
   status.textContent=x.plan?'已生成方案（待实施建议，用户事实待补）':'回答完成';
  }catch(e){append('her','出错：'+e.message);status.textContent=e.message}
  finally{send.disabled=false;input.focus()}
 }
 send.onclick=submit;
 input.onkeydown=e=>{if(e.key==='Enter'&&!e.shiftKey){e.preventDefault();submit()}};
 close.onclick=()=>{panel.hidden=true};
 return{panel,toggle:()=>{panel.hidden=!panel.hidden;if(!panel.hidden)input.focus()},open:()=>{panel.hidden=false;input.focus()}};
}