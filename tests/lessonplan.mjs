import assert from 'node:assert/strict';
import {isBuilder,renderBuilderPlan} from '../web/builder.mjs';
assert(isBuilder('lessonplan-architect'));
const dims=['有明确的问题导向','有明显的创新特色','体现课程思政特色','创新构建教学环境','关注技术应用于教学','注重创新成果的辐射'];
const p={kind:'lessonplan',innovation_line:'以人机协同判断力培养为主线',logic_loop:'目标→过程→评价→证据',word_budget_total:0,duration_minutes:0,sections:[],timeline:[],demo_duration_minutes:0,demo_script:[],lesson_annotations:[],slides:[],
 lesson_plan:{course_name:'数据结构',segment_topic:'二叉树遍历',segment_hours:'1学时（拟议45分钟）',audience:'大二学生（人数待补）',objectives:['掌握遍历（知识）','选择并验证策略（能力）','数字素养（素养）'],learner_analysis:'递归理解薄弱（前测待补）',key_points:'三种遍历',difficulties:'递归栈帧',stages:[{stage:'导入前测',minutes:'5分钟',teacher:'诊断',student:'前测',ai_action:'智能体学情画像',intent:'以学定教',ai_application:'课前诊断',ai_purpose:'定位薄弱点',ai_operation:'出题→采集→画像→审核',ai_expected:'拟形成分布（待实测）'},{stage:'探究核验',minutes:'25分钟',teacher:'追问',student:'核验AI',ai_action:'AI只追问不代答',intent:'逼学生说依据',ai_application:'课中核验',ai_purpose:'训练判断力',ai_operation:'提交→追问→修订→终评',ai_expected:'拟提升解释（待实测）'},{stage:'总结延伸',minutes:'15分钟',teacher:'点评',student:'迁移',ai_action:'',intent:'巩固'}],assessment:'过程性+迁移量规',resources:'课件/平台/智能体',ideology_politics:'工程伦理',consistency:'与大纲/报告/实录对应'},
 scoring_map:dims.map(d=>({dimension:d,section_ids:['lessonplan:process'],evidence_needed:['待补：过程证据']})),
 evidence_needed:['待补：前测数据'],references:[{source_id:'builtin:rules7:scoring',use:'借鉴维度',notice:'省赛校赛口径'}],boundary:'待实施建议',warnings:[]};
const text=renderBuilderPlan(p);
for(const s of ['节段教案方案','节段目标','学情分析','教学重点','教学难点','教学过程（分钟级）','AI应用四项标注','应用环节','设计目的','操作流程','预期效果','节段评价','思政融入点','一致性','6维评分对照','25分钟'])assert(text.includes(s),s);
assert(text.includes('lessonplan:process')===false,'fixed ids should resolve to labels');
console.log('PASS lessonplan builder registration and rendering');