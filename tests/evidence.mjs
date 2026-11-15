import assert from 'node:assert/strict';import {buildEvidence} from '../web/evidence.mjs';
const r={pages:Array.from({length:8},(_,i)=>({id:`${i%2?'lijing':'songxiaofei'}:R${i}`,teacher:i%2?'李静':'宋晓菲',page_text:'教学问题'.repeat(5000),occurrences:Array(400).fill({t0:0,t1:1}),utterances:Array(200).fill({text:'原话'.repeat(1000),version:'ASR未听校'}),official_pages:[{id:'official'+i,title:'官方课件',pdf_page_no:i+1,direct_text:'官方文字'.repeat(500)}],analysis:'x'.repeat(200000)})),document_hits:Array.from({length:6},(_,i)=>({document_id:'doc'+i,text:'报告'.repeat(20000)}))};
const e=buildEvidence(r);assert.equal(e.pages.length,8);assert.equal(e.document_evidence.length,6);assert.equal(e.pages.filter(p=>p.teacher==='李静').length,4);assert.ok(Buffer.byteLength(JSON.stringify(e))<170000);assert.equal(e.pages[0].occurrences.length,3);console.log('Evidence payload keeps every selected page and report within request limits');

assert.equal(e.pages[0].official_pages[0].id,'official0');

const c=buildEvidence({pages:[{id:'song:P12',source_format:'claude',page_text:'页面原文',research_analysis_unverified:'来源库研究判断',utterances:[]}]});assert.equal(c.pages[0].page_facts,'页面原文');assert.equal(c.pages[0].research_analysis_unverified,'来源库研究判断');assert.equal(c.pages[0].source_format,'claude');assert.ok(!c.pages[0].page_facts.includes('研究判断'));
