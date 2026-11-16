"""Private stdio worker: no network listener, no cloud calls, no voice cloning.
stdout is JSONL only; models are loaded once, audio is returned in memory.
"""
import argparse, base64, contextlib, io, json, os, sys, time, wave


def emit(value):
    sys.stdout.write(json.dumps(value, ensure_ascii=False) + '\n')
    sys.stdout.flush()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--model', required=True)
    parser.add_argument('--backend', choices=['mlx', 'torch'], required=True)
    parser.add_argument('--seed', type=int)
    args = parser.parse_args()
    os.environ['HF_HUB_OFFLINE'] = '1'
    os.environ['TRANSFORMERS_OFFLINE'] = '1'
    started = time.monotonic()
    with contextlib.redirect_stdout(sys.stderr):
        import numpy as np
        if args.backend == 'mlx':
            from mlx_audio.tts.utils import load_model
            model = load_model(args.model)
        else:
            import torch
            from qwen_tts import Qwen3TTSModel
            device = 'cuda:0' if torch.cuda.is_available() else 'cpu'
            model = Qwen3TTSModel.from_pretrained(
                args.model, device_map=device,
                dtype=torch.bfloat16 if device.startswith('cuda') else torch.float32,
                attn_implementation='sdpa', local_files_only=True,
            )
    emit({'ready': True, 'backend': args.backend, 'load_seconds': time.monotonic()-started,
          'speakers': ['Serena', 'Uncle_Fu', 'Vivian', 'Dylan', 'Eric', 'Ryan', 'Aiden', 'Ono_Anna', 'Sohee']})
    for line in sys.stdin:
        try:
            request = json.loads(line)
            text = request['text'].strip()
            speaker = request['speaker']
            if speaker not in ('Serena', 'Uncle_Fu', 'Vivian', 'Dylan', 'Eric', 'Ryan', 'Aiden', 'Ono_Anna', 'Sohee') or not text or len(text) > 240:
                raise ValueError('invalid text or speaker')
            started = time.monotonic()
            if args.seed is not None:
                if args.backend == 'mlx':
                    import mlx.core as mx
                    mx.random.seed(args.seed)
                else:
                    torch.manual_seed(args.seed)
            with contextlib.redirect_stdout(sys.stderr):
                if args.backend == 'mlx':
                    pieces = []
                    for result in model.generate_custom_voice(text=text, speaker=speaker,
                            language='Auto', max_tokens=1536, temperature=0.7):
                        pieces.append(np.asarray(result.audio, dtype=np.float32).reshape(-1))
                        sample_rate = result.sample_rate
                    data = np.concatenate(pieces)
                else:
                    waves, sample_rate = model.generate_custom_voice(text=text, speaker=speaker,
                            language='Auto', max_new_tokens=1536)
                    data = np.asarray(waves[0], dtype=np.float32).reshape(-1)
            if not len(data) or not np.isfinite(data).all():
                raise ValueError('invalid generated audio')
            if len(data)/sample_rate > 130:
                raise ValueError('audio exceeded duration limit')
            output = io.BytesIO()
            with wave.open(output, 'wb') as wav:
                wav.setnchannels(1); wav.setsampwidth(2); wav.setframerate(sample_rate)
                wav.writeframes((np.clip(data, -1, 1)*32767).astype('<i2').tobytes())
            emit({'audio': base64.b64encode(output.getvalue()).decode('ascii'),
                  'speaker': speaker, 'sample_rate': sample_rate,
                  'audio_seconds': len(data)/sample_rate, 'seconds': time.monotonic()-started})
        except Exception as error:
            print(type(error).__name__+': '+str(error), file=sys.stderr, flush=True)
            emit({'error': '语音生成失败，请查看本地语音日志。'})

if __name__ == '__main__':
    try:
        main()
    except Exception as error:
        print(type(error).__name__+': '+str(error), file=sys.stderr, flush=True)
        emit({'error': '本地语音模型加载失败，请检查语音包是否完整。'})
        sys.exit(1)
