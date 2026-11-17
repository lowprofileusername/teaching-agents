import assert from 'node:assert/strict';
import {isBuilder,renderBuilderPlan} from '../web/builder.mjs';
assert(isBuilder('syllabus-architect'));
const dims=['有明确的问题导向','有明显的创新特色','体现课程思政特色','创新构建教学环境','关注技术应用于教学','注重创新成果的辐射'];
const p={kind:'syllabus',innovation_line:'以人机协同判断力培养为主线',logic_loop:'目标→内容→评价→证据',word_budget_total:0,duration_minutes:0,sections:[],timeline:[],demo_duration_minutes:0,demo_script:[],lesson_annotations:[],slides:[],
 syllabus:{course_name:'数据结构',course_nature:'专业核心课',credits_hours:'待补：学分/学时（拟议理论32/实践16）',audience:'计算机类专业大二学生（人数待补）',overview:'面向真实工程问题组织教学。',objectives:['掌握结构与算法（知识）','选择并验证数据结构（能力）','数字素养与负责任使用AI（素养）'],ideology_politics:'国产基础软件案例与工程伦理',prerequisites:'先修程序设计基础',content_schedule:[{unit:'线性表',hours:'拟议8学时',key_points:'存储与操作实现',ai_stage:'AI生成测试用例，学生核验边界'},{unit:'树与图',hours:'拟议16学时',key_points:'遍历与最短路径',ai_stage:'AI只追问不代答'}],assessment:'形成性拟议占40% + 终结性拟议占60%',resources:['教材（版次待补）','智慧教育平台资源','课程智能体（截图待补）'],ai_integration:'课前诊断/课中核验/课后个性化，教师终评',consistency:'与报告/实录/教案四项标注一一对应'},
 scoring_map:dims.map(d=>({dimension:d,section_ids:['syllabus:objectives'],evidence_needed:['待补：过程证据']})),
 evidence_needed:['待补：教务系统截图'],references:[{source_id:'builtin:rules7:scoring',use:'借鉴维度对照',notice:'省赛校赛口径'}],boundary:'待实施建议',warnings:['学分学时为待补占位：请按教务系统实际数据核实后填写。']};
const text=renderBuilderPlan(p);
for(const s of ['课程教学大纲方案','课程目标','课程思政目标','教学内容与学时安排','AI应用环节','课程评价与成绩构成','教材与教学资源','AI融合说明','一致性','6维评分对照','建议提醒（不阻断）','课程目标'])assert(text.includes(s),s);
assert(text.includes('syllabus:objectives')===false,'fixed ids should resolve to labels');
assert(text.includes('拟议8学时'));
console.log('PASS syllabus builder registration and rendering');