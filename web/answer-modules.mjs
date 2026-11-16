import {renderMarkdown} from './markdown.mjs';
// Group only actual headings; preserve every source block and its order.
// With no headings, the answer stays one block rather than inventing sections.
export function renderAnswerModules(text){
 const parsed=renderMarkdown(text),out=document.createElement('div');out.className='answerModules';let card=null;
 for(const block of [...parsed.children]){
  if(!card||/^H[234]$/.test(block.tagName)){card=document.createElement('section');card.className='answerModule';out.append(card)}
  card.append(block);
 }
 return out;
}
