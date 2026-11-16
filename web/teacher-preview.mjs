import {TeacherAnimator,wrapView,cardinalViews} from './teacher-animation.mjs';
const $=id=>document.getElementById(id),canvas=$('figure'),animator=new TeacherAnimator(canvas);let view=0,mode='walk',last=0,drag=null;
async function load(){try{await animator.setTemplate($('who').value);document.querySelectorAll('[data-view]').forEach((b,i)=>b.dataset.view=cardinalViews($('who').value)[i])}catch(e){$('status').textContent='素材加载失败：'+e.message}}
$('who').onchange=load;
function setMode(value){mode=value;$('angles').hidden=mode!=='turn';for(const id of ['turn','walk','sit','stand'])$(id).setAttribute('aria-pressed',String(id===mode))}
for(const id of ['turn','walk','sit','stand'])$(id).onclick=()=>setMode(id);
function setView(v){view=wrapView(v);$('angle').value=view}
$('angle').oninput=()=>setView(Number($('angle').value));document.querySelectorAll('[data-view]').forEach(b=>b.onclick=()=>setView(Number(b.dataset.view)));
canvas.onpointerdown=e=>{setMode('turn');drag={x:e.clientX,view};canvas.setPointerCapture(e.pointerId)};canvas.onpointermove=e=>{if(drag)setView(drag.view+(e.clientX-drag.x)/12)};canvas.onpointerup=canvas.onpointercancel=()=>drag=null;
canvas.onkeydown=e=>{if(['ArrowLeft','ArrowRight'].includes(e.key)){e.preventDefault();setMode('turn');setView(view+(e.key==='ArrowLeft'?-1:1))}};
function tick(now){animator.draw(last?now-last:0,{view:mode==='turn'?view:null,sit:mode==='sit',walking:mode==='walk'});last=now;if(animator.ready)$('status').textContent=mode==='turn'?`环绕视角 ${view+1} / 24`:({walking:$('who').value==='female'?'高清分组16帧步态':'原版8帧步态',sitting:'坐下休息',standing:'站立', 'sitting-down':'正在坐下','standing-up':'正在起身'}[canvas.dataset.pose]||'');requestAnimationFrame(tick)}
await load();requestAnimationFrame(tick);
