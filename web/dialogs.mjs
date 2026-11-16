// Every modal has a permanently visible close action. Native dialog semantics
// provide focus trapping, Escape, and focus restoration; closing never cancels jobs.
export function installDialogChrome(){
 const closes={webSearchDialog:'webSearchClose',wordExportDialog:'wordExportClose',workspaceHistoryDialog:'workspaceHistoryClose',settings:'settingsClose',imageDialog:'imageClose',exportDialog:'exportClose',expertDialog:'expertClose',voiceDialog:'voiceClose',teacherProfileDialog:'teacherProfileClose'};
 for(const dialog of document.querySelectorAll('dialog')){
  const close=document.getElementById(closes[dialog.id]);if(!close)continue;
  const title=dialog.querySelector('h2')||Object.assign(document.createElement('h2'),{textContent:'原页预览'});
  title.id||=dialog.id+'Title';dialog.setAttribute('aria-labelledby',title.id);
  close.type='button';close.textContent='关闭 ×';close.setAttribute('aria-label','关闭'+title.textContent);close.classList.add('dialogClose');
  const header=document.createElement('header');header.className='dialogHeader';header.append(title,close);
  const content=document.createElement('div');content.className='dialogContent';content.append(...dialog.childNodes);dialog.append(header,content);
  close.onclick=()=>dialog.close();
  // Only a pointer sequence that both begins and ends on the backdrop closes.
  let outside=false;const isOutside=e=>{const r=dialog.getBoundingClientRect();return e.clientX<r.left||e.clientX>r.right||e.clientY<r.top||e.clientY>r.bottom};
  dialog.addEventListener('pointerdown',e=>outside=e.target===dialog&&isOutside(e));dialog.addEventListener('click',e=>{if(outside&&e.target===dialog&&isOutside(e))dialog.close();outside=false});
 }
 for(const detail of document.querySelectorAll('.projectPanel,.materialPanel,#modelSettings')){
  const b=document.createElement('button');b.type='button';b.className='collapsePanel';b.textContent='收起 ↑';b.setAttribute('aria-label','收起'+detail.querySelector('summary').textContent);b.onclick=()=>{detail.open=false;detail.querySelector('summary').focus()};detail.append(b);
 }
}
