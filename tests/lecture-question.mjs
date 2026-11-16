import assert from 'node:assert/strict';
import {lectureQuestion,lectureBatches} from '../web/lecture-question.mjs';
const context={video:{tag:'song',teacher:'宋晓菲',title:'酶工程',duration:60},current:15,entries:[{page_id:'song:R1',title:'问题',t0:0,t1:10},{page_id:'song:R2',title:'方法',t0:10,t1:30}],utterances:[{t0:0,t1:10,text:'原文1',version:'baseline'},{t0:10,t1:25,text:'原文2',version:'baseline'}]};
assert(lectureQuestion('这场讲座讲了什么',null).error);
let x=lectureQuestion('这场讲座讲了什么',context);assert.equal(x.tag,'song');assert.equal(x.range,'whole');assert(x.material.text.includes('原文1'));assert(x.material.text.includes('原文2'));
x=lectureQuestion('这页是什么意思',context);assert.equal(x.range,'current');assert(!x.material.text.includes('原文1'));assert(x.material.text.includes('原文2'));assert.equal(x.t0,10);
x=lectureQuestion('总结 00:05 到 00:09',context);assert.equal(x.t0,5);assert(!x.material.text.includes('原文2'));
assert(lectureQuestion('片段',context,'range',[55,70]).error);
const lines=['甲'.repeat(4000),'乙'.repeat(4000),'丙'.repeat(4000)],parts=lectureBatches(lines);assert.equal(parts.length,3);assert.equal(parts.join('\n'),lines.join('\n'));console.log('lecture identity, whole transcript, current page, time range and lossless batch boundaries passed');
assert.equal(lectureQuestion('解释当前页',context,'current').page_id,'song:R2');
assert(lectureQuestion('解释当前页',{...context,current:50},'current').error);
assert(lectureQuestion('总结片段',context,'range',[NaN,30]).error);
// A node selection retains decimal boundaries rather than rounding to display time.
const exact=lectureQuestion('总结所选时间段',context,'range',[10.125,29.875]);assert.equal(exact.t0,10.125);assert.equal(exact.t1,29.875);
