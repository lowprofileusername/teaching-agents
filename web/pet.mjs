import {TeacherAnimator} from './teacher-animation.mjs';
const token=location.hash.slice(1);history.replaceState(null,'',location.pathname);
const $=id=>document.getElementById(id),animator=new TeacherAnimator($('teacher'));
const names={idle:'就绪，点击我打开工作台',retrieving:'正在检索教学资料',reviewing:'正在审阅材料',thinking:'正在等待模型回答',answering:'正在生成建议',done:'本次回答已完成',error:'任务遇到问题，请查看工作台'};
let state={template:'female',posture:'active',state:'idle',native:false,direction:'left'},hover=false,speaking=false,generating=false,pendingStep=false,lastStep=0,previous=0,view=null,dragStart=null;
async function call(data){const r=await fetch('/api/desktop',{method:'POST',headers:{'Content-Type':'application/json','X-Session-Token':token},body:JSON.stringify(data)});const x=await r.json();if(!r.ok)throw Error(x.error);return x}
async function update(){try{state=await call({action:'status'});$('template').value=state.template;await animator.setTemplate(state.template);$('state').textContent=names[state.state]||names.idle;$('postureStatus').textContent=state.posture==='sit'?'坐下休息中，仍可继续对话':'自由活动；鼠标移上来会停下';for(const [id,mode] of [['sit','sit'],['rise','active'],['auto','auto']])$(id).setAttribute('aria-pressed',String(state.posture_mode===mode))}catch(e){$('state').textContent=e.message}}
$('template').onchange=async()=>{try{await call({action:'template',template:$('template').value});await update()}catch(e){$('state').textContent=e.message}};
const open=()=>call({action:'open'}).catch(e=>{$('state').textContent=e.message});$('teacher').onclick=()=>{if(view===null)open()};$('teacher').onkeydown=e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();open()}};$('open').onclick=open;$('hide').onclick=()=>call({action:'hide_pet'}).catch(e=>{$('state').textContent=e.message});
for(const [id,posture] of [['sit','sit'],['rise','active'],['auto','auto']])$(id).onclick=async()=>{try{await call({action:'posture',posture});await update()}catch(e){$('state').textContent=e.message}};
document.body.onmouseenter=()=>hover=true;document.body.onmouseleave=()=>hover=false;
const voiceChannel=new BroadcastChannel('desktop-teacher-voice');voiceChannel.onmessage=e=>{if(e.data?.type==='voice-state'){speaking=e.data.state==='speaking';generating=e.data.state==='generating';$('voiceState').textContent=e.data.text;document.body.dataset.speaking=String(speaking);$('stopVoice').hidden=!(speaking||generating);$('speechBubble').hidden=!(speaking||generating);$('speechBubble').textContent=speaking?'正在为你讲解…':'正在准备声音…'}};$('stopVoice').onclick=()=>voiceChannel.postMessage({type:'voice-stop'});
function frame(now){const elapsed=previous?now-previous:0;previous=now;const walking=view===null&&state.posture!=='sit'&&!hover&&!speaking&&!generating&&['idle','done','error'].includes(state.state)&&!document.hidden;animator.draw(elapsed,{sit:state.posture==='sit',walking,direction:state.direction,view});if(walking&&animator.motion.progress===0&&state.native&&!pendingStep&&now-lastStep>33){lastStep=now;pendingStep=true;call({action:'step'}).then(x=>state.direction=x.direction||state.direction).catch(()=>{}).finally(()=>pendingStep=false)}requestAnimationFrame(frame)}
await update();setInterval(update,1800);requestAnimationFrame(frame);

$('rotate').onclick=()=>{view=0;$('angle').hidden=false;$('angle').value='0';$('teacher').setAttribute('aria-label','拖动旋转教师形象；左右方向键切换视角')};
$('animate').onclick=()=>{view=null;$('angle').hidden=true;$('teacher').setAttribute('aria-label','打开备赛工作台')};
$('angle').oninput=()=>view=Number($('angle').value);
$('teacher').onpointerdown=e=>{if(view!==null){dragStart={x:e.clientX,view};$('teacher').setPointerCapture(e.pointerId)}};
$('teacher').onpointermove=e=>{if(dragStart){view=((dragStart.view+Math.round((e.clientX-dragStart.x)/10))%24+24)%24;$('angle').value=String(view)}};
$('teacher').onpointerup=$('teacher').onpointercancel=()=>dragStart=null;
$('teacher').addEventListener('keydown',e=>{if(view!==null&&['ArrowLeft','ArrowRight'].includes(e.key)){e.preventDefault();view=(view+(e.key==='ArrowRight'?1:23))%24;$('angle').value=String(view)}});
