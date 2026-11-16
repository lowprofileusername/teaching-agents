"""Lossless inference-only Qwen3-TTS pack. Preserves every decoder/talker byte.
Never overwrite the source. Audio encoder is only needed for reference-audio
conditioning, which is excluded from this fixed-speaker application.
"""
import argparse, hashlib, json, shutil, struct
from pathlib import Path

def prune_safetensors(source, target):
    with source.open('rb') as f:
        size=struct.unpack('<Q',f.read(8))[0]; header=json.loads(f.read(size)); start=8+size
        retained={}; removed={}; offset=0; entries=[]
        for key, spec in header.items():
            if key=='__metadata__': retained[key]=spec;continue
            a,b=spec['data_offsets']; n=b-a
            if key.startswith('encoder.'):
                removed[key]=n;continue
            retained[key]={**spec,'data_offsets':[offset,offset+n]};offset+=n
            entries.append((key,a,n))
        data=json.dumps(retained,separators=(',',':')).encode();data+=b' '*((-len(data))%8)
        evidence=[]
        with target.open('wb') as out:
            out.write(struct.pack('<Q',len(data)));out.write(data)
            for key,a,n in entries:
                f.seek(start+a);h=hashlib.sha256();remaining=n
                while remaining:
                    chunk=f.read(min(remaining,4<<20));
                    if not chunk:raise ValueError('truncated source')
                    out.write(chunk);h.update(chunk);remaining-=len(chunk)
                evidence.append({'tensor':key,'bytes':n,'sha256':h.hexdigest()})
    return {'removed_encoder_bytes':sum(removed.values()),'retained_tensors':evidence}

def main():
    p=argparse.ArgumentParser();p.add_argument('source',type=Path);p.add_argument('target',type=Path);args=p.parse_args()
    if args.target.exists():raise ValueError('target must not exist')
    args.target.mkdir(parents=True)
    audit={}
    for source in args.source.rglob('*'):
        if not source.is_file():continue
        relative=source.relative_to(args.source);target=args.target/relative;target.parent.mkdir(parents=True,exist_ok=True)
        if str(relative)=='speech_tokenizer/model.safetensors':audit=prune_safetensors(source,target)
        elif str(relative)=='speech_tokenizer/config.json':
            config=json.loads(source.read_text());config.pop('encoder_config',None);target.write_text(json.dumps(config,indent=2))
        else:shutil.copy2(source,target)
    audit.update(source_bytes=sum(p.stat().st_size for p in args.source.rglob('*') if p.is_file()),
                 output_bytes=sum(p.stat().st_size for p in args.target.rglob('*') if p.is_file()),
                 mode='lossless decoder/talker preservation; fixed voices; no audio cloning')
    (args.target.parent/'prune-audit.json').write_text(json.dumps(audit,indent=2))
    print({k:v for k,v in audit.items() if k!='retained_tensors'})
if __name__=='__main__':main()
