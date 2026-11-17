import {createSpeech} from './speech.mjs';
import {TeacherAnimator,wrapView} from './teacher-animation.mjs';
import {PetSound} from './pet-sound.mjs';
import {createPetChat} from './pet-chat.mjs';
import {isDrag,classifyClick,clickCue,clickOpensMain,hoverDue,CLICK_BURST_MS} from './pet-gesture.mjs';
const token=location.hash.slice(1);history.replaceState(null,'',location.pathname);
const $=id=>document.getElementById(id),animator=new TeacherAnimator($('teacher'));
const sound=new PetSound({storage:localStorage});
const chat=createPetChat({token});document.body.append(chat.panel);
$('mmChat').onclick=()=>{closeMenu();chat.toggle()};
$('chatOpen').onclick=()=>{$('controls').classList.remove('expanded');chat.open()};
const names={idle:'就绪，可拖动人物或打开工作台',retrieving:'正在检索教学资料',reviewing:'正在审阅材料',thinking:'正在等待模型回答',answering:'正在生成建议',done:'本次回答已完成',error:'任务遇到问题，请查看工作台'};
let state={template:'female',posture:'active',state:'idle',native:false,direction:'left'},hover=false,speaking=false,generating=false,pendingStep=false,lastStep=0,previous=0,view=null,dragStart=null,windowDrag=null,movePending=null,moveFlight=null;
let lastHover=0,hoverTimer=null,burst=0,burstTimer=null,pointerStart=null,dragMoved=false,braced=false,prevState='';
async function call(data){const r=await fetch('/api/desktop',{method:'POST',headers:{'Content-Type':'application/json','X-Session-Token':token},body:JSON.stringify(data)});const x=await r.json();if(!r.ok)throw Error(x.error);return x}
function stateSound(s){if(s===prevState)return;prevState=s;if(s==='done')sound.play('celebrate');else if(s==='error')sound.play('concern');else if(s==='thinking')sound.play('focus');else if(s==='retrieving'||s==='reviewing')sound.play('notice')}
async function update(){try{state=await call({action:'status'});$('template').value=state.template;await animator.setTemplate(state.template);$('state').textContent=names[state.state]||names.idle;stateSound(state.state);$('postureStatus').textContent=state.posture==='sit'?'坐下休息中，仍可继续对话':'自由活动；鼠标碰到停步，移开继续走';for(const [id,mode] of [['sit','sit'],['rise','active'],['auto','auto']])$(id).setAttribute('aria-pressed',String(state.posture_mode===mode))}catch(e){$('state').textContent=e.message}}
$('template').onchange=async()=>{try{await call({action:'template',template:$('template').value});await update()}catch(e){$('state').textContent=e.message}};
const open=()=>call({action:'open'}).catch(e=>{$('state').textContent=e.message});
$('open').onclick=open;$('mmOpen').onclick=()=>{closeMenu();open()};
$('hide').onclick=()=>call({action:'hide_pet'}).catch(e=>{$('state').textContent=e.message});
$('mmHide').onclick=()=>{closeMenu();call({action:'hide_pet'}).catch(e=>{$('state').textContent=e.message})};
for(const [id,posture] of [['sit','sit'],['rise','active'],['auto','auto']])$(id).onclick=async()=>{try{await call({action:'posture',posture});await update()}catch(e){$('state').textContent=e.message}};
// Hover greets once per cooldown and never stops walking (MaClaw petHoverCooldown).
function greetHover(){const now=Date.now();if(!hoverDue(lastHover,now))return;lastHover=now;sound.play('notice');const b=$('hoverBubble');b.textContent='你好，继续备赛吗？右键打开菜单';b.hidden=false;clearTimeout(hoverTimer);hoverTimer=setTimeout(()=>{b.hidden=true},2600)}
document.body.onmouseenter=()=>{hover=true;greetHover()};
document.body.onmouseleave=()=>{hover=false;$('hoverBubble').hidden=true};
const voiceChannel=new BroadcastChannel('desktop-teacher-voice');voiceChannel.onmessage=e=>{if(e.data?.type==='voice-state'){speaking=e.data.state==='speaking';generating=e.data.state==='generating';sound.speaking=speaking;sound.generating=generating;if(speaking||generating)sound.stop();$('voiceState').textContent=e.data.text;document.body.dataset.speaking=String(speaking);$('stopVoice').hidden=!(speaking||generating);$('speechBubble').hidden=!(speaking||generating);$('speechBubble').textContent=speaking?'正在为你讲解…':'正在准备声音…'}};
$('stopVoice').onclick=()=>voiceChannel.postMessage({type:'voice-stop'});
// Hover pauses walking; moving the pointer away resumes it.
function canWalk(){return !windowDrag&&view===null&&state.posture!=='sit'&&!hover&&!speaking&&!generating&&!$('pin').checked&&['idle','done','error'].includes(state.state)&&!document.hidden}
async function movement(){if(pendingStep)return;pendingStep=true;try{const x=await call({action:'roam',active:canWalk(),speed:Number($('speed').value)});state.direction=x.direction||state.direction}catch{}finally{pendingStep=false}}
function frame(now){const elapsed=previous?now-previous:0;previous=now;const speed=Number($('speed').value);const walking=canWalk();animator.draw(elapsed*Math.max(.7,Math.min(1.6,speed/100)),{sit:state.posture==='sit',walking,direction:state.direction,view});if(walking)sound.step(now);requestAnimationFrame(frame)}
setInterval(movement,250);
document.body.addEventListener('mouseenter',()=>{hover=true;movement()});
document.body.addEventListener('mouseleave',()=>{hover=false;movement()});
document.addEventListener('visibilitychange',movement);window.addEventListener('blur',()=>{hover=false;movement()});
$('pin').onchange=()=>{movement();syncMenu()};$('speed').oninput=()=>{$('speedValue').textContent=$('speed').value+' px/s';movement()};
$('size').onchange=()=>call({action:'size',scale:Number($('size').value)}).catch(e=>$('state').textContent=e.message);
// Sound controls (MaClaw declarative cues + mute gating).
function syncMenu(){$('mmSound').textContent='音效：'+(sound.enabled?'开':'关');$('mmPin').textContent=$('pin').checked?'走动':'定住';$('sound').checked=sound.enabled}
$('sound').onchange=()=>{sound.setEnabled($('sound').checked);syncMenu()};
$('quiet').onchange=()=>{sound.quiet=$('quiet').checked;if(sound.quiet)sound.stop()};
$('mmSound').onclick=()=>{sound.setEnabled(!sound.enabled);syncMenu()};
$('mmPin').onclick=()=>{$('pin').checked=!$('pin').checked;movement();syncMenu();closeMenu()};
const voice=createSpeech({token,template:()=>state.template,onStatus:text=>{$('voiceState').textContent=text}});
$('testVoice').onclick=()=>voice.speak('你好，我是你的教学创新备赛顾问。可以打开工作台，继续上次的任务。');
$('voiceSettings').onclick=()=>call({action:'open_voice'}).catch(e=>$('voiceState').textContent=e.message);
// Right-click minimal menu; "更多选项" expands the full panel.
const menu=$('miniMenu');
function closeMenu(){menu.classList.remove('open')}
function openMenu(x,y){menu.classList.add('open');const mw=menu.offsetWidth||160,mh=menu.offsetHeight||180;menu.style.left=Math.max(2,Math.min(x,220-mw-2))+'px';menu.style.top=Math.max(2,Math.min(y,window.innerHeight-mh-2))+'px'}
document.body.addEventListener('contextmenu',e=>{e.preventDefault();sound.play('notice');openMenu(e.clientX,e.clientY)});
document.addEventListener('pointerdown',e=>{if(!e.target.closest('#miniMenu'))closeMenu()});
$('mmMore').onclick=()=>{closeMenu();$('controls').classList.add('expanded')};
$('collapse').onclick=()=>$('controls').classList.remove('expanded');
window.addEventListener('keydown',e=>{if(e.key==='Escape'){closeMenu();$('controls').classList.remove('expanded')}});
await update();setInterval(update,1800);requestAnimationFrame(frame);syncMenu();
// 360° rotation: drag or wheel cycles all 24 views (full circle).
$('rotate').onclick=()=>{view=0;$('teacher').style.setProperty('--wails-draggable','no-drag');$('angle').hidden=false;$('angle').value='0';$('teacher').setAttribute('aria-label','拖动或滚轮旋转教师形象，完整360°；左右方向键切换视角')};
$('animate').onclick=()=>{view=null;$('teacher').style.setProperty('--wails-draggable','no-drag');$('angle').hidden=true;$('teacher').setAttribute('aria-label','拖动移动桌面教师；右键打开菜单；回车打开工作台')};
$('angle').oninput=()=>view=Number($('angle').value);
$('teacher').addEventListener('wheel',e=>{if(view===null)return;e.preventDefault();view=wrapView(view+(e.deltaY>0?1:-1));$('angle').value=String(view)},{passive:false});
function sendDragMove(point){movePending=point;if(moveFlight)return moveFlight;moveFlight=(async()=>{await windowDrag?.ready;while(movePending){const p=movePending;movePending=null;await call({action:'drag',phase:'move',...p})}})().catch(e=>$('state').textContent=e.message).finally(()=>moveFlight=null);return moveFlight}
function startWindowDrag(e){if(windowDrag||e.button!==0||e.target.closest('button,select,input,#miniMenu,.controls'))return;e.preventDefault();hover=true;pointerStart={x:e.clientX,y:e.clientY};dragMoved=false;braced=false;windowDrag={pointer:e.pointerId,ready:call({action:'drag',phase:'begin',x:Math.round(e.screenX),y:Math.round(e.screenY)})};windowDrag.ready.catch(err=>$('state').textContent=err.message);e.currentTarget.setPointerCapture(e.pointerId)}
async function endWindowDrag(e){if(!windowDrag)return;const d=windowDrag;await sendDragMove({x:Math.round(e.screenX),y:Math.round(e.screenY)});try{await d.ready;await call({action:'drag',phase:'end',x:Math.round(e.screenX),y:Math.round(e.screenY)})}catch(err){$('state').textContent=err.message}finally{if(windowDrag===d)windowDrag=null;if(dragMoved)sound.play('release');movement()}}
function registerClick(){burst++;clearTimeout(burstTimer);burstTimer=setTimeout(()=>{const cls=classifyClick(burst);burst=0;sound.play(clickCue(cls));if(clickOpensMain(cls))open()},CLICK_BURST_MS)}
$('teacher').onpointerdown=e=>{if(view!==null){dragStart={x:e.clientX,view};$('teacher').setPointerCapture(e.pointerId)}else startWindowDrag(e)};
$('teacher').onpointermove=e=>{if(dragStart){view=wrapView(dragStart.view+Math.round((e.clientX-dragStart.x)/10));$('angle').value=String(view)}else if(windowDrag){if(!dragMoved&&pointerStart&&isDrag(e.clientX-pointerStart.x,e.clientY-pointerStart.y)){dragMoved=true;if(!braced){braced=true;sound.play('brace')}}sendDragMove({x:Math.round(e.screenX),y:Math.round(e.screenY)})}};
$('teacher').onpointerup=$('teacher').onpointercancel=e=>{if(dragStart){dragStart=null;return}const wasDrag=dragMoved;endWindowDrag(e);if(!wasDrag&&e.type==='pointerup')registerClick()};
$('teacher').onclick=()=>{};
$('teacher').onkeydown=e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();open()}};
const dragHeader=document.querySelector('.controls header');dragHeader.onpointerdown=startWindowDrag;dragHeader.onpointermove=e=>{if(windowDrag)sendDragMove({x:Math.round(e.screenX),y:Math.round(e.screenY)})};dragHeader.onpointerup=dragHeader.onpointercancel=endWindowDrag;
$('teacher').addEventListener('keydown',e=>{if(view!==null&&['ArrowLeft','ArrowRight'].includes(e.key)){e.preventDefault();view=wrapView(view+(e.key==='ArrowRight'?1:23));$('angle').value=String(view)}});