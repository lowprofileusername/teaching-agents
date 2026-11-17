// Isolated desktop-web UI acceptance. Uses a deterministic local mock, never a
// user's API or lecture DB. Arguments: server.log artifact-dir.
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const fs=require('fs'),path=require('path'),assert=require('assert/strict'),http=require('http'),cp=require('child_process');
(async()=>{
 const qa=process.argv[3];assert(qa&&process.argv[2]);fs.mkdirSync(qa,{recursive:true});
 const url=fs.readFileSync(process.argv[2],'utf8').split('\n').find(s=>s.startsWith('http')).trim();const [base,token]=url.split('/#');
 const post=async(route,data)=>{const r=await fetch(base+route,{method:'POST',headers:{'X-Session-Token':token,'Content-Type':'application/json'},body:JSON.stringify(data)});const x=await r.json();return {status:r.status,...x}};
 const fixture=path.join(qa,'revise-fixture.docx');cp.execFileSync('python3',[path.join(__dirname,'revise-fixture.py'),'create',fixture]);
 let planMode='valid';const calls=[],rows=[],errors=[];let browser,mock,p;
 const record=(name,detail)=>{rows.push({name,status:'PASS',detail});console.log('PASS '+name)};
 const dims=JSON.parse(fs.readFileSync(path.join(__dirname,'../knowledge/rules_7_ai_2026.json'))).scoring.report_dimensions_ai_track.map(d=>d.name);
 const parseMsg=(messages,marker)=>{const m=messages.find(m=>typeof m.content==='string'&&m.content.startsWith(marker));return m?JSON.parse(m.content.slice(m.content.indexOf('\n')+1)):null};
 mock=http.createServer(async(req,res)=>{try{let b='';for await(const c of req)b+=c;const x=JSON.parse(b),sys=x.messages[0].content;calls.push({kind:sys.includes('修订+批注')?'revise':sys.includes('innovation_line')?'generate':'structure'});let answer;
  if(sys.includes('innovation_line')){
   const ev=parseMsg(x.messages,'本轮检索资料'),sources=ev.sources||ev.document_evidence,ref=sources.find(s=>s.id==='builtin:aideep:lijing_ai_track');assert(ref,'priority source reached model');
   const report=sys.includes('本轮depth=full'),n=planMode==='soft'?31:3,seconds=planMode==='over'?721:planMode==='soft'?715:700;
   const plan={kind:report?'report':'presentation',innovation_line:'拟用核验任务连接问题与证据',logic_loop:'问题到目标到行动到证据',word_budget_total:report?4000:0,duration_minutes:0,sections:report?[{id:'s1',title:'课程问题与举措',word_budget:3200,questions:['问题如何溯源'],evidence_needed:['待补：学情记录'],draft:'拟以学情记录识别问题，设计可核验任务并保留修订过程。待补：课程原始过程材料。'},{id:'s2',title:'课程思政与责任治理',word_budget:800,questions:['如何负责任地使用工具'],evidence_needed:['待补：治理规则'],draft:'拟在专业活动中落实责任边界。待补：实际规则及评价记录。'}]:[],scoring_map:dims.map(d=>({dimension:d,section_ids:[report?'s1':'p1'],evidence_needed:['待补：对应过程材料']})),evidence_needed:['待补：前后同口径证据'],timeline:[],demo_duration_minutes:0,demo_script:[],lesson_annotations:[],references:[{source_id:ref.id,use:'借鉴主映射并补齐成效证据',notice:'mock'}],boundary:'待实施方案'};
   if(!report)plan.slides=Array.from({length:n},(_,i)=>({id:'p'+(i+1),page_no:i+1,title:'以核验过程支撑教学判断',section:'创新内容',bullets:['待补：匿名学生任务材料'],visual:'匿名化过程截图',evidence_source:ref.id,speaker_note:'拟说明问题与举措的对应关系',duration_sec:Math.floor(seconds/n)+(i===n-1?seconds%n:0)}));
   answer=JSON.stringify(plan);
  }else if(sys.includes('修订+批注')){
   const mat=parseMsg(x.messages,'以下是用户上传材料');assert(mat?.parts);const items=mat.parts.map(part=>{
    const revisions=[];const add=(word,insert,rid)=>{const i=part.text.indexOf(word);if(i>=0)revisions.push({rid:rid==='complex'?'a':rid,start:Array.from(part.text.slice(0,i)).length,end:Array.from(part.text.slice(0,i+word.length)).length,delete:word,insert,reason:'测试措辞修订'})};
    if(part.text.includes('本课程非常')){add('非常','重点','a');add('问题','难题','b');add('建议开展讨论','拟组织讨论','reject')}
    if(part.text.includes('复杂段落'))add('建议','拟议','complex');
    return {id:part.id,comment:'待补：真实学情与评价口径；复杂段落保留图形。',revisions};});answer=JSON.stringify({items});
  }else answer=JSON.stringify({overview:'测试材料结构可继续逐段审阅。',issues:[]});
  res.setHeader('Content-Type','application/json');res.end(JSON.stringify({choices:[{message:{content:answer},finish_reason:'stop'}]}));
 }catch(e){errors.push('mock: '+e.stack);res.statusCode=500;res.end(JSON.stringify({error:e.message}))}});
 await new Promise(r=>mock.listen(0,'127.0.0.1',r));
 try{
  const saved=await post('/api/profiles',{action:'save',name:'v050 UI mock only',base:'http://127.0.0.1:'+mock.address().port+'/v1',model:'mock',contextWindow:32768,maxTokens:4096,clearKey:true});assert.equal(saved.status,200,JSON.stringify(saved));
  browser=await chromium.launch({executablePath:process.env.CHROME_BIN||undefined,headless:true});p=await browser.newPage({viewport:{width:1500,height:1050}});p.on('pageerror',e=>errors.push(e.stack));
  await p.goto(url);await p.waitForFunction(()=>document.body.dataset.ready==='true');await p.locator('#taskNav [data-task=report]').click();await p.locator('#materialFile').setInputFiles(fixture);await p.waitForFunction(()=>document.querySelector('#materialStatus').textContent.startsWith('revise-fixture.docx ·'));
  await p.locator('#materialTab').click();const mp=p.locator('.materialPanel');if(!await mp.evaluate(e=>e.open))await mp.locator('summary').click();
  await p.locator('#reviewRevise').click();await p.locator('#reviseDialog').waitFor({state:'visible',timeout:60000});
  assert((await p.locator('#reviseWordCount').textContent()).includes('仅提醒'));
  const checks=p.locator('#reviseList .reviseSpan input');assert.equal(await checks.count(),4);await checks.nth(0).check();await checks.nth(1).check();await checks.nth(2).check();await checks.nth(2).uncheck();await checks.nth(3).check();
  assert((await p.locator('#reviseStatus').textContent()).includes('已接受 3 条'));await p.screenshot({path:path.join(qa,'revise-panel.png')});record('修订面板与字数提醒','四条修订，接受三条拒绝一条；超过四千字不阻断');
  await p.locator('#reviseExportWord').click();await p.locator('#wordExportDialog').waitFor({state:'visible'});await p.locator('#wordExportDirectory').fill(qa);
  const resultPromise=p.waitForResponse(r=>r.url().includes('/api/docx-review-export'));await p.locator('#wordExportSave').click();const ex=await (await resultPromise).json();assert(ex.path,JSON.stringify(ex));await p.waitForFunction(()=>document.querySelector('#wordExportStatus').textContent.includes('已保存：'));
  assert((await p.locator('#wordExportStatus').textContent()).includes('字级修订 2 处'));assert(ex.result.warnings.length>0);await p.screenshot({path:path.join(qa,'word-export.png')});
  const ooxml=JSON.parse(cp.execFileSync('python3',[path.join(__dirname,'revise-fixture.py'),'verify',fixture,ex.path],{encoding:'utf8'}));record('Word字级写回与降级批注',ooxml);fs.writeFileSync(path.join(qa,'export-result.json'),JSON.stringify(ex,null,2));
  await p.locator('#wordExportClose').click();
  const chooseTrack=async t=>{await p.getByRole('button',{name:'课程与赛道',exact:true}).click();await p.locator('#competitionTrack').selectOption(t);await p.locator('#projectDialog .dialogClose').click()};await chooseTrack('人工智能');
  await p.locator('#taskNav [data-task=presentation]').click();await p.locator('#assistantTab').click();if(!await p.locator('.advisorOptions').evaluate(e=>e.open))await p.locator('.advisorOptions>summary').click();await p.locator('#specialistRole').selectOption('presentation-architect');
  const ask=async q=>{await p.locator('#question').fill(q);const response=p.waitForResponse(r=>r.url().includes('/api/chat'));await p.locator('#send').click();const res=await response;const data=await res.json();await p.waitForFunction(()=>!document.querySelector('#send').disabled);return {status:res.status(),...data}};
  let r=await ask('请生成汇报PPT方案');assert.equal(r.status,200,JSON.stringify(r));assert((await p.locator('#messages').textContent()).includes('汇报PPT方案'));await p.getByRole('heading',{name:'第 1 页 · 以核验过程支撑教学判断',exact:true}).last().scrollIntoViewIfNeeded();await p.screenshot({path:path.join(qa,'presentation-plan.png')});record('PPT生成与渲染','三页方案及来源、讲稿、时间');
  planMode='over';r=await ask('重新生成汇报方案');assert.equal(r.status,502);assert(r.error.includes('720'));record('PPT 721秒拒绝',r.error);
  planMode='soft';r=await ask('重新生成汇报方案');assert.equal(r.status,200,JSON.stringify(r));assert.equal(r.plan.slides.length,31);assert.equal(r.plan.warnings.length,3);record('PPT建议性限制不阻断','31页、715秒、不使用3+6+3仍通过，并有三条提醒');await p.screenshot({path:path.join(qa,'presentation-warnings.png')});
  planMode='valid';await p.locator('#specialistRole').selectOption('architect');await p.locator('#builderDepth').selectOption('full');r=await ask('请逐段扩写报告');assert.equal(r.status,200,JSON.stringify(r));assert(r.plan.sections.every(s=>s.draft));assert((await p.locator('#messages').textContent()).includes('正文草稿'));record('报告逐段扩写入口','depth=full实际送达，正文草稿可见');
  for(const [track,names] of [['新工科',['刘学莘','熊芸']],['新农科',['向妙莲']],['实验教学',['刘战合']]]){
   r=await post('/api/search',{q:track+' 报告打法'});assert.equal(r.status,200);for(const name of names)assert(r.document_hits.some(s=>s.kind==='track_migration'&&s.origin.includes(name)),track+name);record('赛道检索 '+track,names.join('、'));
  }
  assert.deepEqual(errors,[]);fs.writeFileSync(path.join(qa,'ui-results.json'),JSON.stringify({status:'PASS',rows,export:ex.path,screenshots:['revise-panel.png','word-export.png','presentation-plan.png','presentation-warnings.png'],provider:'deterministic local mock; UI and OOXML verification, not actual Word redline rendering',calls,errors},null,2));
 }catch(e){if(p)await p.screenshot({path:path.join(qa,'failure.png')}).catch(()=>{});fs.writeFileSync(path.join(qa,'ui-results.json'),JSON.stringify({status:'FAIL',rows,error:e.stack,errors,calls},null,2));throw e}finally{await browser?.close();mock.close();}
})().catch(e=>{console.error(e);process.exit(1)});
