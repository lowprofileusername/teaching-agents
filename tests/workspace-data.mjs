import assert from 'node:assert/strict';
import {subtitleCues,timelineAudit,numericMentions,slideStats} from '../web/workspace-data.mjs';
const cues=subtitleCues('1\n00:00:01,200 --> 00:00:03,400\n教师：为什么？\n\n2\n00:00:03.000 --> 00:00:05.000\n学生：因为需求变化。\n\n3\n00:00:40.000 --> 00:00:42.000\n反馈');
assert.equal(cues.length,3);assert.equal(cues[0].start,1.2);assert.equal(cues[0].text,'教师：为什么？');assert.equal(timelineAudit(cues).overlap,1);assert.equal(timelineAudit(cues).gaps.length,1);assert.equal(subtitleCues('没有时间戳的文本').length,0);
const d={name:'报告',units:[{no:1,label:'第1页',text:'2026年参与32人，通过率85%。'},{no:2,label:'第2页',text:''}]};assert.equal(numericMentions(d,'report').length,3);assert.equal(slideStats(d)[1].empty,true);
console.log('workspace timeline, provenance and slide metrics passed');
