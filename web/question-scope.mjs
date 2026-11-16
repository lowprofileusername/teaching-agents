export function questionScope(question,document,selectedIndex=0){
 const explicit=question.match(/第\s*(\d+)\s*[页張张]/),deictic=/(这[一张个]*页|这张(?:ppt|幻灯片)?|本页|当前页|当前幻灯片|当前这页)/i.test(question),whole=/(整份|全部|所有|整体|整套|全稿|全文|全篇)/.test(question);
 if(!explicit&&(!deictic||whole))return null;
 if(document?.name?.toLowerCase().endsWith('.docx'))return{error:'Word目前按章节和段落定位，未获取排版页码。请在报告工作区选择相应章节或段落再审阅。'};
 if(!document?.units?.length)return{error:'还没有可定位的材料。请先添加文件并选中要讨论的页面。'};
 const i=explicit?document.units.findIndex(u=>u.no===Number(explicit[1])):Number(selectedIndex)||0;
 if(i<0||!document.units[i])return{error:'这份材料没有第'+explicit?.[1]+'页，请检查页码。'};
 const u=document.units[i];return{kind:'single_unit',index:i,label:u.label,material:{file:document.name,total_units:document.units.length,selected_unit:u.label,scope:'本轮仅此单元；不代表已审阅其他页。'+document.scope,text:u.text,text_truncated:false}};
}
