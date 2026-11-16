// Match whole social messages only: a greeting followed by a real question is
// still a model request. Uploading a document never initiates another review.
export function socialReply(question){
 const q=String(question).toLowerCase().replace(/[\s\p{P}\p{S}]/gu,'');
 if(/^(?:专家|老师|顾问|ai)?(?:你好|您好|你好呀|您好呀|早上好|下午好|晚上好|hi|hello)$/.test(q))return '你好，我在。你想讨论什么问题？';
 if(/^(?:谢谢|谢谢你|谢谢老师|谢谢专家|好的|好|收到|明白了|ok|thanks)$/.test(q))return '好的，需要继续时告诉我。';
 return '';
}
