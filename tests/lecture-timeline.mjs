import assert from 'node:assert/strict';
import {intervalAt,imageAt,audioLocalTime} from '../web/lecture-timeline.mjs';
const xs=[{page_id:'a',t0:0,t1:5},{page_id:'b',t0:5,t1:9},{page_id:'a',t0:12,t1:15}];assert.equal(intervalAt(xs,5).entry.page_id,'b');assert.equal(intervalAt(xs,10).entry,null);assert.equal(intervalAt(xs,12).entry.page_id,'a');assert.equal(intervalAt([...xs,{page_id:'b',t0:3,t1:8}],4).ambiguous,true);const frame={page_id:'a',t0:2,t1:4,priority:1,asset_id:'frame'};assert.equal(intervalAt([...xs,frame],3).entry.asset_id,'frame');assert.equal(imageAt(frame,[],3).asset_id,'frame');assert.equal(imageAt(xs[2],[{page_id:'a',asset_id:'early',source_seconds:1},{page_id:'a',asset_id:'revisit',source_seconds:13}],12.5).asset_id,'revisit');assert.equal(audioLocalTime({t0:120,t1:240,encoded_duration:120},150),30);assert.equal(audioLocalTime({t0:0,t1:1500,encoded_duration:1500},150),150);console.log('replay boundaries, revisits, gaps, overlapping frames and audio chunk offsets passed');
const {intervalSelection}=await import('../web/lecture-timeline.mjs');
assert.deepEqual(intervalSelection({t0:43.022,t1:73.504},425.79),[43.022,73.504]);
assert.equal(intervalSelection({t0:70,t1:60},100),null);
assert.equal(intervalSelection({t0:null,t1:30},100),null);
