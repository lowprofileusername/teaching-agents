import {build} from 'esbuild';
await build({entryPoints:['src/Shell.tsx'],bundle:true,minify:true,format:'esm',target:['es2022'],outfile:'../web/shell.mjs',legalComments:'linked',define:{'process.env.NODE_ENV':'"production"'}});
