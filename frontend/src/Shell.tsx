import React, {useLayoutEffect, useRef, useState} from 'react';
import {createRoot} from 'react-dom/client';
import {flushSync} from 'react-dom';

type Nodes = Record<string, HTMLElement>;
function Slot({node, className=''}: {node?: HTMLElement; className?: string}) {
 const ref=useRef<HTMLDivElement>(null);
 useLayoutEffect(()=>{if(node&&ref.current)ref.current.append(node);return()=>{node?.remove()}},[node]);
 return <div className={'shellSlot '+className} ref={ref}/>;
}
function Shell({nodes:n}: {nodes:Nodes}) {
 const hub=useRef<HTMLDialogElement>(null),project=useRef<HTMLDialogElement>(null);
 const [compact,setCompact]=useState(false);
 const launch=(id:string)=>{hub.current?.close();document.getElementById(id)?.click()};
 return <>
  <header className="shellTopbar topbar"><button className="sidebarToggle" aria-label="展开或收起任务导航" aria-expanded={!compact} onClick={()=>setCompact(!compact)}>☰</button><Slot node={n.brand}/><span className="shellVersion">v0.5.0</span><div className="shellTopActions"><Slot node={n.memory}/><button id="newTeacherProject">＋ 新老师 / 新项目</button><button onClick={()=>project.current?.showModal()}>课程与赛道</button><button id="settingsCenterOpen" onClick={()=>hub.current?.showModal()}>设置与连接</button></div></header>
  <main className={'appShell redesignedShell'+(compact?' sidebarCollapsed':'')}>
   <aside className="persona"><div className="sidebarLabel">备赛工作区</div><Slot node={n.nav}/><section className="historyColumn"><Slot node={n.tasks}/></section><div className="teacherDock"><Slot node={n.portrait}/><div className="teacherDockTools"><Slot node={n.template}/><Slot node={n.showTeacher}/><button onClick={()=>launch('voiceOpen')}>声音设置</button></div></div><div className="sidebarFoot"><Slot node={n.counts}/><Slot node={n.browse}/></div></aside>
   <section className="workspace"><Slot node={n.heading}/><Slot node={n.tools}/><Slot node={n.tabs}/><Slot node={n.material}/><Slot node={n.content}/><Slot node={n.answers}/></section><Slot node={n.inspector}/>
  </main>
  <dialog ref={project} id="projectDialog"><div className="dialogHeader"><h2>课程与赛道</h2><button className="dialogClose" onClick={()=>project.current?.close()}>关闭 ×</button></div><div className="dialogContent"><Slot node={n.project}/></div></dialog>
  <dialog ref={hub} id="settingsCenter"><div className="dialogHeader"><h2>设置与连接</h2><button className="dialogClose" onClick={()=>hub.current?.close()}>关闭 ×</button></div><div className="dialogContent settingsGrid">
   <button onClick={()=>launch('settingsOpen')}><b>模型与资料库</b><span>多组 API、思考参数、本地 SQLite</span></button>
   <button onClick={()=>launch('webSearchOpen')}><b>联网搜索</b><span>10 种引擎、执行顺序、浏览器保底</span></button>
   <button onClick={()=>launch('voiceOpen')}><b>教师声音</b><span>语音接口、男女音色、试听</span></button>
   <button onClick={()=>launch('workspaceHistoryOpen')}><b>记录与迁移</b><span>工作台快照、导入与恢复</span></button>
   <button onClick={()=>launch('layoutReset')}><b>恢复默认布局</b><span>重置拖动调整过的面板尺寸</span></button>
  </div></dialog><div className="shellLegacyActions" hidden><Slot node={n.legacyActions}/><Slot node={n.rail}/></div>
 </>;
}
export function mountWorkbenchShell() {
 const get=(s:string)=>{const x=document.querySelector<HTMLElement>(s);if(!x)throw Error('Missing workbench slot: '+s);return x};
 const n:Nodes={brand:get('body>header>div:first-child'),memory:get('#memoryStatus'),nav:get('#taskNav'),tasks:get('.moduleTasks'),portrait:get('.portrait'),template:get('#teacherTemplate'),showTeacher:get('#showPet'),counts:get('.stats'),browse:get('#browse'),heading:get('.workbenchHeader'),tools:get('.workspaceTools'),tabs:get('.workspaceTabs'),material:get('.materialPanel'),content:get('#workspacePane'),answers:get('.conversation'),inspector:get('.inspector'),project:get('.projectPanel'),legacyActions:get('.headerActions'),rail:get('.navRail')};
 // Service-owned islands keep their DOM identity and handlers. React owns layout and navigation chrome.
 const expert=get('.expertPanel'),toolbar=get('.conversationToolbar'),details=document.createElement('details'),summary=document.createElement('summary');details.className='advisorOptions';summary.textContent='顾问角色与对话工具';details.append(summary,expert,toolbar);n.answers.prepend(details);
 n.project.setAttribute('open','');
 Object.values(n).forEach(x=>x.remove());document.querySelector('body>header')?.remove();document.querySelector('body>.appShell')?.remove();
 const host=document.createElement('div');host.id='workbenchRoot';document.body.prepend(host);document.body.classList.add('shell-v2');flushSync(()=>createRoot(host).render(<Shell nodes={n}/>));
}
