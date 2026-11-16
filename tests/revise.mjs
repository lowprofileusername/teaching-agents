import assert from 'node:assert/strict';
import {parseRevise,applyRevisions,factRisk,wordCountReminder,revisePrompt} from '../web/document-review.mjs';
import {reviseExportEdits,reviseCounts} from '../web/revise.mjs';

const text='本课程存在若干问题。';
const batch={section:'第一章',parts:[{id:'p1@0:10',paragraph_id:'p1',start:0,end:10,text}]};
const ok={items:[{id:'p1@0:10',comment:'成效数据缺口，需用户补充前后测口径。',revisions:[{rid:'r1',start:3,end:5,delete:'存在',insert:'仍有',reason:'措辞更准确'}]}]};
const items=parseRevise(JSON.stringify(ok),batch);
assert.equal(items.length,1);
assert.equal(items[0].revisions.length,1);
assert.equal(applyRevisions(text,items[0].revisions,new Set(['r1'])),'本课程仍有若干问题。');
assert.equal(applyRevisions(text,items[0].revisions,new Set()),text);

// delete must match the original slice exactly
assert.throws(()=>parseRevise(JSON.stringify({items:[{id:'p1@0:10',comment:'批注',revisions:[{rid:'r1',start:3,end:5,delete:'不对',insert:'仍有',reason:'理由'}]}]}),batch),/delete与原文不一致/);
// overlapping spans rejected
assert.throws(()=>parseRevise(JSON.stringify({items:[{id:'p1@0:10',comment:'批注',revisions:[{rid:'r1',start:0,end:5,delete:'本课程存在',insert:'本课',reason:'理由'},{rid:'r2',start:3,end:5,delete:'存在',insert:'仍有',reason:'理由'}]}]}),batch),/重叠/);
// fabricated facts in insert rejected; placeholder allowed
assert.throws(()=>parseRevise(JSON.stringify({items:[{id:'p1@0:10',comment:'批注',revisions:[{rid:'r1',start:3,end:5,delete:'存在',insert:'提升32%',reason:'理由'}]}]}),batch),/虚构/);
assert.ok(factRisk('学生满意度提升32%'));
assert.ok(!factRisk('待补：需用户提供的真实证据'));
const ph=parseRevise(JSON.stringify({items:[{id:'p1@0:10',comment:'批注',revisions:[{rid:'r1',start:3,end:5,delete:'存在',insert:'待补：需用户提供的真实证据',reason:'无出处断言改占位'}]}]}),batch);
assert.equal(ph[0].revisions[0].insert,'待补：需用户提供的真实证据');
// pure insertion (start==end, delete empty) allowed
assert.ok(parseRevise(JSON.stringify({items:[{id:'p1@0:10',comment:'批注',revisions:[{rid:'r1',start:10,end:10,delete:'',insert:'（待补证据）',reason:'补占位'}]}]}),batch));
// comment-only part allowed; missing/extra ids rejected
assert.ok(parseRevise(JSON.stringify({items:[{id:'p1@0:10',comment:'建议保留',revisions:[]}]}),batch));
assert.throws(()=>parseRevise(JSON.stringify({items:[]}),batch),/漏审/);
assert.throws(()=>parseRevise(JSON.stringify({items:[{id:'p1@0:10',comment:'批注',revisions:[]},{id:'extra',comment:'批注',revisions:[]}]}),batch),/编号不匹配/);
// word count is reminder only
assert.match(wordCountReminder(4600),/仅提醒/);
assert.equal(wordCountReminder(3900),'');
assert.match(revisePrompt('任务',{}),/不得为凑4000字自动删减/);

// Word export bridge: part-relative offsets become paragraph-absolute
const doc={ext:'.docx',source_id:'src',paragraphs:[{id:'p1',text}]};
const edits=reviseExportEdits(doc,items,new Map([['r1',true]]));
assert.equal(edits[0].replacement,'本课程仍有若干问题。');
assert.deepEqual(edits[0].spans,[{start:3,end:5,delete:'存在',insert:'仍有',reason:'措辞更准确'}]);
assert.match(edits[0].comment,/前后测口径/);
assert.equal(reviseExportEdits(doc,items,new Map([['r1',false]]))[0].replacement,'');
assert.deepEqual(reviseExportEdits(doc,items,new Map([['r1',false]]))[0].spans,[]);
const batch2={section:'第一章',parts:[{id:'p1@5:10',paragraph_id:'p1',start:5,end:10,text:'若干问题。'}]};
const items2=parseRevise(JSON.stringify({items:[{id:'p1@5:10',comment:'批注',revisions:[{rid:'r2',start:0,end:2,delete:'若干',insert:'一些',reason:'措辞'}]}]}),batch2);
assert.equal(reviseExportEdits(doc,items2,new Map([['r2',true]]))[0].replacement,'本课程存在一些问题。');
assert.deepEqual(reviseExportEdits(doc,items2,new Map([['r2',true]]))[0].spans,[{start:5,end:7,delete:'若干',insert:'一些',reason:'措辞'}]);
assert.deepEqual(reviseCounts(items,new Map([['r1',true]])),{total:1,accepted:1});
assert.throws(()=>reviseExportEdits({ext:'.pdf'},items,new Map()),/DOCX原件/);
console.log('PASS revise protocol, accept/reject application, word-count reminder and Word export bridge');