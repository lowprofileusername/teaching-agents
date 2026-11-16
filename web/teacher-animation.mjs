// 24 actual image poses per stride. Animation time is independent of RAF rate.
export const WALK_FRAMES=8, WALK_FRAME_MS=110;
export const wrapView=n=>((Math.round(n)%24)+24)%24;
export class TeacherMotion {
 constructor(frames=8,frameMS=110){this.frames=frames;this.frameMS=frameMS;this.progress=0;this.target=0;this.walkTime=0}
 tick(ms,sit,walking){this.target=sit?7:0;const dt=Math.max(0,Math.min(ms,100)),delta=dt/105;if(this.progress<this.target)this.progress=Math.min(this.target,this.progress+delta);else if(this.progress>this.target)this.progress=Math.max(this.target,this.progress-delta);if(walking&&this.progress===0)this.walkTime+=dt;return this.frame(walking)}
 frame(walking){if(this.progress>0||this.target===7)return {sheet:'sit',frame:Math.round(this.progress),pose:this.progress===7?'sitting':this.target===7?'sitting-down':'standing-up'};return {sheet:'walk',frame:walking?Math.floor(this.walkTime/this.frameMS)%this.frames:0,pose:walking?'walking':'standing'}}
}
// Measure source alpha once. Draw without stretching and anchor feet to a common
// baseline. Upper-torso centering avoids arm swings moving the entire person.
export class SpriteAtlas {
 constructor(image,columns,rows){this.image=image;this.columns=columns;this.rows=rows;this.cw=image.width/columns;this.ch=image.height/rows;this.frames=[];
 const c=document.createElement('canvas');c.width=image.width;c.height=image.height;const ctx=c.getContext('2d',{willReadFrequently:true});ctx.drawImage(image,0,0);const data=ctx.getImageData(0,0,c.width,c.height).data;
 const rowCuts=[0];
 for(let row=1;row<rows;row++){const expected=row*this.ch,lo=Math.max(0,Math.floor(expected-this.ch*.22)),hi=Math.min(c.height-1,Math.ceil(expected+this.ch*.22));let runStart=-1,best=null;
 for(let y=lo;y<=hi;y++){let occupied=0;for(let x=0;x<c.width;x++)if(data[(y*c.width+x)*4+3]>100)occupied++;if(occupied===0&&runStart<0)runStart=y;if((occupied>0||y===hi)&&runStart>=0){const end=occupied>0?y-1:y,run={start:runStart,end,length:end-runStart+1};if(!best||run.length>best.length)best=run;runStart=-1}}
 rowCuts.push(best?Math.round((best.start+best.end)/2):Math.round(expected));}rowCuts.push(c.height);
 for(let i=0;i<columns*rows;i++){const row=Math.floor(i/columns),ox=Math.round(i%columns*this.cw),oy=rowCuts[row],ch=rowCuts[row+1]-oy;let y0=ch,y1=0,x0=this.cw,x1=0;
 for(let y=0;y<ch;y++)for(let x=0;x<this.cw;x++)if(data[((oy+y)*c.width+ox+x)*4+3]>100){x0=Math.min(x0,x);x1=Math.max(x1,x);y0=Math.min(y0,y);y1=Math.max(y1,y)}
 let sum=0,count=0;for(let y=Math.round(y0+(y1-y0)*.22);y<y0+(y1-y0)*.5;y++)for(let x=x0;x<=x1;x++)if(data[((oy+y)*c.width+ox+x)*4+3]>100){sum+=x;count++}
 this.frames.push({ox,oy,ch,y0,y1,center:count?sum/count:this.cw/2,height:y1-y0+1});}
 this.height=Math.max(...this.frames.map(f=>f.height));}
 draw(ctx,index,w,h,mirror=false,normalizeHeight=false){const f=this.frames[index],scale=(h*.94)/(normalizeHeight?f.height:this.height);ctx.clearRect(0,0,w,h);ctx.save();if(mirror){ctx.translate(w,0);ctx.scale(-1,1)}ctx.drawImage(this.image,f.ox,f.oy,this.cw,f.ch,w/2-f.center*scale,h*.985-f.y1*scale,this.cw*scale,f.ch*scale);ctx.restore()}
}
export const femaleTurnOrder=[[0,0],[1,0],[1,1],[2,0],[1,2],[0,1],[2,1],[3,3],[2,2],[3,1],[0,2],[3,2],[4,1],[4,2],[2,3],[4,0],[3,0],[5,0],[4,3],[0,3],[1,3],[5,1],[5,2],[5,3]];
export const cardinalViews=template=>template==='female'?[0,5,10,19]:[0,6,12,18];
export class SpriteSequence {
 constructor(atlases,order){this.atlases=atlases;this.order=order;this.frames=order.map(([a,f])=>atlases[a].frames[f]);}
 draw(ctx,index,w,h,mirror=false,normalize=false){const [a,f]=this.order[index];this.atlases[a].draw(ctx,f,w,h,mirror,normalize)}
}
export class TeacherAnimator {
 constructor(canvas){this.canvas=canvas;this.ctx=canvas.getContext('2d');this.ctx.imageSmoothingQuality='high';this.motion=new TeacherMotion();this.template='';this.images={};this.ready=false;this.loadID=0}
 async setTemplate(template){if(this.template===template&&this.ready)return;this.template=template;this.ready=false;const id=++this.loadID;const images={};
 const load=async(name,cols,rows)=>{const image=new Image();image.src=`assets/${name}_sheet.png`;await image.decode();return new SpriteAtlas(image,cols,rows)};
 if(template==='female'){
 const [sit,walks,turns]=await Promise.all([load('female_sit',4,2),Promise.all(Array.from({length:4},(_,i)=>load(`female_walk_hd${i}`,2,2))),Promise.all(Array.from({length:6},(_,i)=>load(`female_turn_hd${i}`,2,2)))]);
 images.sit=sit;images.walk=new SpriteSequence(walks,Array.from({length:16},(_,i)=>[Math.floor(i/4),i%4]));images.turn=new SpriteSequence(turns,femaleTurnOrder);
 }else{[images.sit,images.walk,images.turn]=await Promise.all([load('male_sit',4,2),load('male_walk',4,2),load('male_turn24',6,4)])}
 if(id!==this.loadID)return;this.images=images;this.ready=true;this.motion=new TeacherMotion(template==='female'?16:8,template==='female'?75:110)
 }
 draw(ms,{sit=false,walking=false,direction='right',view=null}){if(!this.ready)return;let state=view===null?this.motion.tick(ms,sit,walking):{sheet:'turn',frame:wrapView(view),pose:'rotating'};if(state.pose==='standing')state={sheet:'turn',frame:cardinalViews(this.template)[direction==='left'?3:1],pose:'standing'};this.images[state.sheet].draw(this.ctx,state.frame,this.canvas.width,this.canvas.height,direction==='left'&&state.sheet==='walk',state.sheet!=='sit');this.canvas.dataset.pose=state.pose;this.canvas.dataset.frame=String(state.frame);this.canvas.dataset.template=this.template}
}
