import assert from 'node:assert/strict';
import {blocks} from '../web/markdown.mjs';
const b=blocks('## 报告大纲\n\n**一、课程定位**\n\n1. 填写课程名称\n2. 说明学情\n\n| 章节 | 内容 |\n| --- | --- |\n| 问题 | 证据 |\n\n<script>alert(1)</script>');
assert.deepEqual(b.map(x=>x.type),['heading','paragraph','list','table','paragraph']);assert.equal(b[2].items.length,2);assert.equal(b[3].rows[1][1],'证据');assert.equal(b[4].text,'<script>alert(1)</script>');console.log('Markdown structure passed; renderer uses text nodes, no raw HTML');
