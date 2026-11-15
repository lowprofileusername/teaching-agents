// Compatibility for browser runtimes without ES2025 Uint8Array encoding helpers.
if(!Uint8Array.prototype.toHex)Object.defineProperty(Uint8Array.prototype,'toHex',{value:function(){return Array.from(this,b=>b.toString(16).padStart(2,'0')).join('')},configurable:true,writable:true});
if(!Uint8Array.fromBase64)Uint8Array.fromBase64=(s)=>Uint8Array.from(atob(s.replace(/-/g,'+').replace(/_/g,'/')),c=>c.charCodeAt(0));
if(!Uint8Array.prototype.toBase64)Object.defineProperty(Uint8Array.prototype,'toBase64',{value:function(){let s='';for(let i=0;i<this.length;i+=32768)s+=String.fromCharCode(...this.subarray(i,i+32768));return btoa(s)},configurable:true,writable:true});
