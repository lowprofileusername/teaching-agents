import {createWorkspaces} from './workspaces.mjs';
import {createSpeech} from './speech.mjs';
import {reviewSections,reviewBatches,parseReview,reviewPrompt,combineParagraphEdits} from './document-review.mjs';
import {tasks,taskByID,rubricFor,reviewContext,taskQuestion} from './workbench.mjs';
import {cleanTurn,branchSnapshot,exportConversation,consumeChatStream} from './conversation.mjs';
import {buildEvidence} from './evidence.mjs';
import {renderMarkdown} from './markdown.mjs';
import {timecode,retrievalQuery} from './retrieval.mjs';
const $=s=>document.querySelector(s),el=(tag,text,cls)=>{const x=document.createElement(tag);if(text!==undefined)x.textContent=text;if(cls)x.className=cls;return x};
let token=location.hash.slice(1)||sessionStorage.getItem('sessionToken')||'',profile={persona:'original'},ready=false,libraryReady=false,busy=false,history=[],videos=[],audioState=null,currentAudioURL=null,assetURLs=new Map(),currentLibraryPath='',uploaded=null,requestController=null,stageTimer=null,conversation=[],currentTask='overview',ruleCatalog=null,reviewItems=[],reviewResults=new Map(),reviewFile=null;
const branchID=new URLSearchParams(location.search).get('branch');
if(token){sessionStorage.setItem('sessionToken',token);window.history.replaceState(null,'',location.pathname)}
const taskSessions=new Map();let workspace=null,importing=false,exportMode='chat';let nativeDesktop=null,activeExpert='',expertCatalog=null;
const teacherSpeech=createSpeech({token,template:()=>$('#teacherTemplate').value,onStatus:text=>{$('#voiceStatus').textContent=text;$('#voiceFloatingStatus').textContent=text}});
async function api(path,data){const ctrl=new AbortController(),timeout=path==='/api/chat'?600000:45000;const onCancel=()=>ctrl.abort();if(requestController&&['/api/search','/api/chat'].includes(path)){if(requestController.signal.aborted)ctrl.abort();else requestController.signal.addEventListener('abort',onCancel,{once:true})}const timer=setTimeout(()=>ctrl.abort(),timeout);try{const r=await fetch(path,{method:'POST',signal:ctrl.signal,headers:{'Content-Type':'application/json','X-Session-Token':token},body:JSON.stringify(data)});if(!r.ok){const e=await r.json().catch(()=>({error:'服务响应错误'}));throw Error(e.error==='context deadline exceeded'?'本次数据库检索超时，请确认已更新EXE和优化版资料库；模型尚未调用。':e.error)}return r}catch(e){if(e.name==='AbortError')throw Error(requestController?.signal.aborted?'本次请求已取消。':path==='/api/chat'?'等待模型超过10分钟，请检查模型运行情况、输出预算和思考等级。':'数据库或本机接口等待超过45秒；请检查SMB连接或数据库锁。');throw e}finally{clearTimeout(timer);requestController?.signal.removeEventListener('abort',onCancel)}}
function lockProject(value){for(const id of ['competitionTrack','competitionEdition','courseName','expertPerspective','scope'])$('#'+id).disabled=value}
function stage(text){clearInterval(stageTimer);const start=Date.now();$('#status').textContent=text;stageTimer=setInterval(()=>{$('#status').textContent=text+'（已等待 '+Math.floor((Date.now()-start)/1000)+' 秒）'},1000)}

const json=async(path,data)=>(await api(path,data)).json();
function inspector(tab='assistant'){const evidence=tab==='evidence';$('.conversation').hidden=evidence;$('.evidence').hidden=!evidence;$('#assistantTab').setAttribute('aria-selected',String(!evidence));$('#evidenceTab').setAttribute('aria-selected',String(evidence));$('.appShell').classList.add('show-inspector')}
workspace=createWorkspaces({
 state:()=>({task:currentTask,uploaded,reviewItems,videos,track:$('#competitionTrack').value,course:$('#courseName').value,documents:Object.fromEntries(['report','lesson','presentation'].map(id=>[id,id===currentTask?uploaded:taskSessions.get(id)?.uploaded]).filter(([,d])=>d))}),
 select:selectTask,upload:()=>{if(busy||importing)return;$('.materialPanel').open=true;$('#materialFile').click()},notice:text=>$('#status').textContent=text,
 section:id=>{$('#reviewSection').value=id},unit:i=>$('#materialUnit').value=String(i),reviewAll:()=>$('#reviewAll').click(),
 ask:(q,m)=>{inspector();ask(q,m)},browse:async tag=>{$('#scope').value=tag;inspector('evidence');await browse()}
});
for(const task of tasks){const b=el('button',task.title);b.type='button';b.dataset.task=task.id;b.onclick=()=>selectTask(task.id);$('#taskNav').append(b)}
$('#workspaceUpload').onclick=()=>{if(!busy&&!importing){$('.materialPanel').open=true;$('#materialFile').click()}};
$('#assistantTab').onclick=()=>inspector();$('#evidenceTab').onclick=()=>inspector('evidence');
$('#railWork').onclick=()=>selectTask('overview');$('#railLibrary').onclick=()=>selectTask('library');$('#railTeacher').onclick=()=>$('#showPet').click();$('#railAI').onclick=()=>{$('.appShell').classList.toggle('show-inspector')};$('#railSettings').onclick=()=>$('#settingsOpen').click();
$('#workspaceExport').onclick=async()=>{exportMode='workspace';$('#exportDialog h2').textContent='导出工作区记录';$('#exportFormat').value='json';$('#exportFormat').disabled=true;$('#exportDialog').showModal();$('#exportStatus').textContent='导出备赛简报、时间轴标注、讲稿与一致性表；不包含原始媒体或API密钥。';try{const x=await json('/api/conversation-export',{action:'directory'});$('#exportDirectory').value=x.directory}catch(e){$('#exportStatus').textContent=e.message}};
selectTask(currentTask);
function renderTurn(turn){const a=el('article',undefined,'message '+turn.role),body=el('div',undefined,'answerBody');a._turn=turn;a.append(el('small',turn.role==='user'?'你':turn.expert_name?'参考'+turn.expert_name+'视角 · AI备赛顾问':'AI备赛顾问'),body);$('#messages').append(a);updateTurn(a);return a}
function updateTurn(a){const t=a._turn,body=a.querySelector('.answerBody');body.replaceChildren(t.role==='user'?el('p',t.text):renderMarkdown(t.text||''));if(t.role==='assistant'&&(t.reasoning||t.modelResponse)){let d=a.querySelector('.modelReasoning');if(!d){d=el('details',undefined,'modelReasoning');d.append(el('summary'),el('pre'));a.insertBefore(d,body)}d.querySelector('summary').textContent=t.reasoning?'接口返回的思考文本（非核验证据）':t.status==='generating'?'接口思考文本（尚未返回）':'接口未返回可展示的思考文本';d.querySelector('pre').textContent=t.reasoning||'开启思考不代表接口会提供思考文本；这里不会推测或补写。'}$('#messages').scrollTop=$('#messages').scrollHeight;}
function msg(role,text,extra={}){const turn={role,text,expert_name:role==='assistant'?(selectedExpert()?.name||''):'',reasoning:'',status:'complete',at:new Date().toISOString(),sources:[],...extra};conversation.push(turn);return renderTurn(turn)}
async function chatRequest(data,onEvent){data.expert_id=$('#expertPerspective').value;const ctrl=requestController;let timedOut=false;const timer=setTimeout(()=>{timedOut=true;ctrl.abort()},600000);try{const r=await fetch('/api/chat',{method:'POST',signal:ctrl.signal,headers:{'Content-Type':'application/json','X-Session-Token':token},body:JSON.stringify(data)});if(!r.ok){const x=await r.json().catch(()=>({error:'模型请求失败'}));throw Error(x.error)}return await consumeChatStream(r,onEvent)}catch(e){if(e.name==='AbortError')throw Error(timedOut?'等待模型超过10分钟，请检查模型服务。':'本次请求已取消，已显示内容可能不完整。');throw e}finally{clearTimeout(timer)}}

function clearImages(){for(const u of assetURLs.values())URL.revokeObjectURL(u);assetURLs.clear()}
async function asset(id){if(assetURLs.has(id))return assetURLs.get(id);const u=URL.createObjectURL(await(await api('/api/asset',{id})).blob());assetURLs.set(id,u);return u}
async function listen(tag,t0,t1){try{const c=await json('/api/audio',{tag,time:t0+.001});const v=videos.find(v=>v.tag===tag);$('#audioRange').textContent=`${v?.teacher||tag} ${timecode(t0)}–${timecode(t1)}（视频时间）`;if(currentAudioURL)URL.revokeObjectURL(currentAudioURL);currentAudioURL=URL.createObjectURL(await(await api('/api/asset',{id:c.asset_id})).blob());const a=$('#audio');a.src=currentAudioURL;audioState={tag,end:t1,chunkEnd:c.t1,chunkStart:c.t0};a.currentTime=Math.max(0,t0-c.t0);await a.play()}catch(e){$('#status').textContent=e.message}}
$('#audio').addEventListener('timeupdate',()=>{if(audioState&&$('#audio').currentTime+audioState.chunkStart>=audioState.end){$('#audio').pause();audioState=null}});
$('#audio').addEventListener('ended',()=>{if(audioState&&audioState.end>audioState.chunkEnd+.02){const s=audioState;listen(s.tag,s.chunkEnd,s.end)}});
function pageCard(p){const box=el('article',undefined,'source');box.dataset.id=p.id;const imgs=el('div');box.append(imgs);for(const meta of p.images||[]){const img=el('img');img.alt=`${p.teacher} ${p.id} ${p.title}`;img.loading='lazy';imgs.append(img);const load=el('button','加载原页截图');imgs.append(load);load.onclick=async()=>{load.disabled=true;try{img.src=await asset(meta.asset_id);load.remove()}catch(e){load.textContent=e.message;load.disabled=false}};img.onclick=()=>{$('#imageDialog img').src=img.src;$('#imageDialog p').textContent=`${p.teacher} · ${p.institution} · ${p.course||'非单一课程'} · ${p.id} · 原PPT页码 ${p.original_page_no||'未确认'}`;$('#imageDialog').showModal()}}
 box.append(el('h3',`${p.teacher} · ${p.id} · ${p.title}`),el('p',`${p.institution||''} / ${p.course||'非单一课程'} / 原PPT页码：${p.original_page_no||'未确认'}`,'fine'),el('p','【页】'+(p.page_text||'')));
 const ts=el('div',undefined,'times');for(const t of p.occurrences||[]){const b=el('button',timecode(t.t0)+'–'+timecode(t.t1));b.onclick=()=>listen(p.tag,t.t0,t.t1);ts.append(b)}box.append(ts);
 const det=el('details');det.append(el('summary',`多版本口述转写 · ${p.utterances.length} 条 · 未整场听校`));for(const u of p.utterances){const who=u.teacher_attribution?`【${u.teacher_attribution}】`:'';const b=el('button',`${timecode(u.t0)} ${who}${u.text}\n${u.version}`,'utter');b.onclick=()=>listen(p.tag,u.t0,u.t1);det.append(b)}box.append(det);
 if(p.official_pages.length){const d=el('details');d.append(el('summary',`对应官方课件 · ${p.official_pages.length} 页`));for(const s of p.official_pages){d.append(el('p',`【官方PDF】${s.title} / 物理第${s.pdf_page_no}页 / ${s.relation}`),el('pre',s.direct_text||s.ocr_text||'无可提取文字'));if(s.notes)d.append(el('p',s.notes));const b=el('button','打开官方 PDF');b.onclick=async()=>{try{window.open(await asset(s.asset_id),'_blank','noopener')}catch(e){$('#status').textContent=e.message}};d.append(b)}box.append(d)}
 if(p.source_format==='claude'&&p.analysis){const d=el('details');d.append(el('summary','来源库逐页研究标注（非讲者原话）'));for(const [k,label] of [['section','环节'],['main_message','主信息'],['elements','元素'],['function','功能'],['relation','前后关系'],['visual','视觉'],['evidence','证据'],['borrow','可借鉴'],['migrate','迁移建议']])if(p.analysis[k])d.append(el('p',label+'：'+p.analysis[k]));box.append(d)}
 for(const [key,label] of [['issues','核查问题'],['annotations','转写修订与疑项']])if(p[key].length){const d=el('details');d.append(el('summary',label),el('pre',JSON.stringify(p[key],null,2)));box.append(d)}return box}
function showSources(pages,raw=[]){clearImages();$('#sources').replaceChildren(...pages.map(pageCard));$('#found').textContent=`${pages.length} 单元 · ${raw.length} OCR`;for(const o of raw){const d=el('details',undefined,'source'),v=videos.find(v=>v.tag===o.tag);d.append(el('summary',`${v?.teacher||o.tag} · ${timecode(o.seconds)} · 原始OCR未核`),el('pre',o.text));const b=el('button','回听此时刻');b.onclick=()=>listen(o.tag,Math.max(0,o.seconds-2),o.seconds+8);d.append(b);$('#sources').append(d)}}
function answerTools(a,text,pages){const tools=el('div',undefined,'tools'),sourceList=el('details',undefined,'answer-sources'),sourceButtons=el('div',undefined,'tools');sourceList.append(el('summary','查看本次检索来源（'+pages.length+'项，不代表全部被引用）'),sourceButtons);a.append(sourceList);for(const p of pages){const b=el('button',`${p.teacher} · ${p.title}`);b.onclick=()=>{showSources([p]);inspector('evidence')};sourceButtons.append(b)}tools.append(teacherSpeech.button(text));a.append(tools)}

async function ask(q,suppliedMaterial=null){
 if(busy||!q.trim())return;
 if(!libraryReady){$('#settings').showModal();$('#libraryStatus').textContent='先连接 SQLite 资料库。';return;}
 busy=true;lockProject(true);requestController=new AbortController();$('#cancelRequest').hidden=false;$('#send').disabled=true;$('#forkWindow').disabled=true;let answerNode=null;
 try{
  // Windows share service settings. Refresh them before each request so an old
  // window cannot silently attribute results to a previously connected library.
  const state=await json('/api/library',{});if(state.path!==currentLibraryPath)await connect();renderProfiles(await json('/api/profiles',{action:'list'}));
  msg('user',q);companionState('retrieving');stage('正在只读检索 SQLite 资料库…');
  const bench=reviewContext({edition:$('#competitionEdition').value,track:$('#competitionTrack').value,course:$('#courseName').value,task:currentTask},ruleCatalog);
  let context=q;const material=suppliedMaterial||materialContext();if(material)context=material.text.slice(0,120)+' '+q;else context=retrievalQuery(q,history,videos);
  const pm=q.match(/(?:[RP]\s*0*|第\s*)(\d{1,3})(?:\s*[页頁])?/i);
  const result=await json('/api/search',{q:[selectedExpert()?.name||'', $('#competitionTrack').value?$('#competitionTrack').value+'赛道':'',context].filter(Boolean).join(' ').slice(0,1800),tag:$('#scope').value,page_no:pm?Number(pm[1]):0});
  showSources(result.pages,result.raw_ocr);showDocumentHits(result.document_hits||[]);
  for(const h of result.official_hits||[]){const d=el('details',undefined,'source');d.append(el('summary',`${h.teacher} 官方PDF第${h.pdf_page_no}页`),el('pre',h.direct_text||h.ocr_text||''));const b=el('button','打开官方PDF');b.onclick=async()=>window.open(await asset(h.asset_id),'_blank','noopener');d.append(b);$('#sources').append(d)}
  const sources=result.pages.map(p=>({id:p.id,title:p.title,teacher:p.teacher,occurrences:p.occurrences}));
  for(const d of result.document_hits||[])sources.push({id:d.kind==='expert_profile'?d.id:d.document_id+':PDF'+d.pdf_page_no,title:d.title,teacher:'研究报告/参考文件',occurrences:[]});
  if(!ready){const text=result.document_hits?.length?'已找到以下参考材料（未调用模型）：\n\n'+result.document_hits.map(x=>x.title+'\n'+x.text).join('\n\n')+'\n\n配置 API 后可结合材料生成审阅意见。':result.pages.length?'找到以下资料，可从右侧查看截图、课件与原声：\n\n'+result.pages.map(p=>`${p.teacher} · ${p.id} · ${p.title}\n【页】${p.page_text}`).join('\n\n')+'\n\n配置 API 后可基于这些证据展开对话。':'当前关键词没有找到页面匹配。请改用核心术语，或选择专家后浏览目录。';answerNode=msg('assistant',text,{sources});answerTools(answerNode,text,result.pages);$('#status').textContent='SQLite 检索完成（'+((result.timings?.total_ms||0)/1000).toFixed(2)+'秒），未调用模型。';
  }else{
   const prefix='检索完成（'+((result.timings?.total_ms||0)/1000).toFixed(2)+'秒）：'+result.pages.length+'个讲座单元、'+(result.document_hits||[]).length+'条报告/规则/专家资料来源。';stage(prefix+'正在等待模型接口…');
   companionState('thinking');answerNode=msg('assistant','',{sources,status:'generating',modelResponse:true});let phase='',lastPaint=0;
   const out=await chatRequest({question:q+'\n本轮备赛信息（用户选择，不是官方规则）：'+JSON.stringify(bench),evidence:JSON.stringify(buildEvidence(result)),history,material:material?JSON.stringify(material):'',stream:$('#streamOutput').checked},(event,x)=>{
    if(event==='status'){stage(prefix+x.message);return;}
    if(event==='progress'||event==='done'){answerNode._turn.text=x.answer||'';answerNode._turn.reasoning=x.reasoning||'';const next=x.answer?'正在接收回答…':x.reasoning?'正在接收接口思考文本…':'等待模型输出…';if(next!==phase){phase=next;companionState(x.answer?'answering':'thinking');stage(prefix+next)}if(event==='done'||Date.now()-lastPaint>80){updateTurn(answerNode);lastPaint=Date.now()}}
   });
   if(!out.answer)throw Error('模型未返回最终回答');
   Object.assign(answerNode._turn,{text:out.answer,reasoning:out.reasoning||'',status:out.answer_truncated?'output_limit':'complete'});updateTurn(answerNode);answerTools(answerNode,out.answer,result.pages);
   history.push({role:'user',content:q},{role:'assistant',content:out.answer});history=history.slice(-8);companionState('done');if($('#voiceAuto').checked)teacherSpeech.speak(out.answer);
   $('#status').textContent=(out.context?'本轮实际传给模型 '+out.context.included_sources+'/'+out.context.retrieved_sources+' 条证据摘录。':'回答完成。')+(out.reasoning_available?'可展开查看接口返回的思考文本。':'接口未返回思考文本。')+(out.answer_truncated?' 输出达到上限，回答可能未完。':'');
  }
 }catch(e){companionState('error');if(answerNode){answerNode._turn.status='interrupted';updateTurn(answerNode)}msg('assistant',e.message,{status:'error'});$('#status').textContent='本次操作未完成。';}
 finally{clearInterval(stageTimer);stageTimer=null;requestController=null;busy=false;lockProject(false);$('#cancelRequest').hidden=true;$('#send').disabled=false;$('#forkWindow').disabled=false;}
}

async function connect(path=''){const x=await json('/api/library',{path});if(currentLibraryPath&&currentLibraryPath!==x.path){history=[];conversation=[];for(const snap of taskSessions.values()){snap.history=[];snap.conversation=[]}$('#messages').replaceChildren();$('#sources').replaceChildren();clearImages();$('#audio').pause();audioState=null;if(currentAudioURL){URL.revokeObjectURL(currentAudioURL);currentAudioURL=null}msg('assistant','已切换资料库，新对话将使用这个库的页面编号和证据。')}currentLibraryPath=x.path;libraryReady=true;videos=x.videos;$('#dbPath').value=x.path;$('#counts').textContent=x.format==='builtin'?'内置9赛道参考 · 无外部讲座库':`${x.counts.video} 场 · ${x.counts.page} 个检索单元`;$('#libraryStatus').textContent=`只读连接成功 · ${(x.bytes/1e9).toFixed(3)} GB · ${x.counts.utterance} 条转写 · ${x.format==='claude'?'Claude 格式':'Codex 格式'}`;$('#scope').replaceChildren(el('option',`全部 ${x.counts.video} 场`));$('#scope option').value='';for(const v of videos){const o=el('option',`${v.teacher} · ${v.course||v.title.split('/').pop()} (${v.page_count})`);o.value=v.tag;$('#scope').append(o)}$('#status').textContent=x.path_saved===false?'资料库已连接，但路径未保存；请检查程序目录是否可写。':`${x.counts.video} 场资料库已就绪；${x.coverage_note||'只读连接'}`;workspace?.refresh();return x}
async function browse(){if(!libraryReady||!currentLibraryPath)return;try{const rows=await json('/api/catalog',{tag:$('#scope').value});clearImages();$('#sources').replaceChildren();$('#found').textContent=rows.length+' 个检索单元';for(const p of rows){const b=el('button',`${p.teacher} · ${p.id} · ${p.title}`,'utter');b.onclick=async()=>{try{showSources([await json('/api/page',{id:p.id})])}catch(e){$('#status').textContent=e.message}};$('#sources').append(b)}}catch(e){$('#status').textContent=e.message}}
$('#connectLibrary').onclick=async()=>{try{await connect($('#dbPath').value.trim());await browse()}catch(e){$('#libraryStatus').textContent=e.message}};
$('#ask').onsubmit=e=>{e.preventDefault();const q=$('#question').value;$('#question').value='';ask(q)};$('#question').onkeydown=e=>{if(e.key==='Enter'&&!e.shiftKey){e.preventDefault();$('#ask').requestSubmit()}};document.querySelectorAll('[data-q]').forEach(b=>b.onclick=()=>ask(b.dataset.q));$('#browse').onclick=()=>{inspector('evidence');browse()};$('#scope').onchange=browse;$('#imageClose').onclick=()=>$('#imageDialog').close();$('#settingsOpen').onclick=()=>$('#settings').showModal();$('#settingsClose').onclick=()=>$('#settings').close();
const config=()=>Object.fromEntries(['base','key','model'].map(k=>[k,$('#'+k).value.trim()]));
let apiProfiles=[];const modelOptions=()=>({thinkingMode:$('#thinkingMode').value,thinkingFormat:$('#thinkingFormat').value,reasoningEffort:$('#reasoningEffort').value,contextWindow:Number($('#contextWindow').value),maxTokens:Number($('#maxTokens').value)});
function fillProfile(p){$('#thinkingMode').value=p?.thinkingMode||'default';$('#thinkingFormat').value=p?.thinkingFormat||'default';$('#reasoningEffort').value=p?.reasoningEffort||'';$('#contextWindow').value=p?.contextWindow||32768;$('#maxTokens').value=p?.maxTokens||4096;$('#apiName').value=p?.name||'';$('#base').value=p?.base||'';$('#model').value=p?.model||'';$('#key').value='';$('#clearKey').checked=false;$('#keyHint').textContent=p?.hasKey?'密钥已保存在系统凭据库；留空保留。换电脑请重新输入。':'未保存密钥；无认证的本地服务可留空。'}
function renderProfiles(x){apiProfiles=x.profiles;ready=x.chatReady;$('#apiProfiles').replaceChildren(el('option','新建配置'));$('#apiProfiles option').value='';for(const p of apiProfiles){const o=el('option',p.name+(p.id===x.active?'（当前）':''));o.value=p.id;$('#apiProfiles').append(o)}$('#apiProfiles').value=x.active||'';fillProfile(x.selected);}
$('#apiProfiles').onchange=()=>fillProfile(apiProfiles.find(p=>p.id===$('#apiProfiles').value));
$('#newProfile').onclick=()=>{$('#apiProfiles').value='';fillProfile(null);$('#configStatus').textContent='填写新配置后保存，当前连接暂不改变。'};
$('#activateProfile').onclick=async()=>{try{const x=await json('/api/profiles',{action:'activate',id:$('#apiProfiles').value});renderProfiles(x);history=[];$('#configStatus').textContent='已切换到：'+x.selected.name;$('#status').textContent='当前 API：'+x.selected.name+' / '+x.selected.model;}catch(e){$('#configStatus').textContent=e.message}};
$('#deleteProfile').onclick=async()=>{const id=$('#apiProfiles').value;if(!id)return;try{renderProfiles(await json('/api/profiles',{action:'delete',id}));$('#configStatus').textContent='配置已删除；其他配置保留。'}catch(e){$('#configStatus').textContent=e.message}};
$('#config').onsubmit=async e=>{e.preventDefault();try{const c=config();const x=await json('/api/profiles',{...modelOptions(),action:'save',id:$('#apiProfiles').value,name:$('#apiName').value,base:c.base,model:c.model,key:c.key,clearKey:$('#clearKey').checked});renderProfiles(x);history=[];$('#configStatus').textContent='已保存并启用：'+x.selected.name;$('#status').textContent='当前 API：'+x.selected.name+' / '+x.selected.model;}catch(e){$('#configStatus').textContent=e.message}};
$('#testAPI').onclick=async()=>{$('#testAPI').disabled=true;$('#configStatus').textContent='正在测试已启用的 API…';try{const x=await json('/api/test',{});$('#configStatus').textContent=(x.modelFound?'连接成功，所选模型存在。':'接口可连接，但列表未找到所填模型；请核对模型名。')+' 可用模型：'+x.models.join('、')}catch(e){$('#configStatus').textContent=e.message}finally{$('#testAPI').disabled=false}};
try{renderProfiles(await json('/api/profiles',{action:'list'}))}catch(e){$('#configStatus').textContent=e.message}

function applyTeacher(persona){const male=persona==='male',label=male?'原创男教师':'原创女教师';profile.persona=male?'male':'original';document.title=label+' · 教学创新备赛工作台';$('h1').textContent=label+' · 教学创新备赛工作台';$('.persona .portrait img').src=male?'assets/original_male_teacher.png':'assets/original_teacher.png';$('.persona .portrait img').alt=label+'AI静态形象';$('.badge').textContent='原创 AI 形象 · 不对应真人';$('#teacherTemplate').value=male?'male':'female'}
try{const p=await(await fetch('profile.json')).json();applyTeacher(p.persona);await connect();await browse()}catch(e){$('#status').textContent=e.message;$('#libraryStatus').textContent=e.message;$('#counts').textContent='外部资料库未连接'}
$('#teacherTemplate').onchange=async()=>{try{const value=$('#teacherTemplate').value;await json('/api/desktop',{action:'template',template:value});applyTeacher(value)}catch(e){$('#status').textContent=e.message}};
setInterval(async()=>{try{const x=await json('/api/desktop',{action:'status'});if((x.template==='male')!==(profile.persona==='male'))applyTeacher(x.template)}catch{}},2000);


function materialContext(){if(!uploaded||(uploaded.task&&uploaded.task!==currentTask))return null;const u=uploaded.units[Number($('#materialUnit').value)||0];return {file:uploaded.name,total_units:uploaded.units.length,selected_unit:u.label,scope:uploaded.scope,text:u.text.slice(0,18000),text_truncated:u.text.length>18000};}
function showDocumentHits(hits){for(const h of hits){const d=el('details',undefined,'source');d.append(el('summary',h.title+(h.kind==='expert_profile'?' · 公开专业资料':' · PDF第'+h.pdf_page_no+'页')),el('p',h.authority_status||h.kind,'fine'),el('pre',h.text));if(h.kind==='expert_profile'){const b=el('button','查看资料与来源');b.onclick=()=>showExpertInfo(h.expert_id);d.append(b);$('#sources').prepend(d);continue}const b=el('button','打开来源 PDF');b.onclick=async()=>{try{const r=await api('/api/document-pdf',{id:h.document_id});const u=URL.createObjectURL(await r.blob());window.open(u+'#page='+h.pdf_page_no,'_blank','noopener');setTimeout(()=>URL.revokeObjectURL(u),600000)}catch(e){$('#status').textContent=e.message}};d.append(b);$('#sources').prepend(d)}}
$('#materialFile').onchange=async()=>{if(busy||importing)return;const f=$('#materialFile').files[0];if(!f)return;if(f.size>50*1024*1024){$('#materialStatus').textContent='文件超过50MB，请拆分或压缩。';return}importing=true;uploaded=null;reviewFile=null;reviewItems=[];reviewResults.clear();workspace.resetMaterial();$('#materialUnit').replaceChildren();$('#materialStatus').textContent='正在本机提取文字…';try{const ext=f.name.toLowerCase().split('.').pop();let x;if(ext==='pdf'){const pdfjs=await import('./lib/pdf.mjs');pdfjs.GlobalWorkerOptions.workerSrc=new URL('./lib/pdf.worker.mjs',import.meta.url).href;const task=pdfjs.getDocument({data:new Uint8Array(await f.arrayBuffer()),isEvalSupported:false,useSystemFonts:true});const pdf=await task.promise;if(pdf.numPages>300){await task.destroy();throw Error('单次最多300页，请拆分。')}const units=[];try{for(let n=1;n<=pdf.numPages;n++){const page=await pdf.getPage(n),t=await page.getTextContent();units.push({no:n,label:'PDF第'+n+'页',text:t.items.map(x=>x.str+(x.hasEOL?'\n':' ')).join('')});page.cleanup()}}finally{await task.destroy()}x={name:f.name,units,scope:'PDF文字层提取；未OCR扫描/图片，不判断视觉版式。'}}else if(ext==='txt'||ext==='md'||ext==='srt'||ext==='vtt'){const t=await f.text(),units=[];if(t.length>1000000)throw Error('文本过长，请拆分。');for(let i=0;i<t.length;i+=4000)units.push({no:units.length+1,label:'文字段'+(units.length+1),text:t.slice(i,i+4000)});x={name:f.name,units,raw_text:t,scope:'纯文字段落，非原排版页码。'}}else{const r=await fetch('/api/material',{method:'POST',headers:{'X-Session-Token':token,'X-File-Name':encodeURIComponent(f.name),'Content-Type':'application/octet-stream'},body:f});x=await r.json();if(!r.ok)throw Error(x.error)}if(!x.units.length)throw Error('未找到可审阅文字。');uploaded={...x,task:currentTask};reviewFile=f;reviewItems=[];reviewResults.clear();initReviewSections();$('#materialUnit').replaceChildren(...x.units.map((u,i)=>{const o=el('option',u.label+(u.text.trim()?'':'（无文字层）'));o.value=i;return o}));$('#materialStatus').textContent=x.name+' · '+x.units.length+'单元。仅将所选单元发送给已选模型。'+x.scope;}catch(e){$('#materialStatus').textContent=e.message}finally{importing=false;workspace?.refresh()}};
$('#reviewUnit').onclick=()=>{if(!uploaded)return;const u=materialContext();if(!u){$('#materialStatus').textContent='请为当前任务添加材料，或切回原任务。';return}if(!u.text.trim()){$('#materialStatus').textContent='此页没有可提取文字，需要OCR或视觉模型，当前未审阅。';return}ask(taskQuestion(currentTask)+' 本次材料：'+u.file+' / '+u.selected_unit);};
$('#clearMaterial').onclick=()=>{workspace.resetMaterial();uploaded=null;reviewFile=null;reviewItems=[];reviewResults.clear();initReviewSections();$('#materialFile').value='';$('#materialUnit').replaceChildren();$('#materialStatus').textContent='材料已从本次会话移除。'};
$('#exportReview').onclick=openExport;

$('#cancelRequest').onclick=()=>{requestController?.abort();$('#status').textContent='正在取消本次请求…'};

try{const bank=await(await fetch('questions.json')).json();let group;for(const item of bank){if(!group||group.label!==item.category){group=document.createElement('optgroup');group.label=item.category;$('#questionBank').append(group)}const o=el('option',item.id+'. '+item.question);o.value=item.question;group.append(o)}$('#useQuestion').onclick=()=>{$('#question').value=$('#questionBank').value;$('#question').focus()}}catch(e){console.warn('问题目录未载入')}

async function openExport(){exportMode='chat';$('#exportFormat').disabled=false;$('#exportDialog h2').textContent='导出当前工作区对话';if(!conversation.length){$('#status').textContent='当前窗口没有可导出的问答。';return;}$('#exportDialog').showModal();$('#exportStatus').textContent='导出当前窗口的完整可见问答；上下文仅使用最近4轮，与导出范围不同。';try{const x=await json('/api/conversation-export',{action:'directory'});$('#exportDirectory').value=x.directory}catch(e){$('#exportStatus').textContent=e.message}}
$('#exportChat').onclick=openExport;$('#exportClose').onclick=()=>$('#exportDialog').close();
$('#exportForm').onsubmit=async e=>{e.preventDefault();$('#exportSave').disabled=true;try{const format=exportMode==='workspace'?'json':$('#exportFormat').value,content=exportMode==='workspace'?JSON.stringify(workspace.exportState(),null,2):exportConversation(conversation,format,{library:currentLibraryPath,includeReasoning:$('#exportReasoning').checked});const out=await json('/api/conversation-export',{action:'save',directory:$('#exportDirectory').value.trim(),format,content});$('#exportStatus').textContent='已保存：\n'+out.path+'\n'+out.bytes+' 字节。'}catch(e){$('#exportStatus').textContent=e.message}finally{$('#exportSave').disabled=false}};
async function openConversationWindow(fork){if(busy||importing){$('#status').textContent='请先完成或取消当前操作，再打开新窗口。';return;}
 const snapshot=fork?{...branchSnapshot(conversation,history,currentLibraryPath,$('#scope').value,materialContext()),task:currentTask,project:{track:$('#competitionTrack').value,edition:$('#competitionEdition').value,course:$('#courseName').value},expert:$('#expertPerspective').value}:null;
 // Open the browser placeholder within the click activation.
 if(nativeDesktop===null){$('#status').textContent='窗口服务尚未就绪，请稍后重试。';return}
 let popup=null;try{
 if(!nativeDesktop){popup=window.open('about:blank','_blank','popup,width=1440,height=900');if(!popup)throw Error('浏览器拦截了新窗口，请允许本机页面弹出窗口。')}
 const out=await json('/api/windows',{action:'open',snapshot});if(popup)popup.location.href=out.url;
 $('#status').textContent=fork?'已打开追问窗口，继承最近4轮上下文及当前材料片段。':'已打开空白工作台，不携带原对话或材料。';
 }catch(e){popup?.close();$('#status').textContent=e.message}
}
$('#newWindow').onclick=()=>openConversationWindow(false);$('#forkWindow').onclick=()=>openConversationWindow(true);
async function restoreBranch(){if(!branchID)return;try{
 const data=await json('/api/windows',{action:'consume',id:branchID});if(data.library!==currentLibraryPath)throw Error('资料库已改变，未继承旧上下文。请切回原库后重新打开追问窗口。');
 selectTask(data.task||'overview');for(const [id,key] of [['competitionTrack','track'],['competitionEdition','edition'],['courseName','course']])if(data.project?.[key])$('#'+id).value=data.project[key];
 const snap=branchSnapshot(data.turns||[],data.history||[],data.library,data.scope,data.material);history=snap.history;conversation=snap.turns;$('#messages').replaceChildren();for(const t of conversation)renderTurn(t);
 if(videos.some(v=>v.tag===snap.scope))$('#scope').value=snap.scope;
 if(data.expert&&[...$('#expertPerspective').options].some(o=>o.value===data.expert)){$('#expertPerspective').value=data.expert;activeExpert=data.expert}
 if(snap.material){const m=snap.material;uploaded={name:m.file,scope:m.scope+'（仅继承的选中片段）',task:currentTask,units:[{no:1,label:m.selected_unit,text:m.text}]};restoreMaterialControls('0','all')}
 updateRubric();workspace.refresh();$('#status').textContent='追问窗口已继承上下文；后续对话与原窗口独立。';
 }catch(e){$('#status').textContent=e.message}}

$('#showPet').onclick=async()=>{try{await json('/api/desktop',{action:'show_pet'});$('#status').textContent='桌面教师已打开，可切换原创女教师和原创男教师。'}catch(e){$('#status').textContent=e.message}};
function companionState(state){json('/api/desktop',{action:'state',state}).catch(()=>{})}

function updateRubric(){
 const track=$('#competitionTrack').value,r=rubricFor(ruleCatalog,track,currentTask);
 $('#rubricStatus').textContent=ruleCatalog?.future_note||'内置参考尚未加载，不能据此断言本届要求。';
 $('#rubricBadge').textContent=r?.part?'第六届参考 · 本项'+r.part.score+'分':track?'第六届参考 / 请核对当届文件':'请先选择赛道';
 $('#rubricDimensions').replaceChildren(...(r?.part?.dimensions||[]).map(t=>el('span',t)));
 if(r&&currentTask==='lesson')$('#rubricDimensions').append(el('p',r.group.video_requirements,'fine'));
}
function snapshotTask(){return {uploaded,reviewFile,reviewItems,reviewResults,history,conversation,unit:$('#materialUnit').value,section:$('#reviewSection').value,question:$('#question').value,scope:$('#scope').value,expert:$('#expertPerspective').value}}
function restoreMaterialControls(unit='0',section='all'){
 $('#materialUnit').replaceChildren(...(uploaded?.units||[]).map((u,i)=>{const o=el('option',u.label);o.value=i;return o}));$('#materialUnit').value=unit;initReviewSections();$('#reviewSection').value=section;
 $('#materialStatus').textContent=uploaded?uploaded.name+' · '+uploaded.units.length+'单元 · '+uploaded.scope:'本工作区尚未添加材料。';$('#materialFile').value='';
}
function selectTask(id){
 if(busy||importing){$('#status').textContent='请先完成或取消当前操作，再切换工作区。';return}
 id=taskByID(id).id;
 if(id!==currentTask){taskSessions.set(currentTask,snapshotTask());currentTask=id;const snap=taskSessions.get(id)||{};
 uploaded=snap.uploaded||null;reviewFile=snap.reviewFile||null;reviewItems=snap.reviewItems||[];reviewResults=snap.reviewResults||new Map();history=snap.history||[];conversation=snap.conversation||[];
 $('#question').value=snap.question||'';$('#scope').value=snap.scope||'';$('#expertPerspective').value=snap.expert||'';activeExpert=snap.expert||'';$('#messages').replaceChildren();for(const t of conversation)renderTurn(t);restoreMaterialControls(snap.unit,snap.section);
 }
 for(const b of $('#taskNav').children)b.setAttribute('aria-current',b.dataset.task===currentTask?'page':'false');
 inspector(id==='library'?'evidence':'assistant');$('#taskTitle').textContent=taskByID(id).title;$('#taskSubtitle').textContent=taskByID(id).subtitle;
 $('#materialHeading').textContent=currentTask==='lesson'?'课堂字幕与批次审阅':currentTask==='report'?'报告上传与批次审阅':currentTask==='presentation'?'汇报上传与审阅':'添加材料与审阅范围';
 $('#taskAction').textContent=currentTask==='library'?'搜索专家证据':'向顾问提问';updateRubric();workspace?.refresh();
 $('.materialPanel').hidden=['overview','library','consistency'].includes(id);$('#workspaceUpload').hidden=['overview','library','consistency'].includes(id);
}
$('#taskAction').onclick=()=>{inspector();if(currentTask==='library'){$('#question').focus();return}$('#question').value=taskQuestion(currentTask);$('#question').focus()};
function projectChanged(){history=[];reviewItems=[];reviewResults.clear();for(const s of taskSessions.values()){s.history=[];s.reviewItems=[];s.reviewResults=new Map()}updateRubric();updateReviewPlan();workspace.refresh();$('#status').textContent='备赛信息已更新，原问答保留展示；材料需按新信息重新审阅。'}
$('#competitionTrack').onchange=projectChanged;$('#competitionEdition').onchange=projectChanged;$('#courseName').onchange=projectChanged;
$('#openBuiltinRules').onclick=async()=>{try{const r=rubricFor(ruleCatalog,$('#competitionTrack').value,currentTask),res=await api('/api/document-pdf',{id:'builtin:rules6'}),u=URL.createObjectURL(await res.blob());window.open(u+'#page='+(r?.part?.page||14),'_blank','noopener');setTimeout(()=>URL.revokeObjectURL(u),600000)}catch(e){$('#status').textContent=e.message}};
try{ruleCatalog=await json('/api/builtin',{})}catch(e){$('#rubricStatus').textContent=e.message}selectTask(currentTask);

function selectedSections(){const sections=reviewSections(uploaded),id=$('#reviewSection').value;if(id==='all')return sections;const start=sections.findIndex(s=>s.id===id);if(start<0)return [];let end=start+1;while(end<sections.length&&sections[end].level>sections[start].level)end++;return sections.slice(start,end)}
function initReviewSections(){const opts=[el('option','全文（自动分批）')];opts[0].value='all';for(const s of uploaded?reviewSections(uploaded):[]){const o=el('option','　'.repeat(Math.min(s.level-1,3))+s.title.slice(0,90));o.value=s.id;opts.push(o)}$('#reviewSection').replaceChildren(...opts);updateReviewPlan()}
function updateReviewPlan(){workspace?.refresh();if(!uploaded){$('#reviewCoverage').textContent='尚未添加材料。';return}const batches=reviewBatches(selectedSections()),done=batches.filter(b=>reviewResults.has(b.parts.map(p=>p.id).join('|'))).length;const empty=uploaded.units.filter(u=>!u.text.trim()).length;$('#reviewCoverage').textContent=`当前范围 ${batches.length} 批，已完成AI文字审阅 ${done} 批。每批单独调用所选API；取消后可继续未完成批次。`+(empty?` ${empty} 个单元没有文字层，不计为已审阅。`:'')+(uploaded.paragraphs?.some(p=>p.toc)?' 已排除检测到的目录条目。':'');}
$('#reviewSection').onchange=updateReviewPlan;
$('#reviewAll').onclick=async()=>{
 if(busy)return;if(!uploaded||uploaded.task!==currentTask){$('#materialStatus').textContent='请先为本任务添加材料。';return}
 const apiState=await json('/api/profiles',{action:'list'});renderProfiles(apiState);if(!ready){$('#settings').showModal();return}
 const batches=reviewBatches(selectedSections());if(!batches.length){$('#reviewCoverage').textContent='没有可审阅的正文；扫描页不能计为完成。';return}
 busy=true;lockProject(true);requestController=new AbortController();$('#cancelRequest').hidden=false;$('#send').disabled=true;$('#reviewAll').disabled=true;$('#materialFile').disabled=true;$('#reviewSection').disabled=true;$('#clearMaterial').disabled=true;companionState('reviewing');
 const context=reviewContext({edition:$('#competitionEdition').value,track:$('#competitionTrack').value,course:$('#courseName').value,task:currentTask},ruleCatalog);
 let completed=0;
 try{
  const evidenceResult=await json('/api/search',{q:($('#competitionTrack').value+'赛道 '+taskByID(currentTask).title+' 评分标准 修改建议').trim(),tag:$('#scope').value});showSources(evidenceResult.pages,evidenceResult.raw_ocr);showDocumentHits(evidenceResult.document_hits||[]);
  const evidence=JSON.stringify(buildEvidence(evidenceResult));
  for(let i=0;i<batches.length;i++){
   const batch=batches[i],key=batch.parts.map(p=>p.id).join('|');if(reviewResults.has(key)){completed++;continue}
   if(requestController.signal.aborted)throw Error('审阅已取消，未完成批次保留待审。');
   stage(`正在审阅第 ${i+1}/${batches.length} 批：${batch.section}`);
   const out=await chatRequest({question:reviewPrompt(taskQuestion(currentTask),context),evidence,history:[],material:JSON.stringify({file:uploaded.name,scope:uploaded.scope,batch:i+1,total_batches:batches.length,...batch}),require_full_material:true,stream:$('#streamOutput').checked},()=>{});
   if(out.answer_truncated||out.context?.material_truncated)throw Error('本批输入或输出不完整，未计为已审阅。');
   const items=parseReview(out.answer,batch);reviewResults.set(key,items);reviewItems.push(...items);completed++;
   const content='### '+batch.section+' · 第'+(i+1)+'批\n\n'+items.map(p=>'**'+p.id+'**\n\n'+p.comment+(p.replacement?'\n\n**建议替换文字**\n\n'+p.replacement:'')).join('\n\n');msg('assistant',content,{status:'reviewed_text'});updateReviewPlan();
  }
  companionState('done');$('#reviewCoverage').textContent=`本次选定范围 ${completed}/${batches.length} 批AI文字审阅完成。原文区间均已送达模型并返回对应记录；这不等于人工验收、图片审阅或真实性认证。`;
 }catch(e){companionState('error');msg('assistant',e.message,{status:'error'});updateReviewPlan();$('#reviewCoverage').textContent+=' 本次停止原因：'+e.message;}
 finally{clearInterval(stageTimer);requestController=null;busy=false;lockProject(false);$('#cancelRequest').hidden=true;$('#send').disabled=false;$('#reviewAll').disabled=false;$('#materialFile').disabled=false;$('#reviewSection').disabled=false;$('#clearMaterial').disabled=false;$('#status').textContent='分批审阅本轮结束；覆盖情况见材料区。'}
};

$('.portrait img').onclick=()=>{ $('#imageDialog img').src=$('.portrait img').src; $('#imageDialog p').textContent=(profile.persona==='male'?'原创男教师':'原创女教师')+' · 完整原图'; $('#imageDialog').showModal(); };

function selectedExpert(){return expertCatalog?.experts.find(e=>e.id===$('#expertPerspective').value)}
function showExpertInfo(id){const experts=id?expertCatalog?.experts.filter(e=>e.id===id):expertCatalog?.experts;const box=$('#expertContent');box.replaceChildren();for(const e of experts||[]){const card=el('section',undefined,'expertCard');card.append(el('h3',e.name+' · '+e.institution),el('p',e.public_bio));for(const [label,items] of [['研究方向',e.research_focus],['教学专长',e.teaching_focus],['可参考的咨询方向（AI归纳）',e.advisory_focus],['本库课程/主题',e.lecture_courses]])card.append(el('p',label+'：'+(items.length?items.join('、'):'未确认')));if(e.notes)card.append(el('p',e.notes,'fine'));card.append(el('p','资料核对：'+e.checked+'；不是专家本人或授权代言。','fine'));for(const s of e.sources){const a=el('a',s.title);a.href=s.url;a.target='_blank';a.rel='noopener noreferrer';card.append(a,el('p',s.supports,'fine'))}box.append(card)}$('#expertDialog').showModal()}
$('#expertInfo').onclick=()=>showExpertInfo($('#expertPerspective').value);$('#expertClose').onclick=()=>$('#expertDialog').close();
$('#expertPerspective').onchange=()=>{if(busy){$('#expertPerspective').value=activeExpert;return;}activeExpert=$('#expertPerspective').value;history=[];const e=selectedExpert();$('#expertLabel').textContent=e?'参考'+e.name+'的专业领域 · AI整理，非本人':'原创教师形象 · 综合指导';$('#status').textContent='已切换回答视角；之前对话保留展示，新问题不沿用旧视角上下文。'};
try{expertCatalog=await json('/api/experts',{});for(const e of expertCatalog.experts){const o=el('option',e.name+' · '+e.institution);o.value=e.id;$('#expertPerspective').append(o)}}catch(e){$('#expertLabel').textContent=e.message}

let voiceCatalog=null;
function renderVoiceEngine(){
 const engine=$('#voiceEngine').value,p=voiceCatalog?.providers?.[engine]||{};
 $('#voiceLocalFields').hidden=engine!=='qwen';$('#voiceRemoteFields').hidden=engine==='qwen';$('#voiceModelLabel').hidden=engine==='azure';
 $('#voiceBase').value=p.base||'';$('#voiceModel').value=p.model||'';$('#voiceFemale').value=p.female||'';$('#voiceMale').value=p.male||'';$('#voiceKey').value='';$('#voiceClearKey').checked=false;
 $('#voiceKeyStatus').textContent=p.has_key?'密钥已保存在系统凭据库；留空保留。':'尚未保存密钥。';
 const list=$('#voiceOptions');list.replaceChildren();for(const name of engine==='qwen'?(voiceCatalog?.speakers||[]):engine==='azure'?['zh-CN-XiaoxiaoNeural','zh-CN-XiaoyiNeural','zh-CN-YunxiNeural','zh-CN-YunyangNeural','zh-CN-YunjianNeural']:[]){const o=el('option');o.value=name;list.append(o)}
 $('#voiceChoiceHint').textContent=engine==='qwen'?'可选择9个预设声音：Serena、Uncle_Fu、Vivian、Dylan、Eric、Ryan、Aiden、Ono_Anna、Sohee。原生语言不同，中文优先试听中文音色。':engine==='azure'?'Azure F0 免费层每月50万字符（以微软当前规则为准）；需自行创建F0资源。填写语音合成地址，如 https://eastasia.tts.speech.microsoft.com，密钥仅保存在本机凭据库。不会自动升级付费。':'服务必须支持 POST /audio/speech、model/input/voice/response_format=wav；可填写本机 HTTP 地址或远程 HTTPS 地址。';
}
async function refreshVoice(){try{const out=await (await teacherSpeech.request({action:'status'})).json();voiceCatalog=out;$('#voiceDirectory').value=out.directory||'';$('#voiceEngine').value=out.engine;renderVoiceEngine();$('#voiceStatus').textContent=out.engine==='qwen'?(out.installed?'原版 Qwen3-TTS 语音包已就绪；首次试听加载模型。':out.error):'已选择远程语音引擎；请保存配置后试听，未自动调用。';}catch(e){$('#voiceStatus').textContent=e.message}}
$('#voiceOpen').onclick=()=>{$('#voiceDialog').showModal();refreshVoice()};
$('#voiceClose').onclick=()=>$('#voiceDialog').close();$('#voiceEngine').onchange=()=>{renderVoiceEngine();$('#voiceStatus').textContent='更改尚未生效，保存后再试听。'};
$('#voiceConnect').onclick=async()=>{teacherSpeech.stopEverywhere();try{await teacherSpeech.request({action:'configure',directory:$('#voiceDirectory').value.trim(),engine:$('#voiceEngine').value,provider:{base:$('#voiceBase').value.trim(),model:$('#voiceModel').value.trim(),female:$('#voiceFemale').value.trim(),male:$('#voiceMale').value.trim()},key:$('#voiceKey').value,clear_key:$('#voiceClearKey').checked});await refreshVoice()}catch(e){$('#voiceStatus').textContent=e.message}};
$('#voicePreview').onclick=()=>teacherSpeech.speak('你好，我是你的桌面教师。我们可以一起梳理课程的真实问题，让创新举措和学生成效形成清楚的证据链。');
$('#voiceStop').onclick=()=>teacherSpeech.stop();$('#voiceStopFloating').onclick=()=>teacherSpeech.stop();
$('#voiceUnload').onclick=async()=>{teacherSpeech.stop();try{await teacherSpeech.request({action:'unload'});$('#voiceStatus').textContent='语音模型已卸载，释放内存。'}catch(e){$('#voiceStatus').textContent=e.message}};
$('#voiceAuto').checked=localStorage.getItem('teacherVoiceAuto')==='true';$('#voiceAuto').onchange=()=>localStorage.setItem('teacherVoiceAuto',String($('#voiceAuto').checked));
window.addEventListener('pagehide',()=>teacherSpeech.stop());

try{nativeDesktop=(await json('/api/desktop',{action:'status'})).native}catch{}
await restoreBranch();
