export function showTeacherProfile(template,selectTask){
 const male=template==='male',name=male?'原创男教师':'原创女教师',dialog=document.querySelector('#teacherProfileDialog');
 dialog.querySelector('h2').textContent=name+' · 备赛工作伙伴';dialog.querySelector('img').src=male?'assets/original_male_teacher.png':'assets/original_teacher.png';dialog.querySelector('img').alt=name;
 dialog.querySelector('.teacherBio').textContent=`${male?'他':'她'}是原创虚构的高校教师 AI 形象，作为你的教学创新备赛工作伙伴。不对应现实教师，也不是任何专家本人的数字分身。两种形象使用相同的工作台能力；选择专家视角后，会参考对应专家的公开专长与实际检索到的讲座资料。`;
 for(const button of dialog.querySelectorAll('[data-profile-task]'))button.onclick=()=>{dialog.close();selectTask(button.dataset.profileTask)};
 dialog.showModal();
}
