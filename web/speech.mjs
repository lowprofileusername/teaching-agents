// Natural TTS only: no implicit browser/system-voice fallback.
export function speechChunks(markdown,limit=160){
 const plain=String(markdown).replace(/```[\s\S]*?```/g,'代码内容请查看工作台。').replace(/!\[[^\]]*\]\([^)]*\)/g,'').replace(/\[([^\]]+)\]\([^)]*\)/g,'$1').replace(/\[[a-zA-Z0-9_]+:[RP]\d+\]/g,'').replace(/\[(\d+)\]/g,'').replace(/https?:\/\/\S+/g,'').replace(/^[\s#>*|\-]+/gm,'').replace(/[*`_]/g,'').trim();
 const pieces=plain.match(/[^。！？!?\n]+[。！？!?\n]?/gu)||[];const chunks=[];let buffer='';
 for(const piece of pieces){for(let chars=Array.from(piece);chars.length;){const part=chars.splice(0,limit).join('');if(Array.from(buffer+part).length>limit){if(buffer.trim())chunks.push(buffer.trim());buffer=''}buffer+=part}}
 if(buffer.trim())chunks.push(buffer.trim());return chunks;
}
export function createSpeech({token,template,onStatus=()=>{}}){
 let controller=null,player=null,url=null,run=0,settle=null;
 const channel=typeof BroadcastChannel!=='undefined'?new BroadcastChannel('desktop-teacher-voice'):null;
 function signal(state,text){onStatus(text);channel?.postMessage({type:'voice-state',state,text})}
 function stop(){run++;controller?.abort();controller=null;if(player){player.pause();player.src='';player=null}settle?.();settle=null;if(url){URL.revokeObjectURL(url);url=null}signal('idle','朗读已停止')}
 channel && (channel.onmessage=e=>{if(e.data?.type==='voice-stop')stop()});
 async function request(data,signal){const r=await fetch('/api/local-voice',{method:'POST',signal,headers:{'Content-Type':'application/json','X-Session-Token':token},body:JSON.stringify(data)});if(!r.ok){const e=await r.json();throw Error(e.error||'本地语音请求失败')}return r}
 async function speak(text){
  stop();const mine=run,voiceTemplate=template(),chunks=speechChunks(text);if(!chunks.length)return;
  channel?.postMessage({type:'voice-stop'});
  controller=new AbortController();const ctrl=controller;
  try{const config=await (await request({action:'status'},ctrl.signal)).json();const voice=config.providers[config.engine][voiceTemplate];for(let i=0;i<chunks.length;i++){
   if(mine!==run)return;
   signal('generating',`正在生成 ${voice} 语音（${i+1}/${chunks.length}）；本地首次加载或远端服务可能较慢…`);
   const timer=setTimeout(()=>ctrl.abort(),300000);let blob;
   try{blob=await (await request({action:'synthesize',template:voiceTemplate,text:chunks[i]},ctrl.signal)).blob()}finally{clearTimeout(timer)}
   if(mine!==run)return;
   url=URL.createObjectURL(blob);player=new Audio(url);player.playbackRate=Number(document.getElementById('voiceRate')?.value||1);
   signal('speaking',`正在朗读（${i+1}/${chunks.length}）`);
   const p=player;await new Promise((resolve,reject)=>{settle=resolve;p.onended=resolve;p.onerror=()=>reject(Error('音频播放失败'));p.play().catch(reject)});settle=null;
   if(mine!==run)return;
   URL.revokeObjectURL(url);url=null;player=null;
  }if(mine===run)signal('idle','朗读完成');
  }catch(e){if(mine===run)signal('error',e.name==='AbortError'?'语音已取消或生成超时':e.message)}finally{if(mine===run){controller=null;player?.pause();player=null;if(url)URL.revokeObjectURL(url);url=null}}
 }
 function button(text){const b=document.createElement('button');b.type='button';b.textContent='教师音色朗读';b.onclick=()=>speak(text);return b}
 return {speak,stop,button,request,stopEverywhere(){stop();channel?.postMessage({type:'voice-stop'})}};
}
