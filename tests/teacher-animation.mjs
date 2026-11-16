import assert from 'node:assert/strict';
import {TeacherMotion,wrapView} from '../web/teacher-animation.mjs';
const m=new TeacherMotion();const frames=new Set();for(let i=0;i<8;i++)frames.add(m.tick(110,false,true).frame);assert.equal(frames.size,8);
let p;for(let i=0;i<8;i++)p=m.tick(105,true,false);assert.equal(p.pose,'sitting');assert.equal(p.frame,7);
assert.equal(m.tick(105,false,false).pose,'standing-up');
for(let i=0;i<8;i++)p=m.tick(105,false,false);assert.equal(p.pose,'standing');
for(let i=0;i<3;i++)m.tick(105,true,false);assert.equal(m.tick(105,false,false).pose,'standing-up');
console.log('walk, sit, stand and mid-transition reversal passed');

assert.equal(wrapView(-1),23);assert.equal(wrapView(24),0);

const hd=new TeacherMotion(16,75),hdFrames=new Set();for(let i=0;i<16;i++)hdFrames.add(hd.tick(75,false,true).frame);assert.equal(hdFrames.size,16);
