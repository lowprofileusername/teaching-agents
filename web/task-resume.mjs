// MaClaw-inspired unfinished-task detection. Recovery metadata is diagnostic
// evidence only: resuming restores context and never replays the interrupted
// operation. Side effects are classified so the UI can warn before any new
// mutation (MaClaw sideEffectStateForToolBatch / recoveryModeForSideEffect).
export function sideEffectState(session) {
  if (!session) return 'none';
  const hasMaterial = !!session.uploaded;
  const hasReview = (session.reviewItems || []).length > 0 || (session.reviewResults && session.reviewResults.size > 0);
  const hasPlan = (session.conversation || []).some(t => t && t.builder_plan);
  if (hasMaterial || hasReview || hasPlan) return 'local_committed';
  return 'none';
}
export function recoveryMode(sideEffect) { return sideEffect === 'none' ? 'resume_context' : 'requires_review'; }
export function detectUnfinished(session) {
  if (!session) return null;
  const turns = session.conversation || [];
  const interrupted = session.jobState === 'interrupted' || session.jobState === 'running'
    || turns.some(t => t && (t.status === 'generating' || t.status === 'interrupted'));
  if (!interrupted) return null;
  const lastUser = [...turns].reverse().find(t => t && t.role === 'user');
  const side = sideEffectState(session);
  return {
    interrupted: true,
    lastTask: String(lastUser?.text || session.question || '新任务').slice(0, 120),
    sideEffectState: side,
    recoveryMode: recoveryMode(side),
    summary: '上次任务在保存进度后中断。',
  };
}
export function resumeNotice(slot) {
  const base = '继续只恢复上下文，不会重试该操作。';
  if (slot.recoveryMode === 'requires_review') return '需要审阅：上次任务可能已修改工作区或产生外部副作用。' + base;
  return '这次中断前只做了检索/问答，没有修改工作区。' + base;
}
export function resumeContextNote(slot) {
  return `## 未完成任务恢复\n- 任务：${slot.lastTask}\n- ${slot.summary}\n- 中断的操作可能已开始但最终状态未知；不要仅因任务恢复就重复执行它。\n- ${resumeNotice(slot)}\n- 只继续这个未完成任务；不要重做已完成的工作。`;
}