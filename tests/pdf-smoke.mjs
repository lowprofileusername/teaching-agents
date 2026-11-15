import fs from 'node:fs';
// Exercise the compatibility path used by Node 22 and older browser engines.
Promise.try=undefined;
const pdfjs=await import('../web/lib/pdf.mjs');
const task=pdfjs.getDocument({data:new Uint8Array(fs.readFileSync(new URL('./pdf_fixture.pdf',import.meta.url))),isEvalSupported:false,useSystemFonts:true});
try{const doc=await task.promise;const pages=[];for(let n=1;n<=doc.numPages;n++){const p=await doc.getPage(n);pages.push((await p.getTextContent()).items.map(x=>x.str).join(' '));p.cleanup()}if(pages.length!==2||!pages[0].includes('TEST PAGE ONE')||!pages[1].includes('TEST PAGE TWO'))throw Error('PDF page extraction failed');console.log('Synthetic two-page PDF extraction and cleanup passed')}finally{await task.destroy()}
