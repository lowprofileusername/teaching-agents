import assert from 'node:assert/strict';
import {isBuilder,renderBuilderPlan} from '../web/builder.mjs';
import {audioQualityLabel} from '../web/lecture-timeline.mjs';
import {buildEvidence} from '../web/evidence.mjs';
import {cleanTurn} from '../web/conversation.mjs';
assert(isBuilder('architect'));assert(isBuilder('video-planner'));assert(!isBuilder('report'));
const p={kind:'report',innovation_line:'建议主线',logic_loop:'问题到证据',word_budget_total:4000,sections:[{id:'s1',title:'AI伦理与治理',word_budget:4000,questions:['如何核验'],evidence_needed:['待补：过程记录']}],scoring_map:[{dimension:'技术应用',section_ids:['s1'],evidence_needed:['待补：记录']}],evidence_needed:['待补：基线'],references:[{source_id:'builtin:play:demo',use:'借鉴动作',notice:'迁移建议非专家本人观点'}],boundary:'非用户成果'};
let text=renderBuilderPlan(p);for(const s of ['4000','AI伦理与治理','必答问题','维评分对照','待补：基线','非专家本人观点'])assert(text.includes(s));
assert.deepEqual(cleanTurn({builder_plan:p}).builder_plan,p);
text=renderBuilderPlan({...p,kind:'lesson',duration_minutes:45,sections:[],timeline:[{id:'t1',start_minute:0,end_minute:45,stage:'核验',ai_action:'追问',student_evidence:'学生修订记录',report_measure:'核验修订'}]});assert(text.includes('0–45分'));assert(text.includes('学生修订记录'));
assert.equal(audioQualityLabel('encoded_decode_checked_not_listened'),'音频编码及解码检查通过，未做原声听辨');
const evidence=buildEvidence({document_hits:[{id:'builtin:play:demo',source_file:'knowledge/ai_playbook.json',origin:'来源老师',kind:'ai_migration'}]});assert.equal(evidence.document_evidence[0].origin,'来源老师');assert.equal(evidence.document_evidence[0].kind,'ai_migration');
console.log('PASS builder presentation, saved plan, evidence attribution and audio-status translation');

assert(isBuilder('presentation-architect'));
const pp={...p,kind:'presentation',sections:[],slides:[{id:'p1',page_no:1,title:'可核查问题',section:'创新内容',bullets:['待补：原始证据'],visual:'匿名截图',evidence_source:'builtin:play:demo',speaker_note:'拟议讲稿',duration_sec:715}],warnings:['时长建议，不阻断']};
text=renderBuilderPlan(pp);for(const s of ['汇报PPT方案','715 秒','第 1 页','拟议讲稿','建议提醒（不阻断）'])assert(text.includes(s));
text=renderBuilderPlan({...p,sections:[{...p.sections[0],draft:'拟议正文；待补：真实证据'}]});assert(text.includes('正文草稿'));assert(text.includes('拟议正文'));
