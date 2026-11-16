// Reading takes priority over streaming. Each workspace owns its follow state.
export function createChatScroll(box,button,current){
 const states=new Map();const state=()=>{const id=current();if(!states.has(id))states.set(id,{follow:true,top:0});return states.get(id)};
 const refresh=()=>{button.hidden=state().follow;button.textContent='查看最新回答 ↓'};
 const pause=()=>{state().follow=false;state().top=box.scrollTop;refresh()};
 box.addEventListener('wheel',e=>{if(e.deltaY<0)pause()},{passive:true});box.addEventListener('touchstart',pause,{passive:true});
 box.addEventListener('keydown',e=>{if(['PageUp','Home','ArrowUp'].includes(e.key))pause()});
 box.addEventListener('pointerdown',e=>{if(e.target===box||e.target.closest('.answerModule,.modelReasoning'))pause()});
 box.addEventListener('scroll',()=>{state().top=box.scrollTop},{passive:true});
 box.addEventListener('toggle',e=>{if(e.target.matches('.modelReasoning')&&e.target.open)pause()},true);
 button.onclick=()=>{state().follow=true;box.scrollTop=box.scrollHeight;refresh()};
 return{paint(streaming){if(streaming&&state().follow)box.scrollTop=box.scrollHeight;refresh()},restore(){box.scrollTop=state().follow?box.scrollHeight:state().top;refresh()},pause};
}
