// MaClaw-style declarative action sounds: host-synthesized {hz,ms} tone
// sequences, no audio files. Cues bind to pet actions; the host gates them
// against speech/generation so ambient tones never talk over the voice.
export const actionCues = {
  step: [{hz: 740, ms: 26}, {hz: 620, ms: 22}],
  greet: [{hz: 1046, ms: 36}, {hz: 1318, ms: 44}, {hz: 1568, ms: 56}],
  notice: [{hz: 1174, ms: 28}, {hz: 1396, ms: 36}],
  brace: [{hz: 349, ms: 40}],
  release: [{hz: 523, ms: 28}, {hz: 659, ms: 36}],
  rest: [{hz: 494, ms: 80}],
  celebrate: [{hz: 784, ms: 36}, {hz: 1046, ms: 36}, {hz: 1318, ms: 64}],
  concern: [{hz: 466, ms: 56}, {hz: 392, ms: 72}],
  focus: [{hz: 698, ms: 32}, {hz: 880, ms: 44}],
};
export function resolveCue(name) { return actionCues[name] || []; }
// Mute gating borrowed from MaClaw actionSoundAllowed: disabled/quiet mutes all;
// speaking or generating mutes ambient action cues (voice has priority).
export function actionSoundAllowed({enabled, quiet, speaking, generating}, cue) {
  if (!enabled || quiet) return false;
  if (speaking || generating) return false;
  return resolveCue(cue).length > 0;
}
export class PetSound {
  constructor({storage} = {}) {
    this.storage = storage;
    this.enabled = (storage?.getItem('petSound') ?? '1') === '1';
    this.quiet = false; this.speaking = false; this.generating = false;
    this.ctx = null; this.gen = 0; this.lastStep = 0;
  }
  setEnabled(on) { this.enabled = !!on; this.storage?.setItem('petSound', on ? '1' : '0'); if (!on) this.stop(); }
  stop() { this.gen++; }
  _context() {
    if (typeof window === 'undefined') return null;
    const AC = window.AudioContext || window.webkitAudioContext;
    if (!AC) return null;
    if (!this.ctx) this.ctx = new AC();
    if (this.ctx.state === 'suspended') this.ctx.resume().catch(() => {});
    return this.ctx;
  }
  play(cue) {
    if (!actionSoundAllowed(this, cue)) return;
    const tones = resolveCue(cue);
    const ctx = this._context();
    if (!ctx) return;
    const gen = ++this.gen;
    let t = ctx.currentTime + 0.01;
    for (const tone of tones) {
      if (gen !== this.gen) return;
      const osc = ctx.createOscillator(), gain = ctx.createGain();
      osc.type = 'sine'; osc.frequency.value = tone.hz;
      const dur = tone.ms / 1000;
      gain.gain.setValueAtTime(0.0001, t);
      gain.gain.linearRampToValueAtTime(0.16, t + 0.008);
      gain.gain.exponentialRampToValueAtTime(0.0001, t + dur);
      osc.connect(gain).connect(ctx.destination);
      osc.start(t); osc.stop(t + dur + 0.02);
      t += dur + 0.008;
    }
  }
  // Footstep throttle: at most one step cue per stride interval while walking.
  step(now, strideMS = 220) {
    if (now - this.lastStep < strideMS) return;
    this.lastStep = now;
    this.play('step');
  }
}